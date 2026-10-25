package app

import "github.com/vivaui/viva/internal/platform"

// Invalidator lets background goroutines ask the window to repaint. Create one,
// pass it as Config.Invalidator, and call Invalidate from anywhere after the
// goroutine has published its result. It owns no goroutine, timer, or channel:
// an application that never calls Invalidate pays nothing, and the window
// thread is woken exactly once per call batch.
//
// Calls made before RunCanvas starts or after the window closes do nothing.
// The zero value is ready to use.
type Invalidator struct {
	waker platform.Waker
}

// Invalidate requests one repaint on the window thread. It is safe to call
// from any goroutine and never blocks on the application.
func (invalidator *Invalidator) Invalidate() {
	invalidator.waker.Wake()
}

// wakeHandler runs the application's OnWake hook, then repaints, so results
// that background work published are applied before the next paint reads them.
func wakeHandler(onWake func(), repaint platform.RedrawHandler) platform.RedrawHandler {
	if onWake == nil {
		return repaint
	}
	return func() (platform.Frame, error) {
		onWake()
		return repaint()
	}
}
