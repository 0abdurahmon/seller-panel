package dbtest

import (
	"errors"
	"testing"
)

func TestCheckName(t *testing.T) {
	cases := map[string]bool{
		"seller_test":    true,
		"app_dev_test":   true,
		"seller":         false,
		"postgres":       false,
		"seller_testing": false,
		"test_seller":    false,
		"_test":          false,
		"":               false,
	}
	for name, ok := range cases {
		err := CheckName(name)
		if ok && err != nil {
			t.Errorf("%q: unexpected error %v", name, err)
		}
		if !ok && !errors.Is(err, ErrNotTestDatabase) {
			t.Errorf("%q: err = %v, want ErrNotTestDatabase", name, err)
		}
	}
}
