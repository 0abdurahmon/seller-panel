package httpx

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fpart struct {
	field, value string
	file         []byte
}

func formRequest(t *testing.T, parts ...fpart) (*http.Request, []byte, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, p := range parts {
		if p.file != nil {
			fw, err := mw.CreateFormFile(p.field, "f.bin")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := fw.Write(p.file); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := mw.WriteField(p.field, p.value); err != nil {
			t.Fatal(err)
		}
	}
	mw.Close()
	return nil, body.Bytes(), mw.FormDataContentType()
}

func read(t *testing.T, body []byte, contentType string, spec FormSpec) (Form, error) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	return ReadForm(httptest.NewRecorder(), req, spec)
}

var spec = FormSpec{Fields: []string{"name", "note"}, Lists: []string{"keep"}, FileField: "photos", MaxFiles: 2, MaxFileBytes: 100}

func TestReadFormReadsEverything(t *testing.T) {
	_, body, ct := formRequest(t,
		fpart{field: "name", value: "Phone"}, fpart{field: "keep", value: " a "}, fpart{field: "keep", value: ""}, fpart{field: "keep", value: "b"},
		fpart{field: "photos", file: []byte("one")}, fpart{field: "photos", file: []byte("two")})
	f, err := read(t, body, ct, spec)
	if err != nil {
		t.Fatal(err)
	}
	if f.Values["name"] != "Phone" || len(f.Files) != 2 || string(f.Files[1]) != "two" {
		t.Fatalf("form = %+v", f)
	}
	if got := f.Lists["keep"]; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("keep = %v, want trimmed non empty values", got)
	}
}

func TestReadFormTellsNotSentFromSentEmpty(t *testing.T) {
	_, body, ct := formRequest(t, fpart{field: "name", value: "x"})
	f, err := read(t, body, ct, spec)
	if _, sent := f.Lists["keep"]; err != nil || sent {
		t.Fatalf("keep not sent: sent=%v err=%v", sent, err)
	}
	_, body, ct = formRequest(t, fpart{field: "name", value: "x"}, fpart{field: "keep", value: ""})
	f, err = read(t, body, ct, spec)
	if got, sent := f.Lists["keep"]; err != nil || !sent || len(got) != 0 {
		t.Fatalf("keep sent empty: %v sent=%v err=%v", got, sent, err)
	}
}

func TestReadFormRefusesMistakes(t *testing.T) {
	big := bytes.Repeat([]byte{1}, 101)
	cases := map[string][]fpart{
		"unknown field":         {{field: "color", value: "red"}},
		"field sent twice":      {{field: "name", value: "a"}, {field: "name", value: "b"}},
		"file in a wrong field": {{field: "avatar", file: []byte("x")}},
		"three files":           {{field: "photos", file: []byte("1")}, {field: "photos", file: []byte("2")}, {field: "photos", file: []byte("3")}},
		"empty file":            {{field: "photos", file: []byte{}}},
	}
	for name, parts := range cases {
		t.Run(name, func(t *testing.T) {
			_, body, ct := formRequest(t, parts...)
			_, err := read(t, body, ct, spec)
			var fe *FormError
			if !errors.As(err, &fe) {
				t.Fatalf("err = %v, want a FormError", err)
			}
		})
	}
	t.Run("file over the limit", func(t *testing.T) {
		_, body, ct := formRequest(t, fpart{field: "photos", file: big})
		if _, err := read(t, body, ct, spec); !errors.Is(err, ErrFormTooLarge) {
			t.Fatalf("err = %v, want ErrFormTooLarge", err)
		}
	})
	t.Run("body over the limit", func(t *testing.T) {
		chunk := strings.Repeat("a", 60<<10)
		_, body, ct := formRequest(t, fpart{field: "name", value: chunk}, fpart{field: "note", value: chunk},
			fpart{field: "keep", value: chunk}, fpart{field: "keep", value: chunk}, fpart{field: "keep", value: chunk}, fpart{field: "keep", value: chunk})
		if _, err := read(t, body, ct, spec); !errors.Is(err, ErrFormTooLarge) {
			t.Fatalf("err = %v, want ErrFormTooLarge", err)
		}
	})
	t.Run("a value that is too long", func(t *testing.T) {
		_, body, ct := formRequest(t, fpart{field: "name", value: strings.Repeat("a", maxFormValueBytes+1)})
		var fe *FormError
		if _, err := read(t, body, ct, spec); !errors.As(err, &fe) {
			t.Fatalf("err = %v, want a FormError", err)
		}
	})
	t.Run("no boundary", func(t *testing.T) {
		if _, err := read(t, []byte("x"), "multipart/form-data", spec); !errors.Is(err, ErrFormBody) {
			t.Fatalf("err = %v, want ErrFormBody", err)
		}
	})
}

// A body that stops anywhere before its closing boundary is never accepted as a complete form.
func TestReadFormRefusesATruncatedBodyAtEveryPoint(t *testing.T) {
	_, body, ct := formRequest(t, fpart{field: "name", value: "Phone"}, fpart{field: "keep", value: "a"}, fpart{field: "photos", file: []byte("image bytes")})
	if _, err := read(t, body, ct, spec); err != nil {
		t.Fatalf("the whole body must be accepted: %v", err)
	}
	// The last two bytes are the CRLF after the closing boundary, a body without them is still complete.
	for cut := 1; cut < len(body)-2; cut++ {
		if _, err := read(t, body[:cut], ct, spec); err == nil {
			t.Fatalf("a body cut at %d of %d bytes was accepted", cut, len(body))
		}
	}
}

func TestIsMultipart(t *testing.T) {
	for ct, want := range map[string]bool{
		"multipart/form-data; boundary=x": true, "Multipart/Form-Data; boundary=x": true,
		"application/json": false, "": false, "multipart/mixed; boundary=x": false,
	} {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Content-Type", ct)
		if got := IsMultipart(req); got != want {
			t.Errorf("%q: %v, want %v", ct, got, want)
		}
	}
}
