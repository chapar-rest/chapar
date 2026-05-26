package settings

import (
	"github.com/chapar-rest/chapar/internal/domain"
)

type EditorFontSettings struct {
	FontFamily string
	FontSize   int
}

type EditorEditingSettings struct {
	Indentation       string
	TabWidth          int
	AutoCloseBrackets bool
	AutoCloseQuotes   bool
	ShowLineNumbers   bool
	WrapLines         bool
}

// EditorSettings configures the code editor.
type EditorSettings struct {
	Font    EditorFontSettings
	Editing EditorEditingSettings
}

func (e *EditorSettings) Defaults() {}

func (e *EditorSettings) FromConfig(cfg domain.EditorConfig) {
	e.Font.FontFamily = cfg.FontFamily
	e.Font.FontSize = cfg.FontSize
	e.Editing.Indentation = cfg.Indentation
	e.Editing.TabWidth = cfg.TabWidth
	e.Editing.AutoCloseBrackets = cfg.AutoCloseBrackets
	e.Editing.AutoCloseQuotes = cfg.AutoCloseQuotes
	e.Editing.ShowLineNumbers = cfg.ShowLineNumbers
	e.Editing.WrapLines = cfg.WrapLines
}

func (e *EditorSettings) ToConfig(cfg *domain.EditorConfig) {
	cfg.FontFamily = e.Font.FontFamily
	cfg.FontSize = e.Font.FontSize
	cfg.Indentation = e.Editing.Indentation
	cfg.TabWidth = e.Editing.TabWidth
	cfg.AutoCloseBrackets = e.Editing.AutoCloseBrackets
	cfg.AutoCloseQuotes = e.Editing.AutoCloseQuotes
	cfg.ShowLineNumbers = e.Editing.ShowLineNumbers
	cfg.WrapLines = e.Editing.WrapLines
}
