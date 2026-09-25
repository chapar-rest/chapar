## Package knuth

    import path "modernc.org/knuth"

Package knuth collects utilities common to all other packages in this
repository.

Documentation: [godoc.org/modernc.org/knuth](http://godoc.org/modernc.org/knuth)

## Installation

To install all the included go... commands found in cmd/

    $ go install modernc.org/knuth/cmd...@latest

## Hacking

Make sure you have these utilities from the Tex-live package(s) installed in
your $PATH:

    dvitype
    gftopk
    gftype
    mf
    mft
    pooltype
    tangle
    tex
    tftopl
    vftovp
    vptovf
    weave

These programs are used only to generate test data. Users of
packages/commands in this repository do not need them installed.

After modification of any sources, run '$ make' in the repository root. That
will regenerate all applicable Go code and testdata, run the tests of all
packages in this repository and install all the commands found in ./cmd
