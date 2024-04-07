# explorer [![Go Reference](https://pkg.go.dev/badge/gioui.org/x/explorer.svg)](https://pkg.go.dev/gioui.org/x/explorer)

-----------

Integrates a simple `Save As...` or `Open...` mechanism to your Gio application.

## What can it be used for?

Well, for anything that manipulates user's file. You can use `os.Open` to open and write file, 
but sometimes you want to know where to save the data, in those case `Explorer` is useful.

## Status

Currently, `Explorer` supports most platforms, including Android 6+, JS, Linux (with XDG Portals), Windows 10+, iOS 14+ and macOS 10+. It will
return ErrAvailableAPI for any other platform that isn't supported.

## Limitations

### Edit file content via `explorer.ReadFile()`:

It may not be possible to edit/write data using `explorer.ReadFile()`. Because of that, it returns a `
io.ReadCloser` instead of `io.ReadWriteCloser`, since some operational systems (such as JS) doesn't
allow us to modify the file. However, you can use type-assertion to check if it's possible or not:

```
reader, _ := explorer.ReadFile()
if f, ok := reader.(*os.File); ok {
    // We can use `os.File.Write` in that case. It's NOT possible in all OSes.
    f.Write(...)
}
```

### Select folders:

It's not possible to select folders.
