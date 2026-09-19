# DECISIONS

## 2026-08-05 — no v2; the Unicode/LaTeX effort moves to modernc.org/xetex

**Decision: this repository stays as it is.** The planned `v2/` — freezing the
WEB→Pascal→Go pipeline, editing the generated Go by hand, removing the Pascal
warts (fixed `mem` arena, 8-bit char codes, string pool, index-based containers,
package-level state), then adding Unicode, modern fonts and LaTeX support — is
**not going to happen**. The effort it was meant to serve now lives in
`modernc.org/xetex`; see that repo's `HANDOFF.md`.

### Why

`../xetex` already does what v2 was aiming at, by a different route: TeX Live's
C/C++ XeTeX is cross-compiled to a WASI wasm module with wasi-sdk and translated
to native CGo-free Go by `modernc.org/wa2go`. Verified working on 2026-08-05 —
`xetex` typesets to XDV and a pure-Go `xdvipdfmx` converts that to PDF with
embedded subsetted fonts and searchable text; OpenType shaping via real harfbuzz
and freetype works; LaTeX2e + expl3 load and run. Reaching the same place from
TeX82 by hand would mean reimplementing eTeX, Unicode character handling,
OpenType font support and a PDF backend — years of work to arrive where a
mechanical translation already is.

The two plans failed on opposite axes. v2 would have had the right *form* — a
clean, importable, concurrent Go library — and the wrong *features* (TeX82 only).
`xetex` has the right features and the wrong form (millions of lines of
generated Go, not installable, one set of globals). Form is the cheaper axis to
fix, and those fixes land in `wa2go`, which benefits every other modernc.org
transpile too.

### What this repo is for now

- A faithful, small, pure-Go TeX82 and the WEB tooling around it, for its
  existing users. Unchanged.
- A **differential oracle**: it is a byte-faithful TeX82 and a Go library, so it
  can be run in-process against other implementations on arbitrary input. If a
  hand-written Go TeX is ever revisited, this is the reference.
- The WEB→Pascal→Go transpiler in `web/` remains the way the engine is derived
  from `tex.web`; keeping it runnable is what preserves that property.

### Measurements, if this is ever reopened

Taken from `tex/tex.go` (~1 MB, 330 methods on `*prg`) while scoping v2:

- `prg.mem[` occurs **2740** times; the arena is `mem [30001]memoryWord`.
  `prg.eqtb[` 610, `strNumber` 1121, `prg.fontInfo[` 183.
- `eqtb` holds six contiguous 256-entry char-indexed regions (catCode, lcCode,
  ucCode, sfCode, mathCode, delCode), and they are save-stack'd — Unicode char
  codes would have to keep group save/restore semantics on a sparse table.
- Package-level mutable state is *almost* gone already: every global is a `prg`
  field. The one real offender was `var opener` in `tex/api.go`, which
  `WithInputFile` rewrote — so a replacement persisted into every later `Main`
  call and raced with any concurrent one, and `fmtFile`/`tfmFile` captured the
  opener *before* options were applied while `inputFile` captured it after.
  **Fixed 2026-08-05**: options now populate a per-call `config`, every file is
  built after they are applied, and `tex.Main` is safe to run concurrently.
  Covered by `TestOptionsAreNotGlobal` and `TestConcurrentEngines` (`-race`).
  There is no remaining mutable package-level state in hand-written code.
- The TRIP oracle is less coupled to implementation detail than it looks:
  `trip.log` is 7306 lines but only **22 lines** across trip.log/tripin.log/
  trip.fot report arena or string-pool statistics, and tripman.pdf Appendix A
  (quoted verbatim in `tex/all_test.go`) explicitly permits those to differ.
  `trip.dvi` is compared byte-exact and is the real semantic contract.

### Also considered and rejected

`modernc.org/tfs` — an independent from-scratch Go TeX written with Gemini in
2025, abandoned. Its scanner tokenized a 9.8 MB corpus cleanly, but the project
stalled with macro expansion barely started and no typesetting at all. Removed
from the builder task list on 2026-08-05. Its one good idea — normalization-
independent grapheme-cluster identity — is already handled better in XeTeX, by
TECkit at input and harfbuzz at shaping, which is the right layer for it:
cluster-level *tokens* would break `\catcode`, `\lccode` and every expl3 routine
that inspects characters one at a time.
