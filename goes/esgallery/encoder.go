package esgallery

import (
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"sync"
)

var _ Encoding = (*Encoder)(nil)

// ErrMissingEncoder is returned by [Encoder.Encode] if no encoder is registered
// for the given content-type.
var ErrMissingEncoder = errors.New("missing encoder for this content-type")

// DefaultEncoder is an [*Encoder] with support for encoding "image/png",
// "image/jpeg", and "image/gif" content-types.
//
// # Encoders
//
//   - PNGs are encoded using [png.Encode].
//   - JPEGs are encoded using [jpeg.Encode] with 100% quality.
//   - GIFs are encoded using [gif.Encode] with default options.
var DefaultEncoder *Encoder

func init() {
	DefaultEncoder = &Encoder{}

	DefaultEncoder.Register("image/png", png.Encode)

	DefaultEncoder.Register("image/jpeg", func(w io.Writer, img image.Image) error {
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 100})
	})

	DefaultEncoder.Register("image/gif", func(w io.Writer, img image.Image) error {
		return gif.Encode(w, img, nil)
	})
}

// An Encoding encodes images of different content-types.
type Encoding interface {
	Encode(w io.Writer, contentType string, img image.Image) error
}

// Encoder encodes images of different content-types. It is safe for concurrent
// use. [DefaultEncoder] is an *Encoder with support for encoding "image/png",
// "image/jpeg", and "image/gif" content-types. The zero-value Encoder is
// ready-to-use.
//
//	var enc Encoder
//	enc.Register("image/png", png.Encode)
//
//	var buf bytes.Buffer
//	var img image.Image
//
//	err := enc.Encode(&buf, "image/png", img)
//	encoded := buf.Bytes()
type Encoder struct {
	mux      sync.RWMutex
	once     sync.Once
	encoders map[string]func(io.Writer, image.Image) error
}

// EncoderFunc is a function that can be used as an [Encoding].
type EncoderFunc func(io.Writer, string, image.Image) error

// Encode implements [Encoding].
func (encode EncoderFunc) Encode(w io.Writer, contentType string, img image.Image) error {
	return encode(w, contentType, img)
}

// Register registers an encoder function for the given content-type.
func (enc *Encoder) Register(contentType string, encoder func(io.Writer, image.Image) error) {
	enc.init()
	enc.mux.Lock()
	defer enc.mux.Unlock()
	enc.encoders[contentType] = encoder
}

// Encode encodes the provided [image.Image] and writes the result to `w`, using
// registered encoder for the given content-type. If no encoder was registered
// for this content-type, an error that satisfies errors.Is(err, ErrMissingEncoder)
// is returned.
func (enc *Encoder) Encode(w io.Writer, contentType string, img image.Image) error {
	enc.init()
	enc.mux.RLock()
	defer enc.mux.RUnlock()
	encode, ok := enc.encoders[contentType]
	if !ok {
		return fmt.Errorf("%q content-type: %w", contentType, ErrMissingEncoder)
	}
	return encode(w, img)
}

func (enc *Encoder) init() {
	enc.once.Do(func() { enc.encoders = make(map[string]func(io.Writer, image.Image) error) })
}
