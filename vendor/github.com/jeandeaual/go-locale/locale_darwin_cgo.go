//go:build darwin && !ios

package locale

// Non-CGO implementation is in locale_darwin_nocgo.go

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation

#include <AppKit/AppKit.h>

const char * preferredLocalization();
const char * preferredLocalizations();
*/
import "C"
import (
	"strings"
)

// GetLocale retrieves the IETF BCP 47 language tag set on the system.
func GetLocale() (string, error) {
	str := C.preferredLocalization()
	if output := C.GoString(str); output != "" {
		return strings.Replace(output, "_", "-", 1), nil
	}

	return getLocaleCli()
}

// GetLocales retrieves the IETF BCP 47 language tags set on the system.
func GetLocales() ([]string, error) {
	str := C.preferredLocalizations()
	if output := C.GoString(str); output != "" {
		r := []string{}
		for _, s := range strings.Split(output, ",") {
			r = append(r, strings.Replace(s, "_", "-", 1))
		}
		return r, nil
	}

	return getLocalesCli()
}
