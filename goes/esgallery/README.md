# esgallery – Event-sourced Image Galleries

Package `esgallery` implements an event-sourced image gallery aggregate on top
of [goes](https://github.com/modernice/goes) and the [non event-sourced
implementation](../../gallery) The `Gallery` provided by this package can be
embedded into your own aggregates that need to implement an image gallery.

## Example

In this example, we use UUIDs for the ids of stacks and variants. Read the
documentation of the [non event-sourced implementation](../../gallery) for
more details.

### 1. Setup gallery aggregate

```go
package myapp

import (
  "github.com/google/uuid"
  "github.com/modernice/goes/aggregate"
  "github.com/modernice/media-entity/goes/esgallery"
)

const GalleryAggregate = "myapp.gallery"

type Gallery struct {
  *aggregate.Base
  *esgallery.Gallery[uuid.UUID, uuid.UUID, *Gallery]
}

type GalleryRepository = aggregate.TypedRepository[*Gallery]

func NewGallery(id uuid.UUID) *Gallery {
  g:= &Gallery{Base: aggregate.New(GalleryAggregate, id)}
  g.Gallery = esgallery.New[uuid.UUID, uuid.UUID](g)
  return g
}
```

### 2. Setup uploader

```go
package myapp

import (
  "github.com/google/uuid"
  "github.com/modernice/goes/aggregate"
  "github.com/modernice/goes/aggregate/repository"
  "github.com/modernice/goes/event"
  "github.com/modernice/media-entity/goes/esgallery"
)

// Create an alias to avoid having to type *esgallery.Uploader[uuid.UUID, uuid.UUID] everywhere.
type Uploader = esgallery.Uploader[uuid.UUID, uuid.UUID]

// NewUploader returns an uploader for gallery images. The two type parameters
// specify the ID types for the stacks and variants within the gallery aggregate.
func NewUploader() *Uploader {
  // Create storage for gallery images.
  var storage esgallery.MemoryStorage

  // Create a new Uploader using the storage.
  return esgallery.NewUploader[uuid.UUID, uuid.UUID](&storage)
}
```

### 2. Setup post-processing

```go
package myapp

import (
  "github.com/google/uuid"
  "github.com/modernice/goes/aggregate"
  "github.com/modernice/goes/aggregate/repository"
  "github.com/modernice/goes/event"
  "github.com/modernice/media-entity/goes/esgallery"
)

// Create an alias to avoid having to type *esgallery.PostProcessor[*Gallery, uuid.UUID, uuid.UUID] everywhere.
type PostProcessor = esgallery.PostProcessor[*Gallery, uuid.UUID, uuid.UUID]

func Setup(uploader *Uploader, bus event.Bus, repo aggregate.Repository) *PostProcessor {
  // Create storage for gallery images.
  var storage esgallery.MemoryStorage

  // Create an image processor for galleries.
  p := esgallery.NewProcessor(esgallery.DefaultEncoder, &storage, uploader, uuid.New)

  // Create the PostProcessor from the Processor.
  galleries := repository.Typed(repo, NewGallery)
  pp := esgallery.NewPostProcessor(p, bus, galleries.Fetch)

  return pp
}
```

### 3. Run post-processor

The `PostProcessor` runs in the background and processes images whenever a new
stack is added to a galllery, or when the original variant of a stack is replaced.

```go
package myapp

import (
  "context"
  "github.com/modernice/media-entity/goes/esgallery"
  "github.com/modernice/media-tools/image"
  "github.com/modernice/media-tools/image/compression"
)

func run(pp *PostProcessor, galleries GalleryRepository) {
  // Setup a processing pipeline
  pipeline := image.Pipeline{
    image.Resize(image.DimensionMap{
      "sm": {640},
      "md": {960},
      "lg": {1280},
      "xl": {1920},
    }),
    image.Compress(compression.JPEG(80)),
  }

  // Start the post-processor as a background task
  results, errs, err := pp.Run(context.TODO(), pipeline)
  if err != nil {
    panic(err)
  }

  // Log processing errors
	go func(){
    for err := range errs {
      log.Printf("post-processor: %v", err)
    }
  }()

  for result := range results {
    g, err := galleries.Fetch(context.TODO(), result.Gallery.ID)
    if err != nil {
      panic(fmt.Errorf("fetch gallery: %w", err))
    }

    if err := result.Apply(g); err != nil {
      panic(fmt.Errorf("apply processor result: %w", err))
    }

    if err := galleries.Save(context.TODO(), result); err != nil {
      panic(fmt.Errorf("save gallery: %w", err))
    }
  }
}
```
