//go:build !windows && !linux && !darwin

package app

// controlsDrawnByViva is false where there is no native window to give controls.
const controlsDrawnByViva = false

// TitlebarInsets is how much of a WindowCustomTitlebar's strip the system or Viva
// covers with window buttons: leading at the start of the strip, trailing at its
// end, in logical pixels. An application keeps its own content clear of both.
// There is none here.
func TitlebarInsets() (leading, trailing float32) { return 0, 0 }
