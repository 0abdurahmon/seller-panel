package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/0abdurahmon/seller-panel/backend/internal/product"
	"github.com/0abdurahmon/seller-panel/backend/internal/shop"
	"github.com/0abdurahmon/seller-panel/backend/internal/upload"
)

// productPhotos lets the product feature store images through the upload feature without importing it:
// it implements product.PhotoStore and translates upload errors into product errors.
type productPhotos struct {
	uploads *upload.Service
}

func (p productPhotos) Save(ctx context.Context, shopID int64, image io.Reader) (string, error) {
	img, err := p.uploads.Upload(ctx, shopID, image)
	switch {
	case err == nil:
		return img.URL, nil
	case errors.Is(err, upload.ErrTooLarge):
		return "", fmt.Errorf("%w: at most 5 MB", product.ErrFileTooLarge)
	case errors.Is(err, upload.ErrUnsupportedType):
		return "", fmt.Errorf("%w: only jpeg, png and webp images are accepted", product.ErrUnsupportedMedia)
	case errors.Is(err, upload.ErrBusy):
		return "", product.ErrBusy
	case errors.Is(err, upload.ErrValidation):
		return "", fmt.Errorf("%w: %s", product.ErrValidation, err)
	default:
		return "", err
	}
}

func (p productPhotos) Remove(ctx context.Context, shopID int64, urls []string) {
	for _, u := range urls {
		if err := p.uploads.Remove(ctx, shopID, u); err != nil && !errors.Is(err, upload.ErrNotFound) {
			slog.Error("remove photo", "shop_id", shopID, "url", u, "error", err)
		}
	}
}

// shopLogos does the same for the shop logo: it implements shop.LogoStore on top of upload.Service.
type shopLogos struct {
	uploads *upload.Service
}

func (l shopLogos) Save(ctx context.Context, shopID int64, image io.Reader) (string, error) {
	img, err := l.uploads.Upload(ctx, shopID, image)
	switch {
	case err == nil:
		return img.URL, nil
	case errors.Is(err, upload.ErrTooLarge):
		return "", fmt.Errorf("%w: at most 5 MB", shop.ErrFileTooLarge)
	case errors.Is(err, upload.ErrUnsupportedType):
		return "", fmt.Errorf("%w: only jpeg, png and webp images are accepted", shop.ErrUnsupportedMedia)
	case errors.Is(err, upload.ErrBusy):
		return "", shop.ErrBusy
	case errors.Is(err, upload.ErrValidation):
		return "", fmt.Errorf("%w: %s", shop.ErrValidation, err)
	default:
		return "", err
	}
}

func (l shopLogos) Remove(ctx context.Context, shopID int64, urls []string) {
	productPhotos(l).Remove(ctx, shopID, urls)
}
