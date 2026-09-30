// Command chapar-cli runs Chapar test cases from the command line, without
// the app: in CI, or anywhere a window cannot open. It builds without cgo
// and without the UI, so one static binary runs on any machine.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/chapar-rest/chapar/internal/testcli"
	"github.com/chapar-rest/chapar/version"
)

const usage = `Usage: chapar-cli <command> [arguments]

Commands:
  test      run test cases; see chapar-cli test -h
  version   print the version
  help      show this help
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return testcli.ExitUsage
	}
	switch args[0] {
	case "test":
		return testcli.Main(args[1:], stdout, stderr)
	case "version", "-version", "--version":
		_, _ = fmt.Fprintln(stdout, "chapar-cli "+version.GetAppVersion())
		return testcli.ExitPassed
	case "help", "-h", "-help", "--help":
		_, _ = fmt.Fprint(stdout, usage)
		return testcli.ExitPassed
	}
	_, _ = fmt.Fprintf(stderr, "chapar-cli: unknown command %q\n\n%s", args[0], usage)
	return testcli.ExitUsage
}
