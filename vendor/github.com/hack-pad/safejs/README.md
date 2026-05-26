# SafeJS  [![Go Reference](https://pkg.go.dev/badge/github.com/hack-pad/safejs.svg)](https://pkg.go.dev/github.com/hack-pad/safejs) [![CI](https://github.com/hack-pad/safejs/actions/workflows/ci.yml/badge.svg)](https://github.com/hack-pad/safejs/actions/workflows/ci.yml) [![Coverage Status](https://coveralls.io/repos/github/hack-pad/safejs/badge.svg?branch=main)](https://coveralls.io/github/hack-pad/safejs?branch=main)

A safer, drop-in replacement for Go's `syscall/js` JavaScript package.

## What makes it safer?

Today, `syscall/js` panics when the JavaScript runtime throws errors.
While sensible in a JavaScript runtime, [Go libraries should avoid using `panic`](https://go.dev/doc/effective_go#panic).

SafeJS provides a nearly identical API to `syscall/js`, but returns errors instead of panicking.

Although returned errors aren't pretty, they make it much easier to integrate with existing Go tools and code patterns.

#### Backward compatibility

This package uses the same backward compatibility guarantee as `syscall/js`.

In an effort to align with the Go standard library API, some breaking changes may become necessary and receive their own minor version bumps.

## Quick start

1. Get `safejs`:
```
go get github.com/hack-pad/safejs
```
2. Import `safejs`:
```go
import "github.com/hack-pad/safejs"
```
3. Replace uses of `syscall/js` with the `safejs` alternative. 

Before:
```go
//go:build js && wasm

package buttons

import "syscall/js"

// InsertButton creates a new button, adds it to 'container', and returns it. Usually.
func InsertButton(container js.Value) js.Value {
    // *whisper:* There's a good chance it could panic! Eh, probably don't need to document it, right?
    dom := js.Global().Get("document") // BOOM!
    button := dom.Call("createElement", "button") // BANG!
    container.Call("appendChild", button) // BAM!
    return button
}
```

After:
```go
//go:build js && wasm

package buttons

import "github.com/hack-pad/safejs"

// InsertButton creates a new button, adds it to 'container', and returns the button or the first error.
func InsertButton(container safejs.Value) (safejs.Value, error) {
    dom, err := safejs.Global().Get("document")
    if err != nil {
        return err
    }
    button, err := dom.Call("createElement", "button")
    if err != nil {
        return err
    }
    _, err = container.Call("appendChild", button)
    if err != nil {
        return err
    }
    return button, nil
}
```

## Even safer

For additional JavaScript safety, use the `jsguard` linter too.

`jsguard` reports the locations of unsafe JavaScript calls, which should be replaced with calls to SafeJS.

```bash
# When installed without specifying a version, uses the go.mod version.
go install github.com/hack-pad/safejs/jsguard/cmd/jsguard
export GOOS=js GOARCH=wasm
jsguard ./...
```

It *does not* report use of types like `js.Value` -- only function calls on those types.

This makes it easy to integrate SafeJS into existing libraries which expose only standard library types.
