package windows

import "sync/atomic"

// redrawQueue keeps background invalidations from filling the native message
// queue with identical redraws. It has no timer or idle work.
type redrawQueue struct{ pending atomic.Bool }

func (queue *redrawQueue) post(post func() bool) {
	if queue.pending.CompareAndSwap(false, true) && !post() {
		queue.pending.Store(false) // a failed PostMessage must not suppress retries
	}
}

// Rearm before painting so a change that arrives during OnRedraw schedules the
// following frame instead of being lost behind the frame already in progress.
func (queue *redrawQueue) begin() { queue.pending.Store(false) }
