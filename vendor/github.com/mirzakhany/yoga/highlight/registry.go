package highlight

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Factory builds a fresh highlighter; each editor gets its own.
type Factory func() Highlighter

var (
	regMu sync.RWMutex
	// byExt maps a lowercase extension, with its dot, to the factory chosen
	// for it by Register. It is consulted before the built-in table.
	byExt = map[string]Factory{}
	// byName holds the built-in grammars by language name.
	byName = map[string]Factory{}
)

// Register maps a file extension (with its leading dot, case-insensitive) to
// a highlighter. It overrides the built-in choice for that extension, so an
// app can add languages or rebind extensions from its own configuration. A
// whole file name such as "Makefile" works too, for files without an
// extension; it wins over the extension.
func Register(ext string, f Factory) {
	regMu.Lock()
	defer regMu.Unlock()
	byExt[strings.ToLower(ext)] = f
}

// Unregister removes an extension added with Register; the built-in choice,
// if any, applies again.
func Unregister(ext string) {
	regMu.Lock()
	defer regMu.Unlock()
	delete(byExt, strings.ToLower(ext))
}

// ByName returns the built-in highlighter for a language name such as "go",
// "typescript", "tsx" or "yaml".
func ByName(name string) (Factory, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	f, ok := byName[strings.ToLower(name)]
	return f, ok
}

// Names lists the built-in language names, sorted.
func Names() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(byName))
	for n := range byName {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func registerName(name string, f Factory) {
	regMu.Lock()
	defer regMu.Unlock()
	byName[name] = f
}

// registered returns the highlighter Register chose for path's file name
// or extension, if any.
func registered(path string) (Highlighter, bool) {
	regMu.RLock()
	f, ok := byExt[strings.ToLower(filepath.Base(path))]
	if !ok {
		f, ok = byExt[strings.ToLower(filepath.Ext(path))]
	}
	regMu.RUnlock()
	if !ok {
		return nil, false
	}
	return f(), true
}
