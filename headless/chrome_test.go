//go:build darwin

package headless

import (
	"image"
	"testing"

	"github.com/vivaui/viva/app"
)

// TestWindowChromeTrafficLights checks that a screenshot of a custom-titlebar
// window carries the close, minimise and zoom buttons macOS draws in its titlebar
// strip: AppKit paints them over the content, so the offscreen renderer has to
// draw them itself or the image is missing what the window shows. They sit in the
// middle of the strip, which a custom strip taller than the system's moves.
func TestWindowChromeTrafficLights(t *testing.T) {
	if !rendererAvailable {
		t.Skip("screenshots need the offscreen renderer (-tags vivahead)")
	}
	for _, test := range []struct {
		name    string
		strip   float32
		centerY int
	}{
		{"system titlebar", 0, 14},
		{"taller custom strip", 36, 18},
	} {
		t.Run(test.name, func(t *testing.T) {
			picture := chromeShot(t, app.WindowCustomTitlebar, test.strip)
			for _, light := range windowLightChecks() {
				red, green, blue := pixelAt(picture, light.centerX, test.centerY)
				if !light.isColour(red, green, blue) {
					t.Errorf("%s light at (%d, %d) is %d %d %d, not that button's colour",
						light.name, light.centerX, test.centerY, red, green, blue)
				}
			}
			// The clearance between two lights is not painted.
			gapX := int(windowLightLeading + windowLightDiameter + (windowLightStride-windowLightDiameter)/2)
			gapRed, gapGreen, gapBlue := pixelAt(picture, gapX, test.centerY)
			if gapRed != 255 || gapGreen != 255 || gapBlue != 255 {
				t.Errorf("between the lights is %d %d %d, want the unpainted background", gapRed, gapGreen, gapBlue)
			}
		})
	}
}

// TestWindowChromeOnlyInCustomTitlebar checks that a titled window's screenshot
// keeps its top-left corner: the system titlebar holds the lights there, outside
// the content a screenshot covers.
func TestWindowChromeOnlyInCustomTitlebar(t *testing.T) {
	if !rendererAvailable {
		t.Skip("screenshots need the offscreen renderer (-tags vivahead)")
	}
	picture := chromeShot(t, app.WindowTitled, 0)
	red, green, blue := pixelAt(picture, int(windowLightLeading+windowLightDiameter/2), 14)
	if red != 255 || green != 255 || blue != 255 {
		t.Errorf("a titled window's corner is %d %d %d, want the unpainted background", red, green, blue)
	}
}

// windowLightChecks names the three lights and how each one's colour is told
// apart from the others.
func windowLightChecks() []struct {
	name     string
	centerX  int
	isColour func(red, green, blue uint8) bool
} {
	return []struct {
		name     string
		centerX  int
		isColour func(red, green, blue uint8) bool
	}{
		{"close", int(windowLightLeading + windowLightDiameter/2), func(red, green, blue uint8) bool {
			return red > 200 && green < 150 && blue < 150
		}},
		{"minimise", int(windowLightLeading + windowLightDiameter/2 + windowLightStride), func(red, green, blue uint8) bool {
			return red > 200 && green > 150 && blue < 100
		}},
		{"zoom", int(windowLightLeading + windowLightDiameter/2 + 2*windowLightStride), func(red, green, blue uint8) bool {
			return green > 150 && red < 100
		}},
	}
}

// chromeShot renders a 200x120 window with a white background, so a painted light
// and an untouched pixel are told apart without reading the theme.
func chromeShot(t *testing.T, mode app.WindowMode, strip float32) *image.RGBA {
	t.Helper()
	config := app.Config{
		Title:          "chrome",
		Width:          200,
		Height:         120,
		WindowMode:     mode,
		TitlebarHeight: strip,
		ClearColor:     app.Color{Red: 1, Green: 1, Blue: 1, Alpha: 1},
	}
	session, err := Open(config, nil, Options{Scale: 1})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(session.Close)
	picture, err := session.Image()
	if err != nil {
		t.Fatalf("image: %v", err)
	}
	return picture
}

func pixelAt(picture *image.RGBA, x, y int) (red, green, blue uint8) {
	colour := picture.RGBAAt(x, y)
	return colour.R, colour.G, colour.B
}
