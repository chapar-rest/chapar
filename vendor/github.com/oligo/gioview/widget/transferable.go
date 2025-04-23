package widget

import (
	"bytes"
	"encoding/json"
	"image"
	"io"
	"log"
	"strings"

	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"github.com/oligo/gioview/theme"
)

const (
	mimeText      = "application/text"
	mimeOctStream = "application/octet-stream"
)

type TransferTarget interface {
	// data to copy to clipboard.
	Data() string
	// read data from clipboard.
	OnPaste(data string, removeOld bool) error
}

type Transferable struct {
	Target TransferTarget
	isCut  bool
}

type payload struct {
	IsCut bool   `json:"isCut"`
	Data  string `json:"data"`
}

func asPayload(data string, isCut bool) io.Reader {
	p := payload{Data: data, IsCut: isCut}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(p)
	if err != nil {
		panic(err)
	}

	return strings.NewReader(buf.String())
}

func toPayload(reader io.ReadCloser) (*payload, error) {
	p := payload{}
	defer reader.Close()

	err := json.NewDecoder(reader).Decode(&p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (t *Transferable) Update(gtx C) error {
	for {
		ke, ok := gtx.Event(
			key.FocusFilter{Target: t},
			pointer.Filter{Target: t, Kinds: pointer.Press},
			key.Filter{Focus: t, Name: "C", Required: key.ModShortcut},
			key.Filter{Focus: t, Name: "V", Required: key.ModShortcut},
			key.Filter{Focus: t, Name: "X", Required: key.ModShortcut},
			transfer.TargetFilter{Target: t, Type: mimeText},
			transfer.TargetFilter{Target: t, Type: mimeOctStream}, // not work, why?
		)

		if !ok {
			break
		}

		switch event := ke.(type) {
		case key.Event:
			if !event.Modifiers.Contain(key.ModShortcut) || event.State != key.Press {
				break
			}

			switch event.Name {
			// Initiate a paste operation, by requesting the clipboard contents; other
			// half is in DataEvent.
			case "V":
				gtx.Execute(clipboard.ReadCmd{Tag: t})

			// Copy or Cut selection -- ignored if nothing selected.
			case "C", "X":
				gtx.Execute(clipboard.WriteCmd{Type: mimeOctStream, Data: io.NopCloser(asPayload(t.Target.Data(), event.Name == "X"))})
				t.isCut = true
			}

		case transfer.DataEvent:
			// read the clipboard content:
			if event.Type == mimeText {
				p, err := toPayload(event.Open())
				if err == nil {
					if err := t.Target.OnPaste(p.Data, p.IsCut); err != nil {
						return err
					}
				} else {
					content, err := io.ReadAll(event.Open())
					if err == nil {
						if err := t.Target.OnPaste(string(content), false); err != nil {
							log.Println(err)
							return err
						}
					}
				}
			}

		case pointer.Event:
			if event.Kind == pointer.Press {
				gtx.Execute(key.FocusCmd{Tag: t})
			}
		case key.FocusEvent:
			// pass
		}
	}

	return nil
}

func (t *Transferable) Layout(gtx C, th *theme.Theme, w layout.Widget) D {
	t.Update(gtx)

	macro := op.Record(gtx.Ops)
	dims := w(gtx)
	call := macro.Stop()

	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	if t.isCut {
		defer paint.PushOpacity(gtx.Ops, 0.7).Pop()
	}
	event.Op(gtx.Ops, t)
	call.Add(gtx.Ops)

	return dims
}
