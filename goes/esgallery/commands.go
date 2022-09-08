package esgallery

import (
	"context"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/codec"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/command/handler"
	"github.com/modernice/media-entity/gallery"
)

// Gallery commands
const (
	AddStackCmd       = "goes.gallery.add_stack"
	RemoveStackCmd    = "goes.gallery.remove_stack"
	AddVariantCmd     = "goes.gallery.add_variant"
	RemoveVariantCmd  = "goes.gallery.remove_variant"
	ReplaceVariantCmd = "goes.gallery.replace_variant"
	TagStackCmd       = "goes.gallery.tag_stack"
	UntagStackCmd     = "goes.gallery.untag_stack"
	SortCmd           = "goes.gallery.sort"
)

// Commands is a factory for [Gallery] commands.
type Commands[StackID, ImageID comparable] struct {
	aggregateName string
}

// NewCommands returns a factory for creating commands and command handlers.
func NewCommands[StackID, ImageID comparable](aggregateName string) *Commands[StackID, ImageID] {
	return &Commands[StackID, ImageID]{aggregateName}
}

// AddStack returns the command to add a new [Stack] to a [*Gallery].
func (c *Commands[StackID, ImageID]) AddStack(galleryID uuid.UUID, stackID StackID, img gallery.Image[ImageID]) command.Cmd[addStack[StackID, ImageID]] {
	return command.New(AddStackCmd, addStack[StackID, ImageID]{stackID, img}, command.Aggregate(c.aggregateName, galleryID))
}

type addStack[StackID, ImageID comparable] struct {
	StackID StackID
	Image   gallery.Image[ImageID]
}

// RemoveStack returns the command to remove a [Stack] from a [*Gallery].
func (c *Commands[StackID, ImageID]) RemoveStack(galleryID uuid.UUID, stackID StackID) command.Cmd[removeStack[StackID]] {
	return command.New(RemoveStackCmd, removeStack[StackID]{stackID}, command.Aggregate(c.aggregateName, galleryID))
}

type removeStack[StackID comparable] struct {
	StackID StackID
}

// AddVariant returns the command to add a new [Variant] to a [Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) AddVariant(galleryID uuid.UUID, stackID StackID, variant gallery.Image[ImageID]) command.Cmd[addVariant[StackID, ImageID]] {
	return command.New(AddVariantCmd, addVariant[StackID, ImageID]{stackID, variant}, command.Aggregate(c.aggregateName, galleryID))
}

type addVariant[StackID, ImageID comparable] struct {
	StackID StackID
	Variant gallery.Image[ImageID]
}

// RemoveVariant returns the command to remove a [Variant] from a [Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) RemoveVariant(galleryID uuid.UUID, stackID StackID, variantID ImageID) command.Cmd[removeVariant[StackID, ImageID]] {
	return command.New(RemoveVariantCmd, removeVariant[StackID, ImageID]{stackID, variantID}, command.Aggregate(c.aggregateName, galleryID))
}

type removeVariant[StackID, ImageID comparable] struct {
	StackID   StackID
	VariantID ImageID
}

// ReplaceVariant returns the command to replace a [Variant] in a [Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) ReplaceVariant(galleryID uuid.UUID, stackID StackID, variant gallery.Image[ImageID]) command.Cmd[replaceVariant[StackID, ImageID]] {
	return command.New(ReplaceVariantCmd, replaceVariant[StackID, ImageID]{stackID, variant}, command.Aggregate(c.aggregateName, galleryID))
}

type replaceVariant[StackID, ImageID comparable] struct {
	StackID StackID
	Variant gallery.Image[ImageID]
}

// TagStack returns the command to add tags to a [Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) TagStack(galleryID uuid.UUID, stackID StackID, tags ...string) command.Cmd[tagStack[StackID]] {
	return command.New(TagStackCmd, tagStack[StackID]{stackID, tags}, command.Aggregate(c.aggregateName, galleryID))
}

type tagStack[StackID comparable] struct {
	StackID StackID
	Tags    gallery.Tags
}

// UntagStack returns the command to remove tags from a [Stack] in a [*Gallery].
func (c *Commands[StackID, ImageID]) UntagStack(galleryID uuid.UUID, stackID StackID, tags ...string) command.Cmd[untagStack[StackID]] {
	return command.New(UntagStackCmd, untagStack[StackID]{stackID, tags}, command.Aggregate(c.aggregateName, galleryID))
}

type untagStack[StackID comparable] struct {
	StackID StackID
	Tags    gallery.Tags
}

// Sort returns the command to sort the [Stack]s in a [*Gallery].
func (c *Commands[StackID, _]) Sort(galleryID uuid.UUID, sorting []StackID) command.Cmd[[]StackID] {
	return command.New(SortCmd, sorting, command.Aggregate(c.aggregateName, galleryID))
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
func RegisterCommands[StackID, ImageID comparable](r codec.Registerer) {
	codec.Register[addStack[StackID, ImageID]](r, AddStackCmd)
	codec.Register[removeStack[StackID]](r, RemoveStackCmd)
	codec.Register[addVariant[StackID, ImageID]](r, AddVariantCmd)
	codec.Register[removeVariant[StackID, ImageID]](r, RemoveVariantCmd)
	codec.Register[replaceVariant[StackID, ImageID]](r, ReplaceVariantCmd)
	codec.Register[tagStack[StackID]](r, TagStackCmd)
	codec.Register[untagStack[StackID]](r, UntagStackCmd)
	codec.Register[[]StackID](r, SortCmd)
}
