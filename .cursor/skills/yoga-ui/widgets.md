# Yoga UI widget catalog

All constructors live in package `ui`. Fluent modifiers live on `*ui.Node` unless noted.

## Typography

| Func | Notes |
|---|---|
| `Text(s)` | Body size; `.Size(px)` sets font size |
| `Title(s)` | Theme title ramp |
| `Subtitle(s)` | Semibold subtitle |
| `Caption(s)` | Small muted |
| `Strong(s)` | Semibold body |
| `Muted(s)` | Body + `TokenForegroundMuted` |

Override color: `.Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))` or `TextColorLit(c)`.

## Buttons

```go
ui.Button(id, ui.Text("Label")).Primary().OnClick(fn)
ui.Button(id, ui.Text("Label")).Secondary() // default
ui.Button(id, ui.Text("Label")).Subtle()
ui.Button(id, ui.Text("Save")).IconStart("save").Hint("⌘S").Disabled(busy)
ui.IconButton(id, "settings").OnClick(fn)
```

`id` keys hover/press/focus. Child is usually `Text`.

## Inputs

```go
ui.TextField(id, value).
    Placeholder("…").IconStart("search").IconEnd("close").
    Password(true).
    OnChange(func(s string) { … }).
    OnSubmit(func(s string) { … }). // Enter
    DefaultFocus().Grow(1)

ui.Checkbox(id, "Label").Check(on).OnToggle(func(v bool) { … }).
    LabelMuted(done).LabelStrike(done)

ui.Radio(id, "Option A").Check(sel == 0).OnClick(func() { sel = 0 })

ui.Select(id, []ui.SelectOption{{Label: "Go", Value: "go"}}).
    Width(200).Selected(idx).
    OptionColor("GET", th.Success).
    OnChange(func(v string) { … }).
    OnSelectItem(func(i int, v string) { … })

ui.Segmented(id,
    ui.SegmentItem{Icon: "split_horizontal", Value: "h"},
    ui.SegmentItem{Label: "List", Value: "list"},
).Selected(idx).OnChange(func(v string) { … })

ui.TagEdit(id, tags).OnTags(func(t []string) { tags = t }).Width(400)
```

`TextField` is **controlled**: pass `app.field` every frame; store edits in `OnChange`. Caret/focus live in the widget store under `id`.

## Navigation / chrome

```go
ui.Nav(id, ui.NavVertical, ui.NavIconTop,
    ui.NavItem{ID: "home", Label: "Home", Icon: "folder"},
).Selected(i).OnSelectItem(func(i int, id string) { … }).Width(88).
    NavBackground(&th.Background)

ui.Tabs(id, []ui.TabModel{{Title: "a.go", Modified: true, Badge: "2"}}).
    Selected(active).
    OnSelectItem(func(i int, _ string) { … }).
    OnTabClose(func(i int) { … }).
    TabBackground(th.Background)

ui.Dropdown(id, "File", []ui.MenuItem{
    {Label: "Save", OnSelect: fn},
}).Width(160)

ui.Breadcrumb(id,
    ui.BreadcrumbSegment{Label: "src", OnSelect: fn},
    ui.BreadcrumbSegment{Label: "main.go"},
)

ui.Splitter(id, ui.Horizontal, left, right).Sizes(240, 0).Grow(1)
// ui.Vertical; size 0 = flex remainder. Drag state keyed by id.
```

Nav orientations: `NavVertical`, `NavHorizontal`. Item layouts: `NavIconLeft`, `NavIconRight`, `NavIconTop`, `NavIconBottom`.

## Surfaces / feedback

```go
ui.Card("Title", "Subtitle", body).Elevated() // or .Flat(); default raised
ui.Alert("message", ui.AlertInfo) // Warning, Error, Success
ui.Alert("…", ui.AlertError).Dismissable(func() { … })
ui.Spinner(id, 24)
ui.HLine(th.Stroke.Thin, th.Border)
ui.VLine(1, th.Border)
ui.Icon("circle", 12, th.Accent)
```

## Overlay hosts (construct once, always include in Body)

```go
app.dialogs = ui.NewDialogHost()
app.files = ui.NewFileDialog()
app.toasts = ui.NewToastHost()

// in Body, as children of the root Column:
app.dialogs
app.files
app.toasts

app.dialogs.ShowError("Error", "failed", func() {})
app.dialogs.ShowInput("Name", "placeholder", func(v string) {}, func() {})
app.files.Show(ui.FileDialogOpts{
    Mode: ui.FileDialogOpenFile, // or FileDialogOpenFolder
    Multiple: false,
    Filters: []ui.FileFilter{{Label: "Go files", Exts: []string{".go"}}},
    OnConfirm: func(paths []string) { … },
})
app.toasts.Show("Saved", ui.ToastInfo, 3*time.Second)
// ToastSuccess, ToastWarning, ToastError
```

Hosts return a zero-size element and register overlays via `c.Overlay`.

## Retained views (`ui.ViewOf`)

Construct in `Build*`, not in `Body`.

### Editor

```go
ed := ui.NewEditor([]byte("{\n}"), highlight.NewJSON())
ed := ui.NewEditorFor(path, content) // highlighter from extension
ui.ViewOf(ed).Grow(1)
ed.Bytes(); ed.Modified(); ed.MarkSaved(); ed.Undo(); ed.Redo(); ed.Close()
```

Virtualized, piece-table, LSP overlay self-registered. `CapturesTab() == true` (Tab inserts indent; Ctrl+Tab moves focus).

### Table

```go
t := ui.NewTable([]ui.TableColumn{
    {ID: "sel", Label: "", Kind: ui.TableColCheckbox, Width: 36},
    {ID: "key", Label: "Key", Kind: ui.TableColEditable, Width: 0, Sortable: true},
    {ID: "act", Label: "", Kind: ui.TableColActions, Width: 40, Locked: true},
}, []ui.TableAction{{Icon: "delete", Tooltip: "Delete"}})
t.SetRows([]ui.TableRow{{ID: "r1", Cells: map[string]string{"key": "a"}}})
t.Actions[0].OnClick = func(rowID string) { t.RemoveRow(rowID) }
t.SetFilter(q); t.AddRow(row)
ui.ViewOf(t).Height(220)
```

Row click: set `t.Selectable = true` (and `t.MultiSelect` for Cmd/Ctrl toggle + Shift range). `OnRowClick` / `OnRowActivate` (double-click or Enter). `TableRow.Icon` draws in the first text column.

Column kinds: `TableColText`, `TableColEditable`, `TableColCheckbox`, `TableColActions`. Width `0` = flex.

### Tree / FileTree

```go
root := &ui.TreeNode{Label: "Envs", Data: "root"}
tree := ui.NewTree(root)
tree.Loader = func(n *ui.TreeNode) []*ui.TreeNode {
    return []*ui.TreeNode{{Label: "A", Data: "a", Leaf: true}}
}
tree.SetRoot(root)
tree.OnActivate = func(n *ui.TreeNode) { … }
tree.SetFilter(q)
tree.Background = &theme.Current().Panel
ui.ViewOf(tree).Grow(1)

ft := ui.NewFileTree(cwd)
ft.OnOpenFile = func(path string) { … }
ft.SetContextMenu(func(path string) []ui.MenuItem { return … })
ft.SetFilter(q)
ui.ViewOf(ft).Grow(1)
```

### ListView / Scroll

Prefer `ui.Scroll(id, ui.Column(items...))` for declarative lists. `NewListView` takes `*layout.Element` children (lower level).

```go
ui.Scroll("gallery", ui.Column(...).Gap(th.Spacing.M).Padding(th.Spacing.L)).Grow(1)
```

## Spec (visual style)

```go
ui.Background(ui.TokenAccent).
    TextColor(ui.TokenAccentForeground).
    Radius(4).
    Border(ui.TokenBorder, 1).
    Cursor(ui.CursorPointer).
    Padding(8).Gap(8).
    When(ui.Hovered, ui.Background(ui.TokenAccentHover)).
    When(ui.Pressed, ui.Background(ui.TokenAccentPressed).Scale(0.96, 0.96)).
    When(ui.Focused, ui.Spec{}.Border(ui.TokenFocusRing, 1)).
    When(ui.Disabled, ui.Background(ui.TokenChromeMuted).TextColor(ui.TokenForegroundDisabled))
```

Attach with `.Style(spec)` or `.Background(token)`. Conditions: `Hovered`, `Pressed`, `Focused`, `Disabled`. Pressed overrides Hovered; Disabled last.

Tokens: `TokenSurface`, `TokenChrome`, `TokenChromeMuted`, `TokenForeground`, `TokenForegroundMuted`, `TokenForegroundSubtle`, `TokenForegroundDisabled`, `TokenAccent`, `TokenAccentHover`, `TokenAccentPressed`, `TokenAccentForeground`, `TokenBorder`, `TokenBorderStrong`, `TokenListHover`, `TokenListActive`, `TokenFocusRing`, `TokenSelection`, `TokenScrollTrack`, `TokenScrollThumb`, `TokenScrollThumbHover`, `TokenError`, `TokenWarning`, `TokenSuccess`.

## Ctx API

| Method | Use |
|---|---|
| `Theme()` | Active `*theme.Theme` |
| `Viewport()` | Logical pixel size |
| `Mouse()` / `Keyboard()` | This frame’s input (may be nil in tests) |
| `Text()` | Shaping engine |
| `Focus()` | `*FocusScope` |
| `Overlay(el)` | Portal painted/hit-tested on top |
| `Animate(d)` | Repaint within `d` (min across frame) |
| `Invalidate()` | Wake event loop; **any goroutine** |
| `Widget(id, alloc)` | Retained micro-state |
| `Now()` | Frame clock |

## Focusable (custom controls)

Implement `ui.Focusable`: `Focus`, `Blur`, `Focused`, `HandleText`, `HandleKeys`, `CapturesTab`, `FocusOnClick`, `FocusEl`. Register with `c.Focus().Add(w)` during Layout. Modal dialogs call `c.Focus().SetModal(w)`.

## Icons

Filestem of `render/assets/icons/*.svg`. Common: `add`, `search`, `settings`, `edit`, `delete`, `close`, `folder`, `folder_open`, `file`, `code`, `terminal`, `play_arrow`, `save`, `split_horizontal`, `split_vertical`, `expand_more`, `chevron_right`, `theme`, `check`. Register extras with `render.RegisterIcon` **before** atlas bake.

## Theme names

`yoga-dark` (default), `yoga-light`, `yoga-high-contrast`, `yoga-midnight`, `dark`, `light`, `github-dark`, `github-light`, `catppuccin`, `dracula`, `nord`, `solarized-dark`, `gruvbox-dark`, `gruvbox-light`, `one-dark`, `monokai`, `tokyo-night`, `rose-pine`. List: `theme.Names()`. Switch: `theme.Use(name)`.
