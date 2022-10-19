package file

// Storage provides the storage information of a file.
type Storage struct {
	Provider string `json:"provider"`
	Path     string `json:"path"`
}
