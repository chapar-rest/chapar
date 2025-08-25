package assets

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"

	"gioui.org/op/paint"
)

var (
	ChaparImage     = MustLoadImageOp("chapar.png")
	GRPCImage       = MustLoadImageOp("grpc.png")
	HTTPImage       = MustLoadImageOp("http.png")
	CollectionImage = MustLoadImageOp("collection.png")
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

func MustLoadImage(fileName string) image.Image {
	img, err := LoadImage(fileName)
	if err != nil {
		panic(fmt.Sprintf("failed to load image %s: %v", fileName, err))
	}
	return img
}

func MustLoadImageOp(fileName string) paint.ImageOp {
	img := MustLoadImage(fileName)
	return paint.NewImageOp(img)
}
