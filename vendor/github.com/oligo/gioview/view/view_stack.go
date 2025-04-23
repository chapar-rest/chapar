package view

import (
	"container/list"
	"iter"
)

// ViewStack is for view navigation history
type ViewStack struct {
	viewList *list.List
}

func (vs *ViewStack) Pop() View {
	head := vs.viewList.Front()
	if head == nil {
		return nil
	}

	return vs.viewList.Remove(head).(View)
}

func (vs *ViewStack) Peek() View {
	if vs.viewList == nil || vs.viewList.Len() <= 0 {
		return nil
	}

	if vs.viewList.Front() == nil {
		return nil
	}

	return vs.viewList.Front().Value.(View)
}

// push a new view to the stack and removes duplicates instance of the same view.
func (vs *ViewStack) Push(vw View) error {
	if vs.viewList == nil {
		vs.viewList = list.New()
		vs.viewList.Init()
	}

	// v := vs.viewList.Front()
	// for v != nil {
	// 	if existing := v.Value.(View); existing.ID() == vw.ID() {
	// 		vs.viewList.Remove(v)
	// 	}
	// 	v = v.Next()
	// }

	vs.viewList.PushFront(vw)
	return nil
}

func (vs *ViewStack) IsEmpty() bool {
	return vs.viewList == nil || vs.viewList.Len() <= 0
}

func (vs *ViewStack) Depth() int {
	if vs.viewList == nil {
		return 0
	}
	return vs.viewList.Len()
}

// All returns a iterator that iterates through the stack of views from back to front,
// or from front to back.
func (vs *ViewStack) All(backward bool) iter.Seq[View] {
	return func(yield func(View) bool) {
		var v *list.Element
		if backward {
			v = vs.viewList.Back()
		} else {
			v = vs.viewList.Front()
		}
		for v != nil {
			if !yield(v.Value.(View)) {
				return
			}

			if backward {
				v = v.Prev()
			} else {
				v = v.Next()
			}
		}
	}
}

func (vs *ViewStack) Clear() {
	if vs.viewList == nil {
		return
	}

	v := vs.viewList.Front()
	for v != nil {
		val := v.Value.(View)
		val.OnFinish()
		v = v.Next()
	}

	vs.viewList.Init()
}

func NewViewStack() *ViewStack {
	return &ViewStack{}
}
