package gallery

import (
	"errors"
	"fmt"

	"github.com/modernice/media-entity/image"
	"github.com/modernice/media-entity/internal"
	"github.com/modernice/media-entity/internal/slicex"
	"golang.org/x/exp/slices"
)

var (
	// ErrEmptyID is returned when trying to add an [Image] with an empty id to
	// a [Stack], or when trying to create a [Stack] with an empty id.
	ErrEmptyID = errors.New("empty id")

	// ErrDuplicateID returned when trying to add a [Stack] with an ID that
	// already exists, or when trying to add an [Image] to a [Stack] that
	// already contains an [Image] with the same id.
	ErrDuplicateID = errors.New("duplicate id")

	// ErrStackNotFound is returned when a [Stack] cannot be found in a gallery.
	ErrStackNotFound = errors.New("stack not found in gallery")

	// ErrVariantNotFound is returned when an [Image] cannot be found in a [Stack].
	ErrVariantNotFound = errors.New("variant not found in stack")
)

// ID is the type constraint for [Stack]s and [Image]s of a gallery.
type ID interface {
	comparable
	fmt.Stringer
}

// Base provides the core implementation for image galleries.
type Base[StackID, ImageID ID] struct {
	DTO[StackID, ImageID]
}

// DTO provides the fields for [*Base].
type DTO[StackID, ImageID ID] struct {
	Stacks []Stack[StackID, ImageID] `json:"stacks"`
}

// Stack returns the [Stack] with the given id, or false if no the gallery does
// not contain a [Stack] with the id.
func (dto DTO[StackID, ImageID]) Stack(id StackID) (Stack[StackID, ImageID], bool) {
	for _, stack := range dto.Stacks {
		if stack.ID == id {
			return stack, true
		}
	}
	return zeroStack[StackID, ImageID](), false
}

// New returns a new gallery [*Base] that can be embedded into structs build
// galleries. The ID type of the gallery's stacks is specified by the StackID
// type parameter.
//
//	type MyGallery struct {
//		*gallery.Base[string]
//	}
//
//	func NewGallery() *MyGallery {
//		return &MyGallery{Base: gallery.New[string]()}
//	}
func New[StackID, ImageID ID]() *Base[StackID, ImageID] {
	return &Base[StackID, ImageID]{}
}

// NewStack adds a new [Stack] to the gallery. The provided [Image] will
// be the original image of the [Stack], with its [image.Metadata.Original]
// field set to `true`. If the gallery already contains a [Stack] with the same
// id, the [Stack] is not added to the gallery, and an error that satisfies
// errors.Is(err, ErrDuplicateID) is returned. If the provided stack id or the
// provided image id is empty (zero value), an error that satisfies
// errors.Is(err, ErrEmptyID) is returned.
func (g *Base[StackID, ImageID]) NewStack(id StackID, img Image[ImageID]) (Stack[StackID, ImageID], error) {
	if id == internal.Zero[StackID]() {
		return zeroStack[StackID, ImageID](), fmt.Errorf("stack id: %w", ErrEmptyID)
	}

	if img.ID == internal.Zero[ImageID]() {
		return zeroStack[StackID, ImageID](), fmt.Errorf("image id: %w", ErrEmptyID)
	}

	// Gallery already contains a Stack with the same id.
	if _, ok := g.Stack(id); ok {
		return zeroStack[StackID, ImageID](), fmt.Errorf("stack id: %w", ErrDuplicateID)
	}

	// Force initialize the "Names" and "Descriptions" fields of the image.
	img.Image = img.Normalize()

	// Mark as the original image.
	img.Original = true

	stack := Stack[StackID, ImageID]{
		ID:       id,
		Variants: []Image[ImageID]{img},
		Tags:     make(Tags, 0),
	}

	g.Stacks = append(g.Stacks, stack)

	return stack, nil
}

// RemoveStack removes the [Stack] with the given id from the gallery. If the
// gallery does not contain a [Stack] with the given id, an error that satisfies
// errors.Is(err, ErrStackNotFound) is returned.
func (g *Base[StackID, ImageID]) RemoveStack(id StackID) (Stack[StackID, ImageID], error) {
	for i, s := range g.Stacks {
		if s.ID == id {
			g.Stacks = append(g.Stacks[:i], g.Stacks[i+1:]...)
			return s, nil
		}
	}
	return zeroStack[StackID, ImageID](), ErrStackNotFound
}

// NewVariant adds an image as a new variant to the [Stack] with the given id,
// and returns the updated [Stack] that contains the new [Image]. If the gallery
// does not contain a [Stack] with the given id, an error that satisfies
// errors.Is(err, ErrStackNotFound) is returned.
func (g *Base[StackID, ImageID]) NewVariant(stackID StackID, variantID ImageID, img image.Image) (Stack[StackID, ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return zeroStack[StackID, ImageID](), ErrStackNotFound
	}

	if _, ok := stack.Variant(variantID); ok {
		return zeroStack[StackID, ImageID](), fmt.Errorf("variant id: %w", ErrDuplicateID)
	}

	variant, err := stack.NewVariant(variantID, img)
	if err != nil {
		return zeroStack[StackID, ImageID](), err
	}

	stack.Variants = append(stack.Variants, variant)
	g.replaceStack(stack.ID, stack)

	return stack, nil
}

// RemoveVariant removes a variant from the given [Stack] and returns the
// removed variant. If the gallery does not contain a [Stack] with the given id,
// an error that satisfies errors.Is(err, ErrStackNotFound) is returned. Similarly,
// if the [Stack] does not contain an [Image] with the given ImageID, an error that
// satisfies errors.Is(err, ErrImageNotFound) is returned.
func (g *Base[StackID, ImageID]) RemoveVariant(stackID StackID, imageID ImageID) (Image[ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return zeroImage[ImageID](), ErrStackNotFound
	}

	img, ok := stack.Image(imageID)
	if !ok {
		return zeroImage[ImageID](), ErrVariantNotFound
	}

	stack.Variants = slicex.Filter(stack.Variants, func(v Image[ImageID]) bool {
		return v.ID != imageID
	})

	g.replaceStack(stack.ID, stack)

	return img, nil
}

// ReplaceVariant replaces a variant of the [Stack] with the given StackID.
// If the gallery does not contain a [Stack] with the given id, an error that
// satisfies errors.Is(err, ErrStackNotFound) is returned. Similarly, if the
// [Stack] does not contain an [Image] with the same id as the provided [Image],
// an error that satisfies errors.Is(err, ErrImageNotFound) is returned.
func (g *Base[StackID, ImageID]) ReplaceVariant(stackID StackID, variant Image[ImageID]) (Stack[StackID, ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return zeroStack[StackID, ImageID](), ErrStackNotFound
	}

	variant.Image = variant.Normalize()

	for i, img := range stack.Variants {
		if img.ID == variant.ID {
			stack.Variants[i] = variant
			g.replaceStack(stack.ID, stack)
			return stack, nil
		}
	}

	return stack, ErrVariantNotFound
}

// Tag adds the given tags to the [Stack] with the given id, and returns the
// updated [Stack]. If the gallery does not contain a [Stack] with the given id,
// an error that satisfies errors.Is(err, ErrStackNotFound) is returned.
func (g *Base[StackID, ImageID]) Tag(stackID StackID, tags ...string) (Stack[StackID, ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return zeroStack[StackID, ImageID](), ErrStackNotFound
	}

	stack.Tags = stack.Tags.With(tags...)
	g.replaceStack(stack.ID, stack)

	return stack, nil
}

// Untag removes the given tags from the [Stack] with the given id, and returns
// the updated [Stack]. If the gallery does not contain a [Stack] with the given
// id, an error that satisfies errors.Is(err, ErrStackNotFound) is returned.
func (g *Base[StackID, ImageID]) Untag(stackID StackID, tags ...string) (Stack[StackID, ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return zeroStack[StackID, ImageID](), ErrStackNotFound
	}

	stack.Tags = stack.Tags.Without(tags...)
	g.replaceStack(stack.ID, stack)

	return stack, nil
}

// Sort sorts the gallery's stacks by the given sorting order.
func (g *Base[StackID, ImageID]) Sort(sorting []StackID) {
	// Filter out invalid stack ids.
	sorting = slicex.Filter(sorting, func(id StackID) bool {
		return slicex.ContainsFunc(g.Stacks, func(s Stack[StackID, ImageID]) bool {
			return s.ID == id
		})
	})

	if len(sorting) == 0 {
		return
	}

	previous := slicex.Map(g.Stacks, func(s Stack[StackID, ImageID]) StackID { return s.ID })

	slices.SortFunc(g.Stacks, func(a, b Stack[StackID, ImageID]) bool {
		idxA := slices.Index(sorting, a.ID)
		idxB := slices.Index(sorting, b.ID)

		if idxA == -1 && idxB == -1 {
			idxA = slices.Index(previous, a.ID)
			idxB = slices.Index(previous, b.ID)
		}

		if idxB < 0 {
			return true
		}

		if idxA < 0 {
			return false
		}

		return idxA <= idxB
	})
}

// Clear removes all stacks from the gallery.
func (g *Base[StackID, ImageID]) Clear() {
	g.Stacks = make([]Stack[StackID, ImageID], 0)
}

// DryRun executes the given function and returns the error that is returned by
// that function. Any changes made to the gallery's stacks are reverted before
// returning.
func (g *Base[StackID, ImageID]) DryRun(fn func(*Base[StackID, ImageID]) error) error {
	backup := cloneStacks(g.Stacks)
	err := fn(g)
	g.Stacks = backup
	return err
}

func (g *Base[StackID, ImageID]) replaceStack(id StackID, stack Stack[StackID, ImageID]) {
	for i, s := range g.Stacks {
		if s.ID == id {
			g.Stacks[i] = stack
			return
		}
	}
}

func zeroStack[StackID, ImageID ID]() (zero Stack[StackID, ImageID]) {
	return zero
}

func cloneStacks[StackID, ImageID ID](stacks []Stack[StackID, ImageID]) []Stack[StackID, ImageID] {
	out := make([]Stack[StackID, ImageID], len(stacks))
	for i, stack := range stacks {
		out[i] = stack.Clone()
	}
	return out
}
