package shop

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

// maxLogoBytes bounds what is read for the logo, the logo store enforces the same limit when it converts.
const maxLogoBytes = 5 << 20

var errInvalidBody = errors.New("request body is not valid")

// readForm reads the multipart form of PUT /shop: the profile fields, a logo file in "logo" and "remove_logo".
// Unknown and repeated fields are refused, like unknown fields in the JSON body.
func readForm(w http.ResponseWriter, r *http.Request) (UpdateRequest, LogoChange, error) {
	f, err := httpx.ReadForm(w, r, httpx.FormSpec{
		Fields:    []string{"name", "description", "contact_phone", "pickup_address", "remove_logo"},
		FileField: "logo", MaxFiles: 1, MaxFileBytes: maxLogoBytes,
	})
	if err != nil {
		var fe *httpx.FormError
		switch {
		case errors.As(err, &fe):
			return UpdateRequest{}, LogoChange{}, invalid("%s", fe.Message)
		case errors.Is(err, httpx.ErrFormTooLarge):
			return UpdateRequest{}, LogoChange{}, ErrFileTooLarge
		default:
			return UpdateRequest{}, LogoChange{}, errInvalidBody
		}
	}
	req := UpdateRequest{
		Name: f.Values["name"], Description: f.Values["description"],
		ContactPhone: f.Values["contact_phone"], PickupAddress: f.Values["pickup_address"],
	}
	lc := LogoChange{}
	if len(f.Files) == 1 {
		lc.Add = f.Files[0]
	}
	if v := strings.TrimSpace(f.Values["remove_logo"]); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return UpdateRequest{}, LogoChange{}, invalid("remove_logo must be true or false")
		}
		lc.Remove = b
	}
	return req, lc, nil
}
