package fribidi

// Shape does all shaping work that depends on the resolved embedding
// levels of the characters. Currently it does mirroring and Arabic shaping,
// but the list may grow in the future.
//
// If arabProps is nil, no Arabic shaping is performed.
//
// Feel free to do your own shaping before or after calling this function,
// but you should take care of embedding levels yourself then.
func Shape(flags Options, embeddingLevels []Level,
	/* input and output */ arabProps []JoiningType, str []rune) {

	if len(arabProps) != 0 {
		shapeArabic(flags, embeddingLevels, arabProps, str)
	}

	if flags&ShapeMirroring != 0 {
		shapeMirroring(embeddingLevels, str)
	}
}

// shapeMirroring replaces mirroring characters on right-to-left embeddings in
// string with their mirrored equivalent as returned by
// getMirrorChar().
//
// This function implements rule L4 of the Unicode Bidirectional Algorithm
// available at http://www.unicode.org/reports/tr9/#L4.
func shapeMirroring(embeddingLevels []Level, str []rune /* input and output */) {
	/* L4. Mirror all characters that are in odd levels and have mirrors. */
	for i, level := range embeddingLevels {
		if level.isRtl() != 0 {
			if mirror, ok := getMirrorChar(str[i]); ok {
				str[i] = mirror
			}
		}
	}
}
