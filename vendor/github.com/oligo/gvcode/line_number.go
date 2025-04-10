package gvcode

import (
	"image"
	"strconv"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	lt "github.com/oligo/gvcode/internal/layout"
)

func paintLineNumber(gtx layout.Context, shaper *text.Shaper, params text.Parameters, viewport image.Rectangle, paragraphs []lt.Paragraph, textMaterial op.CallOp) layout.Dimensions {
	// inherit all other settings from the main text layout.
	params.Alignment = text.End
	params.MinWidth = gtx.Constraints.Min.X
	params.MaxWidth = gtx.Constraints.Max.X
	params.MaxLines = 1

	var dims layout.Dimensions
	glyphs := make([]text.Glyph, 5)

	quit := false
lineLoop:
	for i, line := range paragraphs {
		if quit {
			break
		}

		shaper.LayoutString(params, strconv.Itoa(i+1))
		glyphs = glyphs[:0]

		var bounds image.Rectangle
		visible := false
		for {
			g, ok := shaper.NextGlyph()
			if !ok {
				break
			}

			if int(line.StartY)+g.Descent.Ceil() < viewport.Min.Y {
				break
			} else if int(line.StartY)-g.Ascent.Ceil() > viewport.Max.Y {
				quit = true
				goto lineLoop
			}

			bounds.Min.X = min(bounds.Min.X, g.X.Floor())
			bounds.Min.Y = min(bounds.Min.Y, int(g.Y)-g.Ascent.Floor())
			bounds.Max.X = max(bounds.Max.X, (g.X + g.Advance).Ceil())
			bounds.Max.Y = max(bounds.Max.Y, int(g.Y)+g.Descent.Ceil())

			glyphs = append(glyphs, g)
			visible = true
		}

		if !visible {
			continue
		}

		dims.Size = image.Point{X: max(bounds.Dx(), dims.Size.X), Y: dims.Size.Y + bounds.Dy()}

		trans := op.Affine(f32.Affine2D{}.Offset(f32.Point{Y: float32(line.StartY)}.Sub(layout.FPt(viewport.Min)))).Push(gtx.Ops)

		// draw glyph
		path := shaper.Shape(glyphs)
		outline := clip.Outline{Path: path}.Op().Push(gtx.Ops)
		textMaterial.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		outline.Pop()
		trans.Pop()
	}

	return dims
}
