package textview

import (
	"github.com/oligo/gvcode/textstyle/decoration"
	"github.com/oligo/gvcode/textstyle/syntax"
)

func (e *TextView) AddDecorations(styles ...decoration.Decoration) error {
	if e.decorations == nil {
		panic("TextView is not properly initialized.")
	}

	return e.decorations.Insert(styles...)
}

func (e *TextView) ClearDecorations(source string) error {
	if e.decorations == nil {
		panic("TextView is not properly initialized.")
	}

	if source == "" {
		return e.decorations.RemoveAll()
	} else {
		return e.decorations.RemoveBySource(source)
	}
}

func (e *TextView) SetColorScheme(scheme *syntax.ColorScheme) {
	e.syntaxStyles = syntax.NewTextTokens(scheme)
}

func (e *TextView) SetSyntaxTokens(tokens ...syntax.Token) {
	if e.syntaxStyles == nil {
		panic("TextView is not properly initialized.")
	}
	e.syntaxStyles.Set(tokens...)
}
