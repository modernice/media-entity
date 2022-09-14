package imagepb

import "github.com/modernice/media-entity/image"

func New(img image.Image) *Image {
	return &Image{
		Storage:      NewStorage(img.Storage),
		Filename:     img.Filename,
		Filesize:     int64(img.Filesize),
		Dimensions:   NewDimensions(img.Dimensions),
		Names:        img.Names,
		Descriptions: img.Descriptions,
	}
}

func (img *Image) AsImage() image.Image {
	return image.Image{
		Storage:      img.GetStorage().AsStorage(),
		Filename:     img.GetFilename(),
		Filesize:     int(img.GetFilesize()),
		Dimensions:   img.GetDimensions().AsDimensions(),
		Names:        img.GetNames(),
		Descriptions: img.GetDescriptions(),
	}
}

func NewStorage(s image.Storage) *Storage {
	return &Storage{
		Provider: s.Provider,
		Path:     s.Path,
	}
}

func (s *Storage) AsStorage() image.Storage {
	return image.Storage{
		Provider: s.GetProvider(),
		Path:     s.GetPath(),
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
