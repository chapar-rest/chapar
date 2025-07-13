package textview

import (
	"github.com/oligo/gvcode/textstyle/decoration"
	"github.com/oligo/gvcode/textstyle/syntax"
)

func (e *TextView) AddDecorations(styles ...decoration.Decoration) {
	if e.decorations == nil {
		panic("TextView is not properly  initialized.")
	}

	e.decorations.Insert(styles...)
}

func (e *TextView) ClearDecorations(source string) {
	if e.decorations == nil {
		panic("TextView is not properly  initialized.")
	}

	if source == "" {
		e.decorations.RemoveAll()
	} else {
		e.decorations.RemoveBySource(source)
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
