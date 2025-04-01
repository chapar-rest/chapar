# gvcode

gvcode is a Gio based text editor for code editing.

## Key Features:

- Uses a PieceTable backed text buffer for efficient text editing.  
- Optimized undo/redo operations with built-in support in the PieceTable.  
- Supports both hard and soft tabs, ensuring alignment with tab stops. 
- Lines can be unwrapped, with horizontal scrolling supported.  
- Syntax highlighting is available by applying text styles.  
- Built-in line numbers for better readability.  
- Auto-complete of bracket pairs and quote pairs.
- Auto-indent new lines.
- Bracket auto-indent.
- Increase or descease indents of multi-lines using Tab key and Shift+Tab.
- Expanded shortcuts support (Work in Progress).  
- Large file rendering(Planned).

## Why another code editor?

I ported Gio's editor component to [gioview](https://github.com/oligo/gioview) and added a few features to support basic code editing, but it is far from a good code editor. Keep expanding the original editor seems a wise decision, but the design of the original editor makes it hard to adding more features. And it is also not performant enough for large file editing/rendering. Most importantly it lacks tab/tab stop support. When I tried to add tab stop support to it, I found it needs a new overall design, so there is gvcode.

## Key Design

Gio's text shaper layout the whole document in one pass, although internally the document is processed by paragraphs. After the shaping flow, there is a iterator style API to get the shaped glyphs one by one. This is what Gio's editor does when layouting the texts. 

Gvcode has chosen another way. gvcode read and layout in a paragraph by paragrah manner. The outcomes are joined together to assemble the final document view. This gives up the oppertunity to process the text only visible in the viewport, making incremental shaping possible. Besides that we can also process tab expanding & tab stops at line level, because we have full control of the paragraph layout. To achive that goal, gvcode implemented its own line wrapper.


## How To Use

Gvcode exports simple APIs to ease the integration with your project. Here is a basic example:

```go
    state := &editor.Editor{}
	var ops op.Ops

	for {
		e := ed.window.Event()

		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			
            es := gvcode.NewEditor(th, state)
            es.Font.Typeface = "monospace"
            es.TextSize = unit.Sp(12)
            es.LineHeightScale = 1.5
			es.Layout(gtx)

			e.Frame(gtx.Ops)
		}
	}
```

For a full working example, please see code in the folder `./example`.


## Cautions

`gvcode` is not intended to be a drop-in replacement for the official Editor widget as it dropped some features such as single line mode, truncator and max lines limit. 

This project is a work in progress, and the APIs may change as development continues. Please use it at your own risk.


## Contributing

See the [contribution guide](CONTRIBUTING.md) for details.


## Acknowledgments

This project uses code from the [Gio](https://gioui.org/) project, which is licensed under the Unlicense OR MIT License.

