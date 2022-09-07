package gallery_test

import (
	"github.com/google/uuid"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/image"
)

func newImage() gallery.Image[uuid.UUID] {
	return gallery.Image[uuid.UUID]{
		ID: uuid.New(),
		Image: image.Image{
			Storage: image.Storage{
				Provider: "fs",
				Path:     "/foo/bar/baz.jpg",
			},
			Filesize:   12345,
			Dimensions: image.Dimensions{1920, 1080},
			Names: map[string]string{
				"en": "Foo image",
				"de": "Foo Bild",
			},
			Descriptions: map[string]string{
				"en": "An image of Foo",
				"de": "Ein Bild von Foo",
			},
		},
	}
}
