package esgallery

import (
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/event"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/image"
	"github.com/modernice/media-entity/internal/slicex"
	"golang.org/x/exp/slices"
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

// ID is the type constraint for [gallery.Stack]s and [gallery.Image]s of a gallery.
type ID = gallery.ID

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
type Gallery[StackID, ImageID ID, T Target] struct {
	*gallery.Base[StackID, ImageID]

	target          T
	processedStacks []StackID
}

// ProcessedStacks returns ids of the stacks that have been processed by a post-processor.
func (g *Gallery[StackID, ImageID, T]) ProcessedStacks() []StackID {
	out := make([]StackID, len(g.processedStacks))
	copy(out, g.processedStacks)
	return out
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
func New[StackID, ImageID ID, T Target](target T) *Gallery[StackID, ImageID, T] {
	g := &Gallery[StackID, ImageID, T]{
		Base:   gallery.New[StackID, ImageID](),
		target: target,
	}

	event.ApplyWith(target, g.newStack, StackAdded)
	event.ApplyWith(target, g.removeStack, StackRemoved)
	event.ApplyWith(target, g.clearStack, StackCleared)
	event.ApplyWith(target, g.addVariants, VariantsAdded)
	event.ApplyWith(target, g.addVariant, VariantAdded)
	event.ApplyWith(target, g.removeVariant, VariantRemoved)
	event.ApplyWith(target, g.replaceVariant, VariantReplaced)
	event.ApplyWith(target, g.tag, StackTagged)
	event.ApplyWith(target, g.untag, StackUntagged)
	event.ApplyWith(target, g.sort, Sorted)
	event.ApplyWith(target, g.clear, Cleared)
	event.ApplyWith(target, g.stackProcessed, StackProcessed)

	command.ApplyWith(target, func(load addStack[StackID, ImageID]) error {
		_, err := g.NewStack(load.StackID, load.Image)
		return err
	}, AddStackCmd)

	command.ApplyWith(target, func(load removeStack[StackID]) error {
		_, err := g.RemoveStack(load.StackID)
		return err
	}, RemoveStackCmd)

	command.ApplyWith(target, func(load addVariants[StackID, ImageID]) error {
		_, err := g.AddVariants(load.StackID, load.Variants)
		return err
	}, AddVariantsCmd)

	command.ApplyWith(target, func(load addVariant[StackID, ImageID]) error {
		_, err := g.AddVariant(load.StackID, load.Variant)
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

	command.ApplyWith(target, func(struct{}) error {
		g.Clear()
		return nil
	}, ClearCmd)

	return g
}

// Target returns the actual aggregate that embeds this Gallery.
func (g *Gallery[StackID, ImageID, Target]) Target() Target {
	return g.target
}

// NewStack is the event-sourced variant of [*gallery.Base.NewStack].
func (g *Gallery[StackID, ImageID, Target]) NewStack(id StackID, img gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error) {
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

func (g *Gallery[StackID, ImageID, Target]) newStack(evt event.Of[gallery.Stack[StackID, ImageID]]) {
	stack := evt.Data()
	g.Base.NewStack(stack.ID, stack.Original())
}

// RemoveStack is the event-sourced variant of [*gallery.Base.RemoveStack].
func (g *Gallery[StackID, ImageID, Target]) RemoveStack(id StackID) (gallery.Stack[StackID, ImageID], error) {
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

func (g *Gallery[StackID, ImageID, Target]) removeStack(evt event.Of[StackID]) {
	g.Base.RemoveStack(evt.Data())
}

// ClearStacks removes all variants from a [gallery.Stack] except the original.
func (g *Gallery[StackID, ImageID, T]) ClearStack(id StackID) (gallery.Stack[StackID, ImageID], error) {
	stack, ok := g.Stack(id)
	if !ok {
		return gallery.Stack[StackID, ImageID]{}, gallery.ErrStackNotFound
	}

	if len(stack.Variants) == 0 {
		return stack, nil
	}

	if len(stack.Variants) < 2 && stack.ContainsOriginal() {
		return stack, nil
	}

	aggregate.Next(g.target, StackCleared, id)

	stack, _ = g.Stack(id)

	return stack, nil
}

func (g *Gallery[StackID, ImageID, T]) clearStack(evt event.Of[StackID]) {
	for i, stack := range g.Base.Stacks {
		if stack.ID == evt.Data() {
			g.Base.Stacks[i] = stack.Clear()
			return
		}
	}
}

// NewVariant is the event-sourced variant of [*gallery.Base.NewVariant].
func (g *Gallery[StackID, ImageID, Target]) NewVariant(stackID StackID, variantID ImageID, img image.Image) (gallery.Stack[StackID, ImageID], error) {
	var stack gallery.Stack[StackID, ImageID]
	if err := g.DryRun(func(g *gallery.Base[StackID, ImageID]) error {
		var err error
		stack, err = g.NewVariant(stackID, variantID, img)
		return err
	}); err != nil {
		return stack, err
	}

	aggregate.Next(g.target, VariantAdded, VariantAddedData[StackID, ImageID]{
		StackID: stackID,
		Variant: stack.Last(),
	})

	return stack, nil
}

// AddVariant adds multiple variants to a [gallery.Stack]. Read the
// documentation of g.AddVariant for more information.
func (g *Gallery[StackID, ImageID, Target]) AddVariants(stackID StackID, variants []gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return gallery.ZeroStack[StackID, ImageID](), gallery.ErrStackNotFound
	}

	var zeroID ImageID

	for _, variant := range variants {
		if variant.ID == zeroID {
			return gallery.ZeroStack[StackID, ImageID](), gallery.ErrEmptyID
		}

		if _, ok := stack.Variant(variant.ID); ok {
			return gallery.ZeroStack[StackID, ImageID](), gallery.ErrDuplicateID
		}
	}

	aggregate.Next(g.target, VariantsAdded, VariantsAddedData[StackID, ImageID]{
		StackID:  stackID,
		Variants: variants,
	})

	return stack, nil
}

func (g *Gallery[StackID, ImageID, Target]) addVariants(evt event.Of[VariantsAddedData[StackID, ImageID]]) {
	data := evt.Data()
	for _, variant := range data.Variants {
		g.Base.NewVariant(data.StackID, variant.ID, variant.Image)
	}
}

// AddVariant adds a variant to a [gallery.Stack]. If the [gallery.Stack] cannot
// be found in the gallery, an error that satisfies
// errors.Is(err, [gallery.ErrStackNotFound]) is returned. If the ID of the
// provided variant is empty (zero-value), an error that satisfies
// errors.Is(err, [gallery.ErrEmptyID]) is returned. If the ID of the variant
// already exists within the same [gallery.Stack], an error that satisfies
// errors.Is(err, [gallery.ErrDuplicateID]) is returned.
func (g *Gallery[StackID, ImageID, Target]) AddVariant(stackID StackID, variant gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return gallery.ZeroStack[StackID, ImageID](), gallery.ErrStackNotFound
	}

	var zeroID ImageID
	if variant.ID == zeroID {
		return gallery.ZeroStack[StackID, ImageID](), gallery.ErrEmptyID
	}

	if _, ok := stack.Variant(variant.ID); ok {
		return gallery.ZeroStack[StackID, ImageID](), gallery.ErrDuplicateID
	}

	aggregate.Next(g.target, VariantAdded, VariantAddedData[StackID, ImageID]{
		StackID: stackID,
		Variant: variant,
	})

	return stack, nil
}

func (g *Gallery[StackID, ImageID, Target]) addVariant(evt event.Of[VariantAddedData[StackID, ImageID]]) {
	data := evt.Data()
	g.Base.NewVariant(data.StackID, data.Variant.ID, data.Variant.Image)
}

// RemoveVariant is the event-sourced variant of [*gallery.Base.RemoveVariant].
func (g *Gallery[StackID, ImageID, Target]) RemoveVariant(stackID StackID, imageID ImageID) (gallery.Image[ImageID], error) {
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

func (g *Gallery[StackID, ImageID, Target]) removeVariant(evt event.Of[VariantRemovedData[StackID, ImageID]]) {
	data := evt.Data()
	g.Base.RemoveVariant(data.StackID, data.ImageID)
}

// ReplaceVariant is the event-sourced variant of [*gallery.Base.ReplaceVariant].
func (g *Gallery[StackID, ImageID, Target]) ReplaceVariant(stackID StackID, variant gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error) {
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

func (g *Gallery[StackID, ImageID, Target]) replaceVariant(evt event.Of[VariantReplacedData[StackID, ImageID]]) {
	data := evt.Data()
	g.Base.ReplaceVariant(data.StackID, data.Variant)
}

// Tag is the event-sourced variant of [*gallery.Base.Tag].
func (g *Gallery[StackID, ImageID, Target]) Tag(stackID StackID, tags ...string) (gallery.Stack[StackID, ImageID], error) {
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

func (g *Gallery[StackID, ImageID, Target]) tag(evt event.Of[StackTaggedData[StackID]]) {
	data := evt.Data()
	g.Base.Tag(data.StackID, data.Tags...)
}

// Untag is the event-sourced variant of [*gallery.Base.Untag].
func (g *Gallery[StackID, ImageID, Target]) Untag(stackID StackID, tags ...string) (gallery.Stack[StackID, ImageID], error) {
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

func (g *Gallery[StackID, ImageID, Target]) untag(evt event.Of[StackUntaggedData[StackID]]) {
	data := evt.Data()
	g.Base.Untag(data.StackID, data.Tags...)
}

// Sort is the event-sourced variant of [*gallery.Base.Sort].
func (g *Gallery[StackID, ImageID, Target]) Sort(sorting []StackID) {
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

func (g *Gallery[StackID, ImageID, Target]) sort(evt event.Of[[]StackID]) {
	g.Base.Sort(evt.Data())
}

func (g *Gallery[StackID, ImageID, T]) Clear() {
	if len(g.Stacks) == 0 {
		return
	}
	aggregate.Next(g.target, Cleared, struct{}{})
}

func (g *Gallery[StackID, ImageID, T]) clear(event.Of[struct{}]) {
	g.Base.Clear()
}

// MarkAsProcessed marks a [gallery.Stack] as being processed by a post-processor.
func (g *Gallery[StackID, ImageID, T]) MarkAsProcessed(stackID StackID) {
	if !slices.Contains(g.ProcessedStacks(), stackID) {
		aggregate.Next(g.target, StackProcessed, stackID)
	}
}

func (g *Gallery[StackID, ImageID, T]) stackProcessed(evt event.Of[StackID]) {
	g.processedStacks = append(g.processedStacks, evt.Data())
}
