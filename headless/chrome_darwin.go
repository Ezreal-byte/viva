//go:build darwin

package headless

import (
	"image"
	"math"

	"github.com/vivaui/viva/app"
)

// The traffic lights macOS draws into a custom titlebar, in logical pixels: three
// 12-point circles 20 points apart, the first 8 points from the leading edge,
// centred on the middle of the titlebar strip — the system's 28-point one, or the
// taller custom strip the platform moves them into.
//
// AppKit draws them over the content, so they are not in the scene the offscreen
// renderer paints; without drawing them here a screenshot of such a window would
// be missing the three buttons the window shows.
const (
	windowLightDiameter = 12.0
	windowLightStride   = 20.0
	windowLightLeading  = 8.0
	// windowLightSystemStrip is the height of the system titlebar. A custom strip
	// no taller than it leaves the lights where AppKit puts them, in that strip's
	// middle; a taller one is what the platform centres them in instead.
	windowLightSystemStrip = 28.0
)

// windowLightColors are the system's colours for an active window, leading to
// trailing: close, minimise, zoom.
var windowLightColors = [3]app.Color{
	{Red: 1, Green: 0.373, Blue: 0.341, Alpha: 1},
	{Red: 0.996, Green: 0.737, Blue: 0.180, Alpha: 1},
	{Red: 0.157, Green: 0.784, Blue: 0.251, Alpha: 1},
}

func drawWindowChrome(picture *image.RGBA, config app.Config, scale float64) {
	if config.WindowMode != app.WindowCustomTitlebar {
		return
	}
	strip := float64(config.TitlebarHeight)
	if strip < windowLightSystemStrip {
		strip = windowLightSystemStrip
	}
	centerY := strip / 2 * scale
	for index, color := range windowLightColors {
		centerX := (windowLightLeading + windowLightDiameter/2 + float64(index)*windowLightStride) * scale
		paintDisc(picture, centerX, centerY, windowLightDiameter/2*scale, color)
	}
}

// paintDisc paints one antialiased disc, sampling each pixel four times on a side
// so an edge is smooth at any scale. The frame is opaque, so a sample mixes the
// disc's color into the pixel's own.
func paintDisc(picture *image.RGBA, centerX, centerY, radius float64, color app.Color) {
	const samples = 4
	const inside = samples * samples
	left := int(math.Floor(centerX - radius))
	right := int(math.Ceil(centerX + radius))
	top := int(math.Floor(centerY - radius))
	bottom := int(math.Ceil(centerY + radius))
	left = max(left, picture.Rect.Min.X)
	right = min(right, picture.Rect.Max.X-1)
	top = max(top, picture.Rect.Min.Y)
	bottom = min(bottom, picture.Rect.Max.Y-1)
	for y := top; y <= bottom; y++ {
		for x := left; x <= right; x++ {
			covered := 0
			for sampleY := range samples {
				for sampleX := range samples {
					dx := float64(x) + (float64(sampleX)+0.5)/samples - centerX
					dy := float64(y) + (float64(sampleY)+0.5)/samples - centerY
					if dx*dx+dy*dy <= radius*radius {
						covered++
					}
				}
			}
			if covered == 0 {
				continue
			}
			mix := float64(covered) / inside
			at := picture.PixOffset(x, y)
			picture.Pix[at] = blend(picture.Pix[at], color.Red, mix)
			picture.Pix[at+1] = blend(picture.Pix[at+1], color.Green, mix)
			picture.Pix[at+2] = blend(picture.Pix[at+2], color.Blue, mix)
		}
	}
}

// blend mixes color into the channel value already in the pixel, rounding to the
// nearest of 255 levels.
func blend(existing uint8, color float64, mix float64) uint8 {
	value := float64(existing)*(1-mix) + color*255*mix
	return uint8(math.Round(min(max(value, 0), 255)))
}
