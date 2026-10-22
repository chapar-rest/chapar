package gitx

import (
	"os/exec"
	"syscall"
)

// createNoWindow keeps git from opening a console window: Chapar is a GUI
// app, so a console program it starts would get a window of its own.
const createNoWindow = 0x08000000

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
