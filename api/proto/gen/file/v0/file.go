package filepb

import "github.com/modernice/media-entity/file"

func NewStorage(s file.Storage) *Storage {
	return &Storage{
		Provider: s.Provider,
		Path:     s.Path,
	}
}

func (s *Storage) AsStorage() file.Storage {
	return file.Storage{
		Provider: s.GetProvider(),
		Path:     s.GetPath(),
	}
}
