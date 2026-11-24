package product

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

const (
	// maxPhotoBytes bounds what is read per photo, the photo store enforces the same limit when it converts.
	maxPhotoBytes     = 5 << 20
	maxFormValueBytes = 64 << 10
	// multipartOverhead leaves room for boundaries, headers and the text fields around the photos.
	multipartOverhead = 256 << 10
	maxFormBody       = maxPhotos*maxPhotoBytes + multipartOverhead

	photosField     = "photos"
	keepPhotosField = "keep_photos"
)

var (
	createFields = map[string]bool{"category_id": true, "title": true, "description": true, "price": true, "sku": true, "stock": true, "status": true}
	updateFields = map[string]bool{"category_id": true, "title": true, "description": true, "price": true, "sku": true, "status": true, keepPhotosField: true}
)

func isMultipart(r *http.Request) bool {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mt == "multipart/form-data"
}

type form struct {
	values map[string]string
	files  [][]byte
	keep   *[]string
}

// readForm reads a multipart body: text fields, files in the field "photos", and repeated "keep_photos" values.
// Unknown fields and repeated fields are refused, like unknown fields in the JSON body.
func readForm(w http.ResponseWriter, r *http.Request, allowed map[string]bool) (form, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBody)
	mr, err := r.MultipartReader()
	if err != nil {
		return form{}, errInvalidBody
	}
	f := form{values: map[string]string{}}
	var keep []string
	keepSeen := false
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if isTooLarge(err) {
				return form{}, ErrFileTooLarge
			}
			return form{}, errInvalidBody
		}
		name := part.FormName()
		if part.FileName() != "" {
			if name != photosField {
				return form{}, invalid("unexpected file field %q, photos go in the field %q", name, photosField)
			}
			if len(f.files) == maxPhotos {
				return form{}, invalid("a product has at most %d photos", maxPhotos)
			}
			data, err := io.ReadAll(io.LimitReader(part, maxPhotoBytes+1))
			if err != nil {
				if isTooLarge(err) {
					return form{}, ErrFileTooLarge
				}
				return form{}, errInvalidBody
			}
			if len(data) > maxPhotoBytes {
				return form{}, ErrFileTooLarge
			}
			if len(data) == 0 {
				return form{}, invalid("photos[%d] is empty", len(f.files))
			}
			f.files = append(f.files, data)
			continue
		}
		if !allowed[name] {
			return form{}, invalid("unknown field %q", name)
		}
		raw, err := io.ReadAll(io.LimitReader(part, maxFormValueBytes+1))
		if err != nil {
			if isTooLarge(err) {
				return form{}, ErrFileTooLarge
			}
			return form{}, errInvalidBody
		}
		if len(raw) > maxFormValueBytes {
			return form{}, invalid("field %q is too long", name)
		}
		value := string(raw)
		if name == keepPhotosField {
			keepSeen = true
			if v := strings.TrimSpace(value); v != "" {
				keep = append(keep, v)
			}
			continue
		}
		if _, dup := f.values[name]; dup {
			return form{}, invalid("field %q is sent twice", name)
		}
		f.values[name] = value
	}
	if keepSeen {
		if keep == nil {
			keep = []string{}
		}
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
	in.Status = Status(f.values["status"])
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
