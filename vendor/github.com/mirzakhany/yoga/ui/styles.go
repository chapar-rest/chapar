package ui

import "github.com/mirzakhany/yoga/theme"

// ComponentStyles is the set of default Specs shipped widgets use. Override
// them on a cloned theme via Theme.Styles (type ComponentStyles) so a replica
// can change control appearance as well as palette.
type ComponentStyles struct {
	ButtonPrimary   Spec
	ButtonSecondary Spec
	ButtonSubtle    Spec
	TextField       Spec
	Checkbox        Spec
}

// DefaultStyles returns token-based control specs. They recolor with theme.Use.
func DefaultStyles() ComponentStyles {
	r := float32(theme.DefaultRadius().Medium)
	stroke := float32(theme.DefaultStroke().Thin)
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
			Border(TokenBorder, stroke).
			When(Hovered, Background(TokenListHover)).
			When(Pressed, Background(TokenListActive).Border(TokenBorderStrong, stroke)).
			When(Disabled, Background(TokenChromeMuted).TextColor(TokenForegroundDisabled)),
		ButtonSubtle: Spec{}.TextColor(TokenForeground).
			Radius(r).
			Cursor(CursorPointer).
			When(Hovered, Background(TokenListHover)).
			When(Pressed, Background(TokenListActive)).
			When(Disabled, Spec{}.TextColor(TokenForegroundDisabled)),
		TextField: Background(TokenChrome).
			TextColor(TokenForeground).
			Radius(r).
			Border(TokenBorder, stroke).
			When(Focused, Spec{}.Border(TokenFocusRing, stroke)),
		Checkbox: Background(TokenChrome).
			TextColor(TokenForeground).
			Radius(float32(theme.DefaultRadius().Small)).
			Border(TokenBorder, stroke).
			When(Hovered, Background(TokenListHover)).
			When(Pressed, Background(TokenAccent)),
	}
}

func init() {
	s := DefaultStyles()
	for _, name := range theme.Names() {
		t, ok := theme.Get(name)
		if !ok {
			continue
		}
		t.Styles = s
		theme.Register(t)
	}
	if cur := theme.Current(); cur != nil && cur.Name != "" {
		theme.Use(cur.Name)
	}
}

// ThemeStyles returns the component specs attached to th, or DefaultStyles.
func ThemeStyles(th *theme.Theme) ComponentStyles {
	if th != nil {
		if s, ok := th.Styles.(ComponentStyles); ok {
			return s
		}
	}
	return DefaultStyles()
}

func (c *Ctx) styles() ComponentStyles {
	return ThemeStyles(c.Theme())
}
