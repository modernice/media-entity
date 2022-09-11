package esgallery_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/modernice/media-entity/goes/esgallery"
	"github.com/modernice/media-entity/internal/galleryx"
)

func TestUploader_Upload(t *testing.T) {
	var storage esgallery.MemoryStorage
	storage.SetRoot("esgallery")

	g := NewTestGallery(uuid.New())

	up := esgallery.NewUploader[uuid.UUID, uuid.UUID](&storage)

	stack, _ := g.NewStack(uuid.New(), galleryx.NewImage(uuid.New()))

	img := newExample()
	gimg := galleryx.NewImage(uuid.New())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	variantID := uuid.New()

	uploaded, err := up.UploadVariant(ctx, g, stack.ID, variantID, img)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	wantPath := fmt.Sprintf("%s/%s/%s/%s", g.ID, stack.ID, variantID, gimg.Filename)

	if uploaded.Storage.Path != wantPath {
		t.Errorf("image should be uploaded to path %q; got %q", wantPath, uploaded.Storage.Path)
	}

	if len(storage.Files()) != 1 {
		t.Fatalf("expected 1 file to be in storage; got %d", len(storage.Files()))
	}

	contentsReader, err := storage.Get(context.TODO(), uploaded.Storage.Path)
	if err != nil {
		t.Fatalf("get storage file: %v", err)
	}
	contents, err := io.ReadAll(contentsReader)
	if err != nil {
		t.Fatalf("read contents: %v", err)
	}
	wantContents := string(example)

	if string(contents) != wantContents {
		t.Fatalf("uploaded file has wrong contents\n%s", cmp.Diff(example, contents))
	}

	wantFilesize := len(example)
	if uploaded.Filesize != wantFilesize {
		t.Fatalf("uploaded file has wrong filesize; got %d; want %d", uploaded.Filesize, wantFilesize)
	}
}
