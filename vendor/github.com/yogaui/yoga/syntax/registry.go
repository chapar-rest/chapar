package syntax

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Factory builds a fresh highlighter; each consumer gets its own.
type Factory func() Highlighter

var (
	regMu sync.RWMutex
	// byExt maps a lowercase extension (with its dot) or whole file name to
	// the factory an app chose with Register. It wins over the defaults.
	byExt = map[string]Factory{}
	// defExt and defFile are the engines' defaults, by extension and by file
	// name; a file name wins over the extension.
	defExt  = map[string]Factory{}
	defFile = map[string]Factory{}
	// byName holds highlighters by language name.
	byName = map[string]Factory{}
)

// Register maps a file extension (with its leading dot, case-insensitive) to
// a highlighter. It overrides the default for that extension, so an app can
// add languages or rebind extensions from its own configuration. A whole file
// name such as "Makefile" works too, for files without an extension; it wins
// over the extension.
func Register(ext string, f Factory) {
	regMu.Lock()
	defer regMu.Unlock()
	byExt[strings.ToLower(ext)] = f
}

// Unregister removes an extension added with Register; the default, if any,
// applies again.
func Unregister(ext string) {
	regMu.Lock()
	defer regMu.Unlock()
	delete(byExt, strings.ToLower(ext))
}

// RegisterDefaults is for highlighting engines: it adds their highlighters by
// extension (".go"), by whole file name ("makefile") and by language name
// ("go"). Keys are case-insensitive. Register still overrides them.
func RegisterDefaults(exts, files, names map[string]Factory) {
	regMu.Lock()
	defer regMu.Unlock()
	for k, f := range exts {
		defExt[strings.ToLower(k)] = f
	}
	for k, f := range files {
		defFile[strings.ToLower(k)] = f
	}
	for k, f := range names {
		byName[strings.ToLower(k)] = f
	}
}

// ForPath returns a highlighter for a file path, chosen by its file name or
// extension: Register's choice first, then the engines' defaults. Unknown
// types get Noop.
func ForPath(path string) Highlighter {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))
	regMu.RLock()
	f, ok := byExt[base]
	if !ok {
		f, ok = byExt[ext]
	}
	if !ok {
		f, ok = defFile[base]
	}
	if !ok {
		f, ok = defExt[ext]
	}
	regMu.RUnlock()
	if !ok {
		return Noop{}
	}
	return f()
}

// ByName returns the highlighter for a language name such as "go",
// "typescript", "tsx" or "yaml".
func ByName(name string) (Factory, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	f, ok := byName[strings.ToLower(name)]
	return f, ok
}

// Names lists the registered language names, sorted.
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
