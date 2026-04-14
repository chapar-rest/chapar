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

	styledText     string
	highlightColor color.NRGBA

	changed   bool
	submitted bool
}

// NewPatternEditor creates a new PatternEditor
func NewPatternEditor() *PatternEditor {
	pe := &PatternEditor{
		Editor: new(giovieweditor.Editor),
		Keys:   make(map[string]string),
	}

	pe.SingleLine = true

	return pe
}

func (p *PatternEditor) SetText(text string) {
	p.Editor.SetText(text, false)
	p.updateStyles(text)
}

func (p *PatternEditor) Changed() bool {
	out := p.changed
	p.changed = false
	return out
}

func (p *PatternEditor) Submitted() bool {
	out := p.submitted
	p.submitted = false
	return out
}

func (p *PatternEditor) Layout(gtx layout.Context, theme *chapartheme.Theme, hint string) layout.Dimensions {
	if p.highlightColor != theme.PatternHighlightColor {
		p.highlightColor = theme.PatternHighlightColor
		p.styledText = ""
	}
	if p.styledText == "" {
		p.updateStyles(p.Text())
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
		event, ok := p.Update(gtx)
		if !ok {
			break
		}

		switch event.(type) {
		// on carriage return event
		case giovieweditor.SubmitEvent:
			p.submitted = true
		// on change event
		case giovieweditor.ChangeEvent:
			p.UpdateStyles()
			p.changed = true
		}
	}
	gtx.Constraints.Max.Y = gtx.Dp(20)
	return giovieweditor.NewEditor(p.Editor, editorConf, hint).Layout(gtx)
}

func (p *PatternEditor) UpdateStyles() {
	p.updateStyles(p.Text())
}

func (p *PatternEditor) updateStyles(text string) {
	if p.styledText == text {
		return
	}

	var styles []*giovieweditor.TextStyle

	keyColor := p.highlightColor
	if keyColor == (color.NRGBA{}) {
		keyColor = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
	}
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
	p.UpdateTextStyles(styles)
}

func nRGBAColorToOp(textColor color.NRGBA) op.CallOp {
	ops := new(op.Ops)

	m := op.Record(ops)
	paint.ColorOp{Color: textColor}.Add(ops)
	return m.Stop()
}
