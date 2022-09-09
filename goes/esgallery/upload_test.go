package esgallery_test

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/modernice/media-entity/goes/esgallery"
	"github.com/modernice/media-entity/internal/galleryx"
)

//go:embed testdata/example.jpg
var example []byte

func newExample() io.Reader {
	return bytes.NewReader(example)
}

func TestGallery_Upload(t *testing.T) {
	storage := make(map[string]string)
	put := esgallery.StorageFunc(func(_ context.Context, path string, contents io.Reader) error {
		b, err := io.ReadAll(contents)
		if err != nil {
			return err
		}
		storage[path] = string(b)
		return nil
	})

	g := NewTestGallery(uuid.New())

	pending := galleryx.NewImage(uuid.New())
	stack, _ := g.NewStack(uuid.New(), pending)

	img := newExample()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	uploaded, err := g.Upload(ctx, put, img, stack.ID, pending.ID)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	wantPath := fmt.Sprintf("esgallery/%s/%s/%s/%s", g.ID, stack.ID, pending.ID, pending.Filename)

	if uploaded.Storage.Path != wantPath {
		t.Errorf("image should be uploaded to path %q; got %q", wantPath, uploaded.Storage.Path)
	}

	if len(storage) != 1 {
		t.Fatalf("expected 1 file to be in storage; got %d", len(storage))
	}

	contents := storage[wantPath]
	wantContents := string(example)

	if contents != wantContents {
		t.Fatalf("uploaded file has wrong contents\n%s", cmp.Diff([]byte(wantContents), []byte(contents)))
	}

	wantFilesize := len(example)
	if uploaded.Filesize != wantFilesize {
		t.Fatalf("uploaded file has wrong filesize; got %d; want %d", uploaded.Filesize, wantFilesize)
	}
}
