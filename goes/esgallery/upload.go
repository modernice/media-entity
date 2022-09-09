package esgallery

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/modernice/goes/helper/pick"
	"github.com/modernice/media-entity/gallery"
)

// Storage is storage for gallery images. It is used by [*Gallery.Upload] to
// upload
type Storage interface {
	Put(ctx context.Context, path string, contents io.Reader) error
}

// StorageFunc allows ordinary functions to be used as [Storage].
type StorageFunc func(ctx context.Context, path string, contents io.Reader) error

// Put implements [Storage].
func (put StorageFunc) Put(ctx context.Context, path string, contents io.Reader) error {
	return put(ctx, path, contents)
}

// Upload uploads an image to storage and returns an [Image] that represents
// the uploaded image. The returned [Image] is not automatically added to the
// Gallery. Instead, it is returned so that the caller can decide if and when
// to add it to the Gallery.
//
//	img, err := g.Upload(context.TODO(), ...)
//	stack, err := g.NewVariant(<stack-id>, img)
func (g *Gallery[StackID, ImageID, Target]) Upload(
	ctx context.Context,
	storage Storage,
	img io.Reader,
	stackID StackID,
	variantID ImageID,
) (gallery.Image[ImageID], error) {
	stack, ok := g.Stack(stackID)
	if !ok {
		return gallery.Image[ImageID]{}, gallery.ErrStackNotFound
	}

	variant, ok := stack.Variant(variantID)
	if !ok {
		return gallery.Image[ImageID]{}, gallery.ErrVariantNotFound
	}

	filename := strings.TrimSpace(variant.Filename)
	if filename == "" {
		filename = fmt.Sprintf("%s", variant.ID)
	}
	if filename == "" {
		filename = fmt.Sprintf("%v", variant.ID)
	}

	galleryID := pick.AggregateID(g.target)
	path := fmt.Sprintf("esgallery/%s/%s/%s/%s", galleryID, stackID, variant.ID, filename)

	var size filesize
	img = io.TeeReader(img, &size)

	if err := storage.Put(ctx, path, img); err != nil {
		return variant, fmt.Errorf("storage: %w", err)
	}

	uploaded := variant.Clone()
	uploaded.Storage.Path = path
	uploaded.Filesize = int(size)

	return uploaded, nil
}

type filesize int

func (f *filesize) Write(p []byte) (int, error) {
	l := len(p)
	s := int(*f)
	*f = filesize(s + l)
	return l, nil
}
