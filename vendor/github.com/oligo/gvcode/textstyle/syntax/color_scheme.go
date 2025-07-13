package syntax

import (
	"slices"
	"strings"

	"github.com/oligo/gvcode/color"
)

// StyleScope is a TextMate style scope notation, eg, 'keyword.control.if',
// 'entity.name.function'
type StyleScope string

// Parent returns the parent of scope s. If s is invalid, it returns an invalid
// empty scope.
func (s StyleScope) Parent() StyleScope {
	if !s.IsValid() {
		return ""
	}

	idx := strings.LastIndex(string(s), ".")
	if idx <= 0 {
		return ""
	}

	return StyleScope(s[:idx])
}

// IsValid checks if s has a valid notation.
func (s StyleScope) IsValid() bool {
	if s == "" {
		return false
	}

	lastIdx := len(s) - 1
	for {
		idx := strings.LastIndex(string(s), ".")
		if idx < 0 {
			break
		}

		if idx == 0 || idx == len(s)-1 {
			return false
		}

		if lastIdx-idx <= 1 {
			return false
		}

		lastIdx = idx
		s = s[:idx]
	}

	return true
}

// IsChild checks if other is a sub scope of s.
func (s StyleScope) IsChild(other StyleScope) bool {
	if !s.IsValid() || !other.IsValid() {
		return false
	}

	return other.Parent() == s
}

const (
	defaultScope = StyleScope("_default_")
)

// ColorScheme defines the token types and their styles used for syntax highlighting.
type ColorScheme struct {
	// Name is the name of the color scheme.
	Name string
	color.ColorPalette
	// scopes are registered style scopes for the color scheme.
	// It can be mapped to captures for Tree-Sitter, and TokenType of Chroma.
	scopes []StyleScope

	// styles maps style scope index to non-packed scope style.
	styles map[int]*scopeStyleRaw
}

type scopeStyleRaw struct {
	textStyle TextStyle
	fg, bg    int
}

func (cs *ColorScheme) addScope(scope StyleScope) int {
	if !scope.IsValid() {
		panic("invalid style scope: " + scope)
	}

	if idx := slices.Index(cs.scopes, scope); idx >= 0 {
		return idx
	}

	cs.scopes = append(cs.scopes, scope)
	return len(cs.scopes) - 1
}

func (cs *ColorScheme) getTokenStyle(scope StyleScope) (*scopeStyleRaw, int) {
	idx := slices.Index(cs.scopes, scope)
	if idx < 0 {
		return nil, idx
	}

	if style, exists := cs.styles[idx]; exists {
		return style, idx
	} else {
		return nil, idx
	}
}

func (cs *ColorScheme) AddStyle(scope StyleScope, textStyle TextStyle, fg, bg color.Color) {
	if !slices.Contains(cs.scopes, defaultScope) {
		cs.addStyle(defaultScope, 0, cs.Foreground, color.Color{})
	}

	cs.addStyle(scope, textStyle, fg, bg)
}

func (cs *ColorScheme) addStyle(scope StyleScope, textStyle TextStyle, fg, bg color.Color) {
	tokenTypeID := cs.addScope(scope)
	fgID := cs.AddColor(fg)
	bgID := cs.AddColor(bg)

	if cs.styles == nil {
		cs.styles = make(map[int]*scopeStyleRaw)
	}

	cs.styles[tokenTypeID] = &scopeStyleRaw{
		textStyle: textStyle,
		fg:        fgID,
		bg:        bgID,
	}
}

func (cs *ColorScheme) GetStyleByID(scopeID int) StyleMeta {
	style, exists := cs.styles[scopeID]
	if !exists || style == nil {
		return StyleMeta(0)
	}

	return packTokenStyle(scopeID, style.fg, style.bg, style.textStyle)
}

// GetTokenStyle finds a proper StyleMeta for the requested scope.
// When the scope has no registered style, search upwards using
// the parent scope. If everything has tried but still failed, it
// returns an empty style.
func (cs *ColorScheme) GetTokenStyle(scope StyleScope) StyleMeta {
	var style *scopeStyleRaw
	var scopeID int = -1
	for {
		if !scope.IsValid() {
			break
		}

		style, scopeID = cs.getTokenStyle(scope)
		if scopeID < 0 {
			scope = scope.Parent()
			continue
		}
		if style != nil {
			break
		}
	}

	if scopeID < 0 || style == nil {
		style, scopeID := cs.getTokenStyle(defaultScope)
		return packTokenStyle(scopeID, style.fg, style.bg, style.textStyle)
	}

	return packTokenStyle(scopeID, style.fg, style.bg, style.textStyle)
}

// Scopes returns all the registered style scopes.
func (cs *ColorScheme) Scopes() []StyleScope {
	return cs.scopes
}
