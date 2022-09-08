package esgallery

import (
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/event"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/internal/slicex"
)

// Target is an event-sourced aggregate that acts as a gallery.
// The [*Gallery] provided by this package provides the core functionality
// for image galleries, and applies events and commands to a provided Target.
//
// An aggregate must implement [event.Registerer] and [command.Registerer] to be
// used a Target.
type Target interface {
	aggregate.Aggregate
	event.Registerer
	command.Registerer
}

// Gallery provides the core implementation for image galleries. An aggregate
// that embeds *Gallery implements a ready-to-use image gallery.
//
//	type MyGallery struct {
//		*aggregate.Base
//		*esgallery.Gallery
//	}
//
//	func NewGallery(id uuid.UUID) *MyGallery {
//		g := &MyGallery{Base: aggregate.New("myapp.gallery", id)}
//		g.Gallery = esgallery.New(g)
//		return g
//	}
type Gallery[StackID, ImageID comparable, Aggregate Target] struct {
	*gallery.Base[StackID, ImageID]

	target Aggregate
}

// New returns a new [*Gallery] that applies events and commands to the provided
// target aggregate. Typically, the target aggregate should embed [*Gallery] and
// initialize it within its constructor.
//
//	type MyGallery struct {
//		*aggregate.Base
//		*esgallery.Gallery
//	}
//
//	func NewGallery(id uuid.UUID) *MyGallery {
//		g := &MyGallery{Base: aggregate.New("myapp.gallery", id)}
//		g.Gallery = esgallery.New(g)
//		return g
//	}
func New[StackID, ImageID comparable, Aggregate Target](target Aggregate) *Gallery[StackID, ImageID, Aggregate] {
	g := &Gallery[StackID, ImageID, Aggregate]{
		Base:   gallery.New[StackID, ImageID](),
		target: target,
	}

	event.ApplyWith(target, g.newStack, StackAdded)
	event.ApplyWith(target, g.removeStack, StackRemoved)
	event.ApplyWith(target, g.newVariant, VariantAdded)
	event.ApplyWith(target, g.removeVariant, VariantRemoved)
	event.ApplyWith(target, g.replaceVariant, VariantReplaced)
	event.ApplyWith(target, g.tag, StackTagged)
	event.ApplyWith(target, g.untag, StackUntagged)
	event.ApplyWith(target, g.sort, Sorted)

	command.ApplyWith(target, func(load addStack[StackID, ImageID]) error {
		_, err := g.NewStack(load.StackID, load.Image)
		return err
	}, AddStackCmd)

	command.ApplyWith(target, func(load removeStack[StackID]) error {
		_, err := g.RemoveStack(load.StackID)
		return err
	}, RemoveStackCmd)

	command.ApplyWith(target, func(load addVariant[StackID, ImageID]) error {
		_, err := g.NewVariant(load.StackID, load.Variant)
		return err
	}, AddVariantCmd)

	command.ApplyWith(target, func(load removeVariant[StackID, ImageID]) error {
		_, err := g.RemoveVariant(load.StackID, load.VariantID)
		return err
	}, RemoveVariantCmd)

	command.ApplyWith(target, func(load replaceVariant[StackID, ImageID]) error {
		_, err := g.ReplaceVariant(load.StackID, load.Variant)
		return err
	}, ReplaceVariantCmd)

	command.ApplyWith(target, func(load tagStack[StackID]) error {
		_, err := g.Tag(load.StackID, load.Tags...)
		return err
	}, TagStackCmd)

	command.ApplyWith(target, func(load untagStack[StackID]) error {
		_, err := g.Untag(load.StackID, load.Tags...)
		return err
	}, UntagStackCmd)

	command.ApplyWith(target, func(sorting []StackID) error {
		g.Sort(sorting)
		return nil
	}, SortCmd)

	return g
}

// Target returns the actual aggregate that embeds this Gallery.
func (g *Gallery[StackID, ImageID, Aggregate]) Target() Aggregate {
	return g.target
}

// NewStack is the event-sourced variant of [*gallery.Base.NewStack].
func (g *Gallery[StackID, ImageID, Aggregate]) NewStack(id StackID, img gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error) {
	var stack gallery.Stack[StackID, ImageID]
	if err := g.DryRun(func(g *gallery.Base[StackID, ImageID]) error {
		var err error
		stack, err = g.NewStack(id, img)
		return err
	}); err != nil {
		return stack, err
	}

	aggregate.Next(g.target, StackAdded, stack)

	return stack, nil
}

func (g *Gallery[StackID, ImageID, Aggregate]) newStack(evt event.Of[gallery.Stack[StackID, ImageID]]) {
	stack := evt.Data()
	g.Base.NewStack(stack.ID, stack.Original())
}

// RemoveStack is the event-sourced variant of [*gallery.Base.RemoveStack].
func (g *Gallery[StackID, ImageID, Aggregate]) RemoveStack(id StackID) (gallery.Stack[StackID, ImageID], error) {
	var stack gallery.Stack[StackID, ImageID]
	if err := g.DryRun(func(g *gallery.Base[StackID, ImageID]) error {
		var err error
		stack, err = g.RemoveStack(id)
		return err
	}); err != nil {
		return stack, err
	}

	aggregate.Next(g.target, StackRemoved, id)

	return stack, nil
}

func (g *Gallery[StackID, ImageID, Aggregate]) removeStack(evt event.Of[StackID]) {
	g.Base.RemoveStack(evt.Data())
}

// NewVariant is the event-sourced variant of [*gallery.Base.NewVariant].
func (g *Gallery[StackID, ImageID, Aggregate]) NewVariant(stackID StackID, variant gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error) {
	var stack gallery.Stack[StackID, ImageID]
	if err := g.DryRun(func(g *gallery.Base[StackID, ImageID]) error {
		var err error
		stack, err = g.NewVariant(stackID, variant)
		return err
	}); err != nil {
		return stack, err
	}

	aggregate.Next(g.target, VariantAdded, VariantAddedData[StackID, ImageID]{
		StackID: stackID,
		Variant: variant,
	})

	return stack, nil
}

func (g *Gallery[StackID, ImageID, Aggregate]) newVariant(evt event.Of[VariantAddedData[StackID, ImageID]]) {
	g.Base.NewVariant(evt.Data().StackID, evt.Data().Variant)
}

// RemoveVariant is the event-sourced variant of [*gallery.Base.RemoveVariant].
func (g *Gallery[StackID, ImageID, Aggregate]) RemoveVariant(stackID StackID, imageID ImageID) (gallery.Image[ImageID], error) {
	var variant gallery.Image[ImageID]
	if err := g.DryRun(func(g *gallery.Base[StackID, ImageID]) error {
		var err error
		variant, err = g.RemoveVariant(stackID, imageID)
		return err
	}); err != nil {
		return variant, err
	}

	aggregate.Next(g.target, VariantRemoved, VariantRemovedData[StackID, ImageID]{
		StackID: stackID,
		ImageID: imageID,
	})

	return variant, nil
}

func (g *Gallery[StackID, ImageID, Aggregate]) removeVariant(evt event.Of[VariantRemovedData[StackID, ImageID]]) {
	data := evt.Data()
	g.Base.RemoveVariant(data.StackID, data.ImageID)
}

// ReplaceVariant is the event-sourced variant of [*gallery.Base.ReplaceVariant].
func (g *Gallery[StackID, ImageID, Aggregate]) ReplaceVariant(stackID StackID, variant gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error) {
	var stack gallery.Stack[StackID, ImageID]
	if err := g.DryRun(func(g *gallery.Base[StackID, ImageID]) error {
		var err error
		stack, err = g.ReplaceVariant(stackID, variant)
		return err
	}); err != nil {
		return stack, err
	}

	aggregate.Next(g.target, VariantReplaced, VariantReplacedData[StackID, ImageID]{
		StackID: stackID,
		Variant: variant,
	})

	return stack, nil
}

func (g *Gallery[StackID, ImageID, Aggregate]) replaceVariant(evt event.Of[VariantReplacedData[StackID, ImageID]]) {
	data := evt.Data()
	g.Base.ReplaceVariant(data.StackID, data.Variant)
}

// Tag is the event-sourced variant of [*gallery.Base.Tag].
func (g *Gallery[StackID, ImageID, Aggregate]) Tag(stackID StackID, tags ...string) (gallery.Stack[StackID, ImageID], error) {
	var stack gallery.Stack[StackID, ImageID]
	if err := g.DryRun(func(g *gallery.Base[StackID, ImageID]) error {
		var err error
		stack, err = g.Tag(stackID, tags...)
		return err
	}); err != nil {
		return stack, err
	}

	aggregate.Next(g.target, StackTagged, StackTaggedData[StackID]{
		StackID: stackID,
		Tags:    gallery.NewTags(tags...),
	})

	return stack, nil
}

func (g *Gallery[StackID, ImageID, Aggregate]) tag(evt event.Of[StackTaggedData[StackID]]) {
	data := evt.Data()
	g.Base.Tag(data.StackID, data.Tags...)
}

// Untag is the event-sourced variant of [*gallery.Base.Untag].
func (g *Gallery[StackID, ImageID, Aggregate]) Untag(stackID StackID, tags ...string) (gallery.Stack[StackID, ImageID], error) {
	var stack gallery.Stack[StackID, ImageID]
	if err := g.DryRun(func(g *gallery.Base[StackID, ImageID]) error {
		var err error
		stack, err = g.Untag(stackID, tags...)
		return err
	}); err != nil {
		return stack, err
	}

	aggregate.Next(g.target, StackUntagged, StackUntaggedData[StackID]{
		StackID: stackID,
		Tags:    gallery.NewTags(tags...),
	})

	return stack, nil
}

func (g *Gallery[StackID, ImageID, Aggregate]) untag(evt event.Of[StackUntaggedData[StackID]]) {
	data := evt.Data()
	g.Base.Untag(data.StackID, data.Tags...)
}

// Sort is the event-sourced variant of [*gallery.Base.Sort].
func (g *Gallery[StackID, ImageID, Aggregate]) Sort(sorting []StackID) {
	sorting = slicex.Filter(sorting, func(id StackID) bool {
		return slicex.ContainsFunc(g.Stacks, func(s gallery.Stack[StackID, ImageID]) bool {
			return s.ID == id
		})
	})

	if len(sorting) == 0 {
		return
	}

	aggregate.Next(g.target, Sorted, sorting)
}

func (g *Gallery[StackID, ImageID, Aggregate]) sort(evt event.Of[[]StackID]) {
	g.Base.Sort(evt.Data())
}
