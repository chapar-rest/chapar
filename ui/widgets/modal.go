package widgets

import (
	"time"

	"gioui.org/layout"
	"gioui.org/widget/material"
)

type Modal struct {
	w layout.Widget
}

func NewModal() *Modal {
	m := &Modal{}
	return m
}

func (m *Modal) Show(t time.Time, w layout.Widget) {
	m.w = w
}

func (m *Modal) Visible() bool {
	return m.w != nil
}

func (m *Modal) Hide(t time.Time) {
	m.w = nil
}

func (m *Modal) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return m.w(gtx)
}
