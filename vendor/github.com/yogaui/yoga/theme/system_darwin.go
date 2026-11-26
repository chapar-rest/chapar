//go:build darwin

package theme

import (
	"os/exec"
	"strings"
)

func osPrefersDark() bool {
	out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "Dark"
}

func osPrefersReducedMotion() bool {
	out, err := exec.Command("defaults", "read", "com.apple.universalaccess", "reduceMotion").Output()
	return err == nil && strings.TrimSpace(string(out)) == "1"
}
