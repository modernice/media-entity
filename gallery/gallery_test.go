package gallery_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/internal/testcmp"
)

func TestGallery_NewStack(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	id := uuid.New()

	img := newImage()
	stack, err := g.NewStack(id, img)
	if err != nil {
		t.Fatalf("add stack: %v", err)
	}

	if stack.ID != id {
		t.Fatalf("stack should have id %q; got %q", id, stack.ID)
	}

	if stack.Tags == nil {
		t.Fatalf("Tags should be initialized")
	}

	if len(stack.Variants) != 1 {
		t.Fatalf("stack should have 1 image; has %d", len(stack.Variants))
	}

	found, ok := g.Stack(id)
	if !ok {
		t.Fatalf("added stack not found in gallery")
	}

	originalImg := img
	originalImg.Original = true

	testcmp.Equal(t, "added stack differs from found stack", stack, found)
	testcmp.EqualImages(t, "stack image differs from provided image", originalImg, stack.Variants[0])
}

func TestGallery_NewStack_ErrEmptyID(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	if _, err := g.NewStack(uuid.Nil, newImage()); !errors.Is(err, gallery.ErrEmptyID) {
		t.Fatalf("adding stack with empty stack id should return ErrEmptyID; got %v", err)
	}

	id := uuid.New()
	if _, err := g.NewStack(id, gallery.Image[uuid.UUID]{}); !errors.Is(err, gallery.ErrEmptyID) {
		t.Fatalf("adding stack with empty image id should return ErrEmptyID; got %v", err)
	}
}

func TestGallery_NewStack_ErrDuplicateID(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	id := uuid.New()

	_, err := g.NewStack(id, newImage())
	if err != nil {
		t.Fatalf("add stack: %v", err)
	}

	if _, err := g.NewStack(id, newImage()); !errors.Is(err, gallery.ErrDuplicateID) {
		t.Fatalf("adding stack with duplicate id should return ErrDuplicateID; got %v", err)
	}
}

func TestGallery_NewStack_normalizeImage(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	id := uuid.New()

	stack, err := g.NewStack(id, gallery.Image[uuid.UUID]{ID: uuid.New()})
	if err != nil {
		t.Fatalf("add stack: %v", err)
	}

	for _, img := range stack.Variants {
		if img.Names == nil {
			t.Fatalf("Names should be instantiated")
		}

		if img.Descriptions == nil {
			t.Fatalf("Descriptions should be instantiated")
		}
	}
}

func TestGallery_RemoveStack(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	g.NewStack(uuid.New(), newImage())
	g.NewStack(uuid.New(), newImage())
	stack, _ := g.NewStack(uuid.New(), newImage())
	g.NewStack(uuid.New(), newImage())

	removed, err := g.RemoveStack(stack.ID)
	if err != nil {
		t.Fatalf("remove stack: %v", err)
	}

	if len(g.Stacks) != 3 {
		t.Fatalf("gallery should have 3 stacks; has %d", len(g.Stacks))
	}

	testcmp.Equal(t, "removed stack differs from provided stack", stack, removed)
}

func TestGallery_NewVariant(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	id := uuid.New()

	original := newImage()
	stack, err := g.NewStack(id, original)
	if err != nil {
		t.Fatalf("add stack: %v", err)
	}

	variant := newImage()
	variant.ID = uuid.New()

	stack, err = g.NewVariant(stack.ID, variant)
	if err != nil {
		t.Fatalf("add variant: %v", err)
	}

	if len(stack.Variants) != 2 {
		t.Fatalf("stack should have 2 images; has %d", len(stack.Variants))
	}

	testcmp.Equal(t, "added variant differs from provided variant", variant, stack.Variants[1])
}

func TestGallery_NewVariant_ErrDuplicateImage(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	id := uuid.New()

	original := newImage()
	stack, err := g.NewStack(id, original)
	if err != nil {
		t.Fatalf("add stack: %v", err)
	}

	_, err = g.NewVariant(stack.ID, original)
	if !errors.Is(err, gallery.ErrDuplicateID) {
		t.Fatalf("adding variant with existing id should return ErrDuplicateID; got %v", err)
	}
}

func TestGallery_RemoveVariant(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	stack, _ := g.NewStack(uuid.New(), newImage())

	variant := newImage()
	g.NewVariant(stack.ID, variant)

	removed, err := g.RemoveVariant(stack.ID, variant.ID)
	if err != nil {
		t.Fatalf("remove variant: %v", err)
	}

	testcmp.Equal(t, "removed variant differs from provided variant", variant, removed)
}

func TestGallery_ReplaceVariant(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	stack, _ := g.NewStack(uuid.New(), newImage())

	old := newImage()
	g.NewVariant(stack.ID, old)

	update := old
	update.Names["en"] = "updated"

	updated, err := g.ReplaceVariant(stack.ID, update)
	if err != nil {
		t.Fatalf("replace variant: %v", err)
	}

	if len(updated.Variants) != 2 {
		t.Fatalf("updated stack should still have 2 variants; has %d", len(stack.Variants))
	}

	testcmp.Equal(t, "updated variant differs from provided variant", update, updated.Variants[1])
}

func TestGallery_Tag_Untag(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	img := newImage()
	stack, err := g.NewStack(uuid.New(), img)
	if err != nil {
		t.Fatalf("add stack: %v", err)
	}

	tagged, err := g.Tag(stack.ID, "foo", "bar", "baz", "baz", "foo")
	if err != nil {
		t.Fatalf("tag stack: %v", err)
	}

	if tagged.ID != stack.ID {
		t.Fatalf("tagged stack should have id %q; got %q", stack.ID, tagged.ID)
	}

	if len(tagged.Tags) != 3 {
		t.Fatalf("tagged stack should have 3 tags; has %d", len(tagged.Tags))
	}

	for _, tag := range []string{"foo", "bar", "baz"} {
		if !tagged.Tags.Contains(tag) {
			t.Fatalf("tagged stack should have tag %q", tag)
		}
	}

	untagged, err := g.Untag(stack.ID, "foo", "baz")
	if err != nil {
		t.Fatalf("untag stack: %v", err)
	}

	if !untagged.Tags.Contains("bar") {
		t.Fatalf("untagged stack should have tag %q", "bar")
	}

	for _, tag := range []string{"foo", "baz"} {
		if untagged.Tags.Contains(tag) {
			t.Fatalf("tagged stack should not have tag %q", tag)
		}
	}
}

func TestGallery_Sort(t *testing.T) {
	g := gallery.New[uuid.UUID, uuid.UUID]()

	var stackIDs []uuid.UUID
	for i := 0; i < 4; i++ {
		id := uuid.New()
		stackIDs = append(stackIDs, id)
		if _, err := g.NewStack(id, newImage()); err != nil {
			t.Fatalf("add stack #%d: %v", i+1, err)
		}
	}

	expectStackSorting(t, stackIDs, g.Stacks)

	g.Sort([]uuid.UUID{stackIDs[3], stackIDs[1]})
	expectStackSorting(t, []uuid.UUID{stackIDs[3], stackIDs[1], stackIDs[0], stackIDs[2]}, g.Stacks)

	g.Sort([]uuid.UUID{stackIDs[2], stackIDs[3]})
	expectStackSorting(t, []uuid.UUID{stackIDs[2], stackIDs[3], stackIDs[1], stackIDs[0]}, g.Stacks)

	g.Sort([]uuid.UUID{stackIDs[0], stackIDs[2]})
	expectStackSorting(t, []uuid.UUID{stackIDs[0], stackIDs[2], stackIDs[3], stackIDs[1]}, g.Stacks)
}

func expectStackSorting[StackID, ImageID comparable](t *testing.T, sorting []StackID, stacks []gallery.Stack[StackID, ImageID]) {
	if len(sorting) != len(stacks) {
		t.Fatalf("sorting and stacks should have the same length; sorting has %d, stacks has %d", len(sorting), len(stacks))
	}

	for i, id := range sorting {
		sid := stacks[i].ID
		if sid != id {
			t.Fatalf("stack #%d should have id %v; got %v", i+1, id, sid)
		}
	}
}
