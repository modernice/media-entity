package imagex

import (
	"image"
	"image/color"
	"image/draw"
)

// Rect returns an image with the given width and height, filled with the given color.
func Rect(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return img
}
