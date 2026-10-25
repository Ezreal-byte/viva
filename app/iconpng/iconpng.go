// Package iconpng decodes a PNG into an app.WindowIcon. It is separate from
// package app so an application that supplies raw RGBA pixels, or no icon at
// all, does not link image/png and its compression dependencies.
package iconpng

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"github.com/vivaui/viva/app"
)

// Decode decodes a square PNG as an app.WindowIcon. It is what an application
// embeds with go:embed, usually the very file viva.toml names as [app] icon, so
// the packaged icon and the running window's come from one image.
func Decode(data []byte) (app.WindowIcon, error) {
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return app.WindowIcon{}, fmt.Errorf("iconpng: %w", err)
	}
	bounds := decoded.Bounds()
	if bounds.Dx() != bounds.Dy() {
		return app.WindowIcon{}, fmt.Errorf("iconpng: icon must be square, got %dx%d", bounds.Dx(), bounds.Dy())
	}
	picture, ok := decoded.(*image.NRGBA)
	if !ok || picture.Stride != bounds.Dx()*4 || bounds.Min != (image.Point{}) {
		picture = image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(picture, picture.Bounds(), decoded, bounds.Min, draw.Src)
	}
	icon := app.WindowIcon{Size: bounds.Dx(), RGBA: picture.Pix}
	if icon.Size < 1 || icon.Size > 1024 {
		return app.WindowIcon{}, fmt.Errorf("iconpng: icon must be between 1 and 1024 pixels, got %d", icon.Size)
	}
	return icon, nil
}
