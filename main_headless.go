//go:build nogpu

package main

import (
	"fmt"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/ui"
)

func main() {
	const w, h = 1100, 720

	text, err := shape.NewEngine(1, false)
	if err != nil {
		panic(err)
	}
	clip := &input.MemClipboard{}
	sheet := render.NewSpriteSheet(text.Atlas)
	yoga.SetResources(text, sheet, clip)

	app := BuildCurly()
	c := ui.New(text, ui.NewFocusScope(), nil)
	c.SetIcons(sheet)
	c.SetClipboard(clip)

	mouse := &input.Mouse{}
	keyboard := &input.Keyboard{}

	root := ui.BuildFrame(c, app.Body, w, h, mouse, keyboard)
	layout.Dispatch(root, mouse)
	c.Focus().Route(keyboard)
	mouse.EndFrame()
	keyboard.EndFrame()

	drawList := &render.DrawList{}
	root = ui.BuildFrame(c, app.Body, w, h, mouse, keyboard)
	layout.Paint(root, drawList, text)

	fmt.Printf("headless frame: %d vertices, %d indices\n", len(drawList.Vertices), len(drawList.Indices))
	fmt.Printf("spaces: %d, current: %s\n", len(spaces), app.activeSpace.Name)
	fmt.Printf("environments: %d, current: %s\n", len(environments), app.activeEnvironment.Name)
}
