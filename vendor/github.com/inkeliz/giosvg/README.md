GIOSVG
-----

Give to your app some SVG icons. (:

------------

Example:

```go
// Embed your SVG into Golang:
//go:embed your_icon.svg
var iconFile []byte

// Give your SVG/XML to the Vector:
vector, err := giosvg.NewVector(iconFile)
if err != nil {
	panic(err)
}

// Create the Icon:
icon := giosvg.NewIcon(vector)

func someWidget(gtx layout.Context) layout.Dimensions {
	// Render your icon anywhere:
	return icon.Layout(gtx)
}
```

You can use `embed` to include your icon. The `Vector` can be reused to avoid parse the SVG multiple times.

If your icon use `currentColor`, you can use `paint.ColorOp`:

```go
func someWidget(gtx layout.Context) layout.Dimensions {
    	// Render your icon anywhere, with custom color:
	paint.ColorOp{Color: color.NRGBA{B: 255, A: 255}}.Add(gtx.Ops)
	return icon.Layout(gtx)
}
```

-----------

It's possible to generate Gio functions from SVG, without need to parse XML at runtime, you can use:

```
go run github.com/inkeliz/giosvg/cmd/svggen -i .\path\to\assets -o .\path\to\pkg\vectors.go
```

It will compile all .SVG into one single file, that will create Gio functions ([here you can see one example of generated file](https://github.com/inkeliz/giosvg/blob/4c5a5409fe5bc9f5cd8680eb87d0d6c2ff148d6d/example/school-bus.go)). You can render the SVG using `icon := giosvg.NewIcon(pkg.IconName)` then `icon.Layout(gtx)` as mentioned above (consider that `pkg.IconName` is the generated Golang code).

------------

Icons in the `example` are from Freepik and from Flaticon Licensed by Creative Commons 3.0. This package
is based on OKSVG.
