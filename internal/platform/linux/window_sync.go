package linux

// The _NET_WM_SYNC_REQUEST protocol: while a window is resized interactively
// the window manager sends a request carrying a counter value before each
// configure, and holds the next one back until the window has set its counter
// to that value, which it does after drawing the new size. Without it the
// compositor shows frames of the old size stretched or cleared, which is the
// flicker and tearing of a dragged corner.

// syncRequest is the value of the latest request not yet answered.
type syncRequest struct {
	lo      uint32
	hi      int32
	pending bool
}

// syncRequestValue reads the counter value from the data of the client message:
// the protocol atom, a timestamp, then the low and the high half of the value.
func syncRequestValue(data [20]byte) (lo uint32, hi int32) {
	word := func(index int) uint32 {
		return uint32(data[index*4]) | uint32(data[index*4+1])<<8 | uint32(data[index*4+2])<<16 | uint32(data[index*4+3])<<24
	}
	return word(2), int32(word(3))
}

// request records a request. A newer one replaces an older that was never
// answered: the counter only needs to reach the latest value.
func (request *syncRequest) request(lo uint32, hi int32) {
	request.lo, request.hi, request.pending = lo, hi, true
}

// take returns the value to answer with, once.
func (request *syncRequest) take() (lo uint32, hi int32, ok bool) {
	if !request.pending {
		return 0, 0, false
	}
	request.pending = false
	return request.lo, request.hi, true
}

// syncInt64 is xcb_sync_int64_t as one word: the high half first in memory, so
// the 64-bit little-endian value that the C ABI passes in a single register.
func syncInt64(lo uint32, hi int32) uint64 {
	return uint64(uint32(hi)) | uint64(lo)<<32
}
