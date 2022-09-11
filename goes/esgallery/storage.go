package esgallery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"sync"

	"github.com/modernice/media-entity/image"
)

var _ Storage = (*MemoryStorage)(nil)

// Storage is the storage for gallery images.
type Storage interface {
	// Put writes the contents of the image in r to the storage at the given
	// path and returns the storage location of the uploaded image.
	Put(ctx context.Context, path string, contents io.Reader) (image.Storage, error)

	// Get returns the contents of the image at the given storage path.
	Get(ctx context.Context, path string) (io.Reader, error)
}

// MemoryStorage is a thread-safe [Storage] that stores images in memory.
type MemoryStorage struct {
	mux   sync.RWMutex
	once  sync.Once
	files map[string][]byte
	root  string
}

// SetRoot sets the root path of the MemoryStorage. The root path is prepended
// to all paths passed to Put and Get.
func (s *MemoryStorage) SetRoot(root string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.root = root
}

// Files returns all stored files as a mapping from paths to contents.
func (s *MemoryStorage) Files() map[string][]byte {
	s.mux.RLock()
	defer s.mux.RUnlock()
	out := make(map[string][]byte, len(s.files))
	for k, v := range s.files {
		out[k] = v
	}
	return out
}

// Put implements [Storage].
func (s *MemoryStorage) Put(_ context.Context, p string, contents io.Reader) (image.Storage, error) {
	joined := path.Join(s.root, p)

	b, err := io.ReadAll(contents)
	if err != nil {
		return image.Storage{}, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	s.once.Do(func() { s.files = make(map[string][]byte) })
	s.files[joined] = b

	return image.Storage{
		Provider: "memory",
		Path:     p,
	}, nil
}

// Put implements [Storage].
func (s *MemoryStorage) Get(_ context.Context, p string) (io.Reader, error) {
	p = path.Join(s.root, p)

	s.mux.RLock()
	defer s.mux.RUnlock()

	contents, ok := s.files[p]
	if !ok {
		return nil, fmt.Errorf("image %q not found in memory storage", p)
	}

	return bytes.NewReader(contents), nil
}
