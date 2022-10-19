package esgallery

import (
	"bytes"
	"context"
	"fmt"
	stdimage "image"
	"io"

	"github.com/modernice/goes/helper/pick"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/image"
)

// Uploader uploads gallery images to (cloud) storage. An Uploader can be passed
// to a [*Processor] to automatically upload processed images to (cloud) storage.
type Uploader[StackID, ImageID ID] struct {
	storage Storage
}

// NewUploader returns an [*Uploader] that uploads images to the provided [Storage].
func NewUploader[StackID, ImageID ID](storage Storage) *Uploader[StackID, ImageID] {
	return &Uploader[StackID, ImageID]{
		storage: storage,
	}
}

// UploadNew uploads a new image to the provided gallery and returns the newly
// created [gallery.Stack].
//
// The filesize and dimensions of the uploaded image are determined while
// uploading to storage, and set on the [gallery.Image] in the returned
// [gallery.Stack]. The Filename of the returned [gallery.Image] is set to
// the provided filename.
//
// To upload a new variant of an existing [gallery.Stack], call u.UploadVariant()
// instead.
func (u *Uploader[StackID, ImageID]) UploadNew(
	ctx context.Context,
	g ProcessableGallery[StackID, ImageID],
	stackID StackID,
	imageID ImageID,
	r io.Reader,
	filename string,
) (gallery.Stack[StackID, ImageID], error) {
	if filename == "" {
		return gallery.Stack[StackID, ImageID]{}, fmt.Errorf("empty filename")
	}

	if _, ok := g.Stack(stackID); ok {
		return gallery.Stack[StackID, ImageID]{}, fmt.Errorf("stack id: %w", gallery.ErrDuplicateID)
	}

	var info detectFileInfo
	r = io.TeeReader(r, &info)

	galleryID := pick.AggregateID(g)
	path := variantPath(galleryID, stackID, imageID, filename)

	storage, err := u.storage.Put(ctx, path, r)
	if err != nil {
		return gallery.Stack[StackID, ImageID]{}, fmt.Errorf("storage: %w", err)
	}

	dims, err := info.Dimensions()
	if err != nil {
		return gallery.Stack[StackID, ImageID]{}, fmt.Errorf("detect image dimensions: %w", err)
	}

	img := image.Image{
		Storage:    storage,
		Filename:   filename,
		Filesize:   info.size,
		Dimensions: dims,
	}.Normalize()

	gimg := gallery.Image[ImageID]{
		ID:       imageID,
		Image:    img,
		Original: true,
	}

	stack, err := g.NewStack(stackID, gimg)
	if err != nil {
		return gallery.Stack[StackID, ImageID]{}, fmt.Errorf("create stack: %w", err)
	}

	return stack, nil
}

// UploadVariant writes the image in `r` to the underlying [Storage] and returns
// a [gallery.Image] that represents the uploaded image. The returned
// [gallery.Image] can be added to a [*Gallery], either by calling
// [*Gallery.AddVariant], or by applying a [ProcessorResult] to the gallery
// with [ApplyProcessorResult].
//
// The provided StackID specifies the [gallery.Stack] the image should be added
// to. The provided ImageID is used as the ID of the returned [gallery.Image].
// The storage path of the uploaded image is determined by the StackID, ImageID,
// and the ID of the provided gallery.
//
// The filesize and dimensions of the uploaded image are determined while
// uploading to storage, and set on the returned [gallery.Image]. The Filename
// of the returned [gallery.Image] is set to the Filename of the original image
// of the [gallery.Stack].
func (u *Uploader[StackID, ImageID]) UploadVariant(
	ctx context.Context,
	g ProcessableGallery[StackID, ImageID],
	stackID StackID,
	variantID ImageID,
	r io.Reader,
) (gallery.Image[ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return gallery.Image[ImageID]{}, gallery.ErrStackNotFound
	}
	original := stack.Original()

	var info detectFileInfo
	r = io.TeeReader(r, &info)

	galleryID := pick.AggregateID(g)
	path := variantPath(galleryID, stackID, variantID, original.Filename)

	storage, err := u.storage.Put(ctx, path, r)
	if err != nil {
		return gallery.Image[ImageID]{}, fmt.Errorf("storage: %w", err)
	}

	dims, err := info.Dimensions()
	if err != nil {
		return gallery.Image[ImageID]{}, fmt.Errorf("detect image dimensions: %w", err)
	}

	variantImg := original.Image.Clone()
	variantImg.Storage = storage
	variantImg.Filename = original.Filename
	variantImg.Filesize = info.size
	variantImg.Dimensions = dims

	variant, err := stack.NewVariant(variantID, variantImg)
	if err != nil {
		return gallery.Image[ImageID]{}, fmt.Errorf("create variant: %w", err)
	}

	return variant, nil
}

type detectFileInfo struct {
	size int
	data []byte
}

func (f *detectFileInfo) Write(p []byte) (int, error) {
	f.data = append(f.data, p...)
	l := len(p)
	f.size += l
	return l, nil
}

func (f *detectFileInfo) Dimensions() (image.Dimensions, error) {
	img, _, err := stdimage.Decode(bytes.NewReader(f.data))
	if err != nil {
		return image.Dimensions{}, fmt.Errorf("decode image: %w", err)
	}
	return image.Dimensions{img.Bounds().Dx(), img.Bounds().Dy()}, nil
}
