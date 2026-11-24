package upload

const (
	// MaxFileBytes is the largest accepted image.
	MaxFileBytes = 5 << 20
	// MaxSide is the longest side kept after processing, larger images are scaled down.
	MaxSide = 2048
	// MaxPixels caps the decoded size, so a small compressed file cannot expand into gigabytes.
	MaxPixels = 20_000_000
)

// Image is a stored image.
type Image struct {
	URL    string
	Size   int
	Width  int
	Height int
}
