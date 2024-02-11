package widgets

import "gioui.org/layout"

type Modal struct {
	Visible bool
	Content layout.Widget

	onClose func()
}

func (m *Modal) Layout(gtx layout.Context) layout.Dimensions {
	if m.Visible {
		return m.Content(gtx)
	}
	return layout.Dimensions{}
}

func (m *Modal) Show() {
	m.Visible = true
}

func (m *Modal) Hide() {
	m.Visible = false
	if m.onClose != nil {
		m.onClose()
	}
}

func (m *Modal) OnClose(f func()) {
	m.onClose = f
}

func NewModal(content layout.Widget) *Modal {
	return &Modal{
		Content: content,
	}
}
