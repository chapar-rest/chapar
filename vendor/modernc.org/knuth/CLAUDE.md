# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **Direction:** there is no `v2` and no rewrite planned — see `DECISIONS.md`.
> The Unicode/modern-font/LaTeX effort lives in `modernc.org/xetex` (XeTeX
> cross-compiled to wasm and translated to Go by `wa2go`). This repo stays a
> faithful pure-Go TeX82 and doubles as a differential oracle.

## What this repository is

`modernc.org/knuth` holds Go transpilations of D. E. Knuth's TeX-family Pascal/WEB
programs, plus the transpiler that produces them. Three kinds of code live here:

1. **`web/`** — a WEB → Pascal → Go transpiler (WEB tangle, Pascal scanner/parser/type
   checker, Go backend). This is the engine; it is hand-written and is where translation
   bugs get fixed.
2. **Transpiled tool packages** — `tex`, `mf`, `tangle`, `weave`, `dvitype`, `gftype`,
   `gftodvi`, `gftopk`, `mft`, `pktype`, `pltotf`, `pooltype`, `tftopl`, `vftovp`,
   `vptovf`. Each is mostly generated Go plus a small hand-written API, with a thin
   `cmd/go<tool>` CLI wrapper.
3. **Hand-written support** — the root `knuth` package (Pascal runtime + WEB change-file
   machinery), and `dvi`, `font/{afm,fixed,pkf,tfm}`, `kpath`, `internal/{iobuf,tds}`,
   which are adapted from the star-tex project (see `LICENSE-STAR-TEX` and the
   `README-STAR-TEX` files).

## Commands

```sh
go build ./...
go test ./...                      # whole suite ~12s; web/ dominates (~9s)
go test -v -count=1 ./tex/         # one package
go test -v -run TestPascalParser ./web/
make                               # full cycle, see below
make test                          # go test -v -count=1 ./...
```

Root `make` runs, in order: `go generate -v ./...` (needs TeX Live, see below),
`gofmt -l -s -w .`, `go build -v ./...`, `go test -count=1 ./... | tee log-editor`,
`go install -v ./cmd...`, `staticcheck`.

### Regenerating transpiled code

Each tool package has `//go:generate make generate` in its `api.go`, and a `Makefile`
whose `generate` target runs `go run generate.go`, rebuilds, runs the *real* `weave` on
the `.web`/`.ch` pair as a cross-check, and deletes the intermediate `.pas`/`.tex` files.
So `cd tex && make generate` regenerates just TeX; root `make` regenerates everything.

Regeneration (and refreshing `testdata/`) requires these TeX Live binaries in `$PATH`:
`dvitype gftopk gftype mf mft pooltype tangle tex tftopl vftovp vptovf weave`.
Plain `go build` / `go test` does **not** need them.

### Debugging the transpiler

`web/all_test.go` accepts flags that make the corpus-driven tests usable as a workbench:

```sh
go test -run Go -re mf.web -tmp ~/tmp/knuth ./web/   # == `make tmp` in web/ (tmp2 = tex.web)
```

- `-re <regexp>` — restrict the `web/testdata` corpus to matching paths
- `-tmp <dir>` — keep the generated `test.go`/`test.pas`/`test.pool` instead of a TempDir
- `-mw`/`-mp`/`-mg <file>` — additionally dump the Pascal / pool / Go output
- `-panic` — do not recover panics in the generator, so you get real stack traces
- `-trcw` — trace the WEB scanner

Building with `-tags knuth.debug` turns on tracing of `knuth.Open` / `Reset` / `Rewrite`.

## Architecture

### Generation pipeline

For each transpiled tool `X`:

```
X.web   verbatim CTAN source — never edit
X.ch    change file — the only place to patch Knuth's source
  │  knuth.NewChanger applies the @x/@y/@z blocks
  ▼
web.Tangle (web/tangle.go)  →  X.pas (Pascal) + X.pool (string pool, go:embed'ed)
  ▼
newPasScanner → pasParse → ast.check   (web/pascal.go)
  ▼
gen (web/backend.go) → Go → gofmt -s -r '(x) -> x'
  ▼
X.go   checked in, header says DO NOT EDIT
```

The whole chain is `web.Go(dest, pascal, pool, src, pkg, opts...)`, driven by
`X/generate.go` (`//go:build ignore`).

**Where a change belongs:**

- Tool misbehaves because of Knuth's source or a Pascal-ism the backend can't take →
  edit `X.ch`.
- Wrong Go emitted for correct Pascal → fix `web/*.go`, then regenerate *all* tools and
  re-run the full suite; a backend change touches every generated file.
- Never edit `X.go` (regenerated) or `X.web` (pristine; `web/testdata/ctan.org/...` holds
  byte-identical copies used as the transpiler's corpus).

### Change-file conventions

`@x`-block headers carry an advisory source location, e.g. `@x tex.web:193:`.
`knuth.NewChanger` locates blocks by exact text search of the trimmed `@x` body — not by
line number — so those annotations can go stale without breaking anything. Blocks that
exist only for the torture-test variant are additionally tagged, e.g.
`@x tex.web:309: TRIP` / `@x mf.web:272: TRAP`.

### Shape of the generated code

- Every Pascal global becomes a field of one `type prg struct`; every Pascal
  procedure/function becomes a method on `*prg` (330 of them in `tex.go`). Entry point is
  `(*prg).main()`.
- Identifiers are lowerCamelCased from `snake_case` with numeric suffixes on collisions
  (`ns.xid0`, `web/backend.go`).
- Preamble defines `char = byte`, `signal`, and helpers `strcopy`, `arraystr`, `abs`,
  `fabs`, `round`.
- Local `goto`s become Go labels; non-local ones (`final_end`, `end_of_TEX`) become
  `panic(signal(n))`, recovered by the hand-written `Main`.
- WEB module positions survive as comments (`// tangle:pos tex.web:95:22:`), so any line
  of generated Go can be traced back to the `.web`.

### Hand-written per-tool files

- `api.go` — package doc, `//go:generate`, `//go:embed X.pool`, and `Main(...)`, which
  builds `&prg{...}` wiring each Pascal file variable to a `knuth.File` and recovers
  `signal` / `knuth.Error` into an `error`. Signatures take `io.Reader`/`io.Writer`
  rather than file names, e.g.
  `tftopl.Main(tfmFile io.Reader, plFile, stdout, stderr io.Writer) error`. `tex.Main`
  also takes `Option`s (`WithInputFile`, `WithDVIFile`, `WithLogFile`).
- `etc.go` — the same `origin`/`todo`/`trc` boilerplate, deliberately duplicated in every
  package.

### Pascal runtime (root `knuth` package)

- `File` + `NewTextFile` / `NewBinaryFile` / `NewPoolFile` implement Pascal file semantics
  (`Reset`, `Rewrite`, `Get`, `Put`, `EOF`, `EOLN`, `Read`/`Readln`, `Write`/`Writeln`,
  `ErStat`, `CurPos`/`SetPos`). `WriteWidth` models `write(x:n)` field widths.
- `RuneSource` / `NewRuneSource` / `Changer` / `ReadLine` — the WEB change-file machinery,
  used both by the transpiler and by the transpiled `tangle`/`weave`.
- `Open(name, search)` resolves TeX area prefixes (`TeXfonts:`, `TeXinputs:`,
  `TeXformats:`, `MFbases:`, `MFinputs:`) against the filesystem, then falls back to
  `Assets` — an `fs.FS` over the embedded `assets.tar.gz` (Computer Modern `.mf` sources
  and `tfm`s, `lib/plain.tex`, `lib/hyphen.tex`).

`gotex` therefore starts as INITEX with no format preloaded: `./gotex '\input plain \input story'`
works out of the box because `plain.tex` comes from the embedded assets.

### TRIP / TRAP variants

`tex/internal/trip` and `mf/internal/trap` are *second* transpilations of the same `.web`
with a different `.ch` supplying the settings Knuth's torture tests require (`mem_max=3000`,
`mem_min=1`, `error_line=64`, null `stat`/`tats`, …). They are emitted by the second half
of `tex/generate.go` / `mf/generate.go`. Their own `all_test.go` is a stub — the real work
is in `tex/all_test.go`, which follows the `tripman.pdf` Appendix A procedure end to end
(pltotf → tftopl → INITEX twice → dvitype) and byte-compares against `tex/testdata/trip.*`.

### Testing approach

Tool tests are byte-exact comparisons against output produced by the real TeX Live
binaries, checked into `testdata/` and refreshed by each `Makefile`'s `generate` target.
Mismatches are reported as a `go-difflib` unified diff plus hex dumps. Version/date
banners are stripped by local `noBanner` helpers before comparing.

`web/all_test.go` walks the 17 `.web` files under `web/testdata/ctan.org/` through
progressively deeper stages — `TestTangleScanner`, `TestTangle{,2,3}`,
`TestPascalScanner`, `TestPascalParser`, `TestPascalCheck`, `TestGo` — with `TestGo`
additionally `go build`ing each generated file in a scratch module. A regression in the
backend usually shows up here first.

## Conventions

- Every file opens with the BSD-style copyright header, and each `package` clause carries
  a trailing import-path comment (`package tex // modernc.org/knuth/tex`).
- `log`, `log-diff`, `log-editor`, `log-make`, `log-test` are scratch output from the
  `make` targets, globally gitignored — not sources, don't commit or reason from them.
- `builder.json` lists the GOOS/GOARCH matrix used by the maintainer's out-of-band CI.
