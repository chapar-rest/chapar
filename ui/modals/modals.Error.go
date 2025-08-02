package modals

import "gioui.org/widget"

func NewError(err error) *Message {
	return &Message{
		Title: "Error",
		Body:  err.Error(),
		Type:  MessageTypeErr,
		OKBtn: widget.Clickable{},
	}
}
