package esgallery_test

import (
	"bytes"
	_ "embed"
	stdimage "image"
	"image/jpeg"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/media-entity/gallery"
	"github.com/modernice/media-entity/goes/esgallery"
)

//go:embed testdata/example.jpg
var example []byte
var exampleImg stdimage.Image

func init() {
	exampleImg, _ = jpeg.Decode(bytes.NewReader(example))
}

func newExample() io.Reader {
	return bytes.NewReader(example)
}

type TestGallery struct {
	*aggregate.Base
	*esgallery.Gallery[uuid.UUID, uuid.UUID, *TestGallery]
}

func NewTestGallery(id uuid.UUID) *TestGallery {
	g := &TestGallery{Base: aggregate.New("test.esgallery", id)}
	g.Gallery = esgallery.New[uuid.UUID, uuid.UUID](g)
	return g
}

func expectStackSorting[StackID, ImageID comparable](t *testing.T, sorting []StackID, stacks []gallery.Stack[StackID, ImageID]) {
	if len(sorting) != len(stacks) {
		t.Fatalf("sorting and stacks should have the same length; sorting has %d, stacks has %d", len(sorting), len(stacks))
	}

	for i, id := range sorting {
		sid := stacks[i].ID
		if sid != id {
			t.Fatalf("stack #%d should have id %v; got %v", i+1, id, sid)
		}
	}
}

func UUIDConfig() esgallery.Config[uuid.UUID, uuid.UUID] {
	return esgallery.Configure(esgallery.DefaultEncoder, uuid.New, uuid.New)
}
