package gallery

import (
	"fmt"

	"github.com/modernice/media-entity/image"
	"github.com/modernice/media-entity/internal"
	"github.com/modernice/media-entity/internal/slicex"
	imgtools "github.com/modernice/media-tools/image"
)

// A Stack represents one or multiple variants of the same image. For example,
// an image may exist in multiple sizes.
type Stack[StackID, ImageID ID] struct {
	ID       StackID          `json:"id"`
	Variants []Image[ImageID] `json:"variants"`
	Tags     Tags             `json:"tags"`
}

// Tags are the tags of a [Stack].
type Tags = imgtools.Tags

// NewTags returns a new [Tags] with the given tags. Duplicates are removed.
func NewTags(tags ...string) Tags {
	return imgtools.NewTags(tags...)
}

// Image is an image of a [Stack]. An image may be the original image of the
// stack, or a variant of the original image. The ID of an Image is unique
// within a [Stack].
type Image[ImageID ID] struct {
	image.Image

	ID       ImageID `json:"id"`
	Original bool    `json:"original"`
}

// Clone returns a deep-copy of the image.
func (img Image[ID]) Clone() Image[ID] {
	img.Image = img.Image.Clone()
	return img
}

// ZeroStack returns the zero-value [Stack].
func ZeroStack[StackID, ImageID ID]() (zero Stack[StackID, ImageID]) {
	return zero
}

// Last returns the last [Image] of the [Stack].
func (s Stack[StackID, ImageID]) Last() Image[ImageID] {
	if len(s.Variants) == 0 {
		return zeroImage[ImageID]()
	}
	return s.Variants[len(s.Variants)-1]
}

// Clone returns a deep-copy of the Stack.
func (s Stack[StackID, ImageID]) Clone() Stack[StackID, ImageID] {
	variants := make([]Image[ImageID], len(s.Variants))
	for i, img := range s.Variants {
		variants[i] = img.Clone()
	}
	s.Variants = variants
	return s
}

// Original returns the original image of the stack, or the zero [Image] if the
// stack does not contain an original image.
func (s Stack[StackID, ImageID]) Original() Image[ImageID] {
	for _, img := range s.Variants {
		if img.Original {
			return img
		}
	}
	return zeroImage[ImageID]()
}

// ContainsOriginal returns whether the Stack contains an [Image] that has its
// Original field set to true.
func (s Stack[StackID, ImageID]) ContainsOriginal() bool {
	for _, img := range s.Variants {
		if img.Original {
			return true
		}
	}
	return false
}

// Clear returns a copy of the Stack will all variants removed except for the original.
func (s Stack[StackID, ImageID]) Clear() Stack[StackID, ImageID] {
	s = s.Clone()
	s.Variants = slicex.Filter(s.Variants, func(img Image[ImageID]) bool {
		return img.Original
	})
	return s
}

// Image returns the [Image] with the given id, or false if the stack does not
// contain an [Image] with that id.
func (s Stack[StackID, ImageID]) Image(id ImageID) (Image[ImageID], bool) {
	for _, img := range s.Variants {
		if img.ID == id {
			return img, true
		}
	}
	return zeroImage[ImageID](), false
}

// Variant is an alias for s.Image.
func (s Stack[StackID, ImageID]) Variant(id ImageID) (Image[ImageID], bool) {
	return s.Image(id)
}

// NewVariant returns a new gallery [Image] with the given id. No error is
// returned if the provided ImageID already exists in the [Stack].
func (s Stack[StackID, ImageID]) NewVariant(id ImageID, img image.Image) (Image[ImageID], error) {
	if id == internal.Zero[ImageID]() {
		return zeroImage[ImageID](), fmt.Errorf("image id: %w", ErrEmptyID)
	}

	return Image[ImageID]{
		ID:       id,
		Image:    img.Normalize(),
		Original: false,
	}, nil
}

// Tag returns a copy of the [Stack] with the given tags added.
func (s Stack[StackID, ImageID]) Tag(tags ...string) Stack[StackID, ImageID] {
	s.Tags = s.Tags.With(tags...)
	return s
}

// Tag returns a copy of the [Stack] with the given tags removed.
func (s Stack[StackID, ImageID]) Untag(tags ...string) Stack[StackID, ImageID] {
	s.Tags = s.Tags.Without(tags...)
	return s
}

func zeroImage[ImageID ID]() (zero Image[ImageID]) {
	return zero
}
