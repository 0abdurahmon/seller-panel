package httpx

import (
	"reflect"
	"strings"
	"unicode/utf8"
)

// ValidText reports whether PostgreSQL can store s: valid UTF-8 without a NUL character.
func ValidText(s string) bool {
	return utf8.ValidString(s) && !strings.ContainsRune(s, 0)
}

func validStrings(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return ValidText(v.String())
	case reflect.Pointer, reflect.Interface:
		return v.IsNil() || validStrings(v.Elem())
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() && !validStrings(v.Field(i)) {
				return false
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if !validStrings(v.Index(i)) {
				return false
			}
		}
	case reflect.Map:
		for it := v.MapRange(); it.Next(); {
			if !validStrings(it.Key()) || !validStrings(it.Value()) {
				return false
			}
		}
	}
	return true
}
