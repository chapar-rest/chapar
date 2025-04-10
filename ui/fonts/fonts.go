package fonts

import (
	"embed"
	"fmt"

	"gioui.org/font"
	"gioui.org/font/opentype"
)

//go:embed fonts/*
var fonts embed.FS

func Prepare() ([]font.FontFace, error) {
	var fontFaces []font.FontFace
	sourceSansProRegularOTF, err := getFont("source_sans_pro_regular.otf")
	if err != nil {
		return nil, err
	}

	sourceSansProRegular, err := opentype.Parse(sourceSansProRegularOTF)
	if err != nil {
		return nil, err
	}

	sourceSansProBoldOTF, err := getFont("source_sans_pro_bold.otf")
	if err != nil {
		return nil, err
	}

	sourceSansProBold, err := opentype.Parse(sourceSansProBoldOTF)
	if err != nil {
		return nil, err
	}

	sourceSansProSemiBoldOTF, err := getFont("source_sans_pro_semibold.otf")
	if err != nil {
		return nil, err
	}

	sourceSansProSemiBold, err := opentype.Parse(sourceSansProSemiBoldOTF)
	if err != nil {
		return nil, err
	}

	materialIconsOTF, err := getFont("MaterialIcons-Regular.ttf")
	if err != nil {
		return nil, err
	}

	materialIcons, err := opentype.Parse(materialIconsOTF)
	if err != nil {
		return nil, err
	}

	fontFaces = append(fontFaces,
		font.FontFace{Font: font.Font{Typeface: "sourceSansPro"}, Face: sourceSansProRegular},
		font.FontFace{Font: font.Font{Weight: font.SemiBold}, Face: sourceSansProSemiBold},
		font.FontFace{Font: font.Font{Weight: font.Bold}, Face: sourceSansProBold},
		font.FontFace{Font: font.Font{Typeface: "MaterialIcons"}, Face: materialIcons},
	)
	return fontFaces, nil
}

func getFont(path string) ([]byte, error) {
	data, err := fonts.ReadFile(fmt.Sprintf("fonts/%s", path))
	if err != nil {
		return nil, err
	}

	return data, err
}

func MustGetCodeEditorFont() font.FontFace {
	data, err := getFont("SourceCodePro-Regular.ttf")
	if err != nil {
		panic(err)
	}

	monoFont, err := opentype.Parse(data)
	if err != nil {
		panic(err)
	}

	return font.FontFace{Font: font.Font{}, Face: monoFont}
}
