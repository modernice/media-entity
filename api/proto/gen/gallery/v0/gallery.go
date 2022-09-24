package gallerypb

import (
	imagepb "github.com/modernice/media-entity/api/proto/gen/image/v0"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/internal/slicex"
)

type StringID string

func (id StringID) String() string {
	return string(id)
}

func newStringID(id string) StringID {
	return StringID(id)
}

func New[StackID, ImageID gallery.ID](g gallery.DTO[StackID, ImageID]) *Gallery {
	return &Gallery{
		Stacks: slicex.Map(g.Stacks, NewStack[StackID, ImageID]),
	}
}

func AsGallery[StackID, ImageID gallery.ID](g *Gallery, toStackID func(string) StackID, toImageID func(string) ImageID) gallery.DTO[StackID, ImageID] {
	return gallery.DTO[StackID, ImageID]{
		Stacks: slicex.Map(g.GetStacks(), func(s *Stack) gallery.Stack[StackID, ImageID] {
			return AsStack(s, toStackID, toImageID)
		}),
	}
}

func (g *Gallery) AsGallery() gallery.DTO[StringID, StringID] {
	return AsGallery(g, newStringID, newStringID)
}

func NewStack[StackID, ImageID gallery.ID](s gallery.Stack[StackID, ImageID]) *Stack {
	return &Stack{
		Id:       s.ID.String(),
		Variants: slicex.Map(s.Variants, NewVariant[ImageID]),
		Tags:     s.Tags,
	}
}

func AsStack[StackID, ImageID gallery.ID](s *Stack, toStackID func(string) StackID, toImageID func(string) ImageID) gallery.Stack[StackID, ImageID] {
	return gallery.Stack[StackID, ImageID]{
		ID: toStackID(s.GetId()),
		Variants: slicex.Ensure(slicex.Map(s.GetVariants(), func(img *Image) gallery.Image[ImageID] {
			return AsImage(img, toImageID)
		})),
		Tags: slicex.Ensure(s.GetTags()),
	}
}

func (s *Stack) AsStack() gallery.Stack[StringID, StringID] {
	return AsStack(s, newStringID, newStringID)
}

func NewVariant[ID gallery.ID](img gallery.Image[ID]) *Image {
	return &Image{
		Image:    imagepb.New(img.Image),
		Id:       img.ID.String(),
		Original: img.Original,
	}
}

func AsImage[ID gallery.ID](img *Image, toImageID func(string) ID) gallery.Image[ID] {
	return gallery.Image[ID]{
		Image:    img.GetImage().AsImage(),
		ID:       toImageID(img.GetId()),
		Original: img.GetOriginal(),
	}
}

func (img *Image) AsImage() gallery.Image[StringID] {
	return AsImage(img, newStringID)
}
