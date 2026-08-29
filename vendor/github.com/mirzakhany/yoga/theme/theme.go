// Package theme defines the framework's design tokens and a registry of
// prebuilt themes that can be switched at runtime.
//
// Runtime switching model: there is exactly one live *Theme instance
// (the "active" theme) returned by Current(). Every widget holds that pointer.
// Use(name) overwrites the active instance's fields in place, so a switch is
// reflected by all widgets on the very next paint with no rebuild.
//
// Layering: theme imports only render (for Color) and highlight (for the syntax
// ColorClass keys). The components package imports theme, never the reverse.
package theme

import (
	"sort"

	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/render"
)

// Theme is the Yoga design token set shared by all widgets. Every builtin theme
// fills all tokens; custom themes should do the same or rely on normalize().
type Theme struct {
	Name string // unique registry key, e.g. "yoga-dark"
	Dark bool   // true for dark palettes (lets widgets adapt if needed)

	// Yoga semantic color tokens.
	Surface            render.Color // workspace / editor background
	Chrome             render.Color // sidebars, tab bars, menus
	ChromeMuted        render.Color // tracks, gutters, secondary chrome
	Foreground         render.Color // primary text and icons
	ForegroundMuted    render.Color // secondary labels
	ForegroundSubtle   render.Color // tertiary / de-emphasized
	ForegroundDisabled render.Color // disabled controls
	Accent             render.Color // primary accent fill
	AccentHover        render.Color // accent hover
	AccentPressed      render.Color // accent pressed
	AccentForeground   render.Color // text/icons on accent fills
	Border             render.Color // dividers and control outlines
	BorderStrong       render.Color // emphasized borders
	ListHover          render.Color // hovered list/tab/menu row
	ListActive         render.Color // active/pressed/selected row chrome
	FocusRing          render.Color // keyboard focus indicator
	Selection          render.Color // text selection highlight

	ScrollTrack      render.Color // scrollbar track background
	ScrollThumb      render.Color // scrollbar thumb (handle)
	ScrollThumbHover render.Color // thumb while hovered or dragging

	Error   render.Color
	Warning render.Color
	Success render.Color

	// Non-color design tokens.
	Spacing    Spacing
	Radius     Radius
	Stroke     Stroke
	Typography Typography
	Elevation  Elevation
	Metrics    ComponentMetrics

	// Syntax maps highlight classes to colors for the code editor.
	Syntax map[highlight.ColorClass]render.Color

	// Legacy aliases kept for backward compatibility. normalize() keeps these in
	// sync with the Yoga tokens above; prefer the Yoga names in new code.
	Background render.Color
	Panel      render.Color
	PanelAlt   render.Color
	Text       render.Color
	TextDim    render.Color
	AccentText render.Color
	Hover      render.Color
	Active     render.Color

	// Styles is an opaque bag of component specs attached by the ui package
	// (ui.ComponentStyles). Token-based specs react to palette changes automatically.
	Styles any
}

// Clone returns a deep copy of t, including the Syntax map. Use it to replicate
// a palette, tweak tokens, and Register the result under a new name.
func (t Theme) Clone() Theme {
	c := t
	if t.Syntax != nil {
		c.Syntax = make(map[highlight.ColorClass]render.Color, len(t.Syntax))
		for k, v := range t.Syntax {
			c.Syntax[k] = v
		}
	}
	return c
}

// SyntaxColor resolves a token class to a color, defaulting to plain text.
func (t *Theme) SyntaxColor(c highlight.ColorClass) render.Color {
	if col, ok := t.Syntax[c]; ok {
		return col
	}
	return t.Foreground
}

// registry holds all registered themes by name.
var registry = map[string]Theme{}

// active is the single live theme instance every widget reads from. Use()
// mutates it in place so switches are instant and pointer-stable.
var active = &Theme{}

// Current returns the live active theme. The pointer is stable for the lifetime
// of the process; its contents change when Use is called.
func Current() *Theme { return active }

// Register adds or replaces a theme in the registry (keyed by t.Name).
func Register(t Theme) {
	normalize(&t)
	registry[t.Name] = t
}

// Get returns a copy of the named theme.
func Get(name string) (Theme, bool) {
	t, ok := registry[name]
	return t, ok
}

// Names returns the sorted list of registered theme names.
func Names() []string {
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Use switches the active theme to the named one, updating the shared instance
// in place. Returns false if the name is unknown (active is left unchanged).
// Use(SystemName) resolves to yoga-dark or yoga-light from the OS appearance.
func Use(name string) bool {
	if name == SystemName {
		selectedName = SystemName
		systemResolvedDark = PrefersDark()
		return applyResolved(systemTarget())
	}
	t, ok := registry[name]
	if !ok {
		return false
	}
	selectedName = name
	*active = t
	return true
}

func init() {
	for _, t := range builtins() {
		Register(t)
	}
	Use("yoga-dark")
}
