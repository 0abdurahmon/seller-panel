package httpx

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

const (
	maxFormValueBytes = 64 << 10
	// formOverhead leaves room for boundaries, headers and the text fields around the files.
	formOverhead = 256 << 10
)

var (
	// ErrFormBody is a body that is not a readable multipart form.
	ErrFormBody = errors.New("request body is not a valid multipart form")
	// ErrFormTooLarge is a file, or the whole body, over the limit of the form.
	ErrFormTooLarge = errors.New("a file is too large")
)

// FormError is a mistake of the client in a readable form: an unknown or repeated field, too many
// files, an empty file. Handlers answer it as a validation error with this message.
type FormError struct{ Message string }

func (e *FormError) Error() string { return e.Message }

// FormSpec says which parts a multipart request may have.
type FormSpec struct {
	// Fields are the allowed text fields, each at most once.
	Fields []string
	// Lists are the allowed text fields that may repeat.
	Lists []string
	// FileField is the one field that carries files, at most MaxFiles of MaxFileBytes each.
	FileField    string
	MaxFiles     int
	MaxFileBytes int
}

// Form is what ReadForm found.
type Form struct {
	Values map[string]string
	// Lists has a key for every list field that was sent, even when all its values were empty,
	// so "not sent" and "sent empty" can be told apart. Values are trimmed, empty ones are dropped.
	Lists map[string][]string
	Files [][]byte
}

// IsMultipart reports whether the request body is multipart/form-data.
func IsMultipart(r *http.Request) bool {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mt == "multipart/form-data"
}

// ReadForm reads a multipart body into memory with every limit of the spec applied while reading.
// Unknown and repeated fields are refused, like unknown fields in a JSON body.
func ReadForm(w http.ResponseWriter, r *http.Request, spec FormSpec) (Form, error) {
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || params["boundary"] == "" {
		return Form{}, ErrFormBody
	}
	tail := &tailReader{r: http.MaxBytesReader(w, r.Body, int64(spec.MaxFiles*spec.MaxFileBytes+formOverhead))}
	r.Body = tail
	mr, err := r.MultipartReader()
	if err != nil {
		return Form{}, ErrFormBody
	}
	fields := set(spec.Fields)
	lists := set(spec.Lists)
	f := Form{Values: map[string]string{}, Lists: map[string][]string{}}
	for {
		part, err := mr.NextPart()
		// Only a plain io.EOF is the end of the form; the reader wraps the EOF of a body that stops
		// before the closing boundary, and it also reports a body cut between two parts as a clean end.
		if err == io.EOF {
			if _, err := io.Copy(io.Discard, r.Body); err != nil {
				return Form{}, readError(err)
			}
			if !tail.endsWith("--" + params["boundary"] + "--") {
				return Form{}, ErrFormBody
			}
			return f, nil
		}
		if err != nil {
			return Form{}, readError(err)
		}
		name := part.FormName()
		if part.FileName() != "" {
			if name != spec.FileField {
				return Form{}, &FormError{"unexpected file field " + quote(name) + ", files go in the field " + quote(spec.FileField)}
			}
			if len(f.Files) == spec.MaxFiles {
				return Form{}, &FormError{"at most " + strconv.Itoa(spec.MaxFiles) + " files are allowed in " + quote(spec.FileField)}
			}
			data, err := io.ReadAll(io.LimitReader(part, int64(spec.MaxFileBytes)+1))
			if err != nil {
				return Form{}, readError(err)
			}
			if len(data) > spec.MaxFileBytes {
				return Form{}, ErrFormTooLarge
			}
			if len(data) == 0 {
				return Form{}, &FormError{quote(spec.FileField) + " file number " + strconv.Itoa(len(f.Files)+1) + " is empty"}
			}
			f.Files = append(f.Files, data)
			continue
		}
		if !fields[name] && !lists[name] {
			return Form{}, &FormError{"unknown field " + quote(name)}
		}
		raw, err := io.ReadAll(io.LimitReader(part, maxFormValueBytes+1))
		if err != nil {
			return Form{}, readError(err)
		}
		if len(raw) > maxFormValueBytes {
			return Form{}, &FormError{"field " + quote(name) + " is too long"}
		}
		if !ValidText(string(raw)) {
			return Form{}, &FormError{"field " + quote(name) + " is not valid text"}
		}
		if lists[name] {
			if _, seen := f.Lists[name]; !seen {
				f.Lists[name] = []string{}
			}
			if v := strings.TrimSpace(string(raw)); v != "" {
				f.Lists[name] = append(f.Lists[name], v)
			}
			continue
		}
		if _, dup := f.Values[name]; dup {
			return Form{}, &FormError{"field " + quote(name) + " is sent twice"}
		}
		f.Values[name] = string(raw)
	}
}

func readError(err error) error {
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return ErrFormTooLarge
	}
	return ErrFormBody
}

func set(names []string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

func quote(s string) string { return `"` + s + `"` }

// tailReader remembers the last bytes it read, to check that the form ends with its closing boundary.
type tailReader struct {
	r    io.Reader
	tail []byte
}

const tailSize = 256

func (t *tailReader) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	t.tail = append(t.tail, p[:n]...)
	if len(t.tail) > tailSize {
		t.tail = append(t.tail[:0], t.tail[len(t.tail)-tailSize:]...)
	}
	return n, err
}

func (t *tailReader) Close() error {
	if c, ok := t.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (t *tailReader) endsWith(marker string) bool {
	return strings.HasSuffix(strings.TrimRight(string(t.tail), "\r\n \t"), marker)
}
