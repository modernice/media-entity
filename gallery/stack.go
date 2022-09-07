package gallery

import (
	"github.com/modernice/media-entity/image"
	imgtools "github.com/modernice/media-tools/image"
)

// A Stack represents one or multiple variants of the same image. For example,
// an image may exist in multiple sizes.
type Stack[ID, ImageID comparable] struct {
	ID       ID               `json:"id"`
	Variants []Image[ImageID] `json:"variants"`
	Tags     Tags             `json:"tags"`
}

// Tags are the tags of a [Stack].
type Tags = imgtools.Tags

// Image is an image of a [Stack]. An image may be the original image of the
// stack, or a variant of the original image. The ID of an Image is unique
// within a [Stack].
type Image[ID comparable] struct {
	image.Image

	ID       ID   `json:"id"`
	Original bool `json:"original"`
}

// Original returns the original image of the stack, or the zero [Image] if the
// stack does not contain an original image.
func (s Stack[ID, ImageID]) Original() Image[ImageID] {
	for _, img := range s.Variants {
		if img.Original {
			return img
		}
	}
	return zeroImage[ImageID]()
}

// Image returns the [Image] with the given id, or false if the stack does not
// contain an [Image] with that id.
func (s Stack[ID, ImageID]) Image(id ImageID) (Image[ImageID], bool) {
	for _, img := range s.Variants {
		if img.ID == id {
			return img, true
		}
	}
	return zeroImage[ImageID](), false
}

// Tag returns a copy of the [Stack] with the given tags added.
func (s Stack[ID, ImageID]) Tag(tags ...string) Stack[ID, ImageID] {
	s.Tags = s.Tags.With(tags...)
	return s
}

// Tag returns a copy of the [Stack] with the given tags removed.
func (s Stack[ID, ImageID]) Untag(tags ...string) Stack[ID, ImageID] {
	s.Tags = s.Tags.Without(tags...)
	return s
}

func zeroImage[ID comparable]() (zero Image[ID]) {
	return zero
}
