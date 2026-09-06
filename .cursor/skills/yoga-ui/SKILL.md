---
name: yoga-ui
description: >-
  Build desktop UI with the Yoga Go framework (github.com/mirzakhany/yoga):
  yoga.App, ui.View, Column/Row, widgets, theme tokens, overlays, focus.
  Use when writing or changing Yoga apps, widgets, example/* demos, or anything
  that implements Body(c *ui.Ctx) ui.View.
---

# Yoga UI — agent instructions

Module `github.com/mirzakhany/yoga`. Public UI surface is **`ui/`**. Read [widgets.md](widgets.md) for constructors and [examples.md](examples.md) for `example/` patterns.

## Mental model

Retained app state + per-frame declarative rebuild (SwiftUI `@State` analogue):

1. App struct holds durable state (lists, strings, `*ui.Editor`, `*ui.Table`).
2. `Body(c)` runs **every drawn frame** and returns a `ui.View` tree. There is no host widget cache.
3. `ui.View` is `Layout(c *ui.Ctx) *layout.Element`. `*ui.Node` implements it.
4. Widget **micro-state** (hover, caret, scroll, open menu) lives in `c.Widget(id, alloc)` keyed by a **unique-per-window id**. App data does not.
5. Do not retain per-frame `*ui.Ctx` fields across frames. The Ctx pointer is window-lifetime: `c.Dialogs()`, `c.Files()`, `c.Toasts()`, `c.Focus()`, and `c.Invalidate()` are safe to capture in OnClick.

Heavy widgets (`Editor`, `Table`, `Tree`, `FileTree`, `ListView`) are constructed **once** in `Build*` and placed with `ui.ViewOf(w)`. Stateless DSL nodes (`Column`, `Button`, `TextField`, …) may be allocated in `Body`. Dialogs, file picker, and toasts are window services on `c` — do not construct or thread hosts.

## Bootstrap

`yoga.Run` creates the window/text engine **then** calls `build`. Construct widgets that measure text inside `build`, never before `Run`.

```go
package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/ui"
)

type App struct{ name string }

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
	cfg := yoga.Config{Title: "App", Width: 640, Height: 480}
	if err := yoga.Run(cfg, Build); err != nil { panic(err) }
}
```

GPU entry: `//go:build !nogpu` + `yoga.Run`. Headless: `//go:build nogpu` + `shape.NewEngine` → `yoga.SetResources` → `ui.BuildFrame`. Dual mains live side-by-side in `example/*/`.

Optional App capabilities (type-asserted by the runtime):

- `Close()` — release workers/files when the window closes.
- `OnKey(c *ui.Ctx, k input.KeyEvent) bool` (`yoga.KeyHook`) — global shortcuts after command Dispatch and before focus routing; return true to consume. Prefer `c.Commands().Register` for palette-visible actions.

## Layout DSL (`*ui.Node`)

| Constructor | Behavior |
|---|---|
| `Column(children...)` | Vertical flex, stretch cross-axis |
| `Row(children...)` | Horizontal flex, **vertically centered** |
| `Stack(children...)` | Z-order layers |
| `Center(child)` | Fill parent, center child |
| `Spacer()` | Flex-grow filler |
| `Grid(cols, children...)` | Equal-width columns (`fr` tracks) |
| `Scroll(id, child)` | Clip + scroll; `id` keys offset |
| `ViewOf(v)` | Wrap any `View` so Node modifiers chain |
| `Raw(el)` | Wrap a bare `*layout.Element` |
| `HLine(thick, color)` / `VLine(...)` | Rules |
| `Icon(icon, size, color)` | Lucide atlas sprite |

Children are `ui.View`. Nil children are skipped. Split a pane into helpers that return `ui.View`.

Fluent modifiers (chain on `*Node`): `Gap`, `Padding`/`PaddingXY`/`PaddingLeft|Right|Top|Bottom`, `Margin`/`MarginXY`/sides, `Wrap`, `Grow`, `Shrink`, `Width`, `Height`, `Frame`, `Size` (font size on `Text`, else square), `Align`, `Justify`, `Style(Spec)`, `Background(Token)`, `BackgroundColor`, `Disabled`, `DefaultFocus`.

Alignment constants: `ui.AlignStart|Center|End|Stretch`, `ui.JustifyStart|Center|End|Between`.

Fill remaining space with `.Grow(1)` on the expanding child **and** its ancestors up to the window root. Root Body trees almost always `.Grow(1).Background(ui.TokenSurface)`.

Use `c.Theme().Spacing.*` (`XXS` 2 … `XXXL` 32) — not magic numbers. Common: `S` 8, `M` 12, `L` 16.

## Controlled widgets

Pass current value each frame; callbacks mutate the app struct. Stable unique `id` strings.

```go
ui.TextField("url", app.url).Placeholder("https://…").IconStart(icons.Search).
    OnChange(func(s string) { app.url = s }).OnSubmit(func(s string) { app.go(s) }).Grow(1)

ui.Button("send", ui.Text("Send")).Primary().Hint("⌘↵").IconStart(icons.Play).OnClick(app.send)
ui.Checkbox("n", "Notify").Check(app.on).OnToggle(func(v bool) { app.on = v })
ui.Radio("ra", "A").Check(app.radio == 0).OnClick(func() { app.radio = 0 })
ui.Select("lang", opts).Width(200).Selected(i).OnChange(func(v string) { app.lang = v })
ui.Slider("vol", app.vol).Min(0).Max(100).OnFloatChange(func(v float64) { app.vol = v })
ui.Button("tip", ui.Text("Save")).Tooltip("Save document")
```

Button variants: default **Secondary**; `.Primary()` / `.Subtle()` / `.Ghost()`. Ghost is text-like (no padding or chrome) for footers; chain `.HoverFill()` for a hover background. Supports `.IconStart(icons.Icon)` and `.Tooltip()`. `IconButton(id, icons.Settings)`.

Do **not** construct `NewEditor` / `NewTable` / `NewTree` inside `Body`. Construct in `Build*`, then `ui.ViewOf(app.table).Height(220)` / `.Grow(1)`.

## Theme

`theme.Use(name)` mutates the live theme in place. Default `yoga-dark`. Prefer `ui.Token*` fills so widgets recolor on switch. `c.Theme()` for spacing, radius, stroke, typography, semantic colors (`th.Success`, `th.Error`, …).

```go
ui.Text("hi").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))
ui.Column(...).Background(ui.TokenChrome)
```

Typography helpers: `Text`, `Title`, `Subtitle`, `Caption`, `Strong`, `Muted`. Color/size inherit from parent env (e.g. a Button’s `TextColor`).

Select a theme **before** `yoga.Run` if the app is not dark-default (`example/apitest` uses `yoga-midnight`).

## Overlays, focus, async

- `c.Dialogs()`, `c.Files()`, and `c.Toasts()` are window-owned. `BuildFrame` lays them out after the body — do not put them in the view tree. Capture `c` in OnClick to `Show`.
- `c.Commands()` is the window-owned command registry + searchable palette. Register from `Body` each frame; default toggle is **⌘K** / Ctrl+K. Dispatch runs before `KeyHook` and focus.
- `FileDialog`: pure-Go picker with open file/folder and save modes. Footer holds filename (save), filter, optional New Folder (`AllowCreateFolder`), Cancel, and Open/Select/Save. See [widgets.md](widgets.md).
- `c.Dialogs().Show(DialogOpts)`: custom size, body layout, and footer actions (same modal behavior as FileDialog). `ShowInfo` / `ShowWarning` / `ShowError` / `ShowAction` / `ShowInput` are built on this path.
- `Form`: labeled settings rows (switch, select, number, text, slider, stepper). `Switch`: pill toggle for compact rows.
- Anchored overlays: `.Tooltip(text)` on any node; `Popover` (no scrim, click trigger to toggle); `ContextMenu` (right-click → `Menu`). Dropdowns/Selects/Menus call `c.Overlay` themselves. **Do not use `Dialog` for lightweight history panels** — use `Popover` anchored to a footer/toolbar button with `.Placement(ui.PlacementTop)` when the trigger sits at the bottom of the window.
- `c.Focus().EnsureFocus(w)` / `.DefaultFocus()` on a control when nothing is focused. Tab order = Layout registration order.
- Background work: mutate app state, then `c.Invalidate()` (any goroutine). Capture `c` only for the current frame’s `Invalidate` closure, or keep a wake func — prefer storing results on the app and calling `Invalidate` from a handle the runtime already has. Pattern in `example/apitest`: poll a channel in `Body`, `c.Animate(30*time.Millisecond)` while pending.
- Caret blink / spinner / progress / skeleton: widget calls `c.Animate(d)` during Layout.

## Custom widgets

Implement `ui.View`:

```go
func (w *Panel) Layout(c *ui.Ctx) *layout.Element {
	return ui.Column(...).Grow(1).Layout(c)
}
```

Low-level: `layout.New(layout.Box().…, children...)`, set `Paint` / `OnMouse`, wrap with `ui.Raw`. Hit-test: `e.Frame.Contains(m.X, m.Y)`; stop bubbling with `m.Consumed = true`. Overlays: `el.Overlay = true` then `c.Overlay(el)`.

Store hover in `c.Widget(id, func() any { return &state{} })`.

## Rules of thumb

- Unique widget ids (`"send"`, `"todo-%d"`). Colliding ids share hover/caret.
- Controlled values from the app; never treat TextField as owning the string.
- `Row` children that should stretch vertically: parent `.Align(ui.AlignStretch)`.
- Splitter: `ui.Splitter(id, ui.Horizontal|Vertical, a, b).Sizes(240, 0).Grow(1)` — `0` means flex.
- Drawer: `ui.Drawer(id, panel, page).Open(v).Edge(ui.EdgeRight).Overlay().Size(320).Grow(1)` — or `.Push()`; nest for IDE-style terminal + chat; `.Swipe(true)` for drag gestures. Panel content fills a clipped viewport — use `.Grow(1)` on the panel view, not fixed width/height (the drawer owns main-axis size).
  - **Bottom console / terminal:** `.Edge(ui.EdgeBottom).Push().Resizable(true)` — wrap only the **editor/workspace** pane, not the whole page (e.g. inside a `Splitter` right child so the sidebar tree stays full height). Toggle `.Open(v)` from app state; sync with `.OnOpenChange`.
  - **Avoid `ui.Stack` at the app root** for overlays — defaults center children and shrink the UI. Use `Drawer`, `Popover`, or window services (`c.Dialogs()`, `c.Overlay`) instead.
- Icons: Lucide symbols from `github.com/mirzakhany/yoga/icons` (`icons.Search`, `icons.Plus`, `icons.Settings`, …). Full list in `icons/catalog` for the component gallery. Regenerate: `go run ./cmd/generate-lucide`.
- After async HTTP/highlight: `Invalidate` or `Animate`; idle loop otherwise waits forever.
- Tests/CI: `go test ./...` and `go build -tags nogpu ./...`.

## Example map

| Package | Why read it |
|---|---|
| `example/todo` | Smallest complete app: form, list, controlled fields |
| `example/gallery` | Shell + editor workspace + widget gallery |
| `example/catalog` | Sidebar catalog: grouped nav + per-widget live showcases |
| `example/apitest` | Splitter, editors, Select colors, pending Animate |
| `example/chapar` | Multi-page nav shell, sub-`Layout` helpers |

## Additional resources

- Widget constructors and modifiers: [widgets.md](widgets.md)
- Copied patterns from `example/`: [examples.md](examples.md)
- Human overview: [README.md](../../../README.md)
