package keys

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
)

func OnSaveCommand(gtx layout.Context, receiver any, callback func()) {
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, receiver)
	for {
		keyEvent, ok := gtx.Event(
			key.Filter{
				Required: key.ModShortcut,
				Name:     "S",
			},
		)
		if !ok {
			break
		}

		if ev, ok := keyEvent.(key.Event); ok {
			if ev.Name == "S" && ev.Modifiers.Contain(key.ModShortcut) && ev.State == key.Press {
				callback()
			}
		}
	}
	area.Pop()
}

func OnKey(gtx layout.Context, receiver any, filter key.Filter, callback func()) {
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, receiver)
	for {
		keyEvent, ok := gtx.Event(filter)
		if !ok {
			break
		}

		if ev, ok := keyEvent.(key.Event); ok {
			if ev.Name == filter.Name && ev.Modifiers.Contain(filter.Required) && ev.State == key.Press {
				callback()
			}
		}
	}
	area.Pop()
}
