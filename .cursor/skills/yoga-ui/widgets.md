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

`Text` never overflows once it is given an ellipsis mode; it keeps its measured
width as the flex basis and shortens only when the parent squeezes it:

```go
ui.Text(path).Ellipsis(ui.EllipsisMiddle).Grow(1)  // "/Users/me/…/v1.proto"
ui.Text(name).Ellipsis(ui.EllipsisEnd).MaxWidth(180)
```

`Markdown(id, src)` renders CommonMark + GFM (headings, emphasis, strikethrough,
inline code, links, quotes, bullet/ordered/task lists, highlighted fenced code
with a copy button, tables, rules). Stateless DSL node: the parse is cached
under `id` and redone only when `src` changes, so it suits a file preview and a
streaming LLM reply alike. Each block's text is selectable. It is as tall as its
content; wrap long documents in `Scroll`. Links do nothing until `OnLink`.

```go
ui.Scroll("readme", ui.Markdown("readme-md", src).OnLink(openURL).Padding(th.Spacing.L)).Grow(1)
```

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
ui.MenuButton(id, modelName, items).Ghost().Shrink(1)     // may narrow to icon + chevron, label cut with …

// Row actions shown only on hover: HoverGroup passes whether the pointer is
// over it; Reveal(false) keeps the button's space but hides it and lets
// clicks through (a button whose menu is open stays shown).
ui.HoverGroup(id+"-row", func(hovered bool) ui.View {
    return ui.Row(rowButton, ui.IconButton(id+"-more", icons.Ellipsis).Reveal(hovered).Menu(items))
})
```

`id` keys hover/press/focus. Child is usually `Text`. `HoverGroup` adds no box of its own; modifiers on it apply to the child's.

`ui.Responsive(id, func(width float32) ui.View)` builds its view knowing the width it got at the last layout (0 on the first frame) and rebuilds when that changes: pick a compact form for a narrow pane, e.g. a menu in place of a side list. Like `HoverGroup` it adds no box.

Text with `.Ellipsis(...)` only shortens along a Row's main axis: in a Column, wrap it as `ui.Row(ui.Text(s).Ellipsis(ui.EllipsisEnd).Shrink(1))`, or use `ui.Paragraph` to wrap. To center a block that should shrink with its pane, use `ui.Row(ui.Spacer(), block.Shrink(1), ui.Spacer())` rather than `ui.Center`, whose child keeps its content width.

## Inputs

```go
ui.TextField(id, value).
    Placeholder("…").IconStart(icons.Search).IconEnd(icons.X).
    Password(true).
    OnChange(func(s string) { … }).
    OnSubmit(func(s string) { … }). // Enter
    DefaultFocus().Grow(1)

ui.EditableLabel(id, value).
    Placeholder("Untitled").
    OnSave(func(s string) { … }) // Enter commits; Escape / blur cancels

ui.Checkbox(id, "Label").Check(on).OnToggle(func(v bool) { … }).
    LabelMuted(done).LabelStrike(done)

ui.Switch(id).Check(on).OnToggle(func(v bool) { … })

ui.Radio(id, "Option A").Check(sel == 0).OnClick(func() { sel = 0 })

ui.Select(id, []ui.SelectOption{{Label: "Go", Value: "go"}}).
    Width(200).Selected(idx).
    OptionColor("GET", th.Success).
    OnChange(func(v string) { … }).
    OnSelectItem(func(i int, v string) { … })
ui.Select(id, methods).FitContent().     // as wide as the selected label
    OptionColors(func(o ui.SelectOption) render.Color { … }) // colors menu + trigger labels

// One bordered control: shared border and radius, 1 px dividers, one focus
// ring around the group while any part is focused, outer corners only.
ui.InputGroup("req",
    ui.Select("m", methods).FitContent().OptionColors(methodColor),
    ui.TextField("url", url).Grow(1),
    ui.Button("send", ui.Text("Send")).Primary(),
).Grow(1)

ui.Segmented(id,
    ui.SegmentItem{Icon: "split_horizontal", Value: "h"},
    ui.SegmentItem{Label: "List", Value: "list"},
).Selected(idx).OnChange(func(v string) { … })

ui.TagEdit(id, tags).OnTags(func(t []string) { tags = t }).Width(400)

ui.PathList(id, paths).                      // file/dir paths, one per row
    PathIcon(icons.Folder).                  // default icons.File
    AddLabel("Add import path…").            // default "Add…"
    PathEmptyText("No import paths").
    OnPaths(func(p []string) { paths = p }). // after a row is removed
    OnAdd(func() { /* app opens its own picker */ })
// NoAdd() drops the add button. Rows elide in the middle, so the tail of a
// long path stays readable.

ui.Form(id,
    ui.FormSwitch("notify", "Notifications", "Show alerts", on, func(v bool) { on = v }),
    ui.FormSelect("theme", "Theme", "Color scheme", opts, idx, onChange),
    ui.FormNumber("size", "Font size", "Editor size in pt", 14, 10, 24, 1, onSize),
    ui.FormText("file", "Default file", "Open on startup", name, onName),
    ui.FormFile("cert", "Certificate", "PEM for TLS", path, filters, onPath), // "" = cleared
    ui.FormSlider("vol", "Volume", "Master level", vol, 0, 100, 1, onVol),
    ui.FormStepper("retries", "Retries", "Attempts", n, 0, 10, 1, onN),
)

ui.Slider(id, value).Min(0).Max(100).Step(1).Width(240).
    OnFloatChange(func(v float64) { … })

ui.NumberStepper(id, value).Min(0).Max(20).Step(1).
    OnFloatChange(func(v float64) { … })
```

`ui.Form(id, items...).Grouped()` draws the rows as one inset list with dividers instead of a bordered card per row; a `FormHeading` starts a new list. A form narrower than 360 px puts each control under its label and description instead of beside them.

Switches paint with `th.SwitchTrackOff` / `SwitchKnobOff` (a muted knob, so off reads as off) and `SwitchTrackOn` / `SwitchKnobOn`; each knob clears 3:1 on its track, and themes may set them (tokens `TokenSwitchTrackOff` etc.).


`Form` rows are controlled: pass current values each frame. `Switch` is an unlabeled pill toggle for compact form rows. `OnChange` on text widgets takes `string`; Slider/Stepper use `OnFloatChange`.

`TextField` is **controlled**: pass `app.field` every frame; store edits in `OnChange`. Caret/focus live in the widget store under `id`.

`TextArea(id, value)` is the multi-line TextField: same modifiers (`Placeholder`, `OnChange`, `OnSubmit`, `Suggest`, `Highlight`, `DefaultFocus`). Lines wrap; `.Rows(min, max)` sets the height range (default 3–10), scrolling past max. Enter inserts a newline, unless `OnSubmit` is set: then Enter submits and Shift+Enter inserts the newline (chat composer). The placeholder stays visible while the empty area is focused.

`EditableLabel` is **controlled** for display but edits a local draft until **Enter** (`OnSave`). Hover shows an I-beam cursor; **Escape** or blur discards changes.

## Navigation / chrome

```go
// Opt in at startup: yoga.Config{CustomTitleBar: true}
ui.TitleBar(
    ui.Dropdown("menu-file", "File", items),
    ui.Spacer(),
    ui.TextField("q", query).Placeholder("Search…").Width(240).Grow(1),
    ui.IconButton("settings", icons.Settings).OnClick(openSettings),
)
// TitleBar uses TokenChrome, th.Metrics.TitleBarHeight, Shrink(0).
// Child widgets auto-size to th.Metrics.TitleBarControlHeight (26px) with
// vertical centering — no need to set .Height() on each control.
// macOS: leading inset for traffic lights. Windows/Linux: trailing WindowControls.
// Drag empty area to move; double-click toggles maximize.

ui.Nav(id, ui.NavVertical, ui.NavIconTop,
    ui.NavItem{ID: "home", Label: "Home", Icon: "folder"},
).Selected(i).OnSelectItem(func(i int, id string) { … }).Width(88).
    NavBackground(&th.Background)

ui.Tabs(id, []ui.TabModel{{Title: "a.go", Modified: true, Badge: "2"}}).
    Selected(active).
    OnSelectItem(func(i int, _ string) { … }).
    OnTabClose(func(i int) { … }).
    OnTabContextMenu(func(i int) []ui.MenuItem { … }). // right-click menu per tab
    Closable(false). // hide close buttons for section-style tabs
    TabBackground(th.Background)
// TabModel.Busy swaps the icon for a small spinner (a running task);
// TabModel.Status paints a colored dot after the title (e.g. th.Warning for
// an interrupted task).
// Too many tabs: the strip scrolls (wheel/trackpad), keeps the active tab in
// view, and a "+N" button at the right edge lists the hidden tabs.

ui.Dropdown(id, "File", []ui.MenuItem{
    {Label: "Save", OnSelect: fn},
}).Width(160)

ui.Breadcrumb(id,
    ui.BreadcrumbSegment{Label: "src", OnSelect: fn},
    ui.BreadcrumbSegment{Label: "main.go"},
)

ui.Splitter(id, ui.Horizontal, left, right).Sizes(240, 0).Grow(1)
// Or .Percents(30, 70); .MinSizes(120, 200); .MaxSizes(400, 0); .HandleOnHover()
// ui.Vertical; size/percent 0 = flex remainder. Drag state keyed by id.
// The flex pane takes the room left, never its content's width; a sized pane
// gives way down to its MinSizes entry when the splitter is too small.

ui.Drawer("inspector", panel, page).
    Open(open).
    Edge(ui.EdgeRight). // Left, Top, Bottom
    Overlay().          // or .Push()
    Size(320).
    MinSize(240).       // drag limit; Push also lets the panel give way to it in a small window
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
ui.ClickableCard(id, "Title", "Subtitle", body). // hover fill, pointer, click
    OnClick(open).
    Actions(ui.IconButton(id+"-more", icons.Ellipsis).Menu(items)). // title-row slot; its clicks skip the card
    Footer(ui.Caption("Updated 2h ago")).
    CardSelected(sel) // accent border
// Card grid: fixed-width cards in ui.Row(cards...).Wrap().Gap(…)
ui.Alert("message", ui.AlertInfo) // Warning, Error, Success
ui.Alert("…", ui.AlertError).Dismissable(func() { … })
ui.Spinner(id, 24)
ui.Badge("3").Tone(ui.BadgeAccent) // Muted, Accent, Success, Warning, Error
ui.Badge("GET").Hue(ui.HueGreen).Mono() // categorical hue, monospace label
ui.Tag("Production").Hue(ui.HueRed)     // squared-off chip
ui.Kbd("⌘S")
ui.ProgressBar(id, 0.45).Width(200)
ui.ProgressBar(id, 0).Width(200).Indeterminate()
ui.ProgressRing(id, 0.45)
ui.RangeBar(id, 65, 166, 179).Tone(ui.BadgeWarning).Tooltip("Waiting (TTFB) 101 ms") // span on a track; Tone or Hue colors it; .Width/.Height/.Grow size it
ui.Skeleton(id).Width(160).Height(12)
ui.Skeleton(id).Circle(40)
ui.EmptyState("No results", "Try another filter"). // the detail wraps in a narrow pane
    EmptyIcon("search").
    Action(ui.Button("add", ui.Text("Create")).Primary().OnClick(fn))
ui.HLine(th.Stroke.Thin, th.Border)
ui.VLine(1, th.Border)
ui.Icon("circle", 12, th.Accent)
ui.Image("logo", pngBytes).Width(120)
ui.ImageFile("photo", "cat.jpg").Width(240)
ui.ImageFS("logo", assets, "logo.png").Height(48)
ui.Image("hero", data).Frame(320, 180).Fit(ui.FitContain) // FitCover, FitFill
ui.SVG("mark", svgBytes).Width(96)
ui.SVGFile("logo", "assets/logo.svg").Width(120)
ui.SVGFS("logo", assets, "logo.svg").Height(48)
ui.SVG("mark", data).Size(48).Style(ui.Spec{}.TextColor(ui.TokenAccent)) // currentColor
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

`MenuItem` fields: `Label`, `OnSelect`, `Shortcut` (right hint), `Disabled`, `Separator` (or `ui.MenuSeparator`), `Checked`, `Color`, `Icon` (drawn before the label; labels of the whole menu indent to line up) and `Danger` (error-colored, for Delete/Remove).

Tooltip/Popover/ContextMenu share `placeAnchor` (preferred side + flip + viewport clamp). Popover has **no scrim** (unlike dialogs). Escape / outside click dismisses Popover and ContextMenu. `TableAction.Tooltip` is shown on action-icon hover.

## Overlay hosts (window services)

```go
c.Dialogs().ShowInfo("Info", "note", func() {})
c.Dialogs().ShowWarning("Warning", "careful", func() {})
c.Dialogs().ShowError("Error", "failed", func() {})
c.Dialogs().ShowAction("Delete?", "Cannot undo.", onYes, onNo)
c.Dialogs().ShowDanger("Remove project?", "Cannot undo.", "Remove", onRemove) // Cancel + error-colored confirm
// DialogAction{Label, Primary, Danger, Disabled, OnClick} for custom footers
c.Dialogs().ShowInput("Name", "placeholder", func(v string) {}, func() {})
c.Dialogs().Show(ui.DialogOpts{
    Title: "Settings", Width: 720, Height: 520,
    Body: func(c *ui.Ctx) ui.View { return ui.Form("prefs", rows...) },
    Actions: []ui.DialogAction{{Label: "Close", Primary: true}},
})
c.Dialogs().Show(ui.DialogOpts{          // height follows the body
    Title: "Options", Width: 480, FitContent: true, MaxHeight: 600, // then the body scrolls
    Body: func(c *ui.Ctx) ui.View { return ui.Form("opts", rows...).Grouped() },
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
            Icon("file").Run(func() { a.open("cmd/app/main.go") }),
        ui.Section("Commands"),
        ui.Cmd("file.save").Title("Save File").Shortcut("⌘S").Icon("save").Run(a.save),
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
- Sections keep their registration order and their items. With a query, matches inside each section are ranked best first (ties keep registration order); an empty query lists everything in registration order.
- Default toggle chord is **Mod+K** (⌘K / Ctrl+K); override with `c.Commands().ToggleChord("⌘P")`.
- `Enabled(false)`: listed but greyed; shortcut does not fire. `Hidden(true)`: shortcut only, omitted from the list.
- Palette: search field, subsequence filter (title/id/group/detail), Up/Down/Enter/Escape, trailing `Kbd` chips.
- Chord strings: `"⌘S"`, `"Mod+K"`, `"Ctrl+Shift+P"`. `Mod`/`⌘`/`Cmd`/`Ctrl` mean primary modifier.
- `Scope("tabs")` keeps an entry out of the default palette; `c.Commands().ShowScope("tabs", "Go to tab…")` opens the palette on just that scope (e.g. a "Go to Tab" list). Scoped shortcuts still fire.
- Rich rows: `Leading(view)` replaces the icon with any small view (badge, color dot); `Trailing("Todos")` adds a right-aligned muted detail; `Alt("⌘↵", fn)` (+ optional `AltLabel("Open and send")`) runs a second action, and closes the palette, when the chord is pressed on the highlighted entry or the entry is clicked with ⌘/Ctrl held.
- Characters matching the query are drawn SemiBold (in the link color, or the foreground on the highlighted row). `ui.FuzzyMatch(s, q)` returns the same matched byte ranges for custom lists.
- `c.Commands().Density(ui.DensityCompact)` uses single-line rows of `Metrics.ControlHeight` (subtitle dropped, trailing detail on the same line), fitting 10+ results; `HintFooter(true)` adds a footer listing ↑↓ / ↵ / the highlighted entry's Alt chord / esc. Both are host-wide; set them in `Body` each frame if different lists need different settings.

```go
ui.Item("req.list").Title("List todos").Scope("requests").
    Leading(ui.Row(ui.Badge("GET").Hue(ui.HueGreen).Mono()).Width(64)). // fixed width lines titles up
    Trailing("Todos").
    Run(open).Alt("⌘↵", openAndSend).AltLabel("Open and send")
c.Commands().Density(ui.DensityCompact).HintFooter(true)
```
- `yoga.KeyHook` remains for one-off keys that are not commands; command Dispatch runs first.

## Retained views (`ui.ViewOf`)

Construct in `Build*`, not in `Body`.
### Editor

The code editor is a separate module, `github.com/yogaui/editor` (package `editor`): `editor.NewEditorFor(path, content)`, placed with `ui.ViewOf(ed)`. Its reference is in that repo's README.

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

`t.Editable` (default true) gates click-to-edit on `TableColEditable` columns; set false for read-only. Prefer `TableColText` when a column should never edit. `t.HighlightSelected` (default true) paints `ListActive` behind selected rows; set false to keep checkbox/selection without a row band.

Background matches Tree: `t.Background` nil = transparent body (parent surface shows through); set a `*render.Color` for an opaque panel. `t.HeaderBackground` nil = `ChromeMuted`; override similarly. Height: `t.MinHeight` (default 200); `t.CollapseEmpty = true` sizes the host to the header only when there are no visible rows.

Column kinds: `TableColText`, `TableColEditable`, `TableColCheckbox`, `TableColActions`, `TableColToggle`, `TableColSecret`, `TableColBadges`. Width `0` = flex.

- `TableColSecret`: an editable column shown as bullets, with an eye button per cell (on row hover, or while revealed) to show the value; `t.ToggleReveal(rowID, colID)` does the same from code. `SecretWhen func(row TableRow) bool` limits masking to some rows (nil = all). Masked values are never highlighted, suggested or matched by the filter; the edit field stays masked but still copies the real value (`TextFieldConfig.CopyMasked`). `t.Masked` remains for values that must never be copied.
- `TableColBadges`: a comma-separated value drawn as muted chips.
- `t.CellView = func(row ui.TableRow, col ui.TableColumn) ui.View` draws any view in a cell (return nil to paint it as usual). Views are built for visible rows each frame and receive clicks before the table.
- `t.ActionsOnHover = true` draws the actions column's buttons only on the hovered row (or selected rows while the table is focused); the column keeps its width.

```go
{ID: "val", Kind: ui.TableColSecret, SecretWhen: func(r ui.TableRow) bool { return r.Cells["secret"] == "1" }},
{ID: "flags", Kind: ui.TableColBadges, Width: 160},
```

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
tree.CollapseAll() // close every branch; selection moves to its top-level row
ui.ViewOf(tree).Grow(1)

ft := ui.NewFileTree(cwd)
ft.OnOpenFile = func(path string) { … }
ft.SetContextMenu(func(path string) []ui.MenuItem { return … })
ft.SetIconFor(func(path string, dir, open bool) (icons.Icon, render.Color) { … }) // per file type; nil = defaults
ft.CollapseAll() // fires OnChange
ft.SetFilter(q)
ui.ViewOf(ft).Grow(1)
```

**Slots and reveal.** `tree.Leading = func(n *ui.TreeNode) ui.View` draws a view in place of a node's icon (return nil to keep the icon); `tree.Trailing` draws one right-aligned in the row (counts, status dots), and the label is clipped before it. Views are built for visible rows each frame.

`tree.Reveal(node)` expands the node's ancestors, makes it the selected and active node, and scrolls it into view (centered when it was off screen); it returns false when the node is not in the tree. The active node (`tree.Active()`, `tree.SetActive(n)`) is highlighted whether or not the tree has focus, like the open file in an editor's sidebar. `tree.Selected()` is the keyboard cursor.

```go
tree.Leading = func(n *ui.TreeNode) ui.View {
    if m, ok := n.Data.(string); ok {
        return ui.Row(ui.Badge(m).Hue(ui.HueGreen).Mono()).Width(64)
    }
    return nil
}
tree.Trailing = func(n *ui.TreeNode) ui.View { return ui.Caption(strconv.Itoa(len(n.Children))) }
tree.Reveal(activeRequestNode)
```

### ListView / Scroll

Prefer `ui.Scroll(id, ui.Column(items...))` for declarative lists. A Scroll fills with `TokenSurface`; `.Background(token)` changes it (e.g. `TokenChrome` in a sidebar). `NewListView` takes `*layout.Element` children (lower level).

```go
ui.Scroll("gallery", ui.Column(...).Gap(th.Spacing.M).Padding(th.Spacing.L)).Grow(1)
```

**Scrollbar mode.** By default (`theme.ScrollAlways`) a scrolling widget reserves a `Metrics.ScrollbarSize` gutter and paints a track and thumb while its content overflows. `theme.ScrollOverlay` instead draws a thin rounded thumb (`Metrics.ScrollbarOverlaySize`, default 6 px; 4 px wider while hovered or dragged) over the content: no track, no reserved width. It shows on scroll, wheel, or pointer movement over the scroll area, and fades out ~800 ms after the last activity; dragging and track clicks still work while it is visible, and a faded bar lets clicks through to the content.

```go
theme.UseScrollbarMode(theme.ScrollOverlay) // app-wide; survives theme.Use
ui.Scroll("resp", body).Scrollbars(ui.ScrollOverlay) // one Scroll
ed.SetScrollbars(ui.ScrollOverlay)                   // also Table, Tree, ScrollView
```

Setting `th.Metrics.ScrollbarMode` directly works too, but lasts only until the next `theme.Use`. Every built-in scroller follows the mode: Scroll, ListView, Editor, Table, Tree/FileTree and the command palette. There is no separate "auto" mode: classic bars already appear only while content overflows.

**Categorical hues.** For things with no status meaning (HTTP methods, environments, tags) use `Hue` instead of `Tone`: `HueRed`, `HueOrange`, `HueYellow`, `HueGreen`, `HueTeal`, `HueBlue`, `HuePurple`, `HuePink`. Every theme carries `th.Categorical[hue]` (text) and `th.CategoricalSurface[hue]` (tinted fill), derived during normalization when a theme does not set them. The text color clears 4.5:1 on `Surface`, `Chrome` and its own fill, so small labels stay readable; draw custom views with the same pair.

**Density.** `theme.UseDensity(theme.DensityCompact)` (26 px controls) or `DensityComfortable` (36 px) scales `Metrics.ControlHeight` and `MenuItemHeight` (so table rows, selects, buttons, menus), `Metrics.TreeRowHeight`, and the whole `Spacing` ramp proportionally, all from the theme's own values; it survives `theme.Use`. `DensityDefault` (32 px) leaves every theme exactly as built. Values set directly on the active theme's `Spacing`/`Metrics` are replaced when the density changes.

**Focus-visible.** The focus scope tracks input modality: a click switches to pointer, a key press other than a Cmd/Ctrl/Alt shortcut switches to keyboard (the starting state). Built-in widgets draw their focus ring only under keyboard modality, like CSS `:focus-visible`; text inputs (TextField, the editor, an EditableLabel being edited, an InputGroup holding a field) still show focus after a click. Read it with `c.Focus().Modality()` (`ui.ModalityPointer` / `ui.ModalityKeyboard`) or `c.Focus().FocusVisible()`, and style custom controls with `When(ui.FocusVisible, ui.Spec{}.Border(ui.TokenFocusRing, 1))`. `When(ui.Focused, …)` still applies on any focus.

## Spec (visual style)

```go
ui.Background(ui.TokenAccent).
    TextColor(ui.TokenAccentForeground).
    Radius(4).
    Border(ui.TokenBorder, 1).
    BorderStyle(ui.BorderSolid). // or BorderDotted, BorderDashed
    Cursor(ui.CursorPointer).
    Padding(8).Gap(8).
    When(ui.Hovered, ui.Background(ui.TokenAccentHover)).
    When(ui.Pressed, ui.Background(ui.TokenAccentPressed).Scale(0.96, 0.96)).
    When(ui.Focused, ui.Spec{}.Border(ui.TokenFocusRing, 1)).
    When(ui.Disabled, ui.Background(ui.TokenChromeMuted).TextColor(ui.TokenForegroundDisabled))

// Per-side / per-corner (Node modifiers chain the same way):
ui.Column(...).Padding(th.Spacing.M).
    BorderBottom(ui.TokenBorder, th.Stroke.Thin).
    RadiusTopRight(th.Radius.Large)
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

Pre-rasterized [Lucide](https://lucide.dev) symbols in `github.com/yogaui/yoga/icons`. Use typed vars so the linker drops unused icons:

```go
import "github.com/yogaui/yoga/icons"

ui.Icon(icons.Search, 16, th.Foreground)
ui.Button("save", ui.Text("Save")).IconStart(icons.Save)
ui.IconButton("settings", icons.Settings)
```

Common: `icons.Plus`, `icons.Search`, `icons.Settings`, `icons.Pencil`, `icons.Trash2`, `icons.X`, `icons.Folder`, `icons.FolderOpen`, `icons.File`, `icons.Code`, `icons.Terminal`, `icons.ChevronDown`, `icons.ChevronRight`, `icons.Sun`. Full browse list: `icons/catalog.All`. Regenerate: `go run ./cmd/generate-lucide`. Custom SVGs: `render.RegisterIcon(name, svg)` before first draw.

## Theme names

`yoga-dark` (default), `yoga-light`, `system`, `yoga-high-contrast`, `yoga-high-contrast-light`, `yoga-colorblind-dark`, `yoga-colorblind-light`, `yoga-midnight`, `github-dark`, `github-light`, `catppuccin`, `catppuccin-latte`, `dracula`, `nord`, `solarized-dark`, `solarized-light`, `gruvbox-dark`, `gruvbox-light`, `monokai`, `tokyo-night`, `tokyo-night-day`, `one-dark`, `one-light`, `everforest-dark`, `everforest-light`. List: `theme.Names()`. Switch: `theme.Use(name)`. OS appearance: `theme.PrefersDark()`. User choice while on system: `theme.Selected()`. Family-aware system follow: `theme.UseSystem(name)`. Status text: `th.ErrorForeground` / `th.WarningForeground` / `th.SuccessForeground` (never the raw `th.Error` hue). Focus rings: `th.FocusRingOn(fill)`. Control outlines: `th.BorderControl`. Editor washes: `th.SearchMatch`, `th.SearchMatchActive`, `th.BracketMatch`, `th.CurrentLine`, `th.IndentGuide`, `th.Caret`, `th.SelectionInactive`. Modal backdrop: `th.Scrim`. Links: `th.Link` / `th.LinkHover`.
