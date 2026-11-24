package upload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
		URL:         s.publicURL + "/" + strconv.FormatInt(shopID, 10) + "/" + name,
		ContentType: ContentType,
		Size:        len(out.data),
		Width:       out.width,
		Height:      out.height,
	}, nil
}

// Open returns a stored file of the shop. A name that is not one we generate is ErrNotFound without touching the disk.
func (s *Service) Open(ctx context.Context, shopID int64, name string) (io.ReadCloser, error) {
	if shopID < 1 || !fileName.MatchString(name) {
		return nil, ErrNotFound
	}
	return s.repo.Open(ctx, shopID, name)
}

// Delete removes a file of the shop. Another shop's file is ErrNotFound.
func (s *Service) Delete(ctx context.Context, shopID int64, name string) error {
	if shopID < 1 || !fileName.MatchString(name) {
		return ErrNotFound
	}
	return s.repo.Delete(ctx, shopID, name)
}

// Owns reports whether url is a file this service stored for the shop: our URL prefix, the shop's id,
// a generated name, and the file still exists. Anything else, such as an external or another shop's URL, is false.
func (s *Service) Owns(ctx context.Context, shopID int64, url string) (bool, error) {
	rest, ok := strings.CutPrefix(url, s.publicURL+"/")
	if !ok {
		return false, nil
	}
	shop, name, ok := strings.Cut(rest, "/")
	if !ok || shop != strconv.FormatInt(shopID, 10) {
		return false, nil
	}
	f, err := s.Open(ctx, shopID, name)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, f.Close()
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
