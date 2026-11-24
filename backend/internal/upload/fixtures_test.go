package upload

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/gen2brain/webp"
)

func testImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(x * 255 / w), uint8(y * 255 / h), 128, 255})
		}
	}
	return img
}

func jpegBytes(t testing.TB, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, testImage(w, h), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func pngBytes(t testing.TB, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, testImage(w, h)); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func webpBytes(t testing.TB, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := webp.Encode(&b, testImage(w, h)); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func gifBytes(t testing.TB) []byte {
	t.Helper()
	pal := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{color.Black, color.White})
	var b bytes.Buffer
	if err := gif.Encode(&b, pal, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// withOrientation inserts an EXIF APP1 segment with the given orientation after the JPEG SOI.
func withOrientation(jpg []byte, o uint16) []byte {
	tiff := []byte("MM\x00\x2a\x00\x00\x00\x08")
	entry := make([]byte, 2+12+4)
	binary.BigEndian.PutUint16(entry[0:], 1)
	binary.BigEndian.PutUint16(entry[2:], 0x0112)
	binary.BigEndian.PutUint16(entry[4:], 3)
	binary.BigEndian.PutUint32(entry[6:], 1)
	binary.BigEndian.PutUint16(entry[10:], o)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	payload = append(payload, entry...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	seg = append(seg, payload...)
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}
