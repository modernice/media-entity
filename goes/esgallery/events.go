package esgallery

import (
	"github.com/modernice/goes/codec"
	"github.com/modernice/media-entity/gallery"
)

// Gallery events
const (
	StackAdded      = "goes.gallery.stack_added"
	StackRemoved    = "goes.gallery.stack_removed"
	VariantAdded    = "goes.gallery.variant_added"
	VariantRemoved  = "goes.gallery.variant_removed"
	VariantReplaced = "goes.gallery.variant_replaced"
	StackTagged     = "goes.gallery.stack_tagged"
	StackUntagged   = "goes.gallery.stack_untagged"
	Sorted          = "goes.gallery.sorted"
)

type VariantAddedData[StackID, ImageID ID] struct {
	StackID StackID
	Variant gallery.Image[ImageID]
}

type VariantRemovedData[StackID, ImageID ID] struct {
	StackID StackID
	ImageID ImageID
}

type VariantReplacedData[StackID, ImageID ID] struct {
	StackID StackID
	Variant gallery.Image[ImageID]
}

type StackTaggedData[StackID ID] struct {
	StackID StackID
	Tags    gallery.Tags
}

type StackUntaggedData[StackID ID] struct {
	StackID StackID
	Tags    gallery.Tags
}

// RegisterEvents registers the [*Gallery] events into an event registry.
func RegisterEvents[StackID, ImageID ID](r codec.Registerer) {
	codec.Register[gallery.Stack[StackID, ImageID]](r, StackAdded)
	codec.Register[StackID](r, StackRemoved)
	codec.Register[VariantAddedData[StackID, ImageID]](r, VariantAdded)
	codec.Register[VariantRemovedData[StackID, ImageID]](r, VariantRemoved)
	codec.Register[VariantReplacedData[StackID, ImageID]](r, VariantReplaced)
	codec.Register[StackTaggedData[StackID]](r, StackTagged)
	codec.Register[StackUntaggedData[StackID]](r, StackUntagged)
	codec.Register[[]StackID](r, Sorted)
}
