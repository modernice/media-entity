package image

import (
	"github.com/modernice/media-entity/internal/maps"
	"github.com/modernice/media-tools/image"
)

// Image represents a single image.
type Image struct {
	Storage      Storage           `json:"storage"`
	Filesize     int               `json:"filesize"`
	Dimensions   Dimensions        `json:"dimensions"`
	Names        map[string]string `json:"names"`
	Descriptions map[string]string `json:"descriptions"`
}

// Storage provides the storage information for an [Image].
type Storage struct {
	Provider string `json:"provider"`
	Path     string `json:"path"`
}

// Dimensions are the width and height of an image, in pixels.
type Dimensions = image.Dimensions

// Normalize checks if the "Names" and/or "Descriptions" fields of the [Image]
// are nil. If so, they are initialized with an empty map.
func (img Image) Normalize() Image {
	if img.Names == nil {
		img.Names = make(map[string]string)
	}
	if img.Descriptions == nil {
		img.Descriptions = make(map[string]string)
	}
	return img
}

// Clone returns a deep copy of the image.
func (img Image) Clone() Image {
	img.Names = maps.Clone(img.Names)
	img.Descriptions = maps.Clone(img.Descriptions)
	return img
}
