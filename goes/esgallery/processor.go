package esgallery

import (
	"bytes"
	"context"
	"fmt"
	stdimage "image"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/helper/pick"
	"github.com/modernice/goes/helper/streams"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-tools/image"
)

// ProcessorConfig is the type constraint for a [Config] that can be passed to a [*Processor].
type ProcessorConfig[StackID, ImageID ID] interface {
	Config[StackID, ImageID]

	// Encoding returns the configured [Encoding].
	Encoding() Encoding

	// NewVariantID returns a new ID for a new variant of a processed [gallery.Stack].
	NewVariantID() ImageID
}

// Processor post-processes [gallery.Stack]s and uploads the processed images
// to (cloud) storage.
type Processor[Cfg ProcessorConfig[StackID, ImageID], StackID, ImageID ID] struct {
	config   Cfg
	uploader *Uploader[StackID, ImageID]
	storage  Storage
}

// ProcessorResult is the result post-processing a [gallery.Stack].
type ProcessorResult[StackID, ImageID ID] struct {
	image.PipelineResult

	Gallery aggregate.Ref

	// Trigger is the event that triggered the processing. If the [Processor]
	// was called manually, Trigger is nil.
	Trigger event.Event

	// StackID is the ID of processed [Stack].
	StackID StackID

	// Images are the processed images.
	Images []ProcessedImage[ImageID]
}

// ProcessedImage provides the built [gallery.Image], and the processed image
// from the processing pipeline.
type ProcessedImage[ImageID ID] struct {
	Image     gallery.Image[ImageID]
	Processed image.Processed
}

// ProcessableGallery is the type constraint for gallery aggregates that can be
// processed by a [*Processor].
type ProcessableGallery[StackID, ImageID ID] interface {
	pick.AggregateProvider

	// Stack returns the given [gallery.Stack].
	Stack(StackID) (gallery.Stack[StackID, ImageID], bool)
}

// ResultTarget is the type constraint for the target of [ApplyProcessingResult].
type ResultTarget[StackID, ImageID ID] interface {
	ProcessableGallery[StackID, ImageID]

	// ReplaceVariant replaces a variant of a [gallery.Stack].
	ReplaceVariant(StackID, gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error)

	// AddVariant adds a new variant to a [gallery.Stack].
	AddVariant(StackID, gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error)
}

// Config provides a [*Processor] with the factory functions for [gallery.Stack]
// and [gallery.Image] ids. For example, to use UUIDs for stacks, and strings for
// images/variants, create a config like the following:
//
//	cfg := Configure(uuid.New, func() string { return "<some-unique-string>" })
type Config[StackID, ImageID ID] struct {
	encoding     Encoding
	newStackID   func() StackID
	newVariantID func() ImageID
}

// Configure configures the factory functions for [gallery.Stack] and
// [gallery.Image] ids.
func Configure[StackID, ImageID ID](enc Encoding, newStackID func() StackID, newVariantID func() ImageID) Config[StackID, ImageID] {
	return Config[StackID, ImageID]{
		encoding:     enc,
		newStackID:   newStackID,
		newVariantID: newVariantID,
	}
}

// Encoding returns the configured [Encoding].
func (cfg Config[StackID, ImageID]) Encoding() Encoding {
	return cfg.encoding
}

// NewStackID returns a new ID for a [gallery.Stack].
func (cfg Config[StackID, ImageID]) NewStackID() StackID {
	return cfg.newStackID()
}

// NewVariantID returns a new ID for a [gallery.Image].
func (cfg Config[StackID, ImageID]) NewVariantID() ImageID {
	return cfg.newVariantID()
}

// ApplyProcessorResult applies a [ProcessorResult] to a Gallery by raising the appropriate events.
func ApplyProcessorResult[Gallery ResultTarget[StackID, ImageID], StackID, ImageID ID](result ProcessorResult[StackID, ImageID], g Gallery) error {
	stack, ok := g.Stack(result.StackID)
	if !ok {
		return fmt.Errorf("stack %q: %w", result.StackID, gallery.ErrStackNotFound)
	}

	for _, processed := range result.Images {
		// Variant already exists, so we replace it.
		if _, ok := stack.Variant(processed.Image.ID); ok {
			if _, err := g.ReplaceVariant(result.StackID, processed.Image); err != nil {
				return fmt.Errorf("replace variant %q: %w", processed.Image.ID, err)
			}
			continue
		}

		// Variant does not exist, so we add it.
		if _, err := g.AddVariant(result.StackID, processed.Image); err != nil {
			return fmt.Errorf("add variant: %w", err)
		}
	}

	return nil
}

// Apply is a shortcut for ApplyProcessorResult(result, g).
func (result ProcessorResult[StackID, ImageID]) Apply(g ResultTarget[StackID, ImageID]) error {
	return ApplyProcessorResult(result, g)
}

// NewProcessor returns a post-processor for gallery images.
func NewProcessor[Cfg ProcessorConfig[StackID, ImageID], StackID, ImageID ID](
	cfg Cfg,
	uploader *Uploader[StackID, ImageID],
	storage Storage,
) *Processor[Cfg, StackID, ImageID] {
	return &Processor[Cfg, StackID, ImageID]{
		config:   cfg,
		uploader: uploader,
		storage:  storage,
	}
}

// Process post-processes the given [gallery.Stack] of the provided gallery
// ([StackProvider]). The returned [ProcessorResult] can be applied to
// (gallery) aggregates to actually add the processed images to a gallery.
// The provided [image.Pipeline] runs on the original image of the
// [gallery.Stack].
//
// The returned [ProcessorResult] can be applied to a gallery aggregate by
// calling [ApplyProcessorResult]. Appropriate events will be raised to replace
// the original variant of the [gallery.Stack], and/or to add new variants.
//
//	var gallery *Gallery
//	result, err := p.Process(context.TODO(), image.Pipeline{...}, gallery, stackID)
//	// handle err
//	err := result.Apply(gallery)
func (p *Processor[Config, StackID, ImageID]) Process(
	ctx context.Context,
	pipeline image.Pipeline,
	g ProcessableGallery[StackID, ImageID],
	stackID StackID,
) (ProcessorResult[StackID, ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return zeroResult[StackID, ImageID](), gallery.ErrStackNotFound
	}

	original := stack.Original()

	galleryID, galleryName, _ := g.Aggregate()
	path := variantPath(galleryID, stackID, original.ID, original.Filename)

	// Fetch the original image from storage
	r, err := p.storage.Get(ctx, path)
	if err != nil {
		return zeroResult[StackID, ImageID](), fmt.Errorf("storage: %w", err)
	}

	// Detect content-type while decoding image
	var detectCT detectContentType
	r = io.TeeReader(r, &detectCT)

	img, _, err := stdimage.Decode(r)
	if err != nil {
		return zeroResult[StackID, ImageID](), fmt.Errorf("decode original image: %w", err)
	}

	contentType := detectCT.ContentType()

	result, err := pipeline.Run(ctx, img)
	if err != nil {
		return zeroResult[StackID, ImageID](), fmt.Errorf("pipeline: %w", err)
	}

	processed := make([]ProcessedImage[ImageID], len(result.Images))
	for i, pimg := range result.Images {
		// If the result image is the original image, we keep the variant id,
		// so that the original image will be replaced by [ApplyProcessingResult].
		// Otherwise, we generate an id for the new variant, so that it is appended
		// to the Stack.
		variantID := original.ID
		if !pimg.Original {
			variantID = p.config.NewVariantID()
		}

		// Encode the variant into the original image format that was detected earlier.
		var buf bytes.Buffer
		if err := p.config.Encoding().Encode(&buf, contentType, pimg.Image); err != nil {
			return zeroResult[StackID, ImageID](), fmt.Errorf("encode processed image: %w", err)
		}

		// Upload the variant to storage.
		uploaded, err := p.uploader.Upload(ctx, g, stackID, variantID, &buf)
		if err != nil {
			return zeroResult[StackID, ImageID](), fmt.Errorf("upload processed image: %w", err)
		}

		// Mark the image in the gallery as the original, if it is the original image.
		uploaded.Original = pimg.Original

		processed[i] = ProcessedImage[ImageID]{
			Image:     uploaded,
			Processed: pimg,
		}
	}

	return ProcessorResult[StackID, ImageID]{
		PipelineResult: result,
		Gallery: aggregate.Ref{
			Name: galleryName,
			ID:   galleryID,
		},
		StackID: stackID,
		Images:  processed,
	}, nil
}

type PostProcessor[
	Config ProcessorConfig[StackID, ImageID],
	Gallery ProcessableGallery[StackID, ImageID],
	StackID, ImageID ID,
] struct {
	processor    *Processor[Config, StackID, ImageID]
	bus          event.Bus
	fetchGallery func(context.Context, uuid.UUID) (Gallery, error)
}

func NewPostProcessor[
	Config ProcessorConfig[StackID, ImageID],
	Gallery ProcessableGallery[StackID, ImageID],
	StackID, ImageID ID,
](
	p *Processor[Config, StackID, ImageID],
	bus event.Bus,
	fetchGallery func(context.Context, uuid.UUID) (Gallery, error),
) *PostProcessor[Config, Gallery, StackID, ImageID] {
	return &PostProcessor[Config, Gallery, StackID, ImageID]{
		processor:    p,
		bus:          bus,
		fetchGallery: fetchGallery,
	}
}

func (pp *PostProcessor[Config, Gallery, StackID, ImageID]) Run(ctx context.Context, pipeline image.Pipeline) (
	<-chan ProcessorResult[StackID, ImageID],
	<-chan error,
	error,
) {
	events, errs, err := pp.bus.Subscribe(ctx, ProcessorTriggerEvents...)
	if err != nil {
		return nil, nil, fmt.Errorf("subscribe to %v events: %w", ProcessorTriggerEvents, err)
	}

	results := make(chan ProcessorResult[StackID, ImageID])
	processorErrors := make(chan error)
	outErrors := streams.FanInAll(errs, processorErrors)

	go pp.run(ctx, pipeline, events, results, processorErrors)

	return results, outErrors, nil
}

func (pp *PostProcessor[Config, Gallery, StackID, ImageID]) run(
	ctx context.Context,
	pipeline image.Pipeline,
	events <-chan event.Event,
	result chan<- ProcessorResult[StackID, ImageID],
	errs chan<- error,
) {
	defer close(result)

	fail := func(err error) {
		select {
		case <-ctx.Done():
		case errs <- err:
		}
	}

	push := func(r ProcessorResult[StackID, ImageID]) {
		select {
		case <-ctx.Done():
		case result <- r:
		}
	}

	for evt := range events {
		var (
			result     ProcessorResult[StackID, ImageID]
			err        error
			shouldPush = true
		)

		switch evt.Name() {
		case StackAdded:
			result, err = pp.stackAdded(
				ctx,
				event.Cast[gallery.Stack[StackID, ImageID]](evt),
				pipeline,
			)
		case VariantReplaced:
			result, shouldPush, err = pp.variantReplaced(
				ctx,
				event.Cast[VariantReplacedData[StackID, ImageID]](evt),
				pipeline,
			)
		}

		if err != nil {
			fail(fmt.Errorf("handle %q event: %w", evt.Name(), err))
			continue
		}

		if shouldPush {
			push(result)
		}
	}
}

func (pp *PostProcessor[Config, Gallery, StackID, ImageID]) stackAdded(
	ctx context.Context,
	evt event.Of[gallery.Stack[StackID, ImageID]],
	pipeline image.Pipeline,
) (zero ProcessorResult[StackID, ImageID], _ error) {
	galleryID := pick.AggregateID(evt)
	g, err := pp.fetchGallery(ctx, galleryID)
	if err != nil {
		return zero, fmt.Errorf("fetch gallery: %w", err)
	}

	data := evt.Data()

	result, err := pp.processor.Process(ctx, pipeline, g, data.ID)
	if err != nil {
		return result, fmt.Errorf("run processor: %w", err)
	}

	return result, nil
}

func (pp *PostProcessor[Config, Gallery, StackID, ImageID]) variantReplaced(
	ctx context.Context,
	evt event.Of[VariantReplacedData[StackID, ImageID]],
	pipeline image.Pipeline,
) (zero ProcessorResult[StackID, ImageID], _ bool, _ error) {
	galleryID := pick.AggregateID(evt)
	g, err := pp.fetchGallery(ctx, galleryID)
	if err != nil {
		return zero, false, fmt.Errorf("fetch gallery: %w", err)
	}

	data := evt.Data()

	if !data.Variant.Original {
		return zero, false, nil
	}

	result, err := pp.processor.Process(ctx, pipeline, g, data.StackID)
	if err != nil {
		return result, false, fmt.Errorf("run processor: %w", err)
	}

	return result, true, nil
}

func zeroResult[StackID, ImageID ID]() (zero ProcessorResult[StackID, ImageID]) {
	return zero
}

type detectContentType struct {
	written []byte
	done    bool
}

func (mr *detectContentType) Write(p []byte) (int, error) {
	if mr.done {
		return 0, nil
	}

	const max = 512
	if len(p) > max {
		p = p[:max]
	}

	mr.written = append(mr.written, p...)
	if len(mr.written) >= max {
		mr.done = true
	}

	return len(p), nil
}

func (mr *detectContentType) ContentType() string {
	raw := http.DetectContentType(mr.written)
	ct := strings.Split(raw, ";")
	if len(ct) == 0 {
		return raw
	}
	return ct[0]
}
