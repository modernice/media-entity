package esgallery_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/test"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/goes/esgallery"
	"github.com/modernice/media-entity/internal/galleryx"
	"github.com/modernice/media-entity/internal/testcmp"
)

func TestNew(t *testing.T) {
	base := aggregate.New("foo", uuid.New())
	g := esgallery.New[uuid.UUID, uuid.UUID](base)
	if g.Target() != base {
		t.Fatalf("Target() should return %v; got %v", base, g.Target())
	}
}

func TestGallery_NewStack(t *testing.T) {
	g := NewTestGallery(uuid.New())

	id := uuid.New()
	img := galleryx.NewImage(uuid.New())
	stack, err := g.NewStack(id, img)
	if err != nil {
		t.Fatalf("add stack: %v", err)
	}

	if stack.ID != id {
		t.Fatalf("stack id should be %q; is %q", id, stack.ID)
	}

	found, ok := g.Stack(stack.ID)
	if !ok {
		t.Fatalf("stack %q not found in gallery", stack.ID)
	}

	testcmp.Equal(t, "stack in gallery differs from provided returned stack", stack, found)

	test.Change(t, g, esgallery.StackAdded, test.EventData(stack))
}

func TestGallery_RemoveStack(t *testing.T) {
	g := NewTestGallery(uuid.New())

	stack, _ := g.NewStack(uuid.New(), galleryx.NewImage(uuid.New()))

	removed, err := g.RemoveStack(stack.ID)
	if err != nil {
		t.Fatalf("remove stack: %v", err)
	}

	if _, ok := g.Stack(stack.ID); ok {
		t.Fatalf("removed stack %q still in gallery", stack.ID)
	}

	testcmp.Equal(t, "removed stack differs from created stack", removed, stack)

	test.Change(t, g, esgallery.StackRemoved, test.EventData(stack.ID))
}

func TestGallery_ClearStack(t *testing.T) {
	g := NewTestGallery(uuid.New())

	stack, _ := g.NewStack(uuid.New(), galleryx.NewImage(uuid.New()))
	g.NewVariant(stack.ID, uuid.New(), galleryx.NewImage(uuid.New()).Image)
	g.NewVariant(stack.ID, uuid.New(), galleryx.NewImage(uuid.New()).Image)
	g.NewVariant(stack.ID, uuid.New(), galleryx.NewImage(uuid.New()).Image)

	stack, err := g.ClearStack(stack.ID)
	if err != nil {
		t.Fatalf("clear stack: %v", err)
	}

	if len(stack.Variants) != 1 {
		t.Fatalf("stack should only have 1 variant; has %d", len(stack.Variants))
	}

	testcmp.Equal(t, "cleared stack should contain the original image", stack.Original(), stack.Variants[0])

	test.Change(t, g, esgallery.StackCleared, test.EventData(stack.ID))
}

func TestGallery_NewVariant(t *testing.T) {
	g := NewTestGallery(uuid.New())

	stack, _ := g.NewStack(uuid.New(), galleryx.NewImage(uuid.New()))

	variant := galleryx.NewImage(uuid.New())

	stack, err := g.NewVariant(stack.ID, variant.ID, variant.Image)
	if err != nil {
		t.Fatalf("add variant: %v", err)
	}

	if len(stack.Variants) != 2 {
		t.Fatalf("stack should have 2 variants; has %d", len(stack.Variants))
	}

	testcmp.Equal(t, "variant in stack differs from provided variant", variant, stack.Variants[1])

	found, _ := g.Stack(stack.ID)

	testcmp.Equal(t, "returned stack differs from stack in gallery", stack, found)

	test.Change(t, g, esgallery.VariantAdded, test.EventData(esgallery.VariantAddedData[uuid.UUID, uuid.UUID]{
		StackID: stack.ID,
		Variant: variant,
	}))
}

func TestGallery_RemoveVariant(t *testing.T) {
	g := NewTestGallery(uuid.New())

	stack, _ := g.NewStack(uuid.New(), galleryx.NewImage(uuid.New()))

	variant := galleryx.NewImage(uuid.New())

	stack, err := g.NewVariant(stack.ID, variant.ID, variant.Image)
	if err != nil {
		t.Fatalf("add variant: %v", err)
	}

	removed, err := g.RemoveVariant(stack.ID, variant.ID)
	if err != nil {
		t.Fatalf("remove variant: %v", err)
	}

	testcmp.Equal(t, "removed image differs from returned image", variant, removed)

	stack, _ = g.Stack(stack.ID)
	if _, ok := stack.Variant(variant.ID); ok {
		t.Fatalf("removed variant %q still in stack", variant.ID)
	}

	test.Change(t, g, esgallery.VariantRemoved, test.EventData(esgallery.VariantRemovedData[uuid.UUID, uuid.UUID]{
		StackID: stack.ID,
		ImageID: variant.ID,
	}))
}

func TestGallery_ReplaceVariant(t *testing.T) {
	g := NewTestGallery(uuid.New())

	stack, _ := g.NewStack(uuid.New(), galleryx.NewImage(uuid.New()))
	variant := galleryx.NewImage(uuid.New())
	stack, _ = g.NewVariant(stack.ID, variant.ID, variant.Image)

	replacement := variant.Clone()
	replacement.Names["en"] = "replacement"

	replaced, err := g.ReplaceVariant(stack.ID, replacement)
	if err != nil {
		t.Fatalf("replace variant: %v", err)
	}

	if len(replaced.Variants) != 2 {
		t.Fatalf("stack should have 2 variants; has %d", len(replaced.Variants))
	}

	if replaced.Variants[1].Names["en"] != "replacement" {
		t.Fatalf("replaced variant should have name %q; has %q", "replacement", replaced.Variants[1].Names["en"])
	}

	testcmp.Equal(t, "variant in stack differs from provided variant\n", replacement, replaced.Variants[1])

	found, ok := g.Stack(stack.ID)
	if !ok {
		t.Fatalf("stack %q not found in gallery", stack.ID)
	}

	testcmp.Equal(t, "returned stack differs from stack in gallery", found, replaced)

	test.Change(t, g, esgallery.VariantReplaced, test.EventData(esgallery.VariantReplacedData[uuid.UUID, uuid.UUID]{
		StackID: stack.ID,
		Variant: replacement,
	}))
}

func TestGallery_Tag_Untag(t *testing.T) {
	g := NewTestGallery(uuid.New())

	stack, _ := g.NewStack(uuid.New(), galleryx.NewImage(uuid.New()))
	stack, err := g.Tag(stack.ID, "foo", "bar", "baz")
	if err != nil {
		t.Fatalf("tag stack: %v", err)
	}

	if len(stack.Tags) != 3 {
		t.Fatalf("stack should have 3 tags; has %d", len(stack.Tags))
	}

	for _, tag := range []string{"foo", "bar", "baz"} {
		if !stack.Tags.Contains(tag) {
			t.Fatalf("stack should have tag %q", tag)
		}
	}

	found, _ := g.Stack(stack.ID)

	testcmp.Equal(t, "stack in gallery differs from returned stack", stack, found)

	test.Change(t, g, esgallery.StackTagged, test.EventData(esgallery.StackTaggedData[uuid.UUID]{
		StackID: stack.ID,
		Tags:    gallery.Tags{"foo", "bar", "baz"},
	}))

	untagged, err := g.Untag(stack.ID, "bar")
	if err != nil {
		t.Fatalf("untag stack: %v", err)
	}

	if len(untagged.Tags) != 2 {
		t.Fatalf("stack should have 2 tags; has %d", len(untagged.Tags))
	}

	for _, tag := range []string{"foo", "baz"} {
		if !untagged.Tags.Contains(tag) {
			t.Fatalf("stack should have tag %q", tag)
		}
	}

	found, _ = g.Stack(stack.ID)

	testcmp.Equal(t, "stack in gallery differs from returned stack", untagged, found)

	test.Change(t, g, esgallery.StackUntagged, test.EventData(esgallery.StackUntaggedData[uuid.UUID]{
		StackID: stack.ID,
		Tags:    gallery.NewTags("bar"),
	}))
}

func TestGallery_Sort(t *testing.T) {
	g := NewTestGallery(uuid.New())

	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}

	for _, id := range ids {
		g.NewStack(id, galleryx.NewImage(uuid.New()))
	}

	sorting := []uuid.UUID{ids[3], ids[1]}
	g.Sort(sorting)
	expectStackSorting(t, []uuid.UUID{ids[3], ids[1], ids[0], ids[2]}, g.Stacks)

	test.Change(t, g, esgallery.Sorted, test.EventData(sorting))

	sorting = []uuid.UUID{ids[2], ids[3]}
	g.Sort(sorting)
	expectStackSorting(t, []uuid.UUID{ids[2], ids[3], ids[1], ids[0]}, g.Stacks)

	test.Change(t, g, esgallery.Sorted, test.EventData(sorting))
}

func TestGallery_Clear(t *testing.T) {
	g := NewTestGallery(uuid.New())

	for i := 0; i < 10; i++ {
		g.NewStack(uuid.New(), galleryx.NewImage(uuid.New()))
	}

	g.Clear()

	if len(g.Stacks) != 0 {
		t.Fatalf("cleared gallery should have 0 stacks; has %d", len(g.Stacks))
	}

	test.Change(t, g, esgallery.Cleared)
}
