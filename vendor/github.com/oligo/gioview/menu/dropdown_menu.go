package menu

import (
	"time"

	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/widget"

	"gioui.org/layout"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

type DropdownMenu struct {
	Menu
	modalLayer *widget.ModalLayer
	activated  bool
}

func NewDropdownMenu(options [][]MenuOption) *DropdownMenu {
	m := &DropdownMenu{
		modalLayer: widget.NewModal(),
		Menu:       newMenu(options),
	}
	m.modalLayer.Duration = time.Millisecond * 100
	return m
}

func (m *DropdownMenu) Layout(gtx C, th *theme.Theme) D {
	m.Update(gtx)

	return m.layout(gtx, th, func(gtx C, w layout.Widget) D {
		m.modalLayer.Widget = func(gtx C, th *material.Theme, anim *component.VisibilityAnimation) D {
			return w(gtx)
		}
		return m.modalLayer.Layout(gtx, th.Theme)
	})

}

// Update states and report whether the dropdown menu has just dismissed.
func (m *DropdownMenu) Update(gtx C) bool {
	if m.requestDismiss && m.modalLayer.Visible() {
		m.modalLayer.Disappear(gtx.Now)
		m.requestDismiss = false
	}

	if m.modalLayer.State == component.Appearing {
		// start appearing
		if !m.activated {
			m.activated = true
			m.onActivated(gtx)
		}
	} else if m.modalLayer.State == component.Disappearing {
		// start disappearing
		if m.activated {
			m.activated = false
			m.onDismissed(gtx)
		}
	}

	if m.modalLayer.Visible() {
		m.update(gtx)
	}

	return m.modalLayer.Dismissed()
}

// ToggleVisibility toggles the visibility state and report the changed state.
func (m *DropdownMenu) ToggleVisibility(gtx C) bool {
	m.modalLayer.ToggleVisibility(gtx.Now)
	return m.modalLayer.Visible()
}
