# star-tex

[![Build status](https://builds.sr.ht/~sbinet/star-tex.svg)](https://builds.sr.ht/~sbinet/star-tex?)
[![GoDoc](https://pkg.go.dev/badge/star-tex.org/x/tex)](https://pkg.go.dev/star-tex.org/x/tex)

`star-tex` (or `*TeX`) is a TeX engine in Go.

## cmd/star-tex

`star-tex` provides a `TeX` to `PDF` typesetter.

```
$> star-tex -h
Usage: star-tex [options] FILE.tex [FILE.pdf]

ex:
 $> star-tex ./testdata/hello.tex
 $> star-tex ./testdata/hello.tex ./out.pdf

options:
  -texmf string
    	path to TexMF root

$> star-tex ./testdata/hello.tex out.pdf
$> pdf out.pdf
```

## cmd/dvi-cnv

`dvi-cnv` converts a DVI file into a (set of) PNG or PDF file(s).

```
$> dvi-cnv -help
Usage of dvi-cnv:
  -o string
    	path to output file name
  -texmf string
    	path to TexMF root
  -v	enable verbose mode

$> dvi-cnv -o foo.png ./testdata/hello_golden.dvi
$> open ./foo_1.png

$> dvi-cnv -o foo.pdf ./testdata/hello_golden.dvi
$> open ./foo.pdf
```

## cmd/dvi-dump

`dvi-dump` displays the content of a DVI file in a human readable format or JSON.

The human readable format should be exactly the same than the official [`dvitype`](https://texdoc.org/serve/dvitype/0) command from `TeX Live`.

```
$> dvi-dump -help
Usage of dvi-dump:
  -json
    	enable JSON output
  -texmf string
    	path to TexMF root

$> dvi-dump ./testdata/hello_golden.dvi
numerator/denominator=25400000/473628672
magnification=1000;       0.00006334 pixels per DVI unit
' TeX output 1776.07.04:1200'
Postamble starts at byte 1290.
maxv=43725786, maxh=30785863, maxstackdepth=2, totalpages=1
Font 36: cmti10---loaded at size 655360 DVI units 
Font 23: cmbx10---loaded at size 655360 DVI units 
Font 12: cmsy10---loaded at size 655360 DVI units 
Font 6: cmmi10---loaded at size 655360 DVI units 
Font 0: cmr10---loaded at size 655360 DVI units 
 
42: beginning of page 1 
87: push 
level 0:(h=0,v=0,w=0,x=0,y=0,z=0,hh=0,vv=0) 
88: down3 -917504 v:=0-917504=-917504, vv:=-58 
92: pop 
[...]
```

## cmd/kpath-find

`kpath-find` is a new command that finds files in a `TeX` directory structure:

```
$> kpath-find -help
Usage of kpath-find:
  -all
    	display all matches
  -texmf string
    	path to TEXMF distribution

$> kpath-find -texmf /usr/share/texmf-dist cmr10.pk
/usr/share/texmf-dist/fonts/pk/ljfour/public/cm/dpi600/cmr10.pk

$> kpath-find -all -texmf /usr/share/texmf-dist latex
/usr/share/texmf-dist/makeindex/latex
/usr/share/texmf-dist/tex/latex
/usr/share/texmf-dist/tex4ht/ht-fonts/alias/latex
/usr/share/texmf-dist/tex4ht/ht-fonts/unicode/latex
```

## cmd/pk2bm

`pk2bm` display the content of a `pk` font file.

```
$> pk2bm -help
Usage of pk2bm:
  -H int
    	height of bitmap
  -W int
    	width of bitmap
  -b	generate a bitmap
  -c string
    	character to display
  -h	generate a hexmap

$> pk2bm -b -c a ./internal/tds/fonts/pk/ljfour/public/cm/dpi600/cmr10.pk 

character : 97 (a)
   height : 39
    width : 38
     xoff : -3
     yoff : 37

  ...........********...................
  ........**************................
  ......*****.......******..............
  .....***............*****.............
  ....*****............******...........
  ...*******............******..........
  ...********...........******..........
  ...********............******.........
  ...********............******.........
  ...********.............******........
  ....******..............******........
  .....****...............******........
  ........................******........
  ........................******........
  ........................******........
  ........................******........
  .................*************........
  .............*****************........
  ..........*********.....******........
  ........*******.........******........
  ......*******...........******........
  ....********............******........
  ...*******..............******........
  ..********..............******........
  .********...............******........
  .*******................******........
  .*******................******......**
  *******.................******......**
  *******.................******......**
  *******.................******......**
  *******................*******......**
  *******................*******......**
  ********..............********......**
  .*******.............***.*****......**
  .********............**...*****....**.
  ..********.........****...*****....**.
  ....*******......****......*********..
  ......**************........*******...
  .........********............*****....
```

## cmd/tfm2pl

`tfm2pl` converts a TFM file to human-readable property list file or standard output.
`tfm2pl` is a Go-based reimplementation of `TFtoPL`, distributed with TeX-live.


```
$> tfm2pl -help
Usage: tfm2pl [options] file.tfm [file.pl]

tfm2pl converts a TFM file to human-readable property list file or standard output.

ex:
 $> tfm2pl testdata/simple.tfm
 $> tfm2pl testdata/simple.tfm out.pl

options:

$> tfm2pl /usr/share/texmf-dist/fonts/tfm/public/cm/cmr10.tfm
(FAMILY CMR)
(FACE O 352)
(CODINGSCHEME TEX TEXT)
(DESIGNSIZE R 10.0)
(COMMENT DESIGNSIZE IS IN POINTS)
(COMMENT OTHER SIZES ARE MULTIPLES OF DESIGNSIZE)
(CHECKSUM O 11374260171)
(FONTDIMEN
   (SLANT R 0.0)
   (SPACE R 0.333334)
   (STRETCH R 0.166667)
   (SHRINK R 0.111112)
   (XHEIGHT R 0.430555)
   (QUAD R 1.000003)
   (EXTRASPACE R 0.111112)
   )
(LIGTABLE
   (LABEL O 40)
   (KRN C l R -0.277779)
   (KRN C L R -0.319446)
[...]
```

The output of `tfm2pl` should be exactly the same than the one from the official [`tftopl`](https://texdoc.org/serve/tftopl/0) binary from `TeX Live`.
