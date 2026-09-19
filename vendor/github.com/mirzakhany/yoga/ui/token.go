package ui

import (
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
)

// Token is a deferred color reference resolved against the active theme at
// paint/layout time, so widgets recolor when theme.Use is called.
type Token int

const (
	TokenUnset Token = iota
	TokenSurface
	TokenChrome
	TokenChromeMuted
	TokenForeground
	TokenForegroundMuted
	TokenForegroundSubtle
	TokenForegroundDisabled
	TokenAccent
	TokenAccentHover
	TokenAccentPressed
	TokenAccentForeground
	TokenBorder
	TokenBorderStrong
	TokenListHover
	TokenListActive
	TokenFocusRing
	TokenSelection
	TokenScrollTrack
	TokenScrollThumb
	TokenScrollThumbHover
	TokenError
	TokenWarning
	TokenSuccess
)

// Resolve looks up t on th. TokenUnset yields a fully transparent color.
func (t Token) Resolve(th *theme.Theme) render.Color {
	if th == nil {
		th = theme.Current()
	}
	switch t {
	case TokenSurface:
		return th.Surface
	case TokenChrome:
		return th.Chrome
	case TokenChromeMuted:
		return th.ChromeMuted
	case TokenForeground:
		return th.Foreground
	case TokenForegroundMuted:
		return th.ForegroundMuted
	case TokenForegroundSubtle:
		return th.ForegroundSubtle
	case TokenForegroundDisabled:
		return th.ForegroundDisabled
	case TokenAccent:
		return th.Accent
	case TokenAccentHover:
		return th.AccentHover
	case TokenAccentPressed:
		return th.AccentPressed
	case TokenAccentForeground:
		return th.AccentForeground
	case TokenBorder:
		return th.Border
	case TokenBorderStrong:
		return th.BorderStrong
	case TokenListHover:
		return th.ListHover
	case TokenListActive:
		return th.ListActive
	case TokenFocusRing:
		return th.FocusRing
	case TokenSelection:
		return th.Selection
	case TokenScrollTrack:
		return th.ScrollTrack
	case TokenScrollThumb:
		return th.ScrollThumb
	case TokenScrollThumbHover:
		return th.ScrollThumbHover
	case TokenError:
		return th.Error
	case TokenWarning:
		return th.Warning
	case TokenSuccess:
		return th.Success
	default:
		return render.Color{}
	}
}
