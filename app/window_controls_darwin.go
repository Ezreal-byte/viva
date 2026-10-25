//go:build darwin

package app

// controlsDrawnByViva is false on macOS: the system draws the traffic lights.
const controlsDrawnByViva = false

// TitlebarInsets is how much of a WindowCustomTitlebar's strip the system or Viva
// covers with window buttons: leading at the start of the strip, trailing at its
// end, in logical pixels. An application keeps its own content clear of both.
// On macOS the traffic lights are the system's, at the leading end.
func TitlebarInsets() (leading, trailing float32) { return 70, 0 }
