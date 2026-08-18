# Yoga UI examples (`cmd/`)

Copy these shapes. Do not invent a parallel widget API.

## Smallest app — `cmd/todo`

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
				IconStart("add").
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

GPU main: `yoga.Run(cfg, BuildTodoApp)` with `ClearColor: theme.Current().Background`.

## App shell — `cmd/example`

Page enum + subviews that return `ui.View`. Dialog/toast hosts live on the shell and are always in the tree.

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
		app.dialogs,
		app.files,
		app.toasts,
	).Grow(1).Background(ui.TokenSurface)
}
```

Editor workspace: menu `Dropdown`s, `FileTree` + search `TextField`, `Tabs` + `ViewOf(editor)`, `Splitter`, status bar. Editors constructed with `ui.NewEditorFor` when a file opens — **not** rebuilt each frame.

```go
ui.Splitter("editor-split", ui.Horizontal, explorer, editorCol).Sizes(240, 0).Grow(1)
```

Theme menu: `theme.Names()` → `theme.Use(n)`.

## Widget gallery — `cmd/example/gallery.go`

Scrollable column of every control. Table constructed in `buildComponentGallery`, then `ui.ViewOf(g.kvTable).Height(220)`.

Toasts/dialogs/file picker: `g.toasts.Show(msg, ui.ToastInfo, 3*time.Second)`, `g.dialogs.ShowError(...)`, `g.files.Show(ui.FileDialogOpts{...})`.

## HTTP tester — `cmd/apitest`

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

`theme.Use("yoga-midnight")` in `main` **before** `Config.ClearColor`.

## Multi-page shell — `cmd/chapar`

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
- Forgetting to put `DialogHost`/`FileDialog`/`ToastHost` in the tree.
- Capturing loop variables without `item := it`.
- Reusing the same widget `id` for rows in a list (`fmt.Sprintf("todo-%d", id)`).
