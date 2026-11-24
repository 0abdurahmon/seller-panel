package upload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	maxConcurrent = 3
	busyWait      = 3 * time.Second
)

var fileName = regexp.MustCompile(`^[0-9a-f]{32}\.webp$`)

// Service validates, converts and stores images.
type Service struct {
	repo      Repository
	publicURL string
	slots     chan struct{}
}

// NewService builds the service. publicURL is the prefix of returned URLs, for example "/api/uploads".
func NewService(repo Repository, publicURL string) *Service {
	return &Service{
		repo:      repo,
		publicURL: strings.TrimRight(publicURL, "/"),
		slots:     make(chan struct{}, maxConcurrent),
	}
}

// Upload converts the image in r to WebP and stores it under the shop.
func (s *Service) Upload(ctx context.Context, shopID int64, r io.Reader) (Image, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxFileBytes+1))
	if err != nil {
		return Image{}, fmt.Errorf("%w: upload body could not be read completely: %w", ErrValidation, err)
	}
	if len(raw) > MaxFileBytes {
		return Image{}, fmt.Errorf("%w: at most %d MB", ErrTooLarge, MaxFileBytes>>20)
	}

	release, err := s.acquire(ctx)
	if err != nil {
		return Image{}, err
	}
	out, err := process(raw)
	release()
	if err != nil {
		return Image{}, err
	}

	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Image{}, fmt.Errorf("generate name: %w", err)
	}
	name := hex.EncodeToString(id[:]) + ".webp"
	if err := s.repo.Save(ctx, shopID, name, out.data); err != nil {
		return Image{}, fmt.Errorf("save image: %w", err)
	}
	return Image{
		URL:    s.publicURL + "/" + strconv.FormatInt(shopID, 10) + "/" + name,
		Size:   len(out.data),
		Width:  out.width,
		Height: out.height,
	}, nil
}

// Delete removes a file of the shop. Another shop's file is ErrNotFound.
func (s *Service) Delete(ctx context.Context, shopID int64, name string) error {
	if shopID < 1 || !fileName.MatchString(name) {
		return ErrNotFound
	}
	return s.repo.Delete(ctx, shopID, name)
}

// Remove deletes the file behind a URL this service returned for the shop. A URL that is not ours
// (another prefix or shop, a foreign name) and a file that is already gone are ErrNotFound.
func (s *Service) Remove(ctx context.Context, shopID int64, url string) error {
	rest, ok := strings.CutPrefix(url, s.publicURL+"/")
	if !ok {
		return ErrNotFound
	}
	shop, name, ok := strings.Cut(rest, "/")
	if !ok || shop != strconv.FormatInt(shopID, 10) {
		return ErrNotFound
	}
	return s.Delete(ctx, shopID, name)
}

func (s *Service) acquire(ctx context.Context) (func(), error) {
	t := time.NewTimer(busyWait)
	defer t.Stop()
	select {
	case s.slots <- struct{}{}:
		return func() { <-s.slots }, nil
	case <-t.C:
		return nil, ErrBusy
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
