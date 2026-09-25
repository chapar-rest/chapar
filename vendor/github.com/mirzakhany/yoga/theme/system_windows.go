//go:build windows

package theme

import winreg "golang.org/x/sys/windows/registry"

func osPrefersDark() bool {
	k, err := winreg.OpenKey(winreg.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, winreg.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return false
	}
	return v == 0
}
