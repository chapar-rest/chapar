package assets

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"
)

//go:embed images/*
var images embed.FS

func LoadImage(fileName string) (image.Image, error) {
	file, err := images.ReadFile(fmt.Sprintf("images/%s", fileName))
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(file))
	if err != nil {
		return nil, err
	}

	return img, nil
}
