//go:build windows || linux

package app

// controlsDrawnByViva is true where the system draws no minimize, maximize or
// close buttons for a window without a titlebar, so Viva draws them.
const controlsDrawnByViva = true

// TitlebarInsets is how much of a WindowCustomTitlebar's strip the system or Viva
// covers with window buttons: leading at the start of the strip, trailing at its
// end, in logical pixels. An application keeps its own content clear of both.
// Here the buttons are Viva's, at the trailing end.
func TitlebarInsets() (leading, trailing float32) { return 0, 3 * WindowControlWidth }
