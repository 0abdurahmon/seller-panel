package product

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/0abdurahmon/seller-panel/backend/internal/httpx"
)

const (
	// maxPhotoBytes bounds what is read per photo, the photo store enforces the same limit when it converts.
	maxPhotoBytes = 5 << 20

	photosField     = "photos"
	keepPhotosField = "keep_photos"
)

var formFields = []string{"category_id", "title", "description", "price", "sku"}

type form struct {
	values map[string]string
	files  [][]byte
	keep   *[]string
}

// readForm reads a multipart body: text fields, files in the field "photos" and, on update, repeated
// "keep_photos" values. Unknown and repeated fields are refused, like unknown fields in the JSON body.
func readForm(w http.ResponseWriter, r *http.Request, update bool) (form, error) {
	spec := httpx.FormSpec{Fields: formFields, FileField: photosField, MaxFiles: maxPhotos, MaxFileBytes: maxPhotoBytes}
	if update {
		spec.Lists = []string{keepPhotosField}
	} else {
		spec.Fields = append(append([]string{}, formFields...), "stock")
	}
	raw, err := httpx.ReadForm(w, r, spec)
	if err != nil {
		var fe *httpx.FormError
		switch {
		case errors.As(err, &fe):
			return form{}, invalid("%s", fe.Message)
		case errors.Is(err, httpx.ErrFormTooLarge):
			return form{}, ErrFileTooLarge
		default:
			return form{}, errInvalidBody
		}
	}
	f := form{values: raw.Values, files: raw.Files}
	if keep, sent := raw.Lists[keepPhotosField]; sent {
		f.keep = &keep
	}
	return f, nil
}

func (f form) int64(name string) (int64, error) {
	v, ok := f.values[name]
	if !ok || strings.TrimSpace(v) == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0, invalid("%s must be an integer", name)
	}
	return n, nil
}

func (f form) createInput() (CreateInput, error) {
	var in CreateInput
	var err error
	if in.CategoryID, err = f.int64("category_id"); err != nil {
		return in, err
	}
	if in.Price, err = f.int64("price"); err != nil {
		return in, err
	}
	stock, err := f.int64("stock")
	if err != nil {
		return in, err
	}
	in.Stock = int(stock)
	in.Title, in.Description, in.SKU = f.values["title"], f.values["description"], f.values["sku"]
	return in, nil
}

func (f form) updateInput() (UpdateInput, error) {
	var in UpdateInput
	var err error
	if in.CategoryID, err = f.int64("category_id"); err != nil {
		return in, err
	}
	if in.Price, err = f.int64("price"); err != nil {
		return in, err
	}
	in.Title, in.Description, in.SKU = f.values["title"], f.values["description"], f.values["sku"]
	in.Status = Status(f.values["status"])
	return in, nil
}
