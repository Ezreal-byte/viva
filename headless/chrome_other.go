//go:build windows || linux

package headless

import (
	"image"

	"github.com/vivaui/viva/app"
)

// drawWindowChrome adds nothing on Windows and Linux: the buttons of a custom
// titlebar there are Viva's own (app.WindowControls), which the application paints
// into the scene, so a screenshot already carries them.
func drawWindowChrome(*image.RGBA, app.Config, float64) {}
