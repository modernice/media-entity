package testcmp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/modernice/media-entity/gallery"
)

// Equal compares two values and fails the test if they are not equal.
func Equal(t *testing.T, msg string, a, b any, opts ...cmp.Option) {
	if !cmp.Equal(a, b, opts...) {
		t.Fatalf("%s\n%s", msg, cmp.Diff(a, b, opts...))
	}
}

// EqualImages compares two [gallery.Image]s and fails the test if they are not equal.
func EqualImages[ID comparable](t *testing.T, msg string, a, b gallery.Image[ID]) {
	Equal(t, msg, a, b)
}
