package shop

import (
	"context"
	"io"
)

// Shop is the seller's shop profile as returned by the API.
// Optional fields are null when unset.
type Shop struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	LogoURL       *string `json:"logo_url"`
	Description   string  `json:"description"`
	ContactPhone  *string `json:"contact_phone"`
	PickupAddress *string `json:"pickup_address"`
}

// UpdateRequest is the body of PUT /shop. It replaces all editable fields.
// An empty optional field clears the stored value. The logo is not a field: it is a file in a
// multipart request (see LogoChange), so the API never stores a URL the client chose.
type UpdateRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	ContactPhone  string `json:"contact_phone"`
	PickupAddress string `json:"pickup_address"`
}

// LogoChange is what a request does to the logo. Add is a new image, Remove clears the logo,
// neither keeps it; both at once is a validation error.
type LogoChange struct {
	Add    []byte
	Remove bool
}

// LogoStore turns image bytes into a stored file. Save converts and stores one image and returns its URL;
// Remove deletes stored files and never fails the caller (a file left behind only wastes space).
type LogoStore interface {
	Save(ctx context.Context, shopID int64, image io.Reader) (url string, err error)
	Remove(ctx context.Context, shopID int64, urls []string)
}
