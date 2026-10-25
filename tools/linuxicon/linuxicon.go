// Package linuxicon renders an application icon as the PNG sizes a Linux
// desktop looks for in the hicolor icon theme.
package linuxicon

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/vivaui/viva/internal/iconshape"
	"github.com/vivaui/viva/tools/appbundle"
)

// Sizes are the square sizes written, smallest first. A size larger than the
// source is not made: scaling up only blurs.
var Sizes = []int{48, 64, 128, 256}

// MinPixels is the smallest source accepted; Recommended is the size that
// fills every entry of Sizes.
const (
	MinPixels   = 128
	Recommended = 256
)

// Rendition is one icon size.
type Rendition struct {
	Pixels int
	PNG    []byte
}

// Render decodes a square PNG of at least MinPixels and returns it at every
// size of Sizes not larger than it, smallest first. The output is
// deterministic. largest is the source's width.
func Render(source []byte) (renditions []Rendition, largest int, err error) {
	decoded, err := png.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, 0, fmt.Errorf("the icon is not a PNG: %w", err)
	}
	bounds := decoded.Bounds()
	if bounds.Dx() != bounds.Dy() {
		return nil, 0, fmt.Errorf("the icon is %d x %d px; it must be square", bounds.Dx(), bounds.Dy())
	}
	if bounds.Dx() < MinPixels {
		return nil, 0, fmt.Errorf("the icon is %d px; it must be at least %d px", bounds.Dx(), MinPixels)
	}
	tile := iconshape.Tile(decoded, bounds.Dx(), 1)
	for _, size := range Sizes {
		if size > bounds.Dx() {
			continue
		}
		var scaled image.Image = appbundle.Downscale(tile, size)
		var buffer bytes.Buffer
		if err := png.Encode(&buffer, scaled); err != nil {
			return nil, 0, err
		}
		renditions = append(renditions, Rendition{Pixels: size, PNG: buffer.Bytes()})
	}
	return renditions, bounds.Dx(), nil
}

// Largest is the biggest rendition.
func Largest(renditions []Rendition) Rendition { return renditions[len(renditions)-1] }
