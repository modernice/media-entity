package esgallery_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/event/eventbus"
	"github.com/modernice/goes/event/eventstore"
	"github.com/modernice/goes/test"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/goes/esgallery"
	"github.com/modernice/media-entity/internal/galleryx"
	"github.com/modernice/media-entity/internal/testcmp"
	"github.com/modernice/media-entity/internal/testx"
	imgtools "github.com/modernice/media-tools/image"
	"golang.org/x/exp/maps"
)

func TestProcessor_Process(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var storage esgallery.MemoryStorage
	uploader := esgallery.NewUploader[uuid.UUID, uuid.UUID](&storage)
	pp := esgallery.NewProcessor(esgallery.DefaultEncoder, &storage, uploader, uuid.New)

	pipeline := imgtools.Pipeline{
		imgtools.Resize(imgtools.DimensionMap{
			"sm": {640},
			"md": {960},
			"lg": {1280},
		}),
	}

	g := NewTestGallery(uuid.New())

	r := newExample()
	originalVariant := galleryx.NewImage(uuid.New())
	stack, _ := g.NewStack(uuid.New(), originalVariant)

	if esgallery.WasProcessed(stack) {
		t.Fatalf("WasProcessed() with fresh Stack should return false")
	}

	_, err := uploader.UploadVariant(ctx, g, stack.ID, originalVariant.ID, r)
	if err != nil {
		t.Fatalf("upload original image: %v", err)
	}

	result, err := pp.Process(ctx, pipeline, g, stack.ID)
	if err != nil {
		t.Fatalf("process stack: %v", err)
	}

	testProcessorResult(t, result, &storage, g, stack)
}

func TestProcessor_Run_stackAdded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var storage esgallery.MemoryStorage
	uploader := esgallery.NewUploader[uuid.UUID, uuid.UUID](&storage)
	ebus := eventbus.New()
	estore := eventstore.WithBus(eventstore.New(), ebus)
	repo := repository.New(estore)
	galleries := repository.Typed(repo, NewTestGallery)
	p := esgallery.NewProcessor(esgallery.DefaultEncoder, &storage, uploader, uuid.New)
	pp := esgallery.NewPostProcessor(p, ebus, galleries.Fetch)

	pipeline := imgtools.Pipeline{
		imgtools.Resize(imgtools.DimensionMap{
			"sm": {640},
			"md": {960},
			"lg": {1280},
		}),
	}

	g := NewTestGallery(uuid.New())

	r := newExample()
	originalVariant := galleryx.NewImage(uuid.New())
	stack, _ := g.NewStack(uuid.New(), originalVariant)

	_, err := uploader.UploadVariant(ctx, g, stack.ID, originalVariant.ID, r)
	if err != nil {
		t.Fatalf("upload original image: %v", err)
	}

	results, errs, err := pp.Run(ctx, pipeline)
	if err != nil {
		t.Fatalf("run pipeline: %v", err)
	}
	go testx.PanicOn(errs)

	trigger := g.AggregateChanges()[len(g.AggregateChanges())-1]

	// Trigger post-processor
	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	var result esgallery.ProcessorResult[uuid.UUID, uuid.UUID]
	select {
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for post-processor result")
	case result = <-results:
	}

	testProcessorResult(t, result, &storage, g, stack)

	testcmp.Equal(t, "result has invalid trigger", trigger, result.Trigger)
}

func TestProcessor_Run_variantReplaced_original(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var storage esgallery.MemoryStorage
	uploader := esgallery.NewUploader[uuid.UUID, uuid.UUID](&storage)
	ebus := eventbus.New()
	estore := eventstore.WithBus(eventstore.New(), ebus)
	repo := repository.New(estore)
	galleries := repository.Typed(repo, NewTestGallery)
	p := esgallery.NewProcessor(esgallery.DefaultEncoder, &storage, uploader, uuid.New)
	pp := esgallery.NewPostProcessor(p, ebus, galleries.Fetch)

	pipeline := imgtools.Pipeline{
		imgtools.Resize(imgtools.DimensionMap{
			"sm": {640},
			"md": {960},
			"lg": {1280},
		}),
	}

	g := NewTestGallery(uuid.New())

	r := newExample()
	originalVariant := galleryx.NewImage(uuid.New())
	stack, _ := g.NewStack(uuid.New(), originalVariant)

	_, err := uploader.UploadVariant(ctx, g, stack.ID, originalVariant.ID, r)
	if err != nil {
		t.Fatalf("upload original image: %v", err)
	}

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	<-time.After(200 * time.Millisecond)

	results, errs, err := pp.Run(ctx, pipeline)
	if err != nil {
		t.Fatalf("run pipeline: %v", err)
	}
	go testx.PanicOn(errs)

	// Trigger post-processor
	replacement := stack.Original()
	if _, err := g.ReplaceVariant(stack.ID, replacement); err != nil {
		t.Fatalf("replace variant: %v", err)
	}
	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	var result esgallery.ProcessorResult[uuid.UUID, uuid.UUID]
	select {
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for post-processor result")
	case result = <-results:
	}

	testProcessorResult(t, result, &storage, g, stack)
}

func TestProcessor_Run_variantReplaced_variant(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var storage esgallery.MemoryStorage
	uploader := esgallery.NewUploader[uuid.UUID, uuid.UUID](&storage)
	ebus := eventbus.New()
	estore := eventstore.WithBus(eventstore.New(), ebus)
	repo := repository.New(estore)
	galleries := repository.Typed(repo, NewTestGallery)
	p := esgallery.NewProcessor(esgallery.DefaultEncoder, &storage, uploader, uuid.New)
	pp := esgallery.NewPostProcessor(p, ebus, galleries.Fetch)

	pipeline := imgtools.Pipeline{
		imgtools.Resize(imgtools.DimensionMap{
			"sm": {640},
			"md": {960},
			"lg": {1280},
		}),
	}

	g := NewTestGallery(uuid.New())

	r := newExample()
	originalVariant := galleryx.NewImage(uuid.New())
	stack, _ := g.NewStack(uuid.New(), originalVariant)

	_, err := uploader.UploadVariant(ctx, g, stack.ID, originalVariant.ID, r)
	if err != nil {
		t.Fatalf("upload original image: %v", err)
	}

	variantID := uuid.New()
	stack, _ = g.NewVariant(stack.ID, variantID, galleryx.NewImage(uuid.New()).Image)

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	<-time.After(200 * time.Millisecond)

	results, errs, err := pp.Run(ctx, pipeline)
	if err != nil {
		t.Fatalf("run pipeline: %v", err)
	}
	go testx.PanicOn(errs)

	// Trigger post-processor
	replacement := stack.Last()
	if _, err := g.ReplaceVariant(stack.ID, replacement); err != nil {
		t.Fatalf("replace variant: %v", err)
	}
	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	select {
	case <-time.After(500 * time.Millisecond):
		return
	case <-results:
		t.Fatalf("post-processor should not have triggered")
	}
}

func TestProcessor_Run_WithAutoApply(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var storage esgallery.MemoryStorage
	uploader := esgallery.NewUploader[uuid.UUID, uuid.UUID](&storage)
	ebus := eventbus.New()
	estore := eventstore.WithBus(eventstore.New(), ebus)
	repo := repository.New(estore)
	galleries := repository.Typed(repo, NewTestGallery)
	p := esgallery.NewProcessor(esgallery.DefaultEncoder, &storage, uploader, uuid.New)
	pp := esgallery.NewPostProcessor(p, ebus, galleries.Fetch, esgallery.WithAutoApply[uuid.UUID, uuid.UUID](true, galleries.Save))

	pipeline := imgtools.Pipeline{
		imgtools.Resize(imgtools.DimensionMap{
			"sm": {640},
			"md": {960},
			"lg": {1280},
		}),
	}

	g := NewTestGallery(uuid.New())

	r := newExample()
	originalVariant := galleryx.NewImage(uuid.New())
	stack, _ := g.NewStack(uuid.New(), originalVariant)

	_, err := uploader.UploadVariant(ctx, g, stack.ID, originalVariant.ID, r)
	if err != nil {
		t.Fatalf("upload original image: %v", err)
	}

	results, errs, err := pp.Run(ctx, pipeline)
	if err != nil {
		t.Fatalf("run pipeline: %v", err)
	}
	go testx.PanicOn(errs)

	// Trigger post-processor
	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	var result esgallery.ProcessorResult[uuid.UUID, uuid.UUID]
	select {
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for post-processor result")
	case result = <-results:
	}

	testProcessorResult(t, result, &storage, g, stack)

	if !result.Applied {
		t.Fatalf("Applied field of result should be true")
	}

	if !result.Saved {
		t.Fatalf("Saved field of result should be true")
	}

	g, err = galleries.Fetch(ctx, g.ID)
	if err != nil {
		t.Fatalf("fetch gallery: %v", err)
	}

	if g.AggregateVersion() <= 2 {
		t.Fatalf("expected aggregate version to greater than 2, is %d", g.AggregateVersion())
	}
}

func TestProcessor_Run_WithAutoApply_noSave(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var storage esgallery.MemoryStorage
	uploader := esgallery.NewUploader[uuid.UUID, uuid.UUID](&storage)
	ebus := eventbus.New()
	estore := eventstore.WithBus(eventstore.New(), ebus)
	repo := repository.New(estore)
	galleries := repository.Typed(repo, NewTestGallery)
	p := esgallery.NewProcessor(esgallery.DefaultEncoder, &storage, uploader, uuid.New)
	pp := esgallery.NewPostProcessor(p, ebus, galleries.Fetch, esgallery.WithAutoApply[uuid.UUID, uuid.UUID, *TestGallery](true, nil))

	pipeline := imgtools.Pipeline{
		imgtools.Resize(imgtools.DimensionMap{
			"sm": {640},
			"md": {960},
			"lg": {1280},
		}),
	}

	g := NewTestGallery(uuid.New())

	r := newExample()
	originalVariant := galleryx.NewImage(uuid.New())
	stack, _ := g.NewStack(uuid.New(), originalVariant)

	_, err := uploader.UploadVariant(ctx, g, stack.ID, originalVariant.ID, r)
	if err != nil {
		t.Fatalf("upload original image: %v", err)
	}

	results, errs, err := pp.Run(ctx, pipeline)
	if err != nil {
		t.Fatalf("run pipeline: %v", err)
	}
	go testx.PanicOn(errs)

	// Trigger post-processor
	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	var result esgallery.ProcessorResult[uuid.UUID, uuid.UUID]
	select {
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for post-processor result")
	case result = <-results:
	}

	testProcessorResult(t, result, &storage, g, stack)

	if !result.Applied {
		t.Fatalf("Applied field of result should be true")
	}

	if result.Saved {
		t.Fatalf("Saved field of result should be true")
	}

	g, err = galleries.Fetch(ctx, g.ID)
	if err != nil {
		t.Fatalf("fetch gallery: %v", err)
	}

	if g.AggregateVersion() != 1 {
		t.Fatalf("expected aggregate version to be 1, is %d", g.AggregateVersion())
	}
}

func testProcessorResult(
	t *testing.T,
	result esgallery.ProcessorResult[uuid.UUID, uuid.UUID],
	storage *esgallery.MemoryStorage,
	g *TestGallery,
	stack gallery.Stack[uuid.UUID, uuid.UUID],
) {
	if len(result.Images) != 4 {
		t.Fatalf("expected 4 images in result (including original); got %d", len(result.Images))
	}

	if err := result.Apply(g); err != nil {
		t.Fatalf("apply result: %v", err)
	}

	gstack, ok := g.Stack(stack.ID)
	if !ok {
		t.Fatalf("stack %s not found in gallery", stack.ID)
	}

	if !esgallery.WasProcessed(gstack) {
		t.Fatalf("stack %s should have been processed", stack.ID)
	}

	test.Change(t, g, esgallery.VariantReplaced, test.EventData(esgallery.VariantReplacedData[uuid.UUID, uuid.UUID]{
		StackID: stack.ID,
		Variant: result.Images[0].Image,
	}))

	for _, pimg := range result.Images[1:] {
		test.Change(t, g, esgallery.VariantAdded, test.EventData(esgallery.VariantAddedData[uuid.UUID, uuid.UUID]{
			StackID: stack.ID,
			Variant: pimg.Image,
		}))
	}

	if len(storage.Files()) != 4 {
		t.Fatalf("expected 4 files in storage; got %d\n%s", len(storage.Files()), maps.Keys(storage.Files()))
	}
}
