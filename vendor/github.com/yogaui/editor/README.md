<p align="center">
  <img src="https://raw.githubusercontent.com/yogaui/yoga/main/docs/assets/logo.svg" width="96" alt="Yoga">
</p>

# Yoga editor

The code editor widget for [Yoga](https://github.com/yogaui/yoga), the Go UI framework: a virtualized, piece-table editor with Tree-sitter highlighting, language-server support, find and replace, folding, multiple cursors and a minimap.

It is a separate module so apps that do not need a code editor skip its Cgo grammars.

[Docs](https://yogaui.github.io/docs/editor/) · [Yoga](https://github.com/yogaui/yoga)

```bash
go get github.com/yogaui/editor
```

| Package | |
|---|---|
| `github.com/yogaui/editor` | The `Editor` widget, its keymap and options |
| `github.com/yogaui/editor/highlight` | Tree-sitter and lexical highlighters; importing it also colors Yoga's Markdown code blocks |
| `github.com/yogaui/editor/lsp` | A small Language Server Protocol client, shared across editors |

```go
import (
	"github.com/yogaui/editor"
	"github.com/yogaui/yoga/ui"
)

type App struct{ ed *editor.Editor }

func Build() *App {
	return &App{ed: editor.NewEditorFor("main.go", src)} // highlighter from the extension
}

func (a *App) Body(c *ui.Ctx) ui.View {
	return ui.ViewOf(a.ed).Grow(1)
}
```

Construct editors in `Build`, never in `Body`: an editor keeps the caret, scroll, undo history and its language server across frames.

## Demos

```bash
go run ./example/gallery     # editor workspace + widget gallery
go run ./example/apitest     # HTTP request tester
go run -tags nogpu ./example/gallery   # headless, for CI
```

## Reference



```go
ed := editor.NewEditor([]byte("{\n}"), highlight.NewJSON())
ed := editor.NewEditorFor(path, content) // highlighter from extension
ui.ViewOf(ed).Grow(1)
ed.Bytes(); ed.Modified(); ed.MarkSaved(); ed.Undo(); ed.Redo(); ed.Close()
```

Virtualized, piece-table, LSP overlay self-registered. `CapturesTab() == true` (Tab inserts indent; Ctrl+Tab moves focus).

**Shortcuts.** Every editor shortcut is a row of `editor.DefaultEditorKeymap()` (`EditorBinding{Action, Mac, Other, When}`), listed in `docs/editor-shortcuts.md`. `Mod` is ⌘ on macOS and Ctrl elsewhere; `Cmd`/`Ctrl` name the exact key. Override per editor with `ed.SetKeymap(km)` or for all with `editor.EditorDefaults.Keymap`; `editor.EditorKeymapTable(km)` gives display labels; `ed.Run(action)` and `ed.ShortcutLabel(action)` let app commands share them.

**Folding.** Blocks fold by brackets (strings and comments skipped), block comments, or indentation. Chevrons show in the gutter while the pointer is over the editor (always for a folded block); a folded block shows `first line { … }` and its placeholder opens it. `ed.Fold(line)`, `Unfold`, `ToggleFold`, `FoldAll`, `UnfoldAll`, `Folded`, `Folds`, `FoldRange`; `editor.WithoutFolding()` turns it off. Folds follow edits, open when an edit or the caret (search, go to definition) lands inside, and animate unless `theme.ReducedMotion()`.

**Find bar.** `ed.OpenFind(replace)` opens it (⌘F / ⌥⌘F, Ctrl+F / Ctrl+H). Its inputs are real `TextInput`s, so select all, word clicks, clipboard and undo work there. Options: match case, whole word, regex (`$1`, `${name}`, `$&` in the replacement), preserve case, and find in selection. Replace All is one undo step. `ed.FindFocused()` tells an app to keep its own editor commands off the document while the bar has the keys.

**Wrapping.** `ed.SetSoftWrap(true)` (or `editor.WithSoftWrap(true)`) wraps long lines at the viewport width and can be toggled at runtime; the caret and the row under it (or the top row, if the caret is off screen) keep their place. Continuation rows carry no line number, and caret movement, selection, clicks and Up/Down all work on visual rows.

**Indentation.** `ed.SetTabWidth(n)` lays tabs out every `n` columns for this editor only (`0` follows `shape.FontConfig.TabWidth`), `ed.SetIndentSpaces(v)` makes Tab insert spaces or a tab, and `ed.SetAutoIndent(m)` picks what Enter carries to the new line: `editor.AutoIndentNone`, `editor.AutoIndentKeep` (the line's leading whitespace) or `editor.AutoIndentBrackets` (one more level after `{ [ (` or a trailing `:`, and a closing bracket right after the caret moves to its own line). Unset editors follow `editor.EditorDefaults.IndentSpaces` and `editor.EditorDefaults.AutoIndent`; use these for per-language settings.

`ed.SetWrapIndent(editor.WrapHanging)` (or `editor.WithWrapIndent`) picks where continuation rows start: `WrapIndentNone` (column 0, default), `WrapIndentSame` (under the line's leading whitespace) or `WrapHanging` (one tab stop past it). Indents count rendered columns (tabs expanded) and are capped at half the row width.

**Gutter.** By default the line-number gutter is 52 px, filled with `ChromeMuted`, with a border. `ed.SetGutter(spec)` (or `editor.WithGutter`) sizes it to the digit count instead:

```go
ed.SetGutter(editor.GutterSpec{
    MinDigits: 2,              // room kept even for short docs (0 = 2)
    Padding:   th.Spacing.S,   // each side of the numbers (0 = Spacing.S)
    Fill:      ui.TokenUnset,  // TokenUnset = transparent, like the text area
    Separator: false,          // 1 px border line before the text
})
```


Language servers: `editor.LanguageServers()` is the manager every editor shares (one server per workspace root and language). An app's own `Suggest` / `HoverInfo` sources are asked first; the server answers what they decline.

Large files: Tree-sitter highlighters skip documents over `highlight.MaxBytes` (2 MB by default); `ed.HighlightOversize()` reports when that happened and `ed.HighlightAnyway()` lifts the limit for that editor. `ed.SetText(s)` replaces the document as one undoable edit.

## License

MIT, like Yoga. See [LICENSE](LICENSE).
