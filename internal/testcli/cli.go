// Package testcli is the `chapar-cli test` command: it runs a workspace's test
// cases without the UI, for CI. See docs/testcases-design.md.
package testcli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/sender"
	"github.com/chapar-rest/chapar/internal/testrun"
)

// Exit codes.
const (
	ExitPassed    = 0
	ExitFailed    = 1 // a test case did not pass
	ExitUsage     = 2 // bad flags, workspace or test case files
	ExitCancelled = 130
)

const usage = `Usage: chapar-cli test [flags] [case|file|folder ...]
       chapar-cli test [flags] bundle.yaml ...

Runs test cases and exits 0 when all pass, 1 when any does not, 2 when
the flags, workspace or test case files are wrong.

Arguments name test cases in the workspace (by name or ID), or test case
files and folders of them. With none, every test case in the workspace runs.

A bundle, exported from the app, holds test cases with the requests they
send and optionally an environment, and runs without a workspace.

Environment values can be set over the chosen environment, in this order:
--env-file, then --os-env, then --var. For example, in CI:

  API_TOKEN=... chapar-cli test --os-env API_ --var base=https://staging smoke.yaml

sets {{TOKEN}} from API_TOKEN and {{base}} from the command line.

Flags:
`

type options struct {
	workspace string
	env       string
	tags      list
	reports   list
	bail      bool
	scripts   bool
	noColor   bool
	overrides overrides
}

// list is a flag that can be given more than once, or as a comma list.
type list []string

func (l *list) String() string { return strings.Join(*l, ",") }

func (l *list) Set(v string) error {
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			*l = append(*l, s)
		}
	}
	return nil
}

// Main runs the command with args (without "test") and returns its exit code.
func Main(args []string, stdout, stderr io.Writer) int {
	var o options
	fs := flag.NewFlagSet("chapar-cli test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.workspace, "workspace", "", "workspace folder, or name of a workspace in the app (default: the app's active one)")
	fs.StringVar(&o.env, "env", "", "environment to run with, by name or ID (default: none)")
	fs.Var(&o.tags, "tag", "run only test cases with this tag; repeat or comma-separate for any of several")
	fs.Var(&o.reports, "report", "write a report, as format=path; formats: junit, json")
	fs.StringVar(&o.overrides.file, "env-file", "", "set environment values from a file: a chapar environment, or KEY=VALUE lines")
	fs.StringVar(&o.overrides.osEnv, "os-env", "", "set environment values from OS variables with this prefix, which is removed")
	fs.Var((*list)(&o.overrides.vars), "var", "set an environment value, as key=value; repeat for more")
	fs.BoolVar(&o.bail, "bail", false, "stop after the first test case that does not pass")
	fs.BoolVar(&o.scripts, "scripts", false, "run request scripts even when scripting is off in the app's settings")
	fs.BoolVar(&o.noColor, "no-color", false, "print without colors")
	fs.Usage = func() {
		_, _ = fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
	}

	rest, err := parseInterleaved(fs, args)
	if errors.Is(err, flag.ErrHelp) {
		return ExitPassed
	}
	if err != nil {
		return ExitUsage
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(ctx, o, rest, stdout, stderr)
}

// parseInterleaved parses flags given before, between and after the
// arguments, which the flag package alone stops at.
func parseInterleaved(fs *flag.FlagSet, args []string) ([]string, error) {
	var rest []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return rest, nil
		}
		rest = append(rest, args[0])
		args = args[1:]
	}
}

func run(ctx context.Context, o options, args []string, stdout, stderr io.Writer) int {
	fail := func(format string, a ...any) int {
		_, _ = fmt.Fprintf(stderr, "chapar-cli test: "+format+"\n", a...)
		return ExitUsage
	}
	warn := func(format string, a ...any) {
		_, _ = fmt.Fprintf(stderr, "warning: "+format+"\n", a...)
	}

	var reports []report
	for _, r := range o.reports {
		format, path, ok := strings.Cut(r, "=")
		if !ok || path == "" || (format != "junit" && format != "json") {
			return fail("--report %q: want junit=path or json=path", r)
		}
		reports = append(reports, report{format, path})
	}

	bundles, err := loadBundles(args)
	if err != nil {
		return fail("%v", err)
	}
	var ws *workspace
	if len(bundles) > 0 {
		if o.workspace != "" {
			return fail("--workspace does not apply to a bundle, which holds its own requests")
		}
		ws, args = openBundles(bundles, warn), nil
	} else if ws, err = openWorkspace(o.workspace, warn); err != nil {
		return fail("%v", err)
	}
	cases, err := ws.selectCases(args, o.tags)
	if err != nil {
		return fail("%v", err)
	}

	invalid := false
	for _, tc := range cases {
		for _, p := range testrun.Validate(tc, testrun.Finder(ws.index)) {
			_, _ = fmt.Fprintf(stderr, "%s: %s\n", tc.GetName(), p.Error())
			invalid = true
		}
	}
	if invalid {
		return fail("fix the test cases above and run again")
	}

	var base *domain.Environment
	switch {
	case o.env != "":
		env, warning, err := ws.environment(o.env)
		if err != nil {
			return fail("%v", err)
		}
		if warning != "" {
			warn("%s", warning)
		}
		base = env
	case ws.bundle && len(ws.envs) == 1:
		base = ws.envs[0]
	case ws.bundle && len(ws.envs) > 1:
		return fail("the bundles hold %d environments; pick one with --env", len(ws.envs))
	}
	env, set, err := o.overrides.apply(base, os.Environ())
	if err != nil {
		return fail("%v", err)
	}
	for _, k := range ws.secretsLeftOut {
		if !set[k] {
			warn("secret value %s was left out of the bundle; set it with --os-env, --env-file or --var", k)
		}
	}
	envName := ""
	var runOpts testrun.Options
	if env != nil {
		runOpts.Env, envName = env, env.GetName()
	}

	// persistEnv writes back only what the requests changed, never the
	// values given on the command line.
	var saveEnv func(*domain.Environment) error
	if !ws.bundle && base != nil {
		saveEnv = func(saved *domain.Environment) error {
			changed, removed := requestChanges(env, saved)
			orig := *base
			orig.Spec.Values = append([]domain.KeyValue(nil), base.Spec.Values...)
			for k, v := range changed {
				orig.SetKey(k, v)
			}
			for _, k := range removed {
				orig.UnsetKey(k)
			}
			return ws.saveEnv(&orig)
		}
	}
	for _, tc := range cases {
		if tc.Spec.Options.PersistEnv && saveEnv == nil {
			warn("%s saves environment changes, which needs a workspace environment (--env); they are kept for the run only", tc.GetName())
		}
	}

	cfg := prefs.GetGlobalConfig().Spec.Scripting
	scriptingOn := cfg.Enabled || o.scripts
	var scripts sender.Scripts
	if scriptingOn {
		scripts = &lazyScripts{cfg: cfg}
	}
	runner := testrun.New(testrun.Config{
		NewSender: testrun.NewSender(ws.index.RequestByID, ws.index.CollectionByID, scripts, func() bool { return scriptingOn }),
		Requests:  ws.index,
		SaveEnv:   saveEnv,
	})

	out := &console{w: stdout, color: !o.noColor && colorful(stdout)}
	runOpts.OnEvent = out.event
	start := time.Now()
	var runs []*testrun.Run
	for _, tc := range cases {
		if ctx.Err() != nil {
			break
		}
		out.caseStarted(tc, envName)
		r := runner.Run(ctx, tc, runOpts)
		out.caseFinished(r)
		runs = append(runs, r)
		if o.bail && r.Status != testrun.StatusPassed {
			break
		}
	}
	took := time.Since(start)
	out.summary(runs, took)

	code := ExitPassed
	for _, r := range runs {
		if r.Status != testrun.StatusPassed {
			code = ExitFailed
		}
	}
	if len(runs) < len(cases) && !o.bail {
		code = ExitFailed
	}
	for _, rep := range reports {
		if err := rep.write(runs, took); err != nil {
			_, _ = fmt.Fprintf(stderr, "chapar-cli test: write %s report: %v\n", rep.format, err)
			code = ExitUsage
		}
	}
	if ctx.Err() != nil {
		return ExitCancelled
	}
	return code
}

// colorful reports whether w is a terminal that should get colors.
func colorful(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
