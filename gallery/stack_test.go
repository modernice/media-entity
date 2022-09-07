package gallery_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/internal/testcmp"
)

func TestStack_Original(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	id := uuid.New()

	img := newImage()
	stack, err := g.NewStack(id, img)
	if err != nil {
		t.Fatalf("add stack: %v", err)
	}

	testcmp.EqualImages(t, "Stack.Original() should return the original image", stack.Variants[0], stack.Original())
}

func TestStack_Tag_Untag(t *testing.T) {
	s := gallery.Stack[uuid.UUID, uuid.UUID]{}

	tags := []string{"foo", "bar", "baz"}
	s = s.Tag(tags...)

	if len(s.Tags) != 3 {
		t.Fatalf("Stack should have 3 tags; has %d", len(s.Tags))
	}

	for _, tag := range tags {
		if !s.Tags.Contains(tag) {
			t.Fatalf("Stack should have tag %q", tag)
		}
	}

	s = s.Untag("bar", "baz")

	if len(s.Tags) != 1 {
		t.Fatalf("Stack should have 1 tag; has %d", len(s.Tags))
	}

	if !s.Tags.Contains("foo") {
		t.Fatalf("Stack should have tag %q", "foo")
	}
}
