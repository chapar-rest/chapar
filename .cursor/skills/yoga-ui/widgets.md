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
ui.Button(id, ui.Caption("Ln 12, Col 4")).Ghost().IconStart(icons.Code).Tooltip("Go to line").OnClick(fn)
ui.Button(id, ui.Caption("UTF-8")).Ghost().HoverFill().IconStart(icons.ChevronDown).OnClick(fn)
ui.Button(id, ui.Text("Save")).IconStart(icons.Save).Hint("⌘S").Disabled(busy)
ui.IconButton(id, "settings").OnClick(fn)

// Whole button toggles menu; with OnClick the label is the action and chevron opens menu
ui.MenuButton(id, "Export", []ui.MenuItem{
    {Label: "CSV", OnSelect: fn},
}).Primary().IconStart(icons.Save)
ui.MenuButton(id, "Save", items).Primary().OnClick(save) // split button
```

`id` keys hover/press/focus. Child is usually `Text`.

## Inputs

```go
ui.TextField(id, value).
    Placeholder("…").IconStart(icons.Search).IconEnd(icons.X).
    Password(true).
    OnChange(func(s string) { … }).
    OnSubmit(func(s string) { … }). // Enter
    DefaultFocus().Grow(1)

ui.Checkbox(id, "Label").Check(on).OnToggle(func(v bool) { … }).
    LabelMuted(done).LabelStrike(done)

ui.Switch(id).Check(on).OnToggle(func(v bool) { … })

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

ui.Form(id,
    ui.FormSwitch("notify", "Notifications", "Show alerts", on, func(v bool) { on = v }),
    ui.FormSelect("theme", "Theme", "Color scheme", opts, idx, onChange),
    ui.FormNumber("size", "Font size", "Editor size in pt", 14, 10, 24, 1, onSize),
    ui.FormText("file", "Default file", "Open on startup", name, onName),
    ui.FormSlider("vol", "Volume", "Master level", vol, 0, 100, 1, onVol),
    ui.FormStepper("retries", "Retries", "Attempts", n, 0, 10, 1, onN),
)

ui.Slider(id, value).Min(0).Max(100).Step(1).Width(240).
    OnFloatChange(func(v float64) { … })

ui.NumberStepper(id, value).Min(0).Max(20).Step(1).
    OnFloatChange(func(v float64) { … })
```

`Form` rows are controlled: pass current values each frame. `Switch` is an unlabeled pill toggle for compact form rows. `OnChange` on text widgets takes `string`; Slider/Stepper use `OnFloatChange`.

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

ui.Drawer("inspector", panel, page).
    Open(open).
    Edge(ui.EdgeRight). // Left, Top, Bottom
    Overlay().          // or .Push()
    Size(320).
    Resizable(true).
    Modal(true).        // overlay: scrim + outside click / Escape
    Swipe(true).        // edge drag to open, panel drag to close
    OnOpenChange(func(v bool) { open = v }).
    Grow(1)
// Panel content fills the drawer body (clipped viewport). Use .Grow(1) on the
// panel view; avoid fixed .Width/.Height — the drawer owns main-axis sizing.

// Nested IDE chrome (terminal bottom + chat right):
ui.Drawer("chat", chatPanel,
    ui.Drawer("term", termPanel, editor).
        Edge(ui.EdgeBottom).Push().Open(termOpen).Size(180).Grow(1),
).Edge(ui.EdgeRight).Push().Open(chatOpen).Size(320).Grow(1)

ui.Link(id, "Docs").OnClick(fn)
ui.Disclosure(id, "Section", body).Open(open).OnToggle(func(v bool) { open = v })
ui.Accordion(id,
    ui.AccordionItem{ID: "a", Title: "One", Body: ui.Text("…")},
).OpenIDs(openID).Exclusive().OnAccordionToggle(func(id string, open bool) { … })
```

Nav orientations: `NavVertical`, `NavHorizontal`. Item layouts: `NavIconLeft`, `NavIconRight`, `NavIconTop`, `NavIconBottom`.

## Surfaces / feedback

```go
ui.Card("Title", "Subtitle", body).Elevated() // or .Flat(); default raised
ui.Alert("message", ui.AlertInfo) // Warning, Error, Success
ui.Alert("…", ui.AlertError).Dismissable(func() { … })
ui.Spinner(id, 24)
ui.Badge("3").Tone(ui.BadgeAccent) // Muted, Accent, Success, Warning, Error
ui.Kbd("⌘S")
ui.ProgressBar(id, 0.45).Width(200)
ui.ProgressBar(id, 0).Width(200).Indeterminate()
ui.ProgressRing(id, 0.45)
ui.Skeleton(id).Width(160).Height(12)
ui.Skeleton(id).Circle(40)
ui.EmptyState("No results", "Try another filter").
    EmptyIcon("search").
    Action(ui.Button("add", ui.Text("Create")).Primary().OnClick(fn))
ui.HLine(th.Stroke.Thin, th.Border)
ui.VLine(1, th.Border)
ui.Icon("circle", 12, th.Accent)
```

## Anchored overlays

```go
ui.Button(id, ui.Text("Save")).Tooltip("Save document") // hover delay ~400ms
ui.Tooltip(id, child, "Hint")                            // wrapper form

ui.Popover(id, trigger, content).
    Open(open).OnOpenChange(func(v bool) { open = v }).
    Placement(ui.PlacementBottom). // Top, Left, Right
    Width(260).Height(140)

ui.ContextMenu(id, child, []ui.MenuItem{
    {Label: "Copy", OnSelect: fn},
}).Width(180) // opens on right-click
```

Tooltip/Popover/ContextMenu share `placeAnchor` (preferred side + flip + viewport clamp). Popover has **no scrim** (unlike dialogs). Escape / outside click dismisses Popover and ContextMenu. `TableAction.Tooltip` is shown on action-icon hover.

## Overlay hosts (window services)

```go
c.Dialogs().ShowInfo("Info", "note", func() {})
c.Dialogs().ShowWarning("Warning", "careful", func() {})
c.Dialogs().ShowError("Error", "failed", func() {})
c.Dialogs().ShowAction("Delete?", "Cannot undo.", onYes, onNo)
c.Dialogs().ShowInput("Name", "placeholder", func(v string) {}, func() {})
c.Dialogs().Show(ui.DialogOpts{
    Title: "Settings", Width: 720, Height: 520,
    Body: func(c *ui.Ctx) ui.View { return ui.Form("prefs", rows...) },
    Actions: []ui.DialogAction{{Label: "Close", Primary: true}},
})
c.Files().Show(ui.FileDialogOpts{
    Mode: ui.FileDialogOpenFile, // FileDialogOpenFolder | FileDialogSaveFile
    Multiple: false,
    Filters: []ui.FileFilter{{Label: "Go files", Exts: []string{".go"}}},
    ShowSaveFilter: true,    // save mode: show file-type filter in footer
    AllowCreateFolder: true, // footer: New Folder next to Cancel/Save
    OnConfirm: func(paths []string) { … },
})
c.Toasts().Show("Saved", ui.ToastInfo, 3*time.Second)
// ToastSuccess, ToastWarning, ToastError
```

`FileDialog` is a pure-Go picker (places sidebar, breadcrumb, file table, footer). Save mode adds a filename field; `AllowCreateFolder` puts **New Folder** in the footer after the filter, beside Cancel and Save.

`BuildFrame` lays out the window hosts after the body. Do not put `c.Dialogs()` / `c.Files()` / `c.Toasts()` / `c.Commands()` in the view tree. Dedicated `NewDialogHost()` / `NewFileDialog()` / `NewToastHost()` remain for tests or a second picker — those must still be placed in the tree.

## Command palette / shortcuts

```go
func (a *App) Body(c *ui.Ctx) ui.View {
    c.Commands().Register(
        ui.Section("Recent"),
        ui.Item("recent.main").Title("main.go").Detail("cmd/app/main.go").
            Icon(icons.File).Run(func() { a.open("cmd/app/main.go") }),
        ui.Section("Commands"),
        ui.Cmd("file.save").Title("Save File").Shortcut("⌘S").Icon(icons.Save).Run(a.save),
        ui.Cmd("view.theme").Title("Toggle Theme").Group("View").Shortcut("⌘T").Run(a.toggleTheme),
    )
    return ui.Column(
        ui.Button("palette", ui.Text("Commands")).
            Hint(c.Commands().ToggleLabel()). // default ⌘K
            OnClick(func() { c.Commands().Show() }),
        // ...
    )
}
```

- Register every frame from `Body` (same rhythm as controlled values). Last write wins per id.
- Use `ui.Cmd` for actions (optional `Shortcut`); use `ui.Item` for non-command targets such as recent files (`Title`, `Detail` path, no shortcut).
- Use `ui.Section("Recent")` for labeled separators between groups. Sections are skipped by arrow keys/clicks and hide when none of their following items match the query.
- Registration order is preserved so sections stay with their items.
- Default toggle chord is **Mod+K** (⌘K / Ctrl+K); override with `c.Commands().ToggleChord("⌘P")`.
- `Enabled(false)`: listed but greyed; shortcut does not fire. `Hidden(true)`: shortcut only, omitted from the list.
- Palette: search field, subsequence filter (title/id/group/detail), Up/Down/Enter/Escape, trailing `Kbd` chips.
- Chord strings: `"⌘S"`, `"Mod+K"`, `"Ctrl+Shift+P"`. `Mod`/`⌘`/`Cmd`/`Ctrl` mean primary modifier.
- `yoga.KeyHook` remains for one-off keys that are not commands; command Dispatch runs first.

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
root := &ui.TreeNode{
    Label: "Envs",
    Children: []*ui.TreeNode{
        {Label: "A", Data: "a", Leaf: true},
        {Label: "B", Data: "b", Leaf: true},
    },
}
tree := ui.NewTree(root)
tree.AddChild(nil, &ui.TreeNode{Label: "C", Leaf: true}) // nil parent = root
tree.Remove(node)
tree.Rebuild() // after manual Children edits or OnDrop mutations

// Lazy folders: Loader runs only when a branch has no Children yet
tree.Loader = func(n *ui.TreeNode) []*ui.TreeNode {
    return []*ui.TreeNode{{Label: "lazy.go", Leaf: true}}
}
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
| `Dialogs()` / `Files()` / `Toasts()` / `Commands()` | Window overlay hosts |
| `Overlay(el)` | Portal painted/hit-tested on top |
| `Animate(d)` | Repaint within `d` (min across frame) |
| `Invalidate()` | Wake event loop; **any goroutine** |
| `Widget(id, alloc)` | Retained micro-state |
| `Now()` | Frame clock |

## Focusable (custom controls)

Implement `ui.Focusable`: `Focus`, `Blur`, `Focused`, `HandleText`, `HandleKeys`, `CapturesTab`, `FocusOnClick`, `FocusEl`. Register with `c.Focus().Add(w)` during Layout. Modal dialogs call `c.Focus().SetModal(w)`.

## Icons

Pre-rasterized [Lucide](https://lucide.dev) symbols in `github.com/mirzakhany/yoga/icons`. Use typed vars so the linker drops unused icons:

```go
import "github.com/mirzakhany/yoga/icons"

ui.Icon(icons.Search, 16, th.Foreground)
ui.Button("save", ui.Text("Save")).IconStart(icons.Save)
ui.IconButton("settings", icons.Settings)
```

Common: `icons.Plus`, `icons.Search`, `icons.Settings`, `icons.Pencil`, `icons.Trash2`, `icons.X`, `icons.Folder`, `icons.FolderOpen`, `icons.File`, `icons.Code`, `icons.Terminal`, `icons.ChevronDown`, `icons.ChevronRight`, `icons.Sun`. Full browse list: `icons/catalog.All`. Regenerate: `go run ./cmd/generate-lucide`. Custom SVGs: `render.RegisterIcon(name, svg)` before first draw.

## Theme names

`yoga-dark` (default), `yoga-light`, `yoga-high-contrast`, `yoga-midnight`, `dark`, `light`, `github-dark`, `github-light`, `catppuccin`, `dracula`, `nord`, `solarized-dark`, `gruvbox-dark`, `gruvbox-light`, `one-dark`, `monokai`, `tokyo-night`, `rose-pine`. List: `theme.Names()`. Switch: `theme.Use(name)`.
