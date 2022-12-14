package esgallery

import (
	"bytes"
	"context"
	"fmt"
	stdimage "image"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/helper/pick"
	"github.com/modernice/goes/helper/streams"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-tools/image"
	"golang.org/x/exp/slices"
)

// ProcessedTag is added to [gallery.Stack]s that were processed by a [*PostProcessor].
const ProcessedTag = "processed"

var autoSaveMaxTries = 5

// Processor post-processes [gallery.Stack]s and uploads the processed images
// to (cloud) storage.
type Processor[StackID, ImageID ID] struct {
	encoding     Encoding
	newVariantID func() ImageID
	uploader     *Uploader[StackID, ImageID]
	storage      Storage
}

// ProcessorResult is the result post-processing a [gallery.Stack].
type ProcessorResult[StackID, ImageID ID] struct {
	image.PipelineResult

	Gallery aggregate.Ref

	// Trigger is the event that triggered the processing. If the [*Processor]
	// was called manually, Trigger is nil.
	Trigger event.Event

	// StackID is the ID of processed [gallery.Stack].
	StackID StackID

	// Images are the processed images.
	Images []ProcessedImage[ImageID]

	// Runtime is the time it took to process the [gallery.Stack].
	Runtime time.Duration

	// Applied is set to true if the post-processor applied the result to the
	// gallery. This is the case if the [WithAutoApply] option is enabled.
	// Consider passing the [DiscardResults] option to [PostProcessor.Run] when
	// enabling [WithAutoApply], to avoid having to pull the results from the
	// returned channel.
	Applied bool

	// Saved is set to true if the post-processor saved the gallery after
	// applying the result to it. This is the case if a non-nil "save" function
	// was passed to [WithAutoApply]. Saved can only be true if Applied is true.
	Saved bool
}

// ProcessedImage provides the built [gallery.Image], and the processed image
// from the processing pipeline.
type ProcessedImage[ImageID ID] struct {
	Image     gallery.Image[ImageID]
	Processed image.Processed
}

// ProcessableGallery is the type constraint for gallery aggregates that can be
// handled by [*Processor]s and [*Uploader]s.
type ProcessableGallery[StackID, ImageID ID] interface {
	pick.AggregateProvider

	// Stack returns the given [gallery.Stack].
	Stack(StackID) (gallery.Stack[StackID, ImageID], bool)

	// NewStack adds a new [gallery.Stack] to the gallery.
	NewStack(StackID, gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error)

	// ClearStacks removes all variants from a [gallery.Stack] except the original.
	ClearStack(StackID) (gallery.Stack[StackID, ImageID], error)

	// ReplaceVariant replaces a variant of a [gallery.Stack].
	ReplaceVariant(StackID, gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error)

	// AddVariant adds a new variant to a [gallery.Stack].
	AddVariant(StackID, gallery.Image[ImageID]) (gallery.Stack[StackID, ImageID], error)

	// Tag adds tags to a [gallery.Stack].
	Tag(StackID, ...string) (gallery.Stack[StackID, ImageID], error)

	// MarkAsProcessed marks a [gallery.Stack] as being processed by a post-processor.
	MarkAsProcessed(StackID)
}

// WasProcessed returns whether the given [gallery.Stack] was processed by a [*PostProcessor].
func WasProcessed[StackID, ImageID gallery.ID](s gallery.Stack[StackID, ImageID]) bool {
	return s.Tags.Contains(ProcessedTag)
}

// ApplyResultOption is an option for [ProcessorResult.Apply].
type ApplyResultOption func(*applyResultConfig)

// ClearStack returns an [ApplyResultOption] that clears the variants of the
// [gallery.Stack] before adding the processed variants to the Stack.
func ClearStack(clear bool) ApplyResultOption {
	return func(cfg *applyResultConfig) {
		cfg.clearStack = clear
	}
}

type applyResultConfig struct {
	clearStack bool
}

// ApplyProcessorResult applies a [ProcessorResult] to a Gallery by raising the appropriate events.
func (r ProcessorResult[StackID, ImageID]) Apply(g ProcessableGallery[StackID, ImageID], opts ...ApplyResultOption) error {
	var cfg applyResultConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	stack, ok := g.Stack(r.StackID)
	if !ok {
		return fmt.Errorf("%w [stackId=%s]", gallery.ErrStackNotFound, r.StackID)
	}

	if cfg.clearStack {
		if _, err := g.ClearStack(r.StackID); err != nil {
			return fmt.Errorf("clear stack: %w", err)
		}
	}

	for _, processed := range r.Images {
		if _, ok := stack.Variant(processed.Image.ID); ok {
			// Variant already exists, so we replace it.
			if _, err := g.ReplaceVariant(r.StackID, processed.Image); err != nil {
				return fmt.Errorf("replace variant %q: %w", processed.Image.ID, err)
			}
		} else if _, err := g.AddVariant(r.StackID, processed.Image); err != nil {
			// Variant does not exist, so we add it.
			return fmt.Errorf("add variant: %w", err)
		}

		if _, err := g.Tag(stack.ID, ProcessedTag); err != nil {
			return fmt.Errorf("tag stack as processed: %w", err)
		}
	}

	g.MarkAsProcessed(r.StackID)

	return nil
}

// // Apply calls ApplyProcessorResult(result, g, opts...).
// func (result ProcessorResult[StackID, ImageID]) Apply(g ProcessableGallery[StackID, ImageID], opts ...ApplyResultOption) error {
// 	return ApplyProcessorResult(result, g, opts...)
// }

// NewProcessor returns a post-processor for gallery images.
func NewProcessor[StackID, ImageID ID](
	enc Encoding,
	storage Storage,
	uploader *Uploader[StackID, ImageID],
	newVariantID func() ImageID,
) *Processor[StackID, ImageID] {
	return &Processor[StackID, ImageID]{
		encoding:     enc,
		storage:      storage,
		uploader:     uploader,
		newVariantID: newVariantID,
	}
}

// Process post-processes the given [gallery.Stack] of the provided gallery
// ([StackProvider]). The returned [ProcessorResult] can be applied to
// (gallery) aggregates to actually add the processed images to a gallery.
// The provided [image.Pipeline] runs on the original image of the
// [gallery.Stack].
//
// The returned [ProcessorResult] can be applied to a gallery aggregate by
// calling [ProcessorResult.Apply]. Appropriate events will be raised to replace
// the original variant of the [gallery.Stack], and/or to add new variants.
//
//	var gallery *Gallery
//	result, err := p.Process(context.TODO(), image.Pipeline{...}, gallery, stackID)
//	// handle err
//	err := result.Apply(gallery)
func (p *Processor[StackID, ImageID]) Process(
	ctx context.Context,
	pipeline image.Pipeline,
	g ProcessableGallery[StackID, ImageID],
	stackID StackID,
) (ProcessorResult[StackID, ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return zeroResult[StackID, ImageID](), fmt.Errorf("%w [stackId=%s]", gallery.ErrStackNotFound, stackID)
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
			variantID = p.newVariantID()
		}

		// Encode the variant into the original image format that was detected earlier.
		var buf bytes.Buffer
		if err := p.encoding.Encode(&buf, contentType, pimg.Image); err != nil {
			return zeroResult[StackID, ImageID](), fmt.Errorf("encode processed image: %w", err)
		}

		// Upload the variant to storage.
		uploaded, err := p.uploader.UploadVariant(ctx, g, stackID, variantID, &buf)
		if err != nil {
			return zeroResult[StackID, ImageID](), fmt.Errorf("upload processed image: %w", err)
		}

		// Mark the image in the gallery as the original, if it is the original image.
		uploaded.Original = pimg.Original

		// Add the pipeline tags to the image.
		uploaded.Tags = uploaded.Tags.With(pimg.Tags...)

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

// PostProcessor is a post-processor for gallery images. Whenever a new
// [gallery.Stack] is added to a gallery, or whenever the original image of a
// [gallery.Stack] is replaced, the post-processor is triggered to post-process
// that [gallery.Stack].
//
// # Example
//
// This example makes use of [repository.Typed], which returns a
// [aggregate.TypedRepository] that provides a Fetch method that can be directly
// passed to [NewPostProcessor].
//
//	type MyGallery struct { ... }
//	func NewGallery(id uuid.UUID) *MyGallery { return &MyGallery{ ... } }
//
//	var p *Processor
//	var bus event.Bus
//	var repo aggregate.Repository
//
//	galleries := repository.Typed(repo, NewGallery)
//	pp := NewPostProcessor(p, bus, galleries.Fetch)
type PostProcessor[
	Gallery ProcessableGallery[StackID, ImageID],
	StackID, ImageID ID,
] struct {
	processor    *Processor[StackID, ImageID]
	bus          event.Bus
	fetchGallery func(context.Context, uuid.UUID) (Gallery, error)

	// autoSave is only valid/used if autoApply is true
	autoSave  func(context.Context, Gallery) error
	autoApply bool
}

// PostProcessorOption is an option for [NewPostProcessor].
type PostProcessorOption[
	Gallery ProcessableGallery[StackID, ImageID],
	StackID, ImageID ID,
] func(*PostProcessor[Gallery, StackID, ImageID])

// WithAutoApply returns a [PostProcessorOption] that automatically applies
// [ProcessorResult]s to gallery aggregates. If the provided `save` function
// is non-nil, galleries will also be saved after applying the result.
func WithAutoApply[
	StackID, ImageID ID,
	Gallery ProcessableGallery[StackID, ImageID],
](autoApply bool, save func(context.Context, Gallery) error) PostProcessorOption[Gallery, StackID, ImageID] {
	return func(pp *PostProcessor[Gallery, StackID, ImageID]) {
		pp.autoApply = autoApply
		pp.autoSave = save
	}
}

// NewPostProcessor returns a new post-processor for gallery images.
// Read the documentation of [PostProcessor] for more information.
func NewPostProcessor[
	Gallery ProcessableGallery[StackID, ImageID],
	StackID, ImageID ID,
](
	p *Processor[StackID, ImageID],
	bus event.Bus,
	fetchGallery func(context.Context, uuid.UUID) (Gallery, error),
	opts ...PostProcessorOption[Gallery, StackID, ImageID],
) *PostProcessor[Gallery, StackID, ImageID] {
	pp := &PostProcessor[Gallery, StackID, ImageID]{
		processor:    p,
		bus:          bus,
		fetchGallery: fetchGallery,
	}
	for _, opt := range opts {
		opt(pp)
	}
	return pp
}

// RunProcessorOption is an option for [*PostProcessor.Run].
type RunProcessorOption func(*runProcessorConfig)

type runProcessorConfig struct {
	workers        int
	discardResults bool
	debug          bool
	eventFilters   []func(event.Event) bool
}

// Workers returns a [RunProcessorOption] that sets the number of workers for
// [PostProcessor.Run]. Defaults to 1.
func Workers(workers int) RunProcessorOption {
	if workers < 0 {
		workers = 0
	}
	return func(cfg *runProcessorConfig) {
		cfg.workers = workers
	}
}

// DiscardResults returns a [RunProcessorOption] that discards the
// [ProcessorResult]s instead of returning them in the result channel.
// Typically, you want to use this option in conjunction with [WithAutoApply].
func DiscardResults(discard bool) RunProcessorOption {
	return func(cfg *runProcessorConfig) {
		cfg.discardResults = discard
	}
}

// Debug returns a [RunProcessorOption] that enables debug logs.
func Debug(debug bool) RunProcessorOption {
	return func(cfg *runProcessorConfig) {
		cfg.debug = debug
	}
}

// FilterEvents returns a [PostProcessorOption] that calls the provided filter
// function when receiving a [ProcessorTriggerEvents] event. If the filter
// returns false, the post-processor will not be triggered for that event.
// When multiple filters are provided, all filters must return true for the
// post-processor to be triggered.
func FilterEvents(filter func(event.Event) bool) RunProcessorOption {
	return func(cfg *runProcessorConfig) {
		cfg.eventFilters = append(cfg.eventFilters, filter)
	}
}

// FilterAggregates returns a [PostProcessorOption] that checks if a received
// [ProcessorTriggerEvents] event belongs to the event stream of one of the
// provided aggregates. If the event does not belong to any of these aggregates,
// the post-processor will not be triggered for that event.
func FilterAggregates(aggregates []string) RunProcessorOption {
	return FilterEvents(func(e event.Event) bool {
		return slices.Contains(aggregates, pick.AggregateName(e))
	})
}

// Run runs the post-processor in the background and returns a channel of
// results and a channel of errors. Processing stops when the provided Context
// is canceled. If the underlying event bus fails to subscribe to
// [ProcessorTriggerEvents], nil channels and the event bus error are returned.
func (pp *PostProcessor[Gallery, StackID, ImageID]) Run(ctx context.Context, pipeline image.Pipeline, opts ...RunProcessorOption) (
	<-chan ProcessorResult[StackID, ImageID],
	<-chan error,
	error,
) {
	cfg := runProcessorConfig{workers: 1}
	for _, opt := range opts {
		opt(&cfg)
	}

	cfg.debugLog(
		"starting post-processor with options:\n\tWorkers:\t%d\n\tDiscard results:\t%v",
		cfg.workers, cfg.discardResults,
	)

	cfg.debugLog("subscribing to %v events ...", ProcessorTriggerEvents)

	events, errs, err := pp.bus.Subscribe(ctx, ProcessorTriggerEvents...)
	if err != nil {
		return nil, nil, fmt.Errorf("subscribe to %v events: %w", ProcessorTriggerEvents, err)
	}

	results := make(chan ProcessorResult[StackID, ImageID])
	processorErrors := make(chan error)
	outErrors := streams.FanInAll(errs, processorErrors)

	queue := processorQueue[Gallery, StackID, ImageID]{
		ctx:       ctx,
		cfg:       cfg,
		processor: pp,
		pipeline:  pipeline,
		events:    events,
		results:   results,
		errs:      processorErrors,
	}

	go queue.run()

	return results, outErrors, nil
}

type processorQueue[Gallery ProcessableGallery[StackID, ImageID], StackID, ImageID ID] struct {
	ctx       context.Context
	cfg       runProcessorConfig
	processor *PostProcessor[Gallery, StackID, ImageID]
	pipeline  image.Pipeline
	events    <-chan event.Event
	results   chan<- ProcessorResult[StackID, ImageID]
	errs      chan<- error
}

func (q *processorQueue[Gallery, StackID, ImageID]) run() {
	defer close(q.results)

	var wg sync.WaitGroup
	wg.Add(q.cfg.workers)

	for i := 0; i < q.cfg.workers; i++ {
		go func() {
			defer wg.Done()
			q.work()
		}()
	}

	wg.Wait()
}

func (q *processorQueue[Gallery, StackID, ImageID]) work() {
	for evt := range q.events {
		if !q.shouldProcess(evt) {
			continue
		}

		var (
			result     ProcessorResult[StackID, ImageID]
			err        error
			shouldPush = true
			start      = time.Now()
		)

		q.cfg.debugLog("handling %q event ...", evt.Name())

		switch evt.Name() {
		case StackAdded:
			result, err = q.stackAdded(event.Cast[gallery.Stack[StackID, ImageID]](evt))
			// case VariantReplaced:
			// 	result, shouldPush, err = q.variantReplaced(event.Cast[VariantReplacedData[StackID, ImageID]](evt))
		}

		if err != nil {
			q.fail(fmt.Errorf("handle %q event: %w", evt.Name(), err))
			continue
		}

		result.Trigger = evt
		result.Runtime = time.Since(start)

		galleryID := pick.AggregateID(evt)

		if q.processor.autoApply {
			if err := q.apply(&result, galleryID); err != nil {
				q.fail(fmt.Errorf("apply result: %w", err))
				continue
			}
		}

		if q.cfg.discardResults {
			q.cfg.debugLog("discarding processing result [galleryId=%s, stackId=%s]", galleryID, result.StackID)
			continue
		}

		if shouldPush {
			// Update the Runtime because q.apply() might have taken some time.
			result.Runtime = time.Since(start)
			q.push(result)
		}
	}
}

func (q *processorQueue[Gallery, StackID, ImageID]) shouldProcess(evt event.Event) bool {
	for _, filter := range q.cfg.eventFilters {
		if !filter(evt) {
			return false
		}
	}
	return true
}

func (q *processorQueue[Gallery, StackID, ImageID]) apply(result *ProcessorResult[StackID, ImageID], galleryID uuid.UUID) error {
	// The following is true only if the [WithAutoApply] option is provided with
	// the `autoSave` parameter set to `true`:
	//
	// Between the time of fetching and saving the gallery, the gallery might
	// have raised other aggregate events. To avoid optimistic concurrency
	// issues, we re-try the whole process a few times.
	var tries int
	for {
		tries++

		q.cfg.debugLog("fetching gallery to apply result to ... [id=%s]", galleryID)
		g, err := q.processor.fetchGallery(q.ctx, galleryID)
		if err != nil {
			return fmt.Errorf("fetch gallery: %w", err)
		}

		q.cfg.debugLog("applying processing result ... [galleryId=%s, stackId=%s]", galleryID, result.StackID)
		if err := result.Apply(g); err != nil {
			return err
		}
		result.Applied = true

		if q.processor.autoSave != nil {
			q.cfg.debugLog("auto-saving gallery ... [id=%s]", galleryID)
			if err := q.processor.autoSave(q.ctx, g); err != nil {
				if aggregate.IsConsistencyError(err) {
					q.cfg.debugLog("optimistic concurrency error while auto-saving: %v", err)

					if tries >= autoSaveMaxTries {
						return fmt.Errorf("optimistic concurrency error while auto-saving: %w", err)
					}

					q.cfg.debugLog("retrying auto-save because of optimistic concurrency error ...")
					continue
				}

				return fmt.Errorf("auto-save gallery: %w", err)
			}
			result.Saved = true
		}

		return nil
	}
}

func (q *processorQueue[Gallery, StackID, ImageID]) fail(err error) {
	select {
	case <-q.ctx.Done():
	case q.errs <- err:
	}
}

func (q *processorQueue[Gallery, StackID, ImageID]) push(r ProcessorResult[StackID, ImageID]) {
	select {
	case <-q.ctx.Done():
	case q.results <- r:
	}
}

func (q *processorQueue[Gallery, StackID, ImageID]) stackAdded(evt event.Of[gallery.Stack[StackID, ImageID]]) (zero ProcessorResult[StackID, ImageID], _ error) {
	galleryID := pick.AggregateID(evt)
	g, err := q.processor.fetchGallery(q.ctx, galleryID)
	if err != nil {
		return zero, fmt.Errorf("fetch gallery: %w", err)
	}

	stack := evt.Data()

	if _, ok := g.Stack(stack.ID); !ok {
		return zero, fmt.Errorf("%w [galleryId=%s, stackId=%s]", gallery.ErrStackNotFound, galleryID, stack.ID)
	}

	q.cfg.debugLog("running processor on stack ... [galleryId=%s, stackId=%s]", galleryID, stack.ID)

	result, err := q.processor.processor.Process(q.ctx, q.pipeline, g, stack.ID)
	if err != nil {
		return result, fmt.Errorf("run processor: %w", err)
	}

	return result, nil
}

// func (q *processorQueue[
// 	Gallery,
// 	StackID, ImageID,
// ]) variantReplaced(evt event.Of[VariantReplacedData[StackID, ImageID]]) (
// 	zero ProcessorResult[StackID, ImageID],
// 	_ bool, _ error,
// ) {
// 	galleryID := pick.AggregateID(evt)
// 	g, err := q.processor.fetchGallery(q.ctx, galleryID)
// 	if err != nil {
// 		return zero, false, fmt.Errorf("fetch gallery: %w", err)
// 	}

// 	data := evt.Data()

// 	if !data.Variant.Original {
// 		return zero, false, nil
// 	}

// 	result, err := q.processor.processor.Process(q.ctx, q.pipeline, g, data.StackID)
// 	if err != nil {
// 		return result, false, fmt.Errorf("run processor: %w", err)
// 	}

// 	return result, true, nil
// }

func (cfg runProcessorConfig) debugLog(format string, args ...any) {
	if cfg.debug {
		log.Printf("[DEBUG] %s", fmt.Sprintf(format, args...))
	}
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
