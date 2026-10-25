package linux

import (
	"bytes"
	"testing"
	"time"
)

func TestAStreamHandsOutItsDataInPiecesAndEndsWithAnEmptyOne(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 2*incrChunk+7)
	transfer := &incrTransfer{data: data}
	var collected []byte
	pieces := 0
	for {
		piece, last := transfer.next(time.Unix(0, 0))
		if last {
			if len(piece) != 0 {
				t.Fatalf("the last piece holds %d bytes", len(piece))
			}
			break
		}
		if len(piece) > incrChunk {
			t.Fatalf("a piece of %d bytes is more than a request carries", len(piece))
		}
		collected = append(collected, piece...)
		pieces++
	}
	if pieces != 3 || !bytes.Equal(collected, data) {
		t.Fatalf("%d pieces, %d of %d bytes collected", pieces, len(collected), len(data))
	}
}

func TestStreamsToOnePropertyReplaceEachOtherAndTheOldestGiveWay(t *testing.T) {
	now := time.Unix(100, 0)
	var set incrTransfers
	first := &incrTransfer{requestor: 1, property: 10}
	if dropped := set.begin(first, now); len(dropped) != 0 {
		t.Fatalf("dropped %d", len(dropped))
	}
	second := &incrTransfer{requestor: 1, property: 10}
	if dropped := set.begin(second, now); len(dropped) != 1 || dropped[0] != first {
		t.Fatalf("a new stream to the same property dropped %v", dropped)
	}
	if set.find(1, 10) != second || set.find(1, 11) != nil || !set.active(1) || set.active(2) {
		t.Fatal("find or active is wrong")
	}
	for index := uint32(0); index < maxIncrTransfers; index++ {
		set.begin(&incrTransfer{requestor: 100 + index, property: 1}, now)
	}
	if len(set.list) != maxIncrTransfers {
		t.Fatalf("%d streams, at most %d allowed", len(set.list), maxIncrTransfers)
	}
	if set.find(1, 10) != nil {
		t.Fatal("the oldest stream was kept")
	}
}

func TestAStreamNobodyReadsIsForgotten(t *testing.T) {
	start := time.Unix(100, 0)
	var set incrTransfers
	set.begin(&incrTransfer{requestor: 1, property: 1}, start)
	busy := &incrTransfer{requestor: 2, property: 1}
	set.begin(busy, start)
	busy.next(start.Add(incrTimeout)) // the requestor of this one is still reading
	dropped := set.expire(start.Add(incrTimeout + time.Second))
	if len(dropped) != 1 || dropped[0].requestor != 1 || len(set.list) != 1 || set.list[0] != busy {
		t.Fatalf("expired %v, kept %v", dropped, set.list)
	}
	set.remove(busy)
	if len(set.list) != 0 {
		t.Fatal("remove kept the stream")
	}
}
