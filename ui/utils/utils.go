package utils

import (
	"log"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
)

func IsKeysPressed(gtx layout.Context, modifier key.Modifiers, keyName string) bool {
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	key.InputOp{Tag: area, Keys: key.Set(modifier.String() + "|" + keyName)}.Add(gtx.Ops)
	defer area.Pop()

	for _, ev := range gtx.Events(&key.Event{}) {
		if e, ok := ev.(key.Event); ok {
			if e.State == key.Press && e.Modifiers == key.ModShortcut && e.Name == "S" {
				// Perform your action for Ctrl+S here
				log.Println("Ctrl+S pressed")
				return true
			}
		}
	}

	return false
}
