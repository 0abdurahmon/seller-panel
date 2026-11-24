package upload

const (
	// MaxFileBytes is the largest accepted upload.
	MaxFileBytes = 5 << 20
	// MaxSide is the longest side kept after processing, larger images are scaled down.
	MaxSide = 2048
	// MaxPixels caps the decoded size, so a small compressed file cannot expand into gigabytes.
	MaxPixels = 20_000_000

	ContentType = "image/webp"
)

// Image is the response of POST /uploads.
type Image struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}
