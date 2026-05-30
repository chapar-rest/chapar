package widget

import (
	"fmt"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"

	"github.com/chapar-rest/chapar/uiv2/theme"
)

// Table wraps [core.Table] with a trailing delete action column.
type Table struct {
	core.Table
	onDelete func(row int)
}

// NewTable creates a Table as a child of parent.
func NewTable(parent tree.Node) *Table {
	return tree.New[Table](parent)
}

// OnDelete sets a callback invoked after a row is deleted via the action column.
func (t *Table) OnDelete(fn func(row int)) *Table {
	t.onDelete = fn
	return t
}

func (t *Table) Init() {
	t.Table.Init()
	theme.ConfigureTable(&t.Table)
}

func (t *Table) RowWidgetNs() (nWidgPerRow, idxOff int) {
	n, off := t.Table.RowWidgetNs()
	return n + 1, off
}

func (t *Table) MakeRow(p *tree.Plan, i int) {
	t.Table.MakeRow(p, i)

	tree.AddAt(p, fmt.Sprintf("action-%d", i), func(w *core.Button) {
		w.SetType(core.ButtonAction).SetIcon(icons.Delete)
		w.SetProperty(core.ListRowProperty, i)
		w.Updater(func() {
			si, _, invis := t.SliceIndex(i)
			w.SetState(invis, states.Invisible)
			if t.HasStyler() {
				t.StyleRow(w, si, 0)
			}
		})
		w.OnClick(func(e events.Event) {
			si, _, _ := t.SliceIndex(i)
			t.DeleteAt(si)
			if t.onDelete != nil {
				t.onDelete(si)
			}
		})
	})
}
