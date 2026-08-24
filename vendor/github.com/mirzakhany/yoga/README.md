<p align="center">
  <img src="docs/assets/logo.svg" width="96" alt="Yoga">
</p>

# Yoga

A from-scratch, cross-platform **native UI framework in Go** (module `github.com/mirzakhany/yoga`, Go 1.26.2). It renders with WebGPU + GLFW and lays out with a pure-Go flex/grid/stack engine.

This README is the human guide to **building UI** with Yoga. AI agents should also load the project skill at [`.cursor/skills/yoga-ui/`](.cursor/skills/yoga-ui/SKILL.md).

## Demos

```bash
go run ./example/todo        # smallest app (todos)
go run ./example/gallery     # editor workspace + widget gallery
go run ./example/catalog     # component gallery (sidebar + showcases)
go run ./example/apitest     # HTTP request tester
go run ./example/chapar      # multi-page nav shell

# Headless (no window / GPU) — same UI pipeline, for CI
go run -tags nogpu ./example/todo
go build ./... && go build -tags nogpu ./... && go test ./...
```

There is no Makefile. `-tags nogpu` is the only special flag: it swaps GPU/GLFW files for stubs.

| Command | What it shows |
|---|---|
| `example/todo` | Form + list, controlled `TextField` / `Checkbox` |
| `example/gallery` | File tree, tabs, code editor, component gallery, dialogs/toasts |
| `example/catalog` | Sidebar catalog of widget categories with live showcases |
| `example/apitest` | Splitter, `Select`, `Editor`, async work + `Animate` |
| `example/chapar` | App chrome: top bar, nav, pages as `Layout` helpers |

## Screenshots

The component catalog (`go run ./example/catalog`) in the default `yoga-dark` theme:

<p align="center">
  <img src="docs/assets/catalog-surfaces.png" alt="Catalog surfaces: cards, alerts, badges, and links">
</p>

<p align="center">
  <img src="docs/assets/catalog-form-rows.png" width="49%" alt="Catalog form rows">
  <img src="docs/assets/catalog-commands.png" width="49%" alt="Command palette">
</p>

## Mental model

Yoga is **retained app state + a declarative tree rebuilt every frame** (the SwiftUI `@State` analogue):

1. Your app is a struct that implements `yoga.App` — one method, `Body(c *ui.Ctx) ui.View`.
2. `Body` runs on every drawn frame. There is no framework-level widget cache.
3. **Durable state** (strings, slices, editors, tables) lives on the app struct.
4. **Micro-state** (hover, caret, scroll, open menus) lives in `c.Widget(id, …)`, keyed by a unique id.
5. `ui.View` is `Layout(c *ui.Ctx) *layout.Element`. The DSL (`Column`, `Button`, …) returns `*ui.Node`, which implements `View`.

Do not keep `*ui.Ctx` after `Body` returns. The runtime owns it and resets it each pass.

The event loop is idle until input or an animation request, so idle CPU stays near zero. Each drawn frame builds the tree twice: once to hit-test input against fresh geometry, then again so paint reflects state those events changed.

## First window

`yoga.Run` creates the window and text engine, **then** calls `build`. Construct anything that measures text (editors, tables) inside `build`, never before `Run`.

```go
package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/ui"
)

type App struct {
	name string
}

func Build() *App { return &App{} }

func (a *App) Body(c *ui.Ctx) ui.View {
	th := c.Theme()
	return ui.Column(
		ui.Title("Hello"),
		ui.TextField("name", a.name).
			Placeholder("Your name").
			OnChange(func(s string) { a.name = s }).
			DefaultFocus().
			Grow(1),
		ui.Button("ok", ui.Text("OK")).Primary().OnClick(func() {}),
	).Gap(th.Spacing.M).Padding(th.Spacing.L).Grow(1).Background(ui.TokenSurface)
}

func main() {
	cfg := yoga.Config{
		Title:  "Hello",
		Width:  640,
		Height: 480,
	}
	if err := yoga.Run(cfg, Build); err != nil {
		panic(err)
	}
}
```

Ship a GPU `main` (`//go:build !nogpu`) and a headless `main` (`//go:build nogpu`) in the same package, as the demos do.

Optional capabilities (detected by type assertion):

- `Close()` — stop workers / close files when the window closes.
- `OnKey(c *ui.Ctx, k input.KeyEvent) bool` (`yoga.KeyHook`) — app-global shortcuts before focus routing; return `true` to consume.

## Building a tree

The ergonomic API is package `ui`.

### Layout

| Constructor | Layout |
|---|---|
| `ui.Column(children...)` | Vertical flex (children stretch on the cross axis) |
| `ui.Row(children...)` | Horizontal flex (children **vertically centered**) |
| `ui.Stack(children...)` | Layered (z-order) |
| `ui.Center(child)` | Fill parent, center child |
| `ui.Spacer()` | Flex-grow filler |
| `ui.Grid(cols, children...)` | Equal-width columns |
| `ui.Scroll(id, child)` | Clip and scroll; `id` remembers offset |
| `ui.ViewOf(v)` | Wrap any `View` so modifiers chain |
| `ui.Raw(el)` | Wrap a bare `*layout.Element` |
| `ui.HLine` / `ui.VLine` | Rules |
| `ui.Icon(icon, size, color)` | Atlas sprite |
| `ui.Image(id, data)` | PNG/JPEG bitmap (`ImageFile`, `ImageFS`) |
| `ui.SVG(id, data)` | Custom SVG (`SVGFile`, `SVGFS`; `currentColor` follows text color) |

Children are `ui.View`. `nil` is skipped. Split UI into helpers that return `ui.View`.

Chain modifiers on `*ui.Node`:

```text
.Gap .Padding .PaddingXY .PaddingLeft/Right/Top/Bottom
.Margin .MarginXY .MarginLeft/Right/Top/Bottom
.Wrap .Grow .Shrink .Width .Height .Frame .Size
.Align .Justify .Style .Background .BackgroundColor
.Disabled .DefaultFocus
```

Alignment: `ui.AlignStart|Center|End|Stretch`, `ui.JustifyStart|Center|End|Between`.

A pane that should fill leftover space needs `.Grow(1)` on itself **and** every ancestor up to the Body root. Root trees almost always end with `.Grow(1).Background(ui.TokenSurface)`. A `Row` whose children should stretch vertically needs `.Align(ui.AlignStretch)`.

Use `c.Theme().Spacing` (`XXS` 2px … `XXXL` 32px) instead of magic numbers. Typical: `S` 8, `M` 12, `L` 16.

### Controls are controlled

Pass the current value every frame. Callbacks write back to the app struct. Give every interactive widget a **stable unique `id`** (hover/caret/scroll are stored under that key).

```go
ui.TextField("url", app.url).
    Placeholder("https://…").
    IconStart(icons.Search).
    OnChange(func(s string) { app.url = s }).
    OnSubmit(app.fetch).
    Grow(1)

ui.Button("send", ui.Text("Send")).Primary().Hint("⌘↵").OnClick(app.fetch)
ui.Checkbox("notify", "Notify").Check(app.notify).OnToggle(func(v bool) { app.notify = v })
ui.Radio("ra", "A").Check(app.mode == 0).OnClick(func() { app.mode = 0 })
ui.Select("lang", opts).Width(200).Selected(i).OnChange(func(v string) { app.lang = v })
```

Button variants: default **Secondary**; `.Primary()`, `.Subtle()`, and `.Ghost()`. Ghost is text-like (no padding or chrome) for footers and status bars; chain `.HoverFill()` for a hover background. Supports `.IconStart()` and `.Tooltip()`. Icon-only: `ui.IconButton(id, icons.Settings)`.

Typography: `Text`, `Title`, `Subtitle`, `Caption`, `Strong`, `Muted`. Color inherits from the parent (for example a button’s label uses the button’s text token) unless you override with `.Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))`.

### Heavy widgets: construct once

`Editor`, `Table`, `Tree`, `FileTree`, and `ListView` keep real state. Build them in `Build*`, then place them with `ui.ViewOf`:

```go
func Build() *App {
	app := &App{}
	app.editor = ui.NewEditorFor("main.go", src)
	app.table = ui.NewTable(cols, actions)
	return app
}

func (a *App) Body(c *ui.Ctx) ui.View {
	return ui.Column(
		ui.ViewOf(a.editor).Grow(1),
	).Grow(1)
}
```

Constructing these inside `Body` resets caret, scroll, and selection every frame.

### Splitter, drawer, tabs, nav, menus

```go
ui.Splitter("split", ui.Horizontal, sidebar, main).Sizes(240, 0).Grow(1)
// 0 = flex; drag sizes persist under the id. Axis: Horizontal | Vertical.

ui.Drawer("inspector", panel, page).Open(open).Edge(ui.EdgeRight).Overlay().Size(320).Grow(1)
// .Push() shrinks page; .Modal(true) adds scrim (overlay); .Swipe(true) for drag open/close.
// Panel view should use .Grow(1); drawer owns main-axis size and clips overflow.
// Nest drawers for IDE-style panels (e.g. bottom terminal + right chat).

ui.Tabs("tabs", tabs).Selected(i).OnSelectItem(onSelect).OnTabClose(onClose)
// Section switcher without close buttons:
ui.Tabs("sections", tabs).Selected(i).OnSelectItem(onSelect).Closable(false)

ui.Nav("nav", ui.NavVertical, ui.NavIconTop, items...).
    Selected(i).OnSelectItem(func(i int, id string) { … }).Width(88)

ui.Dropdown("file", "File", []ui.MenuItem{{Label: "Save", OnSelect: save}})
ui.MenuButton("export", "Export", items).Primary().IconStart(icons.Save) // click opens menu
ui.MenuButton("save", "Save", items).Primary().OnClick(save)        // split: label=action, chevron=menu
```

### Dialogs and toasts

Window-owned hosts on `Ctx`. `BuildFrame` lays them out — do not put them in Body:

```go
c.Toasts().Show("Saved", ui.ToastInfo, 3*time.Second)
c.Dialogs().ShowInfo("Info", "helpful note", nil)
c.Dialogs().ShowWarning("Warning", "check before continuing", nil)
c.Dialogs().ShowError("Error", "request failed", nil)
c.Dialogs().ShowAction("Delete?", "This cannot be undone.", onYes, onNo)
c.Dialogs().ShowInput("Name", "placeholder", onOK, onCancel)

c.Dialogs().Show(ui.DialogOpts{
	Title:  "Settings",
	Width:  720,
	Height: 520,
	Body: func(c *ui.Ctx) ui.View {
		return ui.Row(
			ui.Nav("cats", ui.NavVertical, ui.NavIconLeft, items...).Width(200),
			ui.VLine(th.Stroke.Thin, th.Border),
			ui.Scroll("form", ui.Form("settings", rows...)).Grow(1),
		).Align(ui.AlignStretch).Grow(1)
	},
	Actions: []ui.DialogAction{{Label: "Close", Primary: true}},
})
```

Custom dialogs use the same overlay/modal behavior as the file picker. Escape runs `OnDismiss`. Footer actions close the dialog before `OnClick`.

### Form and Switch

`ui.Form` renders labeled settings rows (icon, title, description, control):

```go
ui.Form("prefs",
	ui.FormSwitch("notify", "Notifications", "Show alerts", on, func(v bool) { on = v }),
	ui.FormSelect("theme", "Theme", "Color scheme", opts, idx, onChange),
	ui.FormNumber("size", "Font size", "Editor size in pt", 14, 10, 24, 1, onSize),
	ui.FormText("file", "Default file", "Open on startup", name, onName),
)
```

`ui.Switch(id).Check(on).OnToggle(fn)` is an unlabeled pill toggle for compact rows.

### File dialog

Pure-Go file/folder picker (`ui.FileDialog`) — no native OS dialogs. Call `c.Files().Show` with options:

```go
c.Files().Show(ui.FileDialogOpts{
	Mode: ui.FileDialogOpenFile, // FileDialogOpenFolder | FileDialogSaveFile
	Title: "Open File",
	Dir: "/path/to/start", // optional; defaults to home
	Multiple: false,
	Filters: []ui.FileFilter{
		{Label: "Go files", Exts: []string{".go"}},
		{Label: "All files", Exts: nil},
	},
	ShowHidden: false,
	ShowSaveFilter: true,    // file-type filter in save mode (default off)
	AllowCreateFolder: true, // New Folder button in the footer
	OnConfirm: func(paths []string) { … },
	OnCancel: func() { … },
})
```

**Modes.** `FileDialogOpenFile` lists files (double-click or Open confirms). `FileDialogOpenFolder` selects folders only. `FileDialogSaveFile` adds a filename field; Save confirms the path and applies the selected filter extension when the name has none.

**Layout.** Places sidebar, breadcrumb + searchable file table in the main pane, and a footer with filename (save mode), file-type filter, optional **New Folder** (next to Cancel/Save), and Cancel + Open/Select/Save. When creating a folder, an inline name field with Create/Cancel appears in the footer.

**Keyboard.** Escape cancels. Enter confirms when valid. Tab cycles within the modal dialog.

See `example/gallery/gallery.go` for open file, multi-select, folder pick, save, and a settings dialog demo.

## Theme

There is one live `*theme.Theme` (`theme.Current()`). `theme.Use("yoga-light")` overwrites it in place, so the next paint picks up the new palette with no rebuild.

Prefer **tokens** (`ui.TokenSurface`, `ui.TokenChrome`, `ui.TokenForeground`, `ui.TokenAccent`, `ui.TokenBorder`, `ui.TokenError`, …) over literal colors so chrome follows the theme.

```go
ui.Column(...).Background(ui.TokenChrome)
ui.Text("status").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))
```

Default theme is `yoga-dark`. If you switch before opening the window, call `theme.Use` **before** `yoga.Run` (see `example/apitest`).

Shipped names include `yoga-dark`, `yoga-light`, `yoga-midnight`, `github-dark`, `catppuccin`, `dracula`, `nord`, and others (`theme.Names()`).

Spacing, radius, stroke, and type ramps live on `c.Theme()` (`th.Spacing.M`, `th.Radius.Medium`, `th.Stroke.Thin`, `th.Typography.Body`).

Icons are pre-rasterized [Lucide](https://lucide.dev) symbols from `github.com/mirzakhany/yoga/icons` (e.g. `icons.Search`, `icons.Plus`, `icons.ChevronDown`). Unused icons are dropped by the Go linker. The full catalog lives in `icons/catalog` for the component gallery. Regenerate with `go run ./cmd/generate-lucide`. Custom SVGs: `render.RegisterIcon(name, svg)` before first draw.

Lucide is licensed under the ISC License.

## Input, focus, overlays, async

- **Focus.** Interactive DSL widgets register themselves. Tab order is Layout order. `.DefaultFocus()` / `c.Focus().EnsureFocus(w)` picks a fallback when nothing is focused. Open dialogs call `BeginModal` then `SetModal` so Tab and keys stay inside the dialog.
- **Overlays.** Menus, selects, dialogs, and toasts call `c.Overlay(el)`. Overlays paint and hit-test on top of the body.
- **Animation.** `c.Animate(d)` schedules a repaint within `d` (caret blink, spinner, polling). The runtime waits the minimum requested duration, or sleeps until the next OS event.
- **Background work.** Safe to call `c.Invalidate()` from any goroutine (highlight finished, HTTP returned). Typical pattern: a result channel drained at the top of `Body`, plus `c.Animate(30*time.Millisecond)` while a request is in flight (`example/apitest`).
- **Shortcuts.** Read `c.Keyboard()` in `Body`, or implement `yoga.KeyHook` when a focused editor would otherwise eat the key.

Pointer handlers set `m.Consumed = true` to stop bubbling. Overlay hit-testing runs first.

## Custom widgets

Most screens are functions that return `ui.View`. For a retained component, implement `ui.View`:

```go
func (p *Page) Layout(c *ui.Ctx) *layout.Element {
	return ui.Column(/* … */).Grow(1).Layout(c)
}
```

Chapar-style pages return `ui.View` from a helper also named `Layout` and are composed by the shell — that helper is *not* the `ui.View` interface (which must return `*layout.Element`).

Low-level drawing: `layout.New(layout.Box().…, children…)`, attach `Paint` / `OnMouse`, wrap with `ui.Raw`. Hit-test with `e.Frame.Contains(m.X, m.Y)`.

## Headless / tests

```go
text, _ := shape.NewEngine(1, false)
clip := &input.MemClipboard{}
sheet := render.NewSpriteSheet(text.Atlas)
yoga.SetResources(text, sheet, clip)

c := ui.New(text, ui.NewFocusScope(), nil)
c.SetIcons(sheet)
c.SetClipboard(clip)

root := ui.BuildFrame(c, app.Body, w, h, mouse, keyboard)
layout.Dispatch(root, mouse)
c.Focus().Route(keyboard)
mouse.EndFrame()
keyboard.EndFrame()
root = ui.BuildFrame(c, app.Body, w, h, mouse, keyboard)
layout.Paint(root, drawList, text)
```

## Architecture

Dependency direction is strictly downward:

```
example/*             app: Body(c) trees
  yoga                runtime: window, WebGPU, fonts, event loop
    ui                View DSL, Ctx, focus, widget store
      layout  theme   Element tree, flex/grid/stack, design tokens
        render        DrawList, atlas, WGSL
          input text highlight   mouse/kb, piece table, tree-sitter
```

Cgo exists only in `app_gpu.go` (GLFW + WebGPU) and `render/renderer.go`. Everything else is pure Go. `layout.Element` is the node type; `Calculate` solves then flattens to absolute `Frame`s; `Dispatch` delivers mouse events front-to-back.

## Agent skill

For Cursor (and any agent you paste into):

- [`.cursor/skills/yoga-ui/SKILL.md`](.cursor/skills/yoga-ui/SKILL.md) — rules and bootstrap
- [`.cursor/skills/yoga-ui/widgets.md`](.cursor/skills/yoga-ui/widgets.md) — constructors
- [`.cursor/skills/yoga-ui/examples.md`](.cursor/skills/yoga-ui/examples.md) — patterns from `example/`
