package componentdemo

import (
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/ui"
)

var imageViewComponent = Component{
	ID: "image-view", Title: "Image View", Category: "Media", Summary: "Fit and actual-size bitmap preview independent of display scale.",
	initialize: func(state *State) {
		const w, h = 160, 100
		pix := make([]byte, w*h*4)
		for y := range h {
			for x := range w {
				i := (y*w + x) * 4
				pix[i] = byte(x * 255 / w)
				pix[i+1] = byte(y * 255 / h)
				pix[i+2] = 100
				pix[i+3] = 255
			}
		}
		state.imageBitmap = &ui.Bitmap{Pix: pix, Width: w, Height: h, Stride: w * 4, ID: 1, Version: 1}
	},
	paint: func(state *State, canvas *ui.Canvas, theme ui.Theme, size ui.Size) {
		var zoom float32
		if state.exampleID == "actual" {
			zoom = 1
		}
		components.PaintImageView(canvas, theme, ui.Rect{X: 24, Y: 24, Width: max(0, size.Width-48), Height: max(0, size.Height-48)}, components.ImageViewOptions{Bitmap: *state.imageBitmap, Zoom: zoom})
	},
	Examples: []Example{
		{ID: "fit", Title: "Fit to viewport", Description: "The application supplies bounded decoded pixels; the component only paints them.", Height: 260, Snippet: `components.PaintImageView(canvas, theme, bounds, components.ImageViewOptions{Bitmap: bitmap})`},
		{ID: "actual", Title: "One image pixel per device pixel", Description: "Zoom 1 retains actual pixel size across fractional DPI settings.", Height: 260, Snippet: `components.PaintImageView(canvas, theme, bounds, components.ImageViewOptions{Bitmap: bitmap, Zoom: 1})`},
	},
}
