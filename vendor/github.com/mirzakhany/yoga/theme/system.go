package theme

// SystemName is the registry key for the system appearance theme. Selecting it
// applies yoga-dark or yoga-light based on the OS dark-mode preference.
const SystemName = "system"

var (
	selectedName       = "yoga-dark"
	systemResolvedDark bool
	prefersDarkFn      = osPrefersDark
)

// PrefersDark reports whether the OS is set to a dark appearance.
func PrefersDark() bool {
	return prefersDarkFn()
}

// Selected returns the theme name the user chose (e.g. "system", "yoga-dark").
// When "system" is active, Current().Name reflects the resolved yoga variant.
func Selected() string {
	if selectedName == "" {
		return active.Name
	}
	return selectedName
}

// SyncSystem re-applies the yoga variant when Selected() is "system" and the OS
// preference changed since the last apply. Returns true when the live palette
// was updated.
func SyncSystem() bool {
	if selectedName != SystemName {
		return false
	}
	dark := prefersDarkFn()
	if dark == systemResolvedDark {
		return false
	}
	systemResolvedDark = dark
	return applyResolved(systemTarget())
}

func systemTarget() string {
	if prefersDarkFn() {
		return "yoga-dark"
	}
	return "yoga-light"
}

func applyResolved(name string) bool {
	t, ok := registry[name]
	if !ok {
		return false
	}
	styles := active.Styles
	*active = t
	if styles != nil {
		active.Styles = styles
	} else if t.Styles != nil {
		active.Styles = t.Styles
	}
	return true
}

// yogaSystem registers the virtual system theme entry shown in theme pickers.
func yogaSystem() Theme {
	t := yogaDark()
	t.Name = SystemName
	return t
}
