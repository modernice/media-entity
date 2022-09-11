package esgallery

import (
	"path"

	"github.com/google/uuid"
)

func variantPath[StackID, ImageID ID](galleryID uuid.UUID, stackID StackID, variantID ImageID, filename string) string {
	return path.Join(galleryID.String(), stackID.String(), variantID.String(), filename)
}
