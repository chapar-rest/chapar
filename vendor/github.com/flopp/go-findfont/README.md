[![PkgGoDev](https://pkg.go.dev/badge/github.com/flopp/go-findfont)](https://pkg.go.dev/github.com/flopp/go-findfont)
[![Go Report Card](https://goreportcard.com/badge/github.com/flopp/go-findfont)](https://goreportcard.com/report/github.com/flopp/go-findfont)
[![License MIT](https://img.shields.io/badge/license-MIT-lightgrey.svg?style=flat)](https://github.com/flopp/go-findfont/)

# go-findfont
A platform-agnostic go (golang) library to easily locate truetype font files in your system's user and system font directories.

## What?
`go-findfont` is a golang library that allows you to locate font file on your system. The library is currently aware of the default font directories on Linux/Unix, Windows, and MacOS.

## How?

### Installation

Installing `go-findfont` is as easy as

```bash
go get -u github.com/flopp/go-findfont
```

### Library Usage

```go

import (
  "fmt"
  "io/ioutil"
  
  "github.com/flopp/go-findfont"
  "github.com/golang/freetype/truetype"
)

func main() {
  fontPath, err := findfont.Find("arial.ttf")
  if err != nil {
    panic(err)
  }
  fmt.Printf("Found 'arial.ttf' in '%s'\n", fontPath)

  // load the font with the freetype library
  fontData, err := ioutil.ReadFile(fontPath)
  if err != nil {
    panic(err)
  }
  font, err := truetype.Parse(fontData)
  if err != nil {
    panic(err)
  }

  // use the font...
}
```

## License
Copyright 2016 Florian Pigorsch. All rights reserved.

Use of this source code is governed by a MIT-style license that can be found in the LICENSE file.
