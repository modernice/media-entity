package esgallery

import (
	"context"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/codec"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/command/handler"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/image"
)

// Gallery commands
const (
	AddStackCmd       = "esgallery.add_stack"
	RemoveStackCmd    = "esgallery.remove_stack"
	ClearStackCmd     = "esgallery.clear_stack"
	AddVariantsCmd    = "esgallery.add_variants"
	AddVariantCmd     = "esgallery.add_variant"
	RemoveVariantCmd  = "esgallery.remove_variant"
	ReplaceVariantCmd = "esgallery.replace_variant"
	TagStackCmd       = "esgallery.tag_stack"
	UntagStackCmd     = "esgallery.untag_stack"
	SortCmd           = "esgallery.sort"
	ClearCmd          = "esgallery.clear"
)

// Commands is a factory for [Gallery] commands.
type Commands[StackID, ImageID ID] struct {
	aggregateName string
}

// NewCommands returns a factory for creating commands and command handlers.
func NewCommands[StackID, ImageID ID](aggregateName string) *Commands[StackID, ImageID] {
	return &Commands[StackID, ImageID]{aggregateName}
}

// AddStack returns the command to add a new [gallery.Stack] to a [*Gallery].
func (c *Commands[StackID, ImageID]) AddStack(galleryID uuid.UUID, stackID StackID, img gallery.Image[ImageID]) command.Cmd[addStack[StackID, ImageID]] {
	return command.New(AddStackCmd, addStack[StackID, ImageID]{stackID, img}, command.Aggregate(c.aggregateName, galleryID))
}

type addStack[StackID, ImageID ID] struct {
	StackID StackID
	Image   gallery.Image[ImageID]
}

// RemoveStack returns the command to remove a [gallery.Stack] from a [*Gallery].
func (c *Commands[StackID, ImageID]) RemoveStack(galleryID uuid.UUID, stackID StackID) command.Cmd[removeStack[StackID]] {
	return command.New(RemoveStackCmd, removeStack[StackID]{stackID}, command.Aggregate(c.aggregateName, galleryID))
}

type removeStack[StackID ID] struct {
	StackID StackID
}

// AddVariants returns the command to add a multiple [Variant]s to a [gallery.Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) AddVariants(galleryID uuid.UUID, stackID StackID, variants []VariantToAdd[ImageID]) command.Cmd[addVariants[StackID, ImageID]] {
	return command.New(AddVariantCmd, addVariants[StackID, ImageID]{stackID, variants}, command.Aggregate(c.aggregateName, galleryID))
}

type VariantToAdd[ImageID ID] struct {
	VariantID ImageID
	Image     image.Image
}

type addVariants[StackID, ImageID ID] struct {
	StackID  StackID
	Variants []VariantToAdd[ImageID]
}

// AddVariant returns the command to add a new [Variant] to a [gallery.Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) AddVariant(galleryID uuid.UUID, stackID StackID, variantID ImageID, img image.Image) command.Cmd[addVariant[StackID, ImageID]] {
	return command.New(AddVariantCmd, addVariant[StackID, ImageID]{stackID, VariantToAdd[ImageID]{variantID, img}}, command.Aggregate(c.aggregateName, galleryID))
}

type addVariant[StackID, ImageID ID] struct {
	StackID StackID
	VariantToAdd[ImageID]
}

// ClearStack returns the command to clear the variants of a [gallery.Stack].
func (c *Commands[StackID, ImageID]) ClearStack(galleryID uuid.UUID, stackID StackID) command.Cmd[StackID] {
	return command.New(ClearStackCmd, stackID, command.Aggregate(c.aggregateName, galleryID))
}

// RemoveVariant returns the command to remove a [Variant] from a [gallery.Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) RemoveVariant(galleryID uuid.UUID, stackID StackID, variantID ImageID) command.Cmd[removeVariant[StackID, ImageID]] {
	return command.New(RemoveVariantCmd, removeVariant[StackID, ImageID]{stackID, variantID}, command.Aggregate(c.aggregateName, galleryID))
}

type removeVariant[StackID, ImageID ID] struct {
	StackID   StackID
	VariantID ImageID
}

// ReplaceVariant returns the command to replace a [Variant] in a [gallery.Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) ReplaceVariant(galleryID uuid.UUID, stackID StackID, variant gallery.Image[ImageID]) command.Cmd[replaceVariant[StackID, ImageID]] {
	return command.New(ReplaceVariantCmd, replaceVariant[StackID, ImageID]{stackID, variant}, command.Aggregate(c.aggregateName, galleryID))
}

type replaceVariant[StackID, ImageID ID] struct {
	StackID StackID
	Variant gallery.Image[ImageID]
}

// TagStack returns the command to add tags to a [gallery.Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) TagStack(galleryID uuid.UUID, stackID StackID, tags ...string) command.Cmd[tagStack[StackID]] {
	return command.New(TagStackCmd, tagStack[StackID]{stackID, tags}, command.Aggregate(c.aggregateName, galleryID))
}

type tagStack[StackID ID] struct {
	StackID StackID
	Tags    gallery.Tags
}

// UntagStack returns the command to remove tags from a [gallery.Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) UntagStack(galleryID uuid.UUID, stackID StackID, tags ...string) command.Cmd[untagStack[StackID]] {
	return command.New(UntagStackCmd, untagStack[StackID]{stackID, tags}, command.Aggregate(c.aggregateName, galleryID))
}

type untagStack[StackID ID] struct {
	StackID StackID
	Tags    gallery.Tags
}

// Sort returns the command to sort the [gallery.Stack]s in a [*Gallery].
func (c *Commands[StackID, _]) Sort(galleryID uuid.UUID, sorting []StackID) command.Cmd[[]StackID] {
	return command.New(SortCmd, sorting, command.Aggregate(c.aggregateName, galleryID))
}

// Clear returns the command to clear the [gallery.Stack]s of a [*Gallery].
func (c *Commands[StackID, ImageID]) Clear(galleryID uuid.UUID) command.Cmd[struct{}] {
	return command.New(ClearCmd, struct{}{}, command.Aggregate(c.aggregateName, galleryID))
}

// Register calls RegisterCommands(r).
func (c *Commands[StackID, ImageID]) Register(r codec.Registerer) {
	RegisterCommands[StackID, ImageID](r)
}

// Handle subscribes to gallery commands and executes them on the actual
// gallery aggregate that is returned by calling newFunc with the id of the
// gallery.
func (c *Commands[StackID, ImageID]) Handle(
	ctx context.Context,
	newFunc func(uuid.UUID) handler.Aggregate,
	bus command.Bus,
	repo aggregate.Repository,
) (<-chan error, error) {
	return handler.New(newFunc, repo, bus).Handle(ctx)
}

// RegisterCommands registers [Gallery] commands into a command registry.
func RegisterCommands[StackID, ImageID ID](r codec.Registerer) {
	codec.Register[addStack[StackID, ImageID]](r, AddStackCmd)
	codec.Register[removeStack[StackID]](r, RemoveStackCmd)
	codec.Register[StackID](r, ClearStackCmd)
	codec.Register[addVariants[StackID, ImageID]](r, AddVariantsCmd)
	codec.Register[addVariant[StackID, ImageID]](r, AddVariantCmd)
	codec.Register[removeVariant[StackID, ImageID]](r, RemoveVariantCmd)
	codec.Register[replaceVariant[StackID, ImageID]](r, ReplaceVariantCmd)
	codec.Register[tagStack[StackID]](r, TagStackCmd)
	codec.Register[untagStack[StackID]](r, UntagStackCmd)
	codec.Register[[]StackID](r, SortCmd)
	codec.Register[struct{}](r, ClearCmd)
}
