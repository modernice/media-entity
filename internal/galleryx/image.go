package galleryx

import (
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/image"
)

// NewImage returns a new stub image with the given id.
func NewImage[ID comparable](id ID) gallery.Image[ID] {
	return gallery.Image[ID]{
		ID: id,
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
