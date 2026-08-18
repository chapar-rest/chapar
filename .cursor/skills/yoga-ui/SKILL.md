---
name: yoga-ui
description: >-
  Build desktop UI with the Yoga Go framework (github.com/mirzakhany/yoga):
  yoga.App, ui.View, Column/Row, widgets, theme tokens, overlays, focus.
  Use when writing or changing Yoga apps, widgets, cmd/* demos, or anything
  that implements Body(c *ui.Ctx) ui.View.
---

# Yoga UI — agent instructions

Module `github.com/mirzakhany/yoga`. Public UI surface is **`ui/`**. Read [widgets.md](widgets.md) for constructors and [examples.md](examples.md) for `cmd/` patterns.

## Mental model

Retained app state + per-frame declarative rebuild (SwiftUI `@State` analogue):

1. App struct holds durable state (lists, strings, `*ui.Editor`, `*ui.Table`, hosts).
2. `Body(c)` runs **every drawn frame** and returns a `ui.View` tree. There is no host widget cache.
3. `ui.View` is `Layout(c *ui.Ctx) *layout.Element`. `*ui.Node` implements it.
4. Widget **micro-state** (hover, caret, scroll, open menu) lives in `c.Widget(id, alloc)` keyed by a **unique-per-window id**. App data does not.
5. Do not retain `*ui.Ctx` across frames. The runtime resets it each pass.

Heavy widgets (`Editor`, `Table`, `Tree`, `FileTree`, `ListView`, `DialogHost`, `FileDialog`, `ToastHost`) are constructed **once** in `Build*` and placed with `ui.ViewOf(w)`. Stateless DSL nodes (`Column`, `Button`, `TextField`, …) may be allocated in `Body`.

## Bootstrap

`yoga.Run` creates the window/text engine **then** calls `build`. Construct widgets that measure text inside `build`, never before `Run`.

```go
package main

import (
	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
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

func (a *App) ClearColor() render.Color { return theme.Current().Background }

func main() {
	cfg := yoga.Config{Title: "App", Width: 640, Height: 480, ClearColor: theme.Current().Background}
	if err := yoga.Run(cfg, Build); err != nil { panic(err) }
}
```

GPU entry: `//go:build !nogpu` + `yoga.Run`. Headless: `//go:build nogpu` + `shape.NewEngine` → `yoga.SetResources` → `ui.BuildFrame`. Dual mains live side-by-side in `cmd/*/`.

Optional App capabilities (type-asserted by the runtime):

- `ClearColor() render.Color` — framebuffer clear (keep in sync with theme).
- `Close()` — release workers/files when the window closes.
- `OnKey(c *ui.Ctx, k input.KeyEvent) bool` (`yoga.KeyHook`) — global shortcuts before focus routing; return true to consume.

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
| `Icon(name, size, color)` | Named sprite from the atlas |

Children are `ui.View`. Nil children are skipped. Split a pane into helpers that return `ui.View`.

Fluent modifiers (chain on `*Node`): `Gap`, `Padding`/`PaddingXY`/`PaddingLeft|Right|Top|Bottom`, `Margin`/`MarginXY`/sides, `Wrap`, `Grow`, `Shrink`, `Width`, `Height`, `Frame`, `Size` (font size on `Text`, else square), `Align`, `Justify`, `Style(Spec)`, `Background(Token)`, `BackgroundColor`, `Disabled`, `DefaultFocus`.

Alignment constants: `ui.AlignStart|Center|End|Stretch`, `ui.JustifyStart|Center|End|Between`.

Fill remaining space with `.Grow(1)` on the expanding child **and** its ancestors up to the window root. Root Body trees almost always `.Grow(1).Background(ui.TokenSurface)`.

Use `c.Theme().Spacing.*` (`XXS` 2 … `XXXL` 32) — not magic numbers. Common: `S` 8, `M` 12, `L` 16.

## Controlled widgets

Pass current value each frame; callbacks mutate the app struct. Stable unique `id` strings.

```go
ui.TextField("url", app.url).Placeholder("https://…").IconStart("search").
    OnChange(func(s string) { app.url = s }).OnSubmit(func(s string) { app.go(s) }).Grow(1)

ui.Button("send", ui.Text("Send")).Primary().Hint("⌘↵").IconStart("play_arrow").OnClick(app.send)
ui.Checkbox("n", "Notify").Check(app.on).OnToggle(func(v bool) { app.on = v })
ui.Radio("ra", "A").Check(app.radio == 0).OnClick(func() { app.radio = 0 })
ui.Select("lang", opts).Width(200).Selected(i).OnChange(func(v string) { app.lang = v })
```

Button variants: default **Secondary**; `.Primary()` / `.Subtle()`. `IconButton(id, iconName)`.

Do **not** construct `NewEditor` / `NewTable` / `NewTree` / hosts inside `Body`. Construct in `Build*`, then `ui.ViewOf(app.table).Height(220)` / `.Grow(1)`.

## Theme

`theme.Use(name)` mutates the live theme in place. Default `yoga-dark`. Prefer `ui.Token*` fills so widgets recolor on switch. `c.Theme()` for spacing, radius, stroke, typography, semantic colors (`th.Success`, `th.Error`, …).

```go
ui.Text("hi").Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted))
ui.Column(...).Background(ui.TokenChrome)
```

Typography helpers: `Text`, `Title`, `Subtitle`, `Caption`, `Strong`, `Muted`. Color/size inherit from parent env (e.g. a Button’s `TextColor`).

Select a theme **before** building `Config.ClearColor` if the app is not dark-default (`cmd/apitest` uses `yoga-midnight`).

## Overlays, focus, async

- Place `*ui.DialogHost`, `*ui.FileDialog`, and `*ui.ToastHost` in the Body tree even when closed; `Layout` self-registers overlays.
- Dropdowns/Selects/Menus call `c.Overlay` themselves.
- `c.Focus().EnsureFocus(w)` / `.DefaultFocus()` on a control when nothing is focused. Tab order = Layout registration order.
- Background work: mutate app state, then `c.Invalidate()` (any goroutine). Capture `c` only for the current frame’s `Invalidate` closure, or keep a wake func — prefer storing results on the app and calling `Invalidate` from a handle the runtime already has. Pattern in `cmd/apitest`: poll a channel in `Body`, `c.Animate(30*time.Millisecond)` while pending.
- Caret blink / spinner: widget calls `c.Animate(d)` during Layout.

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
- Icons: names of `render/assets/icons/*.svg` without `.svg` (`search`, `add`, `settings`, …).
- After async HTTP/highlight: `Invalidate` or `Animate`; idle loop otherwise waits forever.
- Tests/CI: `go test ./...` and `go build -tags nogpu ./...`.

## Cmd map

| Package | Why read it |
|---|---|
| `cmd/todo` | Smallest complete app: form, list, controlled fields |
| `cmd/example` | Shell + editor workspace + widget gallery |
| `cmd/apitest` | Splitter, editors, Select colors, pending Animate |
| `cmd/chapar` | Multi-page nav shell, sub-`Layout` helpers |

## Additional resources

- Widget constructors and modifiers: [widgets.md](widgets.md)
- Copied patterns from `cmd/`: [examples.md](examples.md)
- Human overview: [README.md](../../../README.md)
