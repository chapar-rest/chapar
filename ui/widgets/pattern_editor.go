package widgets

import (
	"image/color"
	"regexp"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	giovieweditor "github.com/oligo/gioview/editor"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

var (
	singleBracket = regexp.MustCompile(`(\{[a-zA-Z0-9_]+})`)
	doubleBracket = regexp.MustCompile(`(\{\{[a-zA-Z0-9_]+}})`)
)

// PatternEditor is a widget that allows the user to edit a text like and highlight patterns like {{id}} or {name}
type PatternEditor struct {
	*giovieweditor.Editor
	Keys map[string]string

	styledText string

	onChange func(text string)
	onSubmit func()
}

// NewPatternEditor creates a new PatternEditor
func NewPatternEditor() *PatternEditor {
	return &PatternEditor{
		Editor: new(giovieweditor.Editor),
		Keys:   make(map[string]string),
	}
}

func (p *PatternEditor) SetText(text string) {
	p.Editor.SetText(text, false)
	p.updateStyles(text)
}

func (p *PatternEditor) SetOnSubmit(onSubmit func()) {
	p.onSubmit = onSubmit
}

func (p *PatternEditor) SetOnChanged(onChange func(text string)) {
	p.onChange = onChange
}

func (p *PatternEditor) Layout(gtx layout.Context, theme *chapartheme.Theme, hint string) layout.Dimensions {
	if p.styledText == "" {
		p.updateStyles(p.Editor.Text())
	}

	editorConf := &giovieweditor.EditorConf{
		Shaper:          theme.Shaper,
		TextColor:       theme.Fg,
		Bg:              theme.Bg,
		SelectionColor:  theme.TextSelectionColor,
		TextSize:        unit.Sp(14),
		LineHeightScale: 1,
	}

	for {
		event, ok := p.Editor.Update(gtx)
		if !ok {
			break
		}

		switch event.(type) {
		// on carriage return event
		case giovieweditor.SubmitEvent:
			if p.onSubmit != nil {
				// goroutine to prevent blocking the ui update
				go p.onSubmit()
			}
		// on change event
		case giovieweditor.ChangeEvent:
			p.UpdateStyles()
			if p.onChange != nil {
				p.onChange(p.Editor.Text())
			}
		}
	}

	return giovieweditor.NewEditor(p.Editor, editorConf, hint).Layout(gtx)
}

func (p *PatternEditor) UpdateStyles() {
	p.updateStyles(p.Editor.Text())
}

func (p *PatternEditor) updateStyles(text string) {
	if p.styledText == text {
		return
	}

	var styles []*giovieweditor.TextStyle

	keyColor := color.NRGBA{R: 255, G: 165, B: 0, A: 255}
	// Apply styles based on matches
	applyStyles := func(re *regexp.Regexp) {
		matches := re.FindAllStringIndex(text, -1)
		for _, match := range matches {
			styles = append(styles, &giovieweditor.TextStyle{
				Start: match[0],
				End:   match[1],
				Color: nRGBAColorToOp(keyColor),
			})
		}
	}

	applyStyles(singleBracket)
	applyStyles(doubleBracket)

	p.styledText = text
	p.Editor.UpdateTextStyles(styles)
}

func nRGBAColorToOp(textColor color.NRGBA) op.CallOp {
	ops := new(op.Ops)

	m := op.Record(ops)
	paint.ColorOp{Color: textColor}.Add(ops)
	return m.Stop()
}
