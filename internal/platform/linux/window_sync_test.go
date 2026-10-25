package linux

import "testing"

func TestSyncRequestValueReadsTheCounterWords(t *testing.T) {
	var data [20]byte
	// atom, timestamp, low, high, extended flag
	copy(data[8:], []byte{0x78, 0x56, 0x34, 0x12})
	copy(data[12:], []byte{0xff, 0xff, 0xff, 0xff})
	lo, hi := syncRequestValue(data)
	if lo != 0x12345678 || hi != -1 {
		t.Fatalf("value = %#x, %d", lo, hi)
	}
}

func TestSyncRequestAnswersOnlyTheLatest(t *testing.T) {
	var request syncRequest
	if _, _, ok := request.take(); ok {
		t.Fatal("nothing was requested")
	}
	request.request(2, 0)
	request.request(4, 0)
	if lo, hi, ok := request.take(); !ok || lo != 4 || hi != 0 {
		t.Fatalf("take = %d, %d, %v; want the latest request", lo, hi, ok)
	}
	if _, _, ok := request.take(); ok {
		t.Fatal("a request is answered once")
	}
}

func TestSyncInt64MatchesTheCStruct(t *testing.T) {
	// xcb_sync_int64_t is { int32 hi; uint32 lo }: hi in the low word.
	if got := syncInt64(5, 1); got != 1|5<<32 {
		t.Fatalf("syncInt64 = %#x", got)
	}
	if got := syncInt64(0, -1); got != 0xffffffff {
		t.Fatalf("syncInt64 of a negative high word = %#x", got)
	}
}
