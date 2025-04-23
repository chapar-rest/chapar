package view

import (
	"errors"
	"fmt"
	"iter"
	"log"
	"net/url"
	"slices"
	"sync"

	"gioui.org/app"
)

var _ ViewManager = (*defaultViewManager)(nil)

type defaultViewManager struct {
	window *app.Window
	stacks []*ViewStack
	// views which are to be shown as modal.
	modalStack    *ViewStack
	currentTabIdx int
	views         map[ViewID]ViewProvider

	// title of the window
	currentTitle string
	mu           sync.Mutex
}

func (vm *defaultViewManager) CurrentView() View {
	if len(vm.stacks) <= 0 {
		return nil
	}

	stack := vm.stacks[vm.currentTabIdx]
	vw := stack.Peek()

	if vm.currentTitle != vw.Title() {
		vm.currentTitle = vw.Title()
		vm.window.Option(app.Title(vm.currentTitle))
	}
	return vw
}

func (vm *defaultViewManager) NextModalView() *ModalView {
	if vm.modalStack == nil {
		return nil
	}
	vw := vm.modalStack.Peek()
	if vw == nil {
		return nil
	}

	return &ModalView{View: vw}

}

func (vm *defaultViewManager) ModalViews() iter.Seq[*ModalView] {
	return func(yield func(*ModalView) bool) {
		if vm.modalStack == nil {
			return
		}

		viewIter := vm.modalStack.All(true)

		for vw := range viewIter {
			if !yield(vw.(*ModalView)) {
				return
			}
		}
	}

}

func (vm *defaultViewManager) FinishModalView() {
	vm.modalStack.Pop()
}

func (vm *defaultViewManager) CurrentViewIndex() int {
	return vm.currentTabIdx
}

// func (vm *defaultViewManager) Register(vw View) error {
// 	return vm.RegisterWithProvider(Provide(vw))
// }

func (vm *defaultViewManager) Register(ID ViewID, provider ViewProvider) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if ID == (ViewID{}) {
		return errors.New("cannot register empty view ID")
	}

	if provider == nil {
		return errors.New("view provider is nil")
	}

	if vm.views == nil {
		vm.views = make(map[ViewID]ViewProvider)
	}
	vm.views[ID] = provider
	log.Println("registered view: ", ID)
	return nil
}

func (vm *defaultViewManager) NavBack() View {
	if len(vm.stacks) <= 0 {
		return nil
	}

	stack := vm.stacks[vm.currentTabIdx]
	if stack.Depth() <= 1 {
		// keep the last view
		return stack.Peek()
	}

	vw := stack.Pop()
	vw.OnFinish()
	return stack.Peek()
}

func (vm *defaultViewManager) HasPrev() bool {
	if len(vm.stacks) <= 0 {
		return false
	}
	stack := vm.stacks[vm.currentTabIdx]
	return stack.Depth() > 1
}

func (vm *defaultViewManager) RequestSwitch(intent Intent) error {
	// use mutex to guard the dispatching
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Even if using a empty intent, vm refreshes the window.
	defer vm.window.Invalidate()

	if intent.Target == (ViewID{}) {
		return nil
	}
	provider, ok := vm.views[intent.Target]
	if !ok {
		return fmt.Errorf("no target view found: %v", intent.Target)
	}

	var targetView View
	stack := vm.route(&intent)

	// get target view
	if topVw := stack.Peek(); topVw != nil && topVw.Location() == intent.Location() {
		targetView = topVw
	} else {
		targetView = provider()
		if intent.ShowAsModal {
			targetView = &ModalView{View: targetView}
		}
		err := stack.Push(targetView)
		if err != nil {
			return fmt.Errorf("push to viewstack error: %w", err)
		}
	}

	err := targetView.OnNavTo(intent)
	if err != nil {
		stack.Pop()
		return fmt.Errorf("error handling intent: %w", err)
	}

	location := intent.Location()
	log.Printf("switching to %s", location.String())
	return nil
}

// route the intent to the proper viewstack/tab by intent.URL(). Note that this
// method does not handle modal intent routing.
func (vm *defaultViewManager) routeView(intent *Intent) *ViewStack {
	if len(vm.stacks) <= vm.currentTabIdx {
		// try to fix the illegal state
		stack := NewViewStack()
		vm.stacks = append(vm.stacks, stack)
		vm.currentTabIdx = len(vm.stacks) - 1
		return stack
	}

	// Iterate through all the viewstacks to find the top view with the same location.
	// switch to and replace the existing view.
	for idx, s := range vm.stacks {
		if s.Peek().Location() == intent.Location() {
			// switch to the tab
			vm.currentTabIdx = idx
			return s
		}
	}

	if intent.RequireNew {
		stack := NewViewStack()
		vm.stacks = append(vm.stacks, stack)
		vm.currentTabIdx = len(vm.stacks) - 1
		return stack
	}

	// Respect referer by checking its parent view.
	if intent.Referer != (url.URL{}) && intent.Referer == vm.CurrentView().Location() {
		// push to current view stack
		return vm.stacks[vm.currentTabIdx]
	}

	// then try to match the viewID:
	for idx, s := range vm.stacks {
		if intent.Target == s.Peek().ID() {
			vm.currentTabIdx = idx
			return s
		}
	}

	// create new stack
	stack := NewViewStack()
	vm.stacks = append(vm.stacks, stack)
	vm.currentTabIdx = len(vm.stacks) - 1

	return stack
}

// route the intent to the proper viewstack/tab or to the modal stack.
func (vm *defaultViewManager) route(intent *Intent) *ViewStack {
	if intent.ShowAsModal {
		if vm.modalStack == nil {
			vm.modalStack = NewViewStack()
		}
		return vm.modalStack
	}

	return vm.routeView(intent)
}

func (vm *defaultViewManager) OpenedViews() []View {
	views := make([]View, len(vm.stacks))
	for idx, stack := range vm.stacks {
		views[idx] = stack.Peek()
	}

	return views
}

func (vm *defaultViewManager) CloseTab(idx int) {
	if idx < 0 || idx >= len(vm.stacks) {
		return
	}

	stack := vm.stacks[idx]
	stack.Clear()
	vm.stacks = slices.Delete[[]*ViewStack, *ViewStack](vm.stacks, idx, idx+1)
	if vm.currentTabIdx >= idx && vm.currentTabIdx > 0 {
		vm.currentTabIdx -= 1
	}
}

func (vm *defaultViewManager) SwitchTab(idx int) {
	if idx >= len(vm.stacks) || idx < 0 {
		return
	}

	vm.currentTabIdx = idx
}

func (vm *defaultViewManager) Invalidate() {
	vm.window.Invalidate()
}

func (vm *defaultViewManager) Reset() {
	if vm.modalStack != nil {
		vm.modalStack.Clear()
	}
	for _, stack := range vm.stacks {
		stack.Clear()
	}

	vm.currentTabIdx = 0
	vm.stacks = vm.stacks[:0]
	vm.Invalidate()
}

func DefaultViewManager(window *app.Window) ViewManager {
	return &defaultViewManager{
		window: window,
	}
}
