package view

import (
	"net/url"

	"github.com/oligo/gioview/theme"

	"gioui.org/layout"
)

type Widget func(gtx layout.Context, th *theme.Theme) layout.Dimensions

type BaseView struct {
	location *url.URL
	finished bool
}

type EmptyView struct {
}

// A helper view which implements the View interface.
type SimpleView struct {
	BaseView
	id            ViewID
	title         string
	w             Widget
	intentHandler func(intent Intent) error
}

func (base *BaseView) ID() ViewID { return ViewID{} }

func (base *BaseView) Title() string { return "Base" }

func (base *BaseView) Actions() []ViewAction {
	return nil
}

func (base *BaseView) OnNavTo(intent Intent) error {
	loc := intent.Location()
	base.location = &loc
	return nil
}

func (base *BaseView) OnFinish() {
	base.location = nil
	base.finished = true
	return
}

func (base *BaseView) Finished() bool {
	return base.finished
}

func (base *BaseView) Location() url.URL {
	return *base.location
}

func (base *BaseView) Layout(gtx C, th *theme.Theme) D {
	return layout.Dimensions{}
}

func (sv *SimpleView) ID() ViewID { return sv.id }

func (sv *SimpleView) Title() string { return sv.title }

func (sv *SimpleView) OnNavTo(intent Intent) error {
	sv.BaseView.OnNavTo(intent)
	return sv.intentHandler(intent)
}

func (sv *SimpleView) Location() url.URL {
	return *sv.BaseView.location
}

func (sv *SimpleView) Layout(gtx C, th *theme.Theme) D {
	return sv.w(gtx, th)
}

func Simple(id ViewID, title string, w Widget, intentHandler func(intent Intent) error) View {
	return &SimpleView{
		id:            id,
		title:         title,
		w:             w,
		intentHandler: intentHandler,
	}
}

func (v EmptyView) Actions() []ViewAction {
	return nil
}

func (v EmptyView) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (v EmptyView) OnNavTo(intent Intent) error {
	return nil
}

func (v EmptyView) ID() ViewID    { return NewViewID("Blank") }
func (v EmptyView) Title() string { return "Blank" }
func (v EmptyView) Location() url.URL {
	return BuildURL(v.ID(), nil)
}
