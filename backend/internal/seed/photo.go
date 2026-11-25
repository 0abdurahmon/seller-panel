package seed

import (
	"bytes"
	"embed"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
)

const photoSide = 600

var photoColors = []color.RGBA{
	{R: 0x4f, G: 0x6d, B: 0x7a, A: 0xff},
	{R: 0xc0, G: 0x6c, B: 0x54, A: 0xff},
	{R: 0x6a, G: 0x8d, B: 0x5f, A: 0xff},
	{R: 0x8a, G: 0x6f, B: 0xa8, A: 0xff},
	{R: 0xc9, G: 0xa2, B: 0x4b, A: 0xff},
}

//go:embed photos/*.jpg
var photos embed.FS

// productPhoto returns the picture of a demo product (sources in photos/SOURCES.md), or a one colour image
// for a product that has none.
func productPhoto(sku string, n int) ([]byte, error) {
	data, err := photos.ReadFile("photos/" + sku + ".jpg")
	if errors.Is(err, fs.ErrNotExist) {
		return placeholder(n)
	}
	return data, err
}

func placeholder(n int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, photoSide, photoSide))
	draw.Draw(img, img.Bounds(), image.NewUniform(photoColors[n%len(photoColors)]), image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
