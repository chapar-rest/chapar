package testcli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/internal/prefs"
	"github.com/chapar-rest/chapar/internal/repository"
	"github.com/chapar-rest/chapar/internal/secret"
)

// PassphraseEnv unlocks secret environment values kept with a passphrase.
const PassphraseEnv = "CHAPAR_SECRETS_PASSPHRASE"

// workspace is what a test run reads from a workspace folder.
type workspace struct {
	warn  func(format string, args ...any)
	dir   string
	repo  *repository.FilesystemV2
	index *index
	envs  []*domain.Environment
	cases []*domain.TestCase

	// locked holds the secret values environment left out, by env ID, so
	// saving the env writes them back untouched.
	locked map[string][]domain.KeyValue

	// bundle is set when the cases come from bundle files, not a
	// workspace; there is then no repository to save to.
	bundle         bool
	secretsLeftOut []string
}

// readBundle reads path when it is a test bundle file; ok is false for
// any other file.
func readBundle(path string) (b *domain.TestBundle, ok bool, err error) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	var head struct {
		Kind string `yaml:"kind"`
	}
	if yaml.Unmarshal(data, &head) != nil || head.Kind != domain.KindTestBundle {
		return nil, false, nil
	}
	b = &domain.TestBundle{}
	if err := yaml.Unmarshal(data, b); err != nil {
		return nil, false, fmt.Errorf("%s: %w", path, err)
	}
	return b, true, nil
}

// loadBundles returns the bundles args name. A bundle runs on its own, so
// args are either all bundles or none.
func loadBundles(args []string) ([]*domain.TestBundle, error) {
	var bundles []*domain.TestBundle
	for _, arg := range args {
		b, ok, err := readBundle(arg)
		if err != nil {
			return nil, err
		}
		if ok {
			bundles = append(bundles, b)
		}
	}
	if len(bundles) > 0 && len(bundles) < len(args) {
		return nil, errors.New("bundle files run on their own; give them without workspace test cases")
	}
	return bundles, nil
}

// openBundles builds a workspace from bundle files.
func openBundles(bundles []*domain.TestBundle, warn func(format string, args ...any)) *workspace {
	ws := &workspace{warn: warn, bundle: true, locked: map[string][]domain.KeyValue{}}
	var reqs []*domain.Request
	var colls []*domain.Collection
	for _, b := range bundles {
		reqs = append(reqs, b.Spec.Requests...)
		colls = append(colls, b.Spec.Collections...)
		ws.cases = append(ws.cases, b.Spec.TestCases...)
		if b.Spec.Environment != nil {
			ws.envs = append(ws.envs, b.Spec.Environment)
		}
		ws.secretsLeftOut = append(ws.secretsLeftOut, b.Spec.SecretsLeftOut...)
	}
	ws.index = newIndex(reqs, colls)
	return ws
}

// locate finds the workspace folder: arg can be a directory, or the name
// of a workspace in the app's data folder. Empty means the app's active
// workspace.
func locate(arg string) (dataDir, name string, err error) {
	if arg != "" {
		if info, err := os.Stat(arg); err == nil && info.IsDir() {
			abs, err := filepath.Abs(arg)
			if err != nil {
				return "", "", err
			}
			return filepath.Dir(abs), filepath.Base(abs), nil
		}
	}

	dataDir, name = prefs.GetWorkspacePath(), arg
	if name == "" {
		name = domain.DefaultWorkspaceName
		if aw := prefs.GetAppState().Spec.ActiveWorkspace; aw != nil && aw.Name != "" {
			name = aw.Name
		}
	}
	if info, err := os.Stat(filepath.Join(dataDir, name)); err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("workspace %q not found in %s", name, dataDir)
	}
	return dataDir, name, nil
}

func openWorkspace(arg string, warn func(format string, args ...any)) (*workspace, error) {
	dataDir, name, err := locate(arg)
	if err != nil {
		return nil, err
	}
	repo, err := repository.NewFilesystemV2(dataDir, name)
	if err != nil {
		return nil, err
	}
	ws := &workspace{warn: warn, dir: filepath.Join(dataDir, name), repo: repo, locked: map[string][]domain.KeyValue{}}

	// A malformed file is left out, as the app does; the cases that need
	// it then fail validation with "request not found".
	colls, err := repo.LoadCollections()
	if err := ws.skipped(err); err != nil {
		return nil, err
	}
	reqs, err := repo.LoadRequests()
	if err := ws.skipped(err); err != nil {
		return nil, err
	}
	ws.envs, err = repo.LoadEnvironments()
	if err := ws.skipped(err); err != nil {
		return nil, err
	}
	ws.cases, err = repo.LoadTestCases()
	if err := ws.skipped(err); err != nil {
		return nil, err
	}
	sort.Slice(ws.cases, func(i, j int) bool { return ws.cases[i].GetName() < ws.cases[j].GetName() })
	ws.index = newIndex(reqs, colls)
	return ws, nil
}

func (ws *workspace) skipped(err error) error {
	if err == nil {
		return nil
	}
	if files, ok := repository.SkippedFiles(err); ok {
		for _, f := range files {
			ws.warn("skipped %s: %v", f.Path, f.Err)
		}
		return nil
	}
	return err
}

// environment returns the environment named or with the ID arg, with its
// secret values opened. Values that stay locked are left out, and named
// in the warning.
func (ws *workspace) environment(arg string) (env *domain.Environment, warning string, err error) {
	env, err = findEnv(ws.envs, arg)
	if err != nil || !hasLocked(env) || ws.repo == nil {
		return env, "", err
	}

	// Only now touch the key store: on macOS reading the Keychain can ask
	// the user for permission.
	configDir, err := prefs.GetConfigDir()
	if err != nil {
		return nil, "", err
	}
	m := secret.New(secret.NewOSStore(), filepath.Join(configDir, secret.MetaFileName))
	if pass := os.Getenv(PassphraseEnv); pass != "" && !m.Unlocked() {
		if err := m.UnlockWithPassphrase(pass); err != nil {
			return nil, "", fmt.Errorf("unlock secrets with %s: %w", PassphraseEnv, err)
		}
	}
	ws.repo.SetSecrets(m)
	envs, err := ws.repo.LoadEnvironments()
	if err := ws.skipped(err); err != nil {
		return nil, "", err
	}
	if env, err = findEnv(envs, env.ID()); err != nil {
		return nil, "", err
	}

	var locked []string
	values := env.Spec.Values[:0:0]
	for _, kv := range env.Spec.Values {
		if kv.Locked {
			locked = append(locked, kv.Key)
			ws.locked[env.ID()] = append(ws.locked[env.ID()], kv)
			continue
		}
		values = append(values, kv)
	}
	env.Spec.Values = values
	if len(locked) > 0 {
		warning = fmt.Sprintf("environment %s: secret values %s are locked and left out; unlock them in the app or set %s",
			env.GetName(), strings.Join(locked, ", "), PassphraseEnv)
	}
	return env, warning, nil
}

// saveEnv writes env back, with the locked values environment left out.
func (ws *workspace) saveEnv(env *domain.Environment) error {
	for _, kv := range ws.locked[env.ID()] {
		if !hasKey(env, kv.Key) {
			env.Spec.Values = append(env.Spec.Values, kv)
		}
	}
	return ws.repo.UpdateEnvironment(env)
}

func hasKey(env *domain.Environment, key string) bool {
	for _, kv := range env.Spec.Values {
		if kv.Key == key {
			return true
		}
	}
	return false
}

func findEnv(envs []*domain.Environment, arg string) (*domain.Environment, error) {
	var names []string
	for _, e := range envs {
		if e.ID() == arg || e.GetName() == arg {
			return e, nil
		}
		names = append(names, e.GetName())
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("environment %q not found; the workspace has none", arg)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("environment %q not found; have: %s", arg, strings.Join(names, ", "))
}

func hasLocked(env *domain.Environment) bool {
	for _, kv := range env.Spec.Values {
		if kv.Locked {
			return true
		}
	}
	return false
}

// selectCases picks the cases to run. Each arg is a case name or ID, a
// test case file, or a folder of them; no args means every case in the
// workspace. tags keeps only cases with one of them.
func (ws *workspace) selectCases(args, tags []string) ([]*domain.TestCase, error) {
	var out []*domain.TestCase
	if len(args) == 0 {
		out = ws.cases
	}
	for _, arg := range args {
		cases, err := ws.find(arg)
		if err != nil {
			return nil, err
		}
		out = append(out, cases...)
	}

	if len(tags) > 0 {
		want := map[string]bool{}
		for _, t := range tags {
			want[t] = true
		}
		var tagged []*domain.TestCase
		for _, tc := range out {
			for _, t := range tc.Spec.Tags {
				if want[t] {
					tagged = append(tagged, tc)
					break
				}
			}
		}
		out = tagged
	}
	if len(out) == 0 {
		return nil, errors.New("no test cases to run")
	}
	return out, nil
}

func (ws *workspace) find(arg string) ([]*domain.TestCase, error) {
	for _, tc := range ws.cases {
		if tc.GetName() == arg || tc.ID() == arg {
			return []*domain.TestCase{tc}, nil
		}
	}

	info, err := os.Stat(arg)
	if err != nil {
		return nil, fmt.Errorf("no test case named %q in %s, and no such file", arg, ws.dir)
	}
	files := []string{arg}
	if info.IsDir() {
		yaml, _ := filepath.Glob(filepath.Join(arg, "*.yaml"))
		yml, _ := filepath.Glob(filepath.Join(arg, "*.yml"))
		files = append(yaml, yml...)
		sort.Strings(files)
	}
	var out []*domain.TestCase
	for _, f := range files {
		tc, err := repository.LoadFromYaml[domain.TestCase](f)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		if tc.Kind != domain.KindTestCase {
			if info.IsDir() {
				continue
			}
			return nil, fmt.Errorf("%s is a %q, not a test case", f, tc.Kind)
		}
		if tc.GetName() == "" {
			tc.SetName(strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)))
		}
		out = append(out, tc)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no test cases in %s", arg)
	}
	return out, nil
}

// index is the requests of a workspace, for resolving steps.
type index struct {
	byID  map[string]*domain.Request
	all   []*domain.Request
	colls map[string]*domain.Collection
}

func newIndex(reqs []*domain.Request, colls []*domain.Collection) *index {
	ix := &index{byID: map[string]*domain.Request{}, colls: map[string]*domain.Collection{}}
	add := func(r *domain.Request) {
		ix.byID[r.ID()] = r
		ix.all = append(ix.all, r)
	}
	for _, r := range reqs {
		add(r)
	}
	for _, c := range colls {
		ix.colls[c.ID()] = c
		for _, r := range c.Spec.Requests {
			r.CollectionID, r.CollectionName = c.ID(), c.GetName()
			add(r)
		}
	}
	return ix
}

func (ix *index) RequestByID(id string) *domain.Request { return ix.byID[id] }

func (ix *index) AllRequests() []*domain.Request { return ix.all }

func (ix *index) CollectionByID(id string) *domain.Collection { return ix.colls[id] }
