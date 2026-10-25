package app

import (
	"sync/atomic"
	"time"
)

// frameInterval spaces animated frames for a 120 Hz display. Swap-chain back
// pressure holds slower displays to their own refresh rate, so the shorter
// interval never presents more frames than the display can show; the
// transition itself is time-based and finishes on schedule either way.
const frameInterval = time.Second / 120

// frameTimer coalesces next-frame requests into at most one pending timer.
// The painter calls request from the window thread after a paint that left
// transitions in flight; a settled canvas never calls it, so an idle
// application owns no timer, goroutine, or wake-up. When the timer fires it
// posts one redraw event through wake and the cycle repeats from the repaint.
type frameTimer struct {
	wake    func()
	timer   *time.Timer
	due     time.Time // when the armed timer fires; written by the window thread only
	armed   atomic.Bool
	stopped atomic.Bool
}

func newFrameTimer(wake func()) *frameTimer {
	return &frameTimer{wake: wake}
}

// request arms the timer for a wake-up after delay, never sooner than one
// display frame, unless a wake-up is already pending. A pending wake-up that
// is later than the one asked for is brought forward, so an animation that
// starts while a slow tick is waiting is not held back by it. It must be
// called from the window thread, which is the timer's only writer.
func (frames *frameTimer) request(delay time.Duration) {
	if frames.stopped.Load() {
		return
	}
	delay = max(delay, frameInterval)
	due := time.Now().Add(delay)
	if !frames.armed.CompareAndSwap(false, true) {
		// Stop reports whether the timer was still waiting; if it already
		// fired the wake-up is on its way and the repaint it causes asks again.
		if due.Before(frames.due) && frames.timer.Stop() {
			frames.due = due
			frames.timer.Reset(delay)
		}
		return
	}
	frames.due = due
	if frames.timer == nil {
		frames.timer = time.AfterFunc(delay, frames.fire)
		return
	}
	frames.timer.Reset(delay)
}

func (frames *frameTimer) fire() {
	frames.armed.Store(false)
	if !frames.stopped.Load() {
		frames.wake()
	}
}

// stop cancels any pending wake-up. Later requests are ignored.
func (frames *frameTimer) stop() {
	frames.stopped.Store(true)
	if frames.timer != nil {
		frames.timer.Stop()
	}
}

// monotonicClock returns frame times measured from first use. time.Since
// reads the monotonic clock, so wall-clock adjustments never bend a
// transition.
func monotonicClock() func() time.Duration {
	epoch := time.Now()
	return func() time.Duration { return time.Since(epoch) }
}
