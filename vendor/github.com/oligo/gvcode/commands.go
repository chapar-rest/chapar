package gvcode

import (
	"slices"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"github.com/oligo/gvcode/textview"
)

// CommandHandler defines a callback function for the specific key event. It returns
// an EditorEvent if there is any.
type CommandHandler func(gtx layout.Context, evt key.Event) EditorEvent

type keyCommand struct {
	tag     any
	filter  key.Filter
	handler CommandHandler
}

// RegisterCommand register an extra command handler responding to key events.
// If there is an existing handler, it appends to the existing ones. Only the
// last key filter is checked during event handling. This method is expected to
// be invoked dynamically during layout.
func (e *Editor) RegisterCommand(srcTag any, filter key.Filter, handler CommandHandler) {
	if e.commands == nil {
		e.commands = make(map[key.Name][]keyCommand)
	}

	if len(e.commands) == 0 {
		e.buildBuiltinCommands()
	}

	if srcTag == nil || handler == nil {
		return
	}

	filter.Focus = e
	cmd := keyCommand{tag: srcTag, filter: filter, handler: handler}

	// overwrite the existing handler.
	idx := slices.IndexFunc(e.commands[filter.Name],
		func(c keyCommand) bool {
			return c.filter == cmd.filter && c.tag == cmd.tag
		})
	if idx >= 0 {
		e.commands[filter.Name][idx] = cmd
	} else {
		e.commands[filter.Name] = append(e.commands[filter.Name], cmd)
	}
}

// RemoveCommands unregister command handlers from tag.
func (e *Editor) RemoveCommands(tag any) {
	for k, cmds := range e.commands {
		e.commands[k] = slices.DeleteFunc(cmds, func(cmd keyCommand) bool {
			return cmd.tag == tag
		})

		if len(e.commands[k]) == 0 {
			delete(e.commands, k)
		}
	}
}

func (e *Editor) buildBuiltinCommands() {
	if e.commands == nil {
		e.commands = make(map[key.Name][]keyCommand)
	}
	clear(e.commands)

	registerCommand := func(filter key.Filter, handler CommandHandler) {
		filter.Focus = e
		e.commands[filter.Name] = append(e.commands[filter.Name], keyCommand{filter: filter, handler: handler})
	}

	registerCommand(key.Filter{Focus: e, Name: key.NameEnter, Optional: key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			return e.onInsertLineBreak(evt)
		},
	)

	registerCommand(key.Filter{Focus: e, Name: key.NameReturn, Optional: key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			return e.onInsertLineBreak(evt)
		},
	)

	registerCommand(key.Filter{Focus: e, Name: "C", Required: key.ModShortcut},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			return e.onCopyCut(gtx, evt)
		},
	)

	// Initiate a paste operation, by requesting the clipboard contents; other
	// half is in Editor.processKey() under clipboard.Event.
	registerCommand(key.Filter{Focus: e, Name: "V", Required: key.ModShortcut},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			if !e.readOnly {
				gtx.Execute(clipboard.ReadCmd{Tag: e})
			}
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: "X", Required: key.ModShortcut},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			return e.onCopyCut(gtx, evt)
		})

	registerCommand(key.Filter{Focus: e, Name: "Z", Required: key.ModShortcut, Optional: key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			if !e.readOnly {
				if evt.Modifiers.Contain(key.ModShift) {
					if ev, ok := e.redo(); ok {
						return ev
					}
				} else {
					if ev, ok := e.undo(); ok {
						return ev
					}
				}
			}
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: "A", Required: key.ModShortcut},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			e.text.SetCaret(0, e.text.Len())
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: key.NameHome, Optional: key.ModShortcut | key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			selAct := textview.SelectionClear
			if evt.Modifiers.Contain(key.ModShift) {
				selAct = textview.SelectionExtend
			}

			if evt.Modifiers.Contain(key.ModShortcut) {
				e.text.MoveTextStart(selAct)
			} else {
				e.text.MoveLineStart(selAct)
			}
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: key.NameEnd, Optional: key.ModShortcut | key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			selAct := textview.SelectionClear
			if evt.Modifiers.Contain(key.ModShift) {
				selAct = textview.SelectionExtend
			}
			if evt.Modifiers.Contain(key.ModShortcut) {
				e.text.MoveTextEnd(selAct)
			} else {
				e.text.MoveLineEnd(selAct)
			}
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: key.NameTab, Optional: key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			return e.onTab(evt)
		})

	registerCommand(key.Filter{Focus: e, Name: key.NameDeleteBackward, Optional: key.ModShortcutAlt | key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			if !e.readOnly {
				moveByWord := evt.Modifiers.Contain(key.ModShortcutAlt)

				if moveByWord {
					if e.deleteWord(-1) != 0 {
						return ChangeEvent{}
					}
				} else {
					if e.Delete(-1) != 0 {
						e.updateCompletor("", true)
						return ChangeEvent{}
					}
				}
			}
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: key.NameDeleteForward, Optional: key.ModShortcutAlt | key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			if !e.readOnly {
				moveByWord := evt.Modifiers.Contain(key.ModShortcutAlt)
				if moveByWord {
					if e.deleteWord(1) != 0 {
						return ChangeEvent{}
					}
				} else {
					if e.Delete(1) != 0 {
						return ChangeEvent{}
					}
				}
			}
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: key.NamePageDown, Optional: key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			selAct := textview.SelectionClear
			if evt.Modifiers.Contain(key.ModShift) {
				selAct = textview.SelectionExtend
			}
			e.text.MovePages(+1, selAct)
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: key.NamePageUp, Optional: key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			selAct := textview.SelectionClear
			if evt.Modifiers.Contain(key.ModShift) {
				selAct = textview.SelectionExtend
			}
			e.text.MovePages(-1, selAct)
			return nil
		})

	checkPos := func(gtx layout.Context) (bool, bool) {
		caret, _ := e.text.Selection()
		atBeginning := caret == 0
		atEnd := caret == e.text.Len()
		if gtx.Locale.Direction.Progression() != system.FromOrigin {
			atEnd, atBeginning = atBeginning, atEnd
		}
		return atBeginning, atEnd
	}

	registerCommand(key.Filter{Focus: e, Name: key.NameLeftArrow, Optional: key.ModShortcutAlt | key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			atBeginning, _ := checkPos(gtx)
			if atBeginning {
				return nil
			}

			moveByWord := evt.Modifiers.Contain(key.ModShortcutAlt)
			direction := 1
			if gtx.Locale.Direction.Progression() == system.TowardOrigin {
				direction = -1
			}
			selAct := textview.SelectionClear
			if evt.Modifiers.Contain(key.ModShift) {
				selAct = textview.SelectionExtend
			}

			if moveByWord {
				e.text.MoveWords(-1*direction, selAct)
			} else {
				if selAct == textview.SelectionClear {
					e.text.ClearSelection()
				}
				e.text.MoveCaret(-1*direction, -1*direction*int(selAct))
			}
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: key.NameUpArrow, Optional: key.ModShortcutAlt | key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			atBeginning, _ := checkPos(gtx)
			if atBeginning {
				return nil
			}

			selAct := textview.SelectionClear
			if evt.Modifiers.Contain(key.ModShift) {
				selAct = textview.SelectionExtend
			}
			e.text.MoveLines(-1, selAct)
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: key.NameRightArrow, Optional: key.ModShortcutAlt | key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			_, atEnd := checkPos(gtx)
			if atEnd {
				return nil
			}

			moveByWord := evt.Modifiers.Contain(key.ModShortcutAlt)
			direction := 1
			if gtx.Locale.Direction.Progression() == system.TowardOrigin {
				direction = -1
			}
			selAct := textview.SelectionClear
			if evt.Modifiers.Contain(key.ModShift) {
				selAct = textview.SelectionExtend
			}

			if moveByWord {
				e.text.MoveWords(1*direction, selAct)
			} else {
				if selAct == textview.SelectionClear {
					e.text.ClearSelection()
				}
				e.text.MoveCaret(1*direction, int(selAct)*direction)
			}
			return nil
		})

	registerCommand(key.Filter{Focus: e, Name: key.NameDownArrow, Optional: key.ModShortcutAlt | key.ModShift},
		func(gtx layout.Context, evt key.Event) EditorEvent {
			_, atEnd := checkPos(gtx)
			if atEnd {
				return nil
			}

			selAct := textview.SelectionClear
			if evt.Modifiers.Contain(key.ModShift) {
				selAct = textview.SelectionExtend
			}
			e.text.MoveLines(+1, selAct)
			return nil
		})

}

func (e *Editor) processCommands(gtx layout.Context) EditorEvent {
	if len(e.commands) == 0 {
		e.buildBuiltinCommands()
	}

	for _, cmds := range e.commands {
		cmd := cmds[len(cmds)-1]
		for {
			ke, ok := gtx.Event(cmd.filter)
			if !ok {
				break
			}

			e.blinkStart = gtx.Now
			if ke, ok := ke.(key.Event); ok {
				if !gtx.Focused(e) || ke.State != key.Press {
					break
				}
				e.scrollCaret = true
				e.scroller.Stop()

				if !ke.Modifiers.Contain(cmd.filter.Required) {
					break
				}

				if evt := cmd.handler(gtx, ke); evt != nil {
					return evt
				}
			}
		}
	}

	return nil
}
