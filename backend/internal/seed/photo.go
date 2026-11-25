package seed

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

const photoSide = 600

var photoColors = []color.RGBA{
	{R: 0x4f, G: 0x6d, B: 0x7a, A: 0xff},
	{R: 0xc0, G: 0x6c, B: 0x54, A: 0xff},
	{R: 0x6a, G: 0x8d, B: 0x5f, A: 0xff},
	{R: 0x8a, G: 0x6f, B: 0xa8, A: 0xff},
	{R: 0xc9, G: 0xa2, B: 0x4b, A: 0xff},
}

// placeholder draws a one colour PNG, so every demo product has a photo without shipping image files.
func placeholder(n int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, photoSide, photoSide))
	draw.Draw(img, img.Bounds(), image.NewUniform(photoColors[n%len(photoColors)]), image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
