package linux

import (
	"testing"
)

func TestXIMPacketRoundTrip(t *testing.T) {
	body := []byte{1, 2, 3, 4}
	message := make([]byte, ximHeaderSize+len(body))
	message[0] = byte(ximConnect)
	putUint16(message[2:], uint16(len(body)))
	copy(message[ximHeaderSize:], body)
	if got := getUint16(message[2:]); got != uint16(len(body)) {
		t.Fatalf("length = %d", got)
	}
}

func TestXIMDispatchParsesOpenAndContext(t *testing.T) {
	transport := &ximTransport{}
	open := make([]byte, ximHeaderSize+2)
	open[0] = byte(ximOpenReply)
	putUint16(open[2:], 2)
	putUint16(open[ximHeaderSize:], 7)
	transport.reading = append(transport.reading, open...)
	context := make([]byte, ximHeaderSize+4)
	context[0] = byte(ximCreateICReply)
	putUint16(context[2:], 4)
	putUint16(context[ximHeaderSize:], 7)
	putUint16(context[ximHeaderSize+2:], 11)
	transport.reading = append(transport.reading, context...)
	if err := transport.dispatch(); err != nil {
		t.Fatal(err)
	}
	if transport.imid != 7 || transport.icid != 11 {
		t.Fatalf("ids = %d/%d", transport.imid, transport.icid)
	}
}

func TestXIMDispatchHoldsPartialPacket(t *testing.T) {
	transport := &ximTransport{reading: []byte{byte(ximOpenReply), 0, 0, 4, 1, 0}}
	if err := transport.dispatch(); err != nil {
		t.Fatal(err)
	}
	if len(transport.reading) != 2 {
		t.Fatalf("reading = %v", transport.reading)
	}
}

func TestXIMDispatchParsesCommitAndPreedit(t *testing.T) {
	transport := &ximTransport{imid: 7, icid: 11}
	commit := make([]byte, ximHeaderSize+10)
	commit[0] = byte(ximCommit)
	putUint16(commit[2:], 10)
	putUint16(commit[ximHeaderSize:], 7)
	putUint16(commit[ximHeaderSize+2:], 3)
	copy(commit[ximHeaderSize+4:], "你好")
	preedit := make([]byte, ximHeaderSize+18+len("nihao"))
	preedit[0] = byte(ximPreeditDraw)
	putUint16(preedit[2:], uint16(18+len("nihao")))
	putUint16(preedit[ximHeaderSize:], 7)
	putUint16(preedit[ximHeaderSize+2:], 11)
	putUint32(preedit[ximHeaderSize+12:], 2)
	putUint16(preedit[ximHeaderSize+16:], uint16(len("nihao")))
	copy(preedit[ximHeaderSize+18:], "nihao")
	transport.reading = append(transport.reading, commit...)
	transport.reading = append(transport.reading, preedit...)
	if err := transport.dispatch(); err != nil {
		t.Fatal(err)
	}
	if len(transport.commitValues) != 1 || transport.commitValues[0] != "你好" {
		t.Fatalf("commits = %q", transport.commitValues)
	}
	if len(transport.preeditStrings) != 1 || transport.preeditStrings[0] != "nihao" {
		t.Fatalf("preedits = %q", transport.preeditStrings)
	}
}
