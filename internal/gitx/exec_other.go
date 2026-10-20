//go:build !windows

package gitx

import "os/exec"

func hideWindow(*exec.Cmd) {}
