//go:build !nogpu && !js && !darwin

package yoga

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// clipboardImage reads the clipboard's image as PNG. On Linux it asks
// wl-paste (Wayland) or xclip (X11), whichever is installed; elsewhere
// it finds none.
func clipboardImage() ([]byte, bool) {
	if runtime.GOOS != "linux" && runtime.GOOS != "freebsd" {
		return nil, false
	}
	cmds := [][]string{{"xclip", "-selection", "clipboard", "-t", "image/png", "-o"}}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		cmds = append([][]string{{"wl-paste", "--no-newline", "--type", "image/png"}}, cmds...)
	}
	for _, c := range cmds {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		out, err := exec.CommandContext(ctx, c[0], c[1:]...).Output()
		cancel()
		if err == nil && isPNG(out) {
			return out, true
		}
	}
	return nil, false
}

// isPNG reports whether data starts with the PNG signature; the tools
// print other things (an error, the text on the clipboard) when there is
// no image.
func isPNG(data []byte) bool {
	return len(data) > 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n"
}
