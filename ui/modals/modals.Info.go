package modals

import "gioui.org/widget"

func NewInfo(title, body string) *Message {
	return &Message{
		Title: title,
		Body:  body,
		Type:  MessageTypeInfo,
		OKBtn: widget.Clickable{},
	}
}
