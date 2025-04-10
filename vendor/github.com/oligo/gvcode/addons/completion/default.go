package completion

import (
	"image"
	"strings"
	"sync"

	"gioui.org/io/key"
	"gioui.org/layout"
	"github.com/oligo/gvcode"
)

var _ gvcode.Completion = (*DefaultCompletion)(nil)

type DefaultCompletion struct {
	Editor     *gvcode.Editor
	triggers   []gvcode.Trigger
	completors []gvcode.Completor
	candicates []gvcode.CompletionCandidate
	session    *session
	mu         sync.Mutex
	popup      gvcode.CompletionPopup
}

type session struct {
	ctx     *gvcode.CompletionContext
	trigger gvcode.Trigger
}

func (dc *DefaultCompletion) SetTriggers(triggers ...gvcode.Trigger) {
	dc.triggers = dc.triggers[:0]
	dc.triggers = append(dc.triggers, triggers...)
	if tr, exists := gvcode.GetCompletionTrigger[gvcode.KeyTrigger](dc.triggers); exists {
		dc.Editor.RegisterCommand(dc,
			key.Filter{Name: tr.Name, Required: tr.Modifiers},
			func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
				dc.onKey()
				return nil
			})
	}

}

func (dc *DefaultCompletion) SetCompletors(completors ...gvcode.Completor) {
	dc.completors = dc.completors[:0]
	dc.completors = append(dc.completors, completors...)
}

func (dc *DefaultCompletion) onKey() {
	ctx := dc.Editor.GetCompletionContext()
	if dc.session == nil {
		dc.session = &session{
			ctx: &ctx,
		}
	} else {
		dc.session.ctx = &ctx
	}

	dc.runCompletors(ctx)
}

func (dc *DefaultCompletion) OnText(ctx gvcode.CompletionContext) {
	if ctx.Input == "" {
		dc.Cancel()
		return
	}

	if ctx.New {
		if dc.session == nil {
			dc.session = &session{
				ctx: &ctx,
			}
		} else {
			dc.session.ctx = &ctx
		}
	} else if dc.session == nil {
		return
	}

	if !dc.trigger(&ctx) {
		dc.Cancel()
		return
	}

	dc.runCompletors(ctx)
}

func (dc *DefaultCompletion) runCompletors(ctx gvcode.CompletionContext) {
	dc.candicates = dc.candicates[:0]
	if len(dc.completors) == 0 {
		return
	}

	wg := new(sync.WaitGroup)
	wg.Add(len(dc.completors))

	for _, c := range dc.completors {
		go func() {
			items := c.Suggest(ctx)
			dc.mu.Lock()
			defer dc.mu.Unlock()
			dc.candicates = append(dc.candicates, items...)

			wg.Done()
		}()
	}

	wg.Wait()

	if len(dc.candicates) == 0 {
		dc.Cancel()
		return
	}
	dc.rerank()
}

func (dc *DefaultCompletion) trigger(ctx *gvcode.CompletionContext) bool {
	prefixTrigger, exists := gvcode.GetCompletionTrigger[gvcode.PrefixTrigger](dc.triggers)
	if exists && strings.HasPrefix(ctx.Input, prefixTrigger.Prefix) {
		// strip the trigger prefix from input.
		ctx.Input = strings.TrimPrefix(ctx.Input, prefixTrigger.Prefix)
		return true
	}

	autoTrigger, exists := gvcode.GetCompletionTrigger[gvcode.AutoTrigger](dc.triggers)
	if !exists || len([]rune(ctx.Input)) < autoTrigger.MinSize {
		return false
	}

	return true
}

func (dc *DefaultCompletion) rerank() {
	// TODO
}

func (dc *DefaultCompletion) SetPopup(popup gvcode.CompletionPopup) {
	dc.popup = popup
}

func (dc *DefaultCompletion) IsActive() bool {
	return dc.session != nil
}

func (dc *DefaultCompletion) Offset() image.Point {
	if dc.session == nil {
		return image.Point{}
	}

	return dc.session.ctx.Position.Coords
}

func (dc *DefaultCompletion) Layout(gtx layout.Context) layout.Dimensions {
	return dc.popup(gtx, dc.candicates)
}

func (dc *DefaultCompletion) Cancel() {
	dc.session = nil
	dc.candicates = dc.candicates[:0]
}

func (dc *DefaultCompletion) OnConfirm(idx int) {
	if dc.Editor == nil {
		return
	}
	if idx < 0 || idx >= len(dc.candicates) {
		return
	}

	candidate := dc.candicates[idx]
	dc.Editor.SetCaret(dc.session.ctx.Position.Start, dc.session.ctx.Position.End)
	dc.Editor.Insert(candidate.InsertText)
	dc.Cancel()
}
