package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRefusesNULCharacters(t *testing.T) {
	type nested struct {
		Note *string  `json:"note"`
		Tags []string `json:"tags"`
	}
	type body struct {
		Name  string `json:"name"`
		Inner nested `json:"inner"`
	}
	cases := map[string]bool{
		`{"name":"ok Ў","inner":{"note":"fine","tags":["a"]}}`: true,
		`{"name":"a\u0000b"}`:               false,
		`{"inner":{"note":"a\u0000b"}}`:     false,
		`{"inner":{"tags":["a","\u0000"]}}`: false,
	}
	for raw, ok := range cases {
		var v body
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(raw))
		if err := DecodeJSON(httptest.NewRecorder(), req, &v); (err == nil) != ok {
			t.Errorf("%s: err = %v, want accepted = %v", raw, err, ok)
		}
	}
}

func TestReadFormRefusesTextTheDatabaseCannotStore(t *testing.T) {
	for _, field := range []string{"name", "keep"} {
		for _, value := range []string{"a\x00b", "a\xffb"} {
			_, body, ct := formRequest(t, fpart{field: field, value: value})
			_, err := read(t, body, ct, spec)
			var fe *FormError
			if !errors.As(err, &fe) {
				t.Errorf("%s=%q: err = %v, want FormError", field, value, err)
			}
		}
	}
}
