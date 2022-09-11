package esgallery_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/modernice/goes/event/eventbus"
	"github.com/modernice/goes/test"
	"github.com/modernice/media-entity/goes/esgallery"
	"github.com/modernice/media-entity/internal/galleryx"
	imgtools "github.com/modernice/media-tools/image"
	"golang.org/x/exp/maps"
)

func TestProcessor_Process(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var storage esgallery.MemoryStorage
	cfg := UUIDConfig()
	uploader := esgallery.NewUploader(cfg, &storage)
	ebus := eventbus.New()
	pp := esgallery.NewProcessor(cfg, esgallery.DefaultEncoder, uploader, &storage, ebus)

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

	_, err := uploader.Upload(ctx, g, stack.ID, originalVariant.ID, r)
	if err != nil {
		t.Fatalf("upload original image: %v", err)
	}

	result, err := pp.Process(ctx, pipeline, g, stack.ID)
	if err != nil {
		t.Fatalf("process stack: %v", err)
	}

	if len(result.Images) != 4 {
		t.Fatalf("expected 4 images in result (including original); got %d", len(result.Images))
	}

	if err := esgallery.ApplyProcessorResult(result, g); err != nil {
		t.Fatalf("apply result: %v", err)
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
