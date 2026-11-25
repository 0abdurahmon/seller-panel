package upload

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"net/http"

	"github.com/gen2brain/webp"
	xdraw "golang.org/x/image/draw"
)

const webpQuality = 80

type processed struct {
	data          []byte
	width, height int
}

// process validates the bytes by content (never by the client's name or header), decodes them,
// fixes the orientation, scales down, and encodes WebP. Re-encoding also drops all metadata (EXIF, GPS).
func process(raw []byte) (processed, error) {
	if len(raw) == 0 {
		return processed{}, fmt.Errorf("%w: file is empty", ErrValidation)
	}
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return processed{}, fmt.Errorf("%w: only jpeg, png and webp images are accepted", ErrUnsupportedType)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return processed{}, fmt.Errorf("%w: file is not a valid image", ErrValidation)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width*cfg.Height > MaxPixels {
		return processed{}, fmt.Errorf("%w: image is too large in pixels (at most %d megapixels)", ErrValidation, MaxPixels/1_000_000)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return processed{}, fmt.Errorf("%w: file is not a valid image", ErrValidation)
	}
	img = fit(img, MaxSide)
	if http.DetectContentType(raw) == "image/jpeg" {
		img = orient(img, jpegOrientation(raw))
	}

	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: webpQuality}); err != nil {
		return processed{}, fmt.Errorf("encode webp: %w", err)
	}
	b := img.Bounds()
	return processed{data: buf.Bytes(), width: b.Dx(), height: b.Dy()}, nil
}

func fit(src image.Image, maxSide int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSide && h <= maxSide {
		return src
	}
	if w >= h {
		h = h * maxSide / w
		w = maxSide
	} else {
		w = w * maxSide / h
		h = maxSide
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	// Halving first keeps large reductions smooth. A kernel scaler would need about 250 MB per photo.
	for max(src.Bounds().Dx(), src.Bounds().Dy()) > 2*maxSide {
		half := src.Bounds()
		src = scale(src, max(half.Dx()/2, 1), max(half.Dy()/2, 1))
	}
	return scale(src, w, h)
}

func scale(src image.Image, w, h int) image.Image {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	return dst
}

// orient applies an EXIF orientation so the stored image looks upright without the tag.
func orient(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	tmp := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(tmp, tmp.Bounds(), src, b.Min, draw.Src)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2:
				nx, ny = w-1-x, y
			case 3:
				nx, ny = w-1-x, h-1-y
			case 4:
				nx, ny = x, h-1-y
			case 5:
				nx, ny = y, x
			case 6:
				nx, ny = h-1-y, x
			case 7:
				nx, ny = h-1-y, w-1-x
			case 8:
				nx, ny = y, w-1-x
			}
			dst.SetNRGBA(nx, ny, tmp.NRGBAAt(x, y))
		}
	}
	return dst
}
