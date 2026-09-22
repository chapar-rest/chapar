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

	// LightSibling and DarkSibling name the other half of this theme's family.
	// UseSystem follows them to track the OS appearance without leaving the
	// palette the user picked. Both default to this theme's own name, so a
	// family of one still resolves.
	LightSibling string
	DarkSibling  string

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
	Border             render.Color // dividers and decorative separators
	BorderStrong       render.Color // emphasized borders
	BorderControl      render.Color // input/checkbox outlines; holds 3:1 on Surface
	ListHover          render.Color // hovered list/tab/menu row
	ListActive         render.Color // active/pressed/selected row chrome
	FocusRing          render.Color // keyboard focus indicator, on ordinary surfaces
	FocusRingInverse   render.Color // focus indicator drawn over an accent-filled control
	Selection          render.Color // text selection highlight (focused)
	SelectionInactive  render.Color // selection in an unfocused editor
	Scrim              render.Color // modal backdrop wash, already carrying its alpha
	Link               render.Color // hyperlink text; holds 4.5:1 on Surface and Chrome
	LinkHover          render.Color // hyperlink under the pointer
	LinkVisited        render.Color // followed hyperlink

	ScrollTrack      render.Color // scrollbar track background
	ScrollThumb      render.Color // scrollbar thumb (handle)
	ScrollThumbHover render.Color // thumb while hovered or dragging

	// Status hues. The base token is the saturated brand color used for fills,
	// icons, and bar charts. *Foreground is the readable text variant (4.5:1 on
	// Surface and on the matching *Surface fill); *Surface is the subtle tinted
	// background for badges, alerts, and inline callouts. Text must use the
	// *Foreground token: on light palettes an amber Warning cannot reach 4.5:1.
	Error   render.Color
	Warning render.Color
	Success render.Color
	Info    render.Color

	ErrorForeground   render.Color
	WarningForeground render.Color
	SuccessForeground render.Color
	InfoForeground    render.Color

	ErrorSurface   render.Color
	WarningSurface render.Color
	SuccessSurface render.Color
	InfoSurface    render.Color

	// Editor tokens. Widgets must not invent these: a hardcoded highlight that
	// works on a dark palette disappears on a light one.
	CurrentLine       render.Color // active line wash, carries its own alpha
	IndentGuide       render.Color // indentation rule
	Caret             render.Color // text cursor
	SearchMatch       render.Color // find result
	SearchMatchActive render.Color // the focused find result
	BracketMatch      render.Color // matching bracket pair

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

// FocusRingOn returns the focus indicator to stroke a control whose own fill is
// the given color. The ring is painted on the control's edge, so it sits
// against that fill as well as the surface behind it; when the normal ring
// cannot clear the fill — a primary button, a filled slider thumb — this
// returns the inverse ring instead. Widgets already know their fill, so they
// should always route through this rather than reading FocusRing directly.
func (t *Theme) FocusRingOn(fill render.Color) render.Color {
	if fill.A == 0 {
		return t.FocusRing
	}
	if ContrastRatio(t.FocusRing, fill) >= ContrastNonText {
		return t.FocusRing
	}
	if ContrastRatio(t.FocusRingInverse, fill) >= ContrastNonText {
		return t.FocusRingInverse
	}
	// Mid-tone fills clear neither themed ring. Fall back to the extreme that
	// the fill is furthest from, so focus stays visible on any control.
	if Luminance(fill) < 0.18 {
		return white
	}
	return black
}

// OnColor returns a readable foreground for an arbitrary fill. Use it where a
// widget paints text or an icon on a color that is not one of the surfaces —
// a status fill, a close button's red hover — so the label does not have to
// assume which way that fill leans.
func (t *Theme) OnColor(bg render.Color) render.Color {
	best, bestRatio := t.Foreground, ContrastRatio(t.Foreground, bg)
	for _, c := range []render.Color{t.AccentForeground, white, black} {
		if r := ContrastRatio(c, bg); r > bestRatio {
			best, bestRatio = c, r
		}
	}
	return best
}

// SyntaxColor resolves a token class to a color, defaulting to plain text.
func (t *Theme) SyntaxColor(c highlight.ColorClass) render.Color {
	if col, ok := t.Syntax[c]; ok {
		return col
	}
	switch c {
	case highlight.ClassError:
		return t.Error
	case highlight.ClassWarning:
		return t.Warning
	case highlight.ClassSuccess:
		return t.Success
	case highlight.ClassMuted:
		return t.ForegroundMuted
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
