//go:build !linux

package app

import (
	"github.com/vivaui/viva/internal/platform"
	"github.com/vivaui/viva/ui"
)

// watchDesktopSettings has nothing to watch here: these targets read the fonts and the
// motion preference when the window opens.
func watchDesktopSettings(*ui.Canvas, *platform.Waker) (stop func()) { return func() {} }
