//go:build !darwin && !windows && !js

package theme

import (
	"os/exec"
	"strings"
)

func osPrefersDark() bool {
	if out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output(); err == nil {
		s := strings.ToLower(strings.TrimSpace(string(out)))
		if strings.Contains(s, "dark") {
			return true
		}
		if strings.Contains(s, "light") {
			return false
		}
	}
	if out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme").Output(); err == nil {
		return strings.Contains(strings.ToLower(string(out)), "dark")
	}
	return false
}
