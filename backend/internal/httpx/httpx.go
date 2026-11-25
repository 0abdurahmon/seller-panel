// Package httpx holds HTTP helpers shared by all features: JSON responses,
// the error shape from docs/openapi, and request decoding.
package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
)

var errInvalidText = errors.New("text has a NUL character or is not valid UTF-8")

// Error is the JSON body returned for every failed request.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes v as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a standard error body.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, Error{Code: code, Message: message})
}

// DecodeJSON reads a JSON body into v. It rejects unknown fields, bodies
// larger than 1 MB and strings with a NUL character.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if !validStrings(reflect.ValueOf(v)) {
		return errInvalidText
	}
	return nil
}
