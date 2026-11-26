package ui

import "github.com/yogaui/yoga/theme"

// ComponentStyles is the set of default Specs shipped widgets use. Override
// them on a cloned theme via Theme.Styles (type ComponentStyles) so a replica
// can change control appearance as well as palette.
type ComponentStyles struct {
	ButtonPrimary    Spec
	ButtonSecondary  Spec
	ButtonSubtle     Spec
	ButtonGhost      Spec
	ButtonGhostHover Spec
	TextField        Spec
	Select           Spec
	Checkbox         Spec
	Switch           Spec
}

// DefaultStyles returns token-based control specs built on the default
// radius and stroke ramps. They recolor with theme.Use.
func DefaultStyles() ComponentStyles {
	return stylesFor(theme.DefaultRadius(), theme.DefaultStroke())
}

// StylesFor returns the default control specs shaped by th's own radius and
// stroke ramps, so a theme with squarer corners or heavier strokes (the
// high-contrast themes) gets them on buttons, fields, and toggles too.
func StylesFor(th *theme.Theme) ComponentStyles {
	if th == nil {
		return DefaultStyles()
	}
	return stylesFor(th.Radius, th.Stroke)
}

func stylesFor(radius theme.Radius, strokes theme.Stroke) ComponentStyles {
	r := float32(radius.Medium)
	stroke := float32(strokes.Thin)
	return ComponentStyles{
		ButtonPrimary: Background(TokenAccent).
			TextColor(TokenAccentForeground).
			Radius(r).
			Cursor(CursorPointer).
			When(Hovered, Background(TokenAccentHover)).
			When(Pressed, Background(TokenAccentPressed).Scale(0.96, 0.96)).
			When(Disabled, Background(TokenChromeMuted).TextColor(TokenForegroundDisabled)),
		ButtonSecondary: Background(TokenChromeMuted).
			TextColor(TokenForeground).
			Radius(r).
			Cursor(CursorPointer).
			Border(TokenBorderControl, stroke).
			When(Hovered, Background(TokenListHover)).
			When(Pressed, Background(TokenListActive).Border(TokenBorderStrong, stroke)).
			When(Disabled, Background(TokenChromeMuted).TextColor(TokenForegroundDisabled)),
		ButtonSubtle: Spec{}.TextColor(TokenForeground).
			Radius(r).
			Cursor(CursorPointer).
			When(Hovered, Background(TokenListHover)).
			When(Pressed, Background(TokenListActive)).
			When(Disabled, Spec{}.TextColor(TokenForegroundDisabled)),
		ButtonGhost: Spec{}.TextColor(TokenForegroundMuted).
			Cursor(CursorPointer).
			When(Hovered, Spec{}.TextColor(TokenForeground)).
			When(Pressed, Spec{}.TextColor(TokenForeground)).
			When(Disabled, Spec{}.TextColor(TokenForegroundDisabled)),
		ButtonGhostHover: Spec{}.TextColor(TokenForegroundMuted).
			Radius(r).
			Cursor(CursorPointer).
			When(Hovered, Background(TokenListHover).TextColor(TokenForeground)).
			When(Pressed, Background(TokenListActive).TextColor(TokenForeground)).
			When(Disabled, Spec{}.TextColor(TokenForegroundDisabled)),
		TextField: Background(TokenChrome).
			TextColor(TokenForeground).
			Radius(r).
			Border(TokenBorderControl, stroke).
			When(Focused, Spec{}.Border(TokenFocusRing, stroke)).
			When(Disabled, Background(TokenChromeMuted).TextColor(TokenForegroundDisabled)),
		// Select shares TextField's fill: both are fields, and a select next to
		// a text input in a form row should read as the same kind of control.
		Select: Background(TokenChrome).
			TextColor(TokenForeground).
			Radius(r).
			Border(TokenBorderControl, stroke).
			When(Hovered, Background(TokenListHover)).
			When(FocusVisible, Spec{}.Border(TokenFocusRing, stroke)).
			When(Disabled, Background(TokenChromeMuted).TextColor(TokenForegroundDisabled)),
		Checkbox: Background(TokenChrome).
			TextColor(TokenForeground).
			Radius(float32(radius.Small)).
			Border(TokenBorderControl, stroke).
			When(Hovered, Background(TokenListHover)).
			When(Pressed, Background(TokenAccent)).
			When(Disabled, Background(TokenChromeMuted).TextColor(TokenForegroundDisabled)),
		Switch: Background(TokenSwitchTrackOff).
			Radius(float32(radius.Large)).
			Border(TokenBorderControl, stroke).
			When(Hovered, Background(TokenListHover)).
			When(Pressed, Background(TokenAccent)).
			When(Disabled, Background(TokenChromeMuted).TextColor(TokenForegroundDisabled)),
	}
}

func init() {
	for _, name := range theme.Names() {
		t, ok := theme.Get(name)
		if !ok {
			continue
		}
		t.Styles = StylesFor(&t)
		theme.Register(t)
	}
	if cur := theme.Current(); cur != nil && cur.Name != "" {
		theme.Use(cur.Name)
	}
}

// ThemeStyles returns the component specs attached to th, or StylesFor(th)
// for a theme registered without any (one created after this package's init).
func ThemeStyles(th *theme.Theme) ComponentStyles {
	if th != nil {
		if s, ok := th.Styles.(ComponentStyles); ok {
			return s
		}
	}
	return StylesFor(th)
}

func (c *Ctx) styles() ComponentStyles {
	return ThemeStyles(c.Theme())
}
