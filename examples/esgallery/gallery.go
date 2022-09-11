package esgallery

import (
	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/media-entity/goes/esgallery"
)

// Aggregate is the name of the [*Gallery] aggregate.
const Aggregate = "gallery"

// Repository is the [*Gallery] repository.
type Repository = aggregate.TypedRepository[*Gallery]

// Gallery is an image gallery.
type Gallery struct {
	*aggregate.Base
	*esgallery.Gallery[uuid.UUID, uuid.UUID, *Gallery]
}

// NewGallery returns a new image gallery.
func NewGallery(id uuid.UUID) *Gallery {
	g := &Gallery{Base: aggregate.New(Aggregate, id)}
	g.Gallery = esgallery.New[uuid.UUID, uuid.UUID](g)
	return g
}
