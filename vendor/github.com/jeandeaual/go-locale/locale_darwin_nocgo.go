//go:build darwin && !ios && !cgo

package locale

// GetLocale retrieves the IETF BCP 47 language tag set on the system.
func GetLocale() (string, error) {
	return getLocaleCli()
}

// GetLocales retrieves the IETF BCP 47 language tags set on the system.
func GetLocales() ([]string, error) {
	return getLocalesCli()
}
