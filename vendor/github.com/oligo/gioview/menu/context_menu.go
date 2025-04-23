package menu

import (
	"github.com/oligo/gioview/theme"

	"gioui.org/io/pointer"
	"gioui.org/layout"
)

type ContextMenu struct {
	Menu
	contextArea ContextArea
	// position hint
	PositionHint layout.Direction
}

func NewContextMenu(options [][]MenuOption, absPosition bool) *ContextMenu {
	m := &ContextMenu{
		Menu: newMenu(options),
	}

	if absPosition {
		m.contextArea.AbsolutePosition = true
		m.contextArea.Activation = pointer.ButtonPrimary
	}

	return m
}

func (m *ContextMenu) Layout(gtx C, th *theme.Theme) D {
	m.Update(gtx)

	return m.layout(gtx, th, m.contextArea.Layout)
}

func (m *ContextMenu) Update(gtx C) {
	m.contextArea.PositionHint = m.PositionHint
	if m.contextArea.Activated() {
		m.onActivated(gtx)
	}

	if m.requestDismiss {
		m.contextArea.Dismiss()
		m.requestDismiss = false
	}

	if m.contextArea.Dismissed() {
		m.onDismissed(gtx)
	}

	if m.contextArea.Active() {
		m.update(gtx)
	}
}
