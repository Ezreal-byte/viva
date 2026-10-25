package linux

import "time"

// A selection owner hands its data to a requestor in a window property. A property holds
// as much as one X request carries; more goes in pieces, which the requestor asks for one
// after another by deleting the property it read (ICCCM 2.7.2, INCR). This file is that
// stream's bookkeeping; selection_linux.go speaks to the X server.

const (
	// incrChunk is the most written to a property in one request, and so the size at which
	// data goes by INCR. Every X server accepts a request of 256 KB.
	incrChunk = 128 << 10
	// incrTimeout is how long a transfer waits for the requestor's next deletion before it
	// is forgotten: a requestor that died leaves no other trace.
	incrTimeout = 10 * time.Second
	// maxIncrTransfers bounds the transfers in flight, oldest dropped first.
	maxIncrTransfers = 8
)

// incrTransfer is one stream: the data, and how far the requestor has read.
type incrTransfer struct {
	requestor, property, kind uint32
	data                      []byte
	offset                    int
	touched                   time.Time
}

// next is the next piece to write. The last piece is an empty one, which tells the requestor
// the stream is over.
func (transfer *incrTransfer) next(now time.Time) (piece []byte, last bool) {
	transfer.touched = now
	if transfer.offset >= len(transfer.data) {
		return nil, true
	}
	end := min(transfer.offset+incrChunk, len(transfer.data))
	piece = transfer.data[transfer.offset:end]
	transfer.offset = end
	return piece, false
}

// incrTransfers are the streams a window has under way.
type incrTransfers struct {
	list []*incrTransfer
}

// begin adds a stream. One to the same property of the same window replaces the older one,
// as does the lack of room; the dropped ones are returned.
func (set *incrTransfers) begin(transfer *incrTransfer, now time.Time) (dropped []*incrTransfer) {
	transfer.touched = now
	kept := set.list[:0]
	for _, other := range set.list {
		if other.requestor == transfer.requestor && other.property == transfer.property {
			dropped = append(dropped, other)
			continue
		}
		kept = append(kept, other)
	}
	set.list = kept
	for len(set.list) >= maxIncrTransfers {
		dropped = append(dropped, set.list[0])
		set.list = append(set.list[:0], set.list[1:]...)
	}
	set.list = append(set.list, transfer)
	return dropped
}

// find is the stream to property of the requestor window, nil if there is none.
func (set *incrTransfers) find(requestor, property uint32) *incrTransfer {
	for _, transfer := range set.list {
		if transfer.requestor == requestor && transfer.property == property {
			return transfer
		}
	}
	return nil
}

// active reports whether any stream goes to the requestor window.
func (set *incrTransfers) active(requestor uint32) bool {
	for _, transfer := range set.list {
		if transfer.requestor == requestor {
			return true
		}
	}
	return false
}

func (set *incrTransfers) remove(target *incrTransfer) {
	for index, transfer := range set.list {
		if transfer == target {
			set.list = append(set.list[:index], set.list[index+1:]...)
			return
		}
	}
}

// expire drops the streams the requestor has not asked more of for incrTimeout and returns them.
func (set *incrTransfers) expire(now time.Time) (dropped []*incrTransfer) {
	kept := set.list[:0]
	for _, transfer := range set.list {
		if now.Sub(transfer.touched) > incrTimeout {
			dropped = append(dropped, transfer)
			continue
		}
		kept = append(kept, transfer)
	}
	for index := len(kept); index < len(set.list); index++ {
		set.list[index] = nil
	}
	set.list = kept
	return dropped
}
