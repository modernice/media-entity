package esgallery

import (
	"github.com/modernice/goes/codec"
	"github.com/modernice/media-entity/gallery"
)

// Gallery events
const (
	StackAdded      = "esgallery.stack_added"
	StackRemoved    = "esgallery.stack_removed"
	StackCleared    = "esgallery.stack_cleared"
	VariantAdded    = "esgallery.variant_added"
	VariantRemoved  = "esgallery.variant_removed"
	VariantReplaced = "esgallery.variant_replaced"
	StackTagged     = "esgallery.stack_tagged"
	StackUntagged   = "esgallery.stack_untagged"
	Sorted          = "esgallery.sorted"
	Cleared         = "esgallery.cleared"
)

// ProcessorTriggerEvents are the events that can trigger a [*Processor].
var ProcessorTriggerEvents = []string{
	StackAdded,
	// VariantReplaced,
}

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
	codec.Register[StackID](r, StackCleared)
	codec.Register[VariantAddedData[StackID, ImageID]](r, VariantAdded)
	codec.Register[VariantRemovedData[StackID, ImageID]](r, VariantRemoved)
	codec.Register[VariantReplacedData[StackID, ImageID]](r, VariantReplaced)
	codec.Register[StackTaggedData[StackID]](r, StackTagged)
	codec.Register[StackUntaggedData[StackID]](r, StackUntagged)
	codec.Register[[]StackID](r, Sorted)
	codec.Register[struct{}](r, Cleared)
}
