# Yoga UI examples (`example/`)

Copy these shapes. Do not invent a parallel widget API.

## Smallest app — `example/todo`

Retained slice + draft string. DSL-only widgets. DefaultFocus on the field.

```go
func (app *TodoApp) Body(c *ui.Ctx) ui.View {
	th := c.Theme()
	rows := make([]ui.View, 0, len(app.items))
	for _, it := range app.items {
		item := it // capture
		cb := ui.Checkbox(item.id, item.title).
			Check(item.done).
			OnToggle(func(v bool) { app.setDone(item.id, v) })
		if item.done {
			cb.LabelMuted(true).LabelStrike(true)
		}
		rows = append(rows, cb)
	}
	return ui.Column(
		ui.Title("Todos"),
		ui.Row(
			ui.TextField("draft", app.draft).
				Placeholder("Add a todo and press Enter...").
				IconStart(icons.Plus).
				OnChange(func(s string) { app.draft = s }).
				OnSubmit(func(s string) { app.addTodo(s) }).
				DefaultFocus().
				Grow(1),
			ui.Button("add", ui.Text("Add")).Primary().OnClick(func() { app.addTodo(app.draft) }),
		).Gap(th.Spacing.S),
		ui.Column(rows...).Gap(th.Spacing.S).Grow(1),
	).Gap(th.Spacing.M).Grow(1).Padding(th.Spacing.L).Background(ui.TokenSurface)
}
```

GPU main: `yoga.Run(cfg, BuildTodoApp)`.

## Custom title bar — `example/catalog`

Opt in at window creation, then compose the bar with existing widgets. The catalog demo uses this for its File/Go/Help menus and command palette.

```go
// main_gpu.go
cfg := yoga.Config{
	Title:          "Yoga Components",
	Width:          1100,
	Height:         720,
	CustomTitleBar: true,
}
yoga.Run(cfg, BuildCatalog)

// app.go — menus, search, theme picker in the title bar
func (app *CatalogApp) topBar(c *ui.Ctx) ui.View {
	return ui.TitleBar(
		ui.Dropdown("menu-file", "File", fileItems),
		ui.Dropdown("menu-go", "Go", app.goMenuItems()),
		ui.Spacer(),
		ui.Button("cmd-palette", ui.Text("Commands")).Width(300).
			IconStart(icons.Search).
			Hint(c.Commands().ToggleLabel()).
			OnClick(func() { c.Commands().Show() }),
		ui.Select("theme", themes).Width(180).Selected(idx).OnChange(theme.Use),
	)
}
```

`TitleBar` auto-sizes child controls to `TitleBarControlHeight` (26px) with vertical centering. macOS keeps native traffic lights; Windows/Linux get framework min/max/close. Drag empty area to move; double-click toggles maximize.

## App shell — `example/gallery`

Page enum + subviews that return `ui.View`. Dialogs, file picker, and toasts are window services: `c.Dialogs()`, `c.Files()`, `c.Toasts()`. Do not put hosts in the tree.

```go
func (app *AppShell) Body(c *ui.Ctx) ui.View {
	var content ui.View
	switch app.page {
	case pageComponents:
		content = app.gallery.Layout(c)
	default:
		content = app.editor.Layout(c)
	}
	return ui.Column(
		ui.Row(
			ui.Nav("shell-nav", ui.NavVertical, ui.NavIconTop,
				ui.NavItem{ID: "editor", Label: "Editor", Icon: "edit"},
				ui.NavItem{ID: "gallery", Label: "Components", Icon: "code"},
			).Selected(int(app.page)).OnSelectItem(func(i int, _ string) {
				app.page = appPage(i)
			}).Width(88),
			content,
		).Align(ui.AlignStretch).Grow(1),
	).Grow(1).Background(ui.TokenSurface)
}
```

Editor workspace: menu `Dropdown`s, `FileTree` + search `TextField`, `Tabs` + `ViewOf(editor)`, `Splitter`, status bar. Editors constructed with `ui.NewEditorFor` when a file opens — **not** rebuilt each frame.

```go
ui.Splitter("editor-split", ui.Horizontal, explorer, editorCol).Sizes(240, 0).Grow(1)
```

Theme menu: `theme.Names()` → `theme.Use(n)`.

## Widget gallery — `example/gallery/gallery.go`

Scrollable column of every control. Table constructed in `buildComponentGallery`, then `ui.ViewOf(g.kvTable).Height(220)`.

Toasts/dialogs/file picker: `c.Toasts().Show(msg, ui.ToastInfo, 3*time.Second)`, `c.Dialogs().ShowInfo` / `ShowWarning` / `ShowError` / `ShowAction` / `ShowInput`, `c.Files().Show(ui.FileDialogOpts{...})`.

Settings dialog (sidebar nav + `ui.Form`):

```go
c.Dialogs().Show(ui.DialogOpts{
    Title: "Settings", Width: 720, Height: 520,
    Body: func(c *ui.Ctx) ui.View {
        th := c.Theme()
        return ui.Row(
            ui.Nav("settings-cats", ui.NavVertical, ui.NavIconLeft, items...).Width(200),
            ui.VLine(th.Stroke.Thin, th.Border),
            ui.Column(ui.Subtitle(catName), ui.Scroll("form", form).Grow(1)).Grow(1),
        ).Align(ui.AlignStretch).Grow(1)
    },
    Actions: []ui.DialogAction{{Label: "Close", Primary: true}},
})
```

Form rows: `ui.FormSwitch`, `FormSelect`, `FormNumber`, `FormText`.

File dialog modes and footer options:

```go
c.Files().Show(ui.FileDialogOpts{
    Mode:              ui.FileDialogSaveFile,
    ShowSaveFilter:    true,
    AllowCreateFolder: true, // New Folder in footer, after filter
    Filters: []ui.FileFilter{
        {Label: "Go files", Exts: []string{".go"}},
        {Label: "All files", Exts: nil},
    },
    OnConfirm: func(paths []string) { … },
})
// FileDialogOpenFile (Multiple for multi-select), FileDialogOpenFolder
```

## HTTP tester — `example/apitest`

Toolbar `Row` of Select + TextField + Segmented + Primary button with `.Hint("⌘↵")`. Body polls a result channel and `c.Animate(pendingPoll)` while in flight.

```go
select {
case r := <-app.resultCh:
	app.handleResult(r)
default:
}
if app.pending {
	c.Animate(pendingPoll)
}
```

Editors (`NewEditor` + highlighter) live on the app. Panes are helper methods returning `ui.View`. Split axis toggled via Segmented; `Sizes(0, splitFixedSize)`.

Select option tints: `.OptionColor("GET", th.Success)`.

Global shortcut in `Body` via `c.Keyboard()` (Cmd/Ctrl+Enter). Prefer `yoga.KeyHook` for shortcuts that must run even when a field captures keys.

`theme.Use("yoga-midnight")` in `main` **before** `yoga.Run`.

## Section box with border

Wrap any group of widgets to draw a chrome box around them:

```go
ui.Column(
    ui.Subtitle("Details"),
    ui.Text(app.details),
).Gap(th.Spacing.S).Padding(th.Spacing.M).
    Radius(th.Radius.Medium).
    Border(ui.TokenBorder, th.Stroke.Thin).
    Background(ui.TokenChrome)

// Bottom divider only:
ui.Row(...).BorderBottom(ui.TokenBorder, th.Stroke.Thin)

// Dotted accent rail:
ui.Column(...).BorderLeft(ui.TokenAccent, th.Stroke.Thick).BorderStyle(ui.BorderDotted)
```

## Multi-page shell — `example/chapar`

Subcomponents with `Layout(c *ui.Ctx) ui.View` (returns a Node, not `*layout.Element`). Nav bar owned by the shell; current page content `ui.ViewOf(currentPage.Layout(c)).Grow(1)`.

```go
return ui.Column(
	app.topBar.Layout(c),
	ui.HLine(1, th.Border),
	ui.Row(
		app.navigationBar.Layout(c),
		ui.VLine(1, th.Border),
		ui.ViewOf(currentPage.Layout(c)).Grow(1),
	).Align(ui.AlignStretch).Grow(1),
	ui.HLine(1, th.Border),
	app.footer.Layout(c),
).Grow(1)
```

Environments page: `Tree` with `Loader`, search field, Splitter sidebar.

## Headless (`-tags nogpu`)

```go
text, err := shape.NewEngine(1, false)
clip := &input.MemClipboard{}
sheet := render.NewSpriteSheet(text.Atlas)
yoga.SetResources(text, sheet, clip)

app := BuildTodoApp()
c := ui.New(text, ui.NewFocusScope(), nil)
c.SetIcons(sheet)
c.SetClipboard(clip)

root := ui.BuildFrame(c, app.Body, w, h, mouse, keyboard)
layout.Dispatch(root, mouse)
c.Focus().Route(keyboard)
mouse.EndFrame()
keyboard.EndFrame()
root = ui.BuildFrame(c, app.Body, w, h, mouse, keyboard) // paint reflects input
layout.Paint(root, drawList, text)
```

GPU runtime already does two `BuildFrame` passes per drawn frame (hit-test, then paint).

## Anti-patterns seen in reviews

- `NewEditor` / `NewTable` inside `Body` — leaks and resets caret/scroll.
- Uncontrolled TextField (`OnChange` only, never passing the stored string back).
- Hardcoded `render.RGBA8` for chrome — use tokens so `theme.Use` works.
- Forgetting `.Grow(1)` on the chain from expanding content to the Body root.
- Threading `DialogHost`/`FileDialog`/`ToastHost` through the app; use `c.Dialogs()` / `c.Files()` / `c.Toasts()`.
- Capturing loop variables without `item := it`.
- Reusing the same widget `id` for rows in a list (`fmt.Sprintf("todo-%d", id)`).
