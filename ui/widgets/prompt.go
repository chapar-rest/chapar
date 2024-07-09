package widgets

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

// Prompt is a modal dialog that prompts the user for a response.
const (
	ModalTypeInfo = "info"
	ModalTypeWarn = "warn"
	ModalTypeErr  = "err"
)

var (
	colors = map[string]color.NRGBA{
		// Red
		ModalTypeErr: {R: 0xD1, G: 0x1E, B: 0x35, A: 0xFF},
		// Light blue
		ModalTypeInfo: {R: 0x1D, G: 0xBF, B: 0xEC, A: 0xFF},
		// Yellow
		ModalTypeWarn: {R: 0xFD, G: 0xB5, B: 0x0E, A: 0xFF},
	}
)

type Prompt struct {
	Title   string
	Content string
	Type    string
	Visible bool

	rememberBool *widget.Bool

	options []Option
	result  string

	onSubmit func(selectedOption string, remember bool)
}

type Option struct {
	Text   string
	Button widget.Clickable
	Icon   *widget.Icon
}

func NewPrompt(title, content, modalType string, options ...Option) *Prompt {
	return &Prompt{
		Title:   title,
		Content: content,
		Type:    modalType,
		options: options,
	}
}

func (p *Prompt) SetOptions(options ...Option) {
	p.options = options
}

func (p *Prompt) Show() {
	p.Visible = true
}

func (p *Prompt) Hide() {
	p.Visible = false
}

func (p *Prompt) IsVisible() bool {
	return p.Visible
}

func (p *Prompt) WithRememberBool() {
	p.rememberBool = &widget.Bool{Value: false}
}

func (p *Prompt) WithoutRememberBool() {
	p.rememberBool = nil
}

func (p *Prompt) SetOnSubmit(f func(selectedOption string, remember bool)) {
	p.onSubmit = f
}

func (p *Prompt) submit() {
	if p.onSubmit == nil {
		return
	}

	if !p.Visible {
		return
	}

	if p.rememberBool == nil {
		p.onSubmit(p.result, false)
		return
	}

	p.onSubmit(p.result, p.rememberBool.Value)
}

func (p *Prompt) Result() (string, bool) {
	if p.result == "" {
		return "", false
	}

	if p.rememberBool != nil {
		return p.result, p.rememberBool.Value
	}

	return p.result, false
}

func (p *Prompt) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	if !p.Visible {
		return layout.Dimensions{}
	}

	textColor := theme.ContrastFg
	switch p.Type {
	case ModalTypeErr:
		textColor = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	case ModalTypeInfo:
		textColor = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}
	case ModalTypeWarn:
		textColor = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}
	}

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 8).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, colors[p.Type])
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Vertical,
					Alignment: layout.Middle,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							h := material.H6(theme.Material(), p.Title)
							h.Color = textColor
							return h.Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							b := material.Body1(theme.Material(), p.Content)
							b.Color = textColor
							return b.Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						count := len(p.options)
						if p.rememberBool != nil {
							count++
						}

						items := make([]layout.FlexChild, 0, count)
						if p.rememberBool != nil {
							items = append(
								items,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return material.CheckBox(theme.Material(), p.rememberBool, "Don't ask again").Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
							)
						}
						for i := range p.options {
							i := i

							if p.options[i].Button.Clicked(gtx) {
								p.result = p.options[i].Text
								p.submit()
							}

							items = append(
								items,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									btn := Button(theme.Material(), &p.options[i].Button, nil, IconPositionStart, p.options[i].Text)
									btn.Background = chapartheme.White
									btn.Color = chapartheme.Black
									return btn.Layout(gtx, theme)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
							)
						}
						return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis:      layout.Horizontal,
								Alignment: layout.Middle,
								Spacing:   layout.SpaceStart,
							}.Layout(gtx,
								items...,
							)
						})
					}),
				)
			})
		},
	)
}
