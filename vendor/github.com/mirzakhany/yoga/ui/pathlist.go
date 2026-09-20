package ui

import (
	"strconv"

	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/layout"
)

type pathListData struct {
	paths     []string
	onPaths   func([]string)
	onAdd     func()
	addLabel  string
	emptyText string
	icon      icons.Icon
	noAdd     bool
}

// PathList shows file or directory paths one per row, each with a remove
// button, above an add button. Paths are controlled by the app: removing a row
// calls OnPaths with the remaining ones, and the add button only calls OnAdd,
// leaving the app to run a picker and hand back a new list.
//
// Long paths are elided in the middle, so the tail that identifies the file
// stays readable however narrow the list gets.
func PathList(id string, paths []string) *Node {
	return &Node{kind: kindPathList, id: id, extra: &pathListData{
		paths: append([]string(nil), paths...),
		icon:  icons.File,
	}}
}

// OnPaths is called with the remaining paths after a row is removed.
func (n *Node) OnPaths(fn func([]string)) *Node {
	if d, ok := n.extra.(*pathListData); ok {
		d.onPaths = fn
	}
	return n
}

// OnAdd is called when the add button is clicked. The app opens its own picker
// and pushes the result back as a new PathList value.
func (n *Node) OnAdd(fn func()) *Node {
	if d, ok := n.extra.(*pathListData); ok {
		d.onAdd = fn
	}
	return n
}

// AddLabel sets the text of the add button. The default is "Add…".
func (n *Node) AddLabel(s string) *Node {
	if d, ok := n.extra.(*pathListData); ok {
		d.addLabel = s
	}
	return n
}

// NoAdd drops the add button, for a list the app fills by other means.
func (n *Node) NoAdd() *Node {
	if d, ok := n.extra.(*pathListData); ok {
		d.noAdd = true
	}
	return n
}

// PathEmptyText sets the muted line shown when the list has no paths.
func (n *Node) PathEmptyText(s string) *Node {
	if d, ok := n.extra.(*pathListData); ok {
		d.emptyText = s
	}
	return n
}

// PathIcon sets the leading icon on every row. The default is a file icon;
// a list of directories wants icons.Folder.
func (n *Node) PathIcon(icon icons.Icon) *Node {
	if d, ok := n.extra.(*pathListData); ok {
		d.icon = icon
	}
	return n
}

func (n *Node) layoutPathList(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "pathlist")
	}
	d, _ := n.extra.(*pathListData)
	if d == nil {
		d = &pathListData{}
	}
	th := c.Theme()
	paths := d.paths

	kids := make([]View, 0, len(paths)+1)
	if len(paths) == 0 && d.emptyText != "" {
		kids = append(kids, Muted(d.emptyText).Padding(th.Spacing.XS))
	}
	for i, path := range paths {
		i, path := i, path
		row := make([]View, 0, 3)
		if !d.icon.Empty() {
			row = append(row, Icon(d.icon, th.Metrics.IconSizeSM, th.ForegroundMuted))
		}
		row = append(row,
			Text(path).Ellipsis(EllipsisMiddle).Grow(1).Tooltip(path),
			IconButton(id+"-rm-"+strconv.Itoa(i), icons.Trash2).Subtle().
				Disabled(n.disabled).
				Tooltip("Remove").
				OnClick(func() {
					next := append([]string(nil), paths...)
					next = append(next[:i], next[i+1:]...)
					if d.onPaths != nil {
						d.onPaths(next)
					}
				}),
		)
		kids = append(kids, Row(row...).
			Gap(th.Spacing.S).
			Padding(th.Spacing.XS).
			Align(layout.AlignCenter))
	}
	if !d.noAdd {
		label := d.addLabel
		if label == "" {
			label = "Add…"
		}
		kids = append(kids, Row(
			Button(id+"-add", Text(label)).Subtle().IconStart(icons.Plus).
				Disabled(n.disabled).
				OnClick(func() {
					if d.onAdd != nil {
						d.onAdd()
					}
				}),
			Spacer(),
		))
	}
	return Column(kids...).Gap(th.Spacing.XS).Style(n.spec).Layout(c)
}
