package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chapar-rest/chapar/internal/testcli"
	"github.com/chapar-rest/chapar/version"
)

func TestRun(t *testing.T) {
	tests := []struct {
		args   []string
		code   int
		stdout string
		stderr string
	}{
		{nil, testcli.ExitUsage, "", "Usage: chapar-cli"},
		{[]string{"help"}, testcli.ExitPassed, "Usage: chapar-cli", ""},
		{[]string{"--version"}, testcli.ExitPassed, "chapar-cli " + version.GetAppVersion(), ""},
		{[]string{"nope"}, testcli.ExitUsage, "", `unknown command "nope"`},
		// test hands its arguments to the test command.
		{[]string{"test", "-h"}, testcli.ExitPassed, "", "Usage: chapar-cli test"},
		{[]string{"test", "--nope"}, testcli.ExitUsage, "", "flag provided but not defined"},
	}
	for _, tt := range tests {
		var out, errb bytes.Buffer
		code := run(tt.args, &out, &errb)
		if code != tt.code || !strings.Contains(out.String(), tt.stdout) || !strings.Contains(errb.String(), tt.stderr) {
			t.Errorf("run(%q) = %d\nstdout: %s\nstderr: %s", tt.args, code, out.String(), errb.String())
		}
	}
}
