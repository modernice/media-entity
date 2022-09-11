package esgallery

import (
	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/event"
	"github.com/modernice/media-entity/goes/esgallery"
)

// Setup sets up and returns [*esgallery.PostProcessor] and [*esgallery.Uploader].
func Setup(bus event.Bus, repo aggregate.Repository) (
	*esgallery.PostProcessor[*Gallery, uuid.UUID, uuid.UUID],
	*esgallery.Uploader[uuid.UUID, uuid.UUID],
) {
	// Create storage for gallery images.
	var storage esgallery.MemoryStorage

	// Create a new Uploader using the storage.
	uploader := esgallery.NewUploader[uuid.UUID, uuid.UUID](&storage)

	// Create an image processor for galleries.
	p := esgallery.NewProcessor(esgallery.DefaultEncoder, &storage, uploader, uuid.New)

	// Create a the PostProcessor from the Processor.
	galleries := repository.Typed(repo, NewGallery)
	pp := esgallery.NewPostProcessor(p, bus, galleries.Fetch)

	return pp, uploader
}
