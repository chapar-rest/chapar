package view

import (
	"fmt"
	"iter"
	"net/url"

	"github.com/oligo/gioview/theme"

	"gioui.org/layout"
	"gioui.org/widget"
)

type ViewID struct {
	name string
	path string
}

type Intent struct {
	Target      ViewID
	Params      map[string]interface{}
	Referer     url.URL
	ShowAsModal bool
	// indicates the provider to create a new view instance and show up
	// in a new tab
	RequireNew bool
}

func (i Intent) Location() url.URL {
	return BuildURL(i.Target, i.Params)
}

type ViewAction struct {
	Name      string
	Icon      *widget.Icon
	OnClicked func(gtx C)
}

type View interface {
	Actions() []ViewAction
	Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions
	OnNavTo(intent Intent) error
	ID() ViewID
	Location() url.URL
	Title() string
	// set the view to finished state and do some cleanup ops.
	OnFinish()
	Finished() bool
}

// ViewProvider is used to construct new View instance. Each view
// should provide its own provider. Usually this is the constructor of the view.
type ViewProvider func() View

// View manager is the core of gio-view. It can be used to:
//  1. manages views and modals;
//  2. dispatch view/modal requests via the Intent object.
//  3. views navigation.
//
// Views can have a bounded history stack depending its intent request.
// The history stack is bounded to a tab widget which is part of a tab bar.
// Most the the API of the view manager handles navigating between views.
type ViewManager interface {
	// Register is used to register views before the view rendering happens.
	// Use provider to enable us to use dynamically constructed views.
	Register(ID ViewID, provider ViewProvider) error

	// Try to swith the current view to the requested view. If referer of the intent equals to
	// the current viewID of the current tab, the requested view should be routed and pushed to
	// to the existing viewstack(current tab). Otherwise a new viewstack for the intent is created(a new tab)
	// if there's no duplicate active view (first views of the stacks).
	RequestSwitch(intent Intent) error

	// OpenedViews return the views on top of the stack of each tab.
	OpenedViews() []View
	// Close the current tab and move backwards to the previous one if there's any.
	CloseTab(idx int)
	SwitchTab(idx int)
	// CurrentView returns the top most view of the current tab.
	CurrentView() View
	// current tab index
	CurrentViewIndex() int
	// Navigate back to the last view if there's any and pop out the current view.
	// It returns the view that is to be rendered.
	NavBack() View
	// Check is there are any naviBack-able views in the current stack or not. This should not
	// count for the current view.
	HasPrev() bool
	// return the next view that is intened to be shown in the modal layer. It returns nil if
	// there's no shownAsModal intent request.
	//
	// Deprecated: Please use [ModalViews] to handle multple stacked modal views.
	NextModalView() *ModalView
	// ModalViews returns all stacked modal views from back to front.
	ModalViews() iter.Seq[*ModalView]
	// finish the last modal view handling.
	FinishModalView()
	// refresh the window
	Invalidate()

	//Reset resets internal states of the VM
	Reset()
}

func BuildURL(target ViewID, params map[string]interface{}) url.URL {
	var urlParams = make(url.Values)
	for k, v := range params {
		urlParams.Add(k, fmt.Sprintf("%v", v))
	}

	u := target.Path()
	u.RawQuery = urlParams.Encode()
	return u
}
