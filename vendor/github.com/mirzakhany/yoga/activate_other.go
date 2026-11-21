//go:build !darwin || js

package yoga

// ActivateInstance brings another instance of this app to the front on
// macOS; elsewhere it does nothing and reports false.
func ActivateInstance(pid int) bool { return false }
