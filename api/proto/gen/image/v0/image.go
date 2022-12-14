package imagepb

import (
	filepb "github.com/modernice/media-entity/api/proto/gen/file/v0"
	"github.com/modernice/media-entity/image"
	"github.com/modernice/media-entity/internal/mapx"
	"github.com/modernice/media-entity/internal/slicex"
)

func New(img image.Image) *Image {
	return &Image{
		Storage:      filepb.NewStorage(img.Storage),
		Filename:     img.Filename,
		Filesize:     int64(img.Filesize),
		Dimensions:   NewDimensions(img.Dimensions),
		Names:        mapx.Ensure(img.Names),
		Descriptions: mapx.Ensure(img.Descriptions),
		Tags:         slicex.Ensure(img.Tags),
	}
}

func (img *Image) AsImage() image.Image {
	return image.Image{
		Storage:      img.GetStorage().AsStorage(),
		Filename:     img.GetFilename(),
		Filesize:     int(img.GetFilesize()),
		Dimensions:   img.GetDimensions().AsDimensions(),
		Names:        mapx.Ensure(img.GetNames()),
		Descriptions: mapx.Ensure(img.GetDescriptions()),
		Tags:         slicex.Ensure(img.GetTags()),
	}
}

func NewDimensions(d image.Dimensions) *Dimensions {
	return &Dimensions{
		Width:  int64(d.Width()),
		Height: int64(d.Height()),
	}
}

func (d *Dimensions) AsDimensions() image.Dimensions {
	return image.Dimensions{int(d.GetWidth()), int(d.GetHeight())}
}
