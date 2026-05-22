package codeeditor

import (
	"image"
	"io"
	"math"
	"strings"

	"gioui.org/f32"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget"
	"gioui.org/x/component"

	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type editorContextMenu struct {
	menu      component.MenuState
	selectAll widget.Clickable
	copy      widget.Clickable
	paste     widget.Clickable
	init      bool

	active        bool
	startedActive bool
	position      f32.Point
	dims          layout.Dimensions
}

func (c *CodeEditor) initContextMenu() {
	if c.ctxMenu.init {
		return
	}
	c.ctxMenu.init = true
	c.ctxMenu.menu.Options = []func(gtx layout.Context) layout.Dimensions{
		c.contextMenuItem(&c.ctxMenu.selectAll, "Select All", true),
		c.contextMenuItem(&c.ctxMenu.copy, "Copy", true),
		c.contextMenuItem(&c.ctxMenu.paste, "Paste", true),
	}
}

func (c *CodeEditor) contextMenuItem(clickable *widget.Clickable, label string, enabled bool) func(gtx layout.Context) layout.Dimensions {
	return func(gtx layout.Context) layout.Dimensions {
		return c.layoutContextMenuItem(gtx, clickable, label, enabled)
	}
}

func (c *CodeEditor) layoutContextMenuItem(gtx layout.Context, clickable *widget.Clickable, label string, enabled bool) layout.Dimensions {
	th := c.theme.Material()
	itm := component.MenuItem(th, clickable, label)
	itm.Label.Color = c.theme.Fg
	if !enabled {
		itm.Label.Color.A = 0x80
	}
	return itm.Layout(gtx)
}

func (c *CodeEditor) layoutContextMenuSurface(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	gtx.Constraints.Min = image.Point{}
	m := component.Menu(theme.Material(), &c.ctxMenu.menu)
	m.Fill = theme.MenuBgColor
	return m.Layout(gtx)
}

func (m *editorContextMenu) updatePointer(gtx layout.Context) {
	m.startedActive = m.active
	suppressionTag := &m.active
	dismissTag := &m.dims

	if m.active {
		for {
			ev, ok := gtx.Event(pointer.Filter{
				Target: m,
				Kinds:  pointer.Press | pointer.Release,
			})
			if !ok {
				break
			}
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}
			if e.Buttons.Contain(pointer.ButtonPrimary) {
				clickPos := e.Position.Sub(m.position)
				max := f32.Point{
					X: float32(m.dims.Size.X),
					Y: float32(m.dims.Size.Y),
				}
				if clickPos.X <= 0 || clickPos.Y <= 0 || clickPos.X >= max.X || clickPos.Y >= max.Y {
					m.dismiss()
				}
			}
		}

		for {
			ev, ok := gtx.Event(pointer.Filter{
				Target: suppressionTag,
				Kinds:  pointer.Press,
			})
			if !ok {
				break
			}
			if e, ok := ev.(pointer.Event); ok && e.Kind == pointer.Press {
				m.dismiss()
			}
		}

		for {
			ev, ok := gtx.Event(pointer.Filter{
				Target: dismissTag,
				Kinds:  pointer.Release,
			})
			if !ok {
				break
			}
			if e, ok := ev.(pointer.Event); ok && e.Kind == pointer.Release {
				m.dismiss()
			}
		}
	}

	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: m,
			Kinds:  pointer.Press | pointer.Release,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		if e.Buttons.Contain(pointer.ButtonSecondary) && e.Kind == pointer.Press {
			m.active = true
			m.position = e.Position
		}
	}
}

func (m *editorContextMenu) dismiss() {
	m.active = false
}

func (m *editorContextMenu) clampPosition(areaSize image.Point) {
	if int(m.position.X)+m.dims.Size.X > areaSize.X {
		if newX := int(m.position.X) - m.dims.Size.X; newX >= 0 {
			m.position.X = float32(newX)
		}
	}
	if int(m.position.Y)+m.dims.Size.Y > areaSize.Y {
		if newY := int(m.position.Y) - m.dims.Size.Y; newY >= 0 {
			m.position.Y = float32(newY)
		}
	}
}

func (c *CodeEditor) handleContextMenu(gtx layout.Context, theme *chapartheme.Theme) {
	c.initContextMenu()

	canCopy := c.editor.SelectionLen() > 0
	canPaste := !c.readOnly

	if c.ctxMenu.selectAll.Clicked(gtx) {
		c.editor.SetCaret(0, c.editor.Len())
		gtx.Execute(key.FocusCmd{Tag: c.editor})
	}
	if canCopy && c.ctxMenu.copy.Clicked(gtx) {
		if text := c.editor.SelectedText(); text != "" {
			gtx.Execute(clipboard.WriteCmd{
				Type: "application/text",
				Data: io.NopCloser(strings.NewReader(text)),
			})
		}
		gtx.Execute(key.FocusCmd{Tag: c.editor})
	}
	if canPaste && c.ctxMenu.paste.Clicked(gtx) {
		gtx.Execute(clipboard.ReadCmd{Tag: c.editor})
		gtx.Execute(key.FocusCmd{Tag: c.editor})
	}

	c.ctxMenu.menu.Options = []func(gtx layout.Context) layout.Dimensions{
		c.contextMenuItem(&c.ctxMenu.selectAll, "Select All", true),
		c.contextMenuItem(&c.ctxMenu.copy, "Copy", canCopy),
		c.contextMenuItem(&c.ctxMenu.paste, "Paste", canPaste),
	}
}

func (c *CodeEditor) layoutContextMenu(gtx layout.Context, theme *chapartheme.Theme, editor layout.Widget) layout.Dimensions {
	c.handleContextMenu(gtx, theme)
	menu := &c.ctxMenu

	return layout.Stack{}.Layout(gtx,
		layout.Stacked(editor),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			menu.updatePointer(gtx)
			areaDims := layout.Dimensions{Size: gtx.Constraints.Min}

			var contextual op.CallOp
			if menu.active || menu.startedActive {
				macro := op.Record(gtx.Ops)
				c.layoutContextMenuSurface(gtx, theme)
				contextual = macro.Stop()
			}

			if menu.active {
				macro := op.Record(gtx.Ops)
				menu.dims = c.layoutContextMenuSurface(gtx, theme)
				contextual = macro.Stop()

				menu.clampPosition(areaDims.Size)

				suppressionScrim := func() op.CallOp {
					macro2 := op.Record(gtx.Ops)
					stack := clip.Rect(image.Rectangle{
						Min: image.Point{-1e6, -1e6},
						Max: image.Point{1e6, 1e6},
					}).Push(gtx.Ops)
					event.Op(gtx.Ops, &menu.active)
					stack.Pop()
					return macro2.Stop()
				}()
				op.Defer(gtx.Ops, suppressionScrim)

				pos := image.Point{
					X: int(math.Round(float64(menu.position.X))),
					Y: int(math.Round(float64(menu.position.Y))),
				}
				macro3 := op.Record(gtx.Ops)
				op.Offset(pos).Add(gtx.Ops)
				contextual.Add(gtx.Ops)

				pt := pointer.PassOp{}.Push(gtx.Ops)
				stack := clip.Rect(image.Rectangle{Max: menu.dims.Size}).Push(gtx.Ops)
				event.Op(gtx.Ops, &menu.dims)
				stack.Pop()
				pt.Pop()
				op.Defer(gtx.Ops, macro3.Stop())
			}

			defer pointer.PassOp{}.Push(gtx.Ops).Pop()
			defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
			event.Op(gtx.Ops, menu)

			return areaDims
		}),
	)
}
