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
	robotoRegularTTF, err := getFont("Roboto-Regular.ttf")
	if err != nil {
		return nil, err
	}

	robotoRegular, err := opentype.Parse(robotoRegularTTF)
	if err != nil {
		return nil, err
	}

	robotoBoldTTF, err := getFont("Roboto-Bold.ttf")
	if err != nil {
		return nil, err
	}

	robotoBold, err := opentype.Parse(robotoBoldTTF)
	if err != nil {
		return nil, err
	}

	robotoMediumTTF, err := getFont("Roboto-Medium.ttf")
	if err != nil {
		return nil, err
	}

	robotoMedium, err := opentype.Parse(robotoMediumTTF)
	if err != nil {
		return nil, err
	}

	materialIconsTTF, err := getFont("MaterialIcons-Regular.ttf")
	if err != nil {
		return nil, err
	}

	materialIcons, err := opentype.Parse(materialIconsTTF)
	if err != nil {
		return nil, err
	}

	robotoMonoRegularTTF, err := getFont("RobotoMono-Regular.ttf")
	if err != nil {
		return nil, err
	}

	robotoMonoRegular, err := opentype.Parse(robotoMonoRegularTTF)
	if err != nil {
		return nil, err
	}

	fontFaces = append(fontFaces,
		font.FontFace{Font: font.Font{}, Face: robotoRegular},
		font.FontFace{Font: font.Font{Weight: font.Medium}, Face: robotoMedium},
		font.FontFace{Font: font.Font{Weight: font.Bold}, Face: robotoBold},
		font.FontFace{Font: font.Font{Typeface: "MaterialIcons"}, Face: materialIcons},
		font.FontFace{Font: font.Font{Typeface: "RobotoMono"}, Face: robotoMonoRegular},
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

func MustGetMono() font.FontFace {
	robotoMonoRegularTTF, err := getFont("RobotoMono-Regular.ttf")
	if err != nil {
		panic(err)
	}

	robotoMonoRegular, err := opentype.Parse(robotoMonoRegularTTF)
	if err != nil {
		panic(err)
	}

	return font.FontFace{Font: font.Font{}, Face: robotoMonoRegular}
}
