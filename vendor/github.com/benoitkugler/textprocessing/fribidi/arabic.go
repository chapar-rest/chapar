package fribidi

import "github.com/benoitkugler/textlayout/unicodedata"

func getArabicShapePres(r rune, shape uint8) rune {
	if r < unicodedata.FirstArabicShape || r > unicodedata.LastArabicShape {
		return r
	}
	return rune(unicodedata.ArabicShaping[r-unicodedata.FirstArabicShape][shape])
}

// shapeArabic does Arabic shaping, depending on the flags set.
func shapeArabic(flags Options, embeddingLevels []Level,
	/* input and output */
	arabProps []JoiningType, str []rune) {

	if flags&ShapeArabPres != 0 {
		shapeArabicJoining(arabProps, str)
	}
	if flags&ShapeArabLiga != 0 {
		shapeArabicLigature(mandatoryLigaTable, embeddingLevels, arabProps, str)
	}

	// if flags&FRIBIDI_FLAG_SHAPE_ARAB_CONSOLE != 0 { // TODO: ?
	// 	fribidi_shape_arabic_ligature(console_liga_table, embedding_levels, len, ar_props, str)
	// 	fribidi_shape_arabic_joining(FRIBIDI_GET_ARABIC_SHAPE_NSM, len, ar_props, str)
	// }
}

type pairMap struct {
	pair [2]rune
	to   rune
}

func shapeArabicJoining(arabProps []JoiningType, str []rune /* input and output */) {
	for i, ar := range arabProps {
		if ar.isArabShapes() {
			str[i] = getArabicShapePres(str[i], ar.joinShape())
		}
	}
}

func compPairMap(a, b pairMap) int32 {
	if a.pair[0] != b.pair[0] {
		return a.pair[0] - b.pair[0]
	}
	return a.pair[1] - b.pair[1]
}

func binarySearch(key pairMap, base []pairMap) (pairMap, bool) {
	min, max := 0, len(base)-1
	for min <= max {
		mid := (min + max) / 2
		p := base[mid]
		c := compPairMap(key, p)
		if c < 0 {
			max = mid - 1
		} else if c > 0 {
			min = mid + 1
		} else {
			return p, true
		}
	}
	return pairMap{}, false
}

func findPairMatch(table []pairMap, first, second rune) rune {
	x := pairMap{
		pair: [2]rune{first, second},
	}
	if match, ok := binarySearch(x, table); ok {
		return match.to
	}
	return 0
}

/* Char we place for a deleted slot, to delete later */
const charFill = 0xFEFF

func shapeArabicLigature(table []pairMap, embeddingLevels []Level,
	/* input and output */
	arabProps []JoiningType, str []rune) {
	// TODO: This doesn't form ligatures for even-level Arabic text. no big problem though. */
	L := len(embeddingLevels)
	size := len(table)
	for i := 0; i < L-1; i++ {
		var c rune
		if str[i] >= table[0].pair[0] && str[i] <= table[size-1].pair[0] {
			c = findPairMatch(table, str[i], str[i+1])
		}

		if embeddingLevels[i].isRtl() != 0 && embeddingLevels[i] == embeddingLevels[i+1] && c != 0 {
			str[i] = charFill
			arabProps[i] |= ligatured
			str[i+1] = c
		}
	}
}

var mandatoryLigaTable = []pairMap{
	{pair: [2]rune{0xFEDF, 0xFE82}, to: 0xFEF5},
	{pair: [2]rune{0xFEDF, 0xFE84}, to: 0xFEF7},
	{pair: [2]rune{0xFEDF, 0xFE88}, to: 0xFEF9},
	{pair: [2]rune{0xFEDF, 0xFE8E}, to: 0xFEFB},
	{pair: [2]rune{0xFEE0, 0xFE82}, to: 0xFEF6},
	{pair: [2]rune{0xFEE0, 0xFE84}, to: 0xFEF8},
	{pair: [2]rune{0xFEE0, 0xFE88}, to: 0xFEFA},
	{pair: [2]rune{0xFEE0, 0xFE8E}, to: 0xFEFC},
}
