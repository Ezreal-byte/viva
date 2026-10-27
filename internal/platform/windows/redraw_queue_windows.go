package windows

import "sync/atomic"

const (
	redrawPending uint32 = 1 << iota
	redrawRetry
)

// redrawQueue keeps background invalidations from filling the native message
// queue with identical redraws. It has no timer or idle work.
type redrawQueue struct{ state atomic.Uint32 }

func (queue *redrawQueue) post(post func() bool) {
	for {
		state := queue.state.Load()
		if state&redrawPending != 0 {
			if state&redrawRetry != 0 || queue.state.CompareAndSwap(state, state|redrawRetry) {
				return
			}
			continue
		}
		if !queue.state.CompareAndSwap(0, redrawPending) {
			continue
		}
		for {
			if post() {
				return
			}
			state = queue.state.Load()
			if state&redrawRetry != 0 {
				if queue.state.CompareAndSwap(state, redrawPending) {
					continue
				}
				continue
			}
			if queue.state.CompareAndSwap(state, 0) {
				return
			}
		}
	}
}

// Rearm before painting so a change that arrives during OnRedraw schedules the
// following frame instead of being lost behind the frame already in progress.
func (queue *redrawQueue) begin() { queue.state.Store(0) }
