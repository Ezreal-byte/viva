//go:build unix

package dbus

import (
	"bytes"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestMessageRoundTripKeepsEveryField(t *testing.T) {
	var body Encoder
	body.String("hello")
	body.Int32(-7)
	mark := body.BeginArray(1)
	body.Byte(1)
	body.Byte(2)
	body.EndArray(mark)
	in := &Message{
		Type: TypeCall, Flags: flagNoReply, Serial: 9, Path: "/org/x", Interface: "org.x.I", Member: "Go",
		Destination: "org.x", Sender: ":1.5", Signature: "siay", Body: body.Bytes(),
	}
	wire := in.encode()
	if total, err := messageLength(wire); err != nil || total != len(wire) {
		t.Fatalf("messageLength = %d, %v for %d bytes", total, err, len(wire))
	}
	out, err := decodeMessage(wire)
	if err != nil {
		t.Fatal(err)
	}
	if out.Type != in.Type || out.Flags != in.Flags || out.Serial != 9 || out.Path != in.Path || out.Interface != in.Interface ||
		out.Member != in.Member || out.Destination != in.Destination || out.Sender != in.Sender || out.Signature != in.Signature {
		t.Fatalf("decoded %+v, sent %+v", out, in)
	}
	args := out.Args()
	if args.String() != "hello" || args.Int32() != -7 {
		t.Fatalf("arguments decoded wrong")
	}
	end := args.ArrayLength(1)
	if args.Byte() != 1 || args.Byte() != 2 || args.Pos() != end || args.Err() != nil {
		t.Fatalf("array decoded wrong: err=%v", args.Err())
	}
}

func TestReturnAndErrorCarryTheirReplySerial(t *testing.T) {
	for _, kind := range []byte{TypeReturn, TypeError} {
		in := &Message{Type: kind, Serial: 2, ReplySerial: 77, ErrorName: map[bool]string{true: "org.x.Err"}[kind == TypeError]}
		out, err := decodeMessage(in.encode())
		if err != nil || out.ReplySerial != 77 || out.ErrorName != in.ErrorName {
			t.Fatalf("kind %d: %+v, %v", kind, out, err)
		}
	}
}

func TestArraysOfAlignedElementsPadBeforeTheFirstElement(t *testing.T) {
	// A byte then an array of 8-aligned structs: the length word is followed by
	// four bytes of padding that the length does not count.
	var e Encoder
	e.Byte(1)
	mark := e.BeginArray(8)
	e.Align(8)
	e.Int32(3)
	e.Int32(4)
	e.EndArray(mark)
	d := NewDecoder(e.Bytes())
	d.Byte()
	end := d.ArrayLength(8)
	if d.Int32() != 3 || d.Int32() != 4 || d.Pos() != end || d.Err() != nil {
		t.Fatalf("decoded wrong, err=%v", d.Err())
	}
}

func TestSkipStepsOverNestedContainers(t *testing.T) {
	// a{sv} with a string and an a(iiay) inside variants, then a sentinel.
	var e Encoder
	outer := e.BeginArray(8)
	e.Align(8)
	e.String("Name")
	e.Variant("s")
	e.String("value")
	e.Align(8)
	e.String("Icon")
	e.Variant("a(iiay)")
	pixels := e.BeginArray(8)
	e.Align(8)
	e.Int32(1)
	e.Int32(1)
	bytes := e.BeginArray(1)
	e.Byte(255)
	e.Byte(0)
	e.Byte(0)
	e.Byte(0)
	e.EndArray(bytes)
	e.EndArray(pixels)
	e.EndArray(outer)
	e.Uint32(0xfeed)
	d := NewDecoder(e.Bytes())
	d.Skip("a{sv}")
	if got := d.Uint32(); got != 0xfeed || d.Err() != nil {
		t.Fatalf("after Skip read %#x, err=%v", got, d.Err())
	}
}

func TestDecoderReportsTruncationInsteadOfPanicking(t *testing.T) {
	var e Encoder
	e.String("some text")
	whole := e.Bytes()
	for cut := 0; cut < len(whole); cut++ {
		d := NewDecoder(whole[:cut])
		_ = d.String()
		if d.Err() == nil {
			t.Fatalf("a string cut at %d of %d bytes decoded without error", cut, len(whole))
		}
	}
}

func TestSkipRejectsMalformedSignatures(t *testing.T) {
	for _, sig := range []string{"", "a", "(i", "{ss", "z", "ii", "a(", "(i}", "()", "a()", "a{}"} {
		d := NewDecoder(make([]byte, 64))
		d.Skip(sig)
		if d.Err() == nil {
			t.Errorf("signature %q was accepted", sig)
		}
	}
}

func TestUnixPathUnderstandsBusAddresses(t *testing.T) {
	for address, want := range map[string]string{
		"unix:path=/run/user/1000/bus":          "/run/user/1000/bus",
		"unix:path=/run/user/1000/bus,guid=abc": "/run/user/1000/bus",
		"unix:abstract=/tmp/dbus-x,guid=1":      "@/tmp/dbus-x",
		"unix:guid=1,path=/tmp/a%20b":           "/tmp/a b",
	} {
		if got, ok := unixPath(address); !ok || got != want {
			t.Errorf("unixPath(%q) = %q, %v; want %q", address, got, ok, want)
		}
	}
	if _, ok := unixPath("tcp:host=localhost,port=1"); ok {
		t.Error("a tcp address was accepted")
	}
}

// fakeBus is the other end of a socketpair: it checks the authentication, says
// Hello, and lets a test script the rest.
type fakeBus struct {
	t  *testing.T
	fd int
	in []byte
}

func newFakeBus(t *testing.T) (*fakeBus, *Conn) {
	t.Helper()
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	bus := &fakeBus{t: t, fd: pair[1]}
	t.Cleanup(func() { syscall.Close(bus.fd) })
	var conn *Conn
	var connErr error
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		conn, connErr = NewConn(pair[0])
	}()
	if line := bus.line(); !strings.HasPrefix(line, "\x00AUTH EXTERNAL ") {
		t.Fatalf("first line of the exchange = %q", line)
	}
	bus.write([]byte("OK 0123456789abcdef\r\n"))
	if line := bus.line(); line != "BEGIN" {
		t.Fatalf("second line = %q", line)
	}
	hello := bus.message()
	if hello.Member != "Hello" || hello.Destination != "org.freedesktop.DBus" {
		t.Fatalf("first call = %+v", hello)
	}
	var name Encoder
	name.String(":1.42")
	bus.write((&Message{Type: TypeReturn, Serial: 1, ReplySerial: hello.Serial, Signature: "s", Body: name.Bytes()}).encode())
	wait.Wait()
	if connErr != nil {
		t.Fatal(connErr)
	}
	t.Cleanup(conn.Close)
	return bus, conn
}

func (bus *fakeBus) write(data []byte) {
	bus.t.Helper()
	if _, err := syscall.Write(bus.fd, data); err != nil {
		bus.t.Fatal(err)
	}
}

func (bus *fakeBus) fill() {
	bus.t.Helper()
	buffer := make([]byte, 4096)
	n, err := syscall.Read(bus.fd, buffer)
	if err != nil || n == 0 {
		bus.t.Fatalf("fake bus read: %d, %v", n, err)
	}
	bus.in = append(bus.in, buffer[:n]...)
}

func (bus *fakeBus) line() string {
	bus.t.Helper()
	for {
		if end := bytes.Index(bus.in, []byte("\r\n")); end >= 0 {
			line := string(bus.in[:end])
			bus.in = bus.in[end+2:]
			return line
		}
		bus.fill()
	}
}

func (bus *fakeBus) message() *Message {
	bus.t.Helper()
	for {
		total, err := messageLength(bus.in)
		if err != nil {
			bus.t.Fatal(err)
		}
		if total > 0 && len(bus.in) >= total {
			message, err := decodeMessage(bus.in[:total])
			if err != nil {
				bus.t.Fatal(err)
			}
			message.Body = append([]byte(nil), message.Body...)
			bus.in = bus.in[total:]
			return message
		}
		bus.fill()
	}
}

func TestConnAuthenticatesAndLearnsItsName(t *testing.T) {
	_, conn := newFakeBus(t)
	if conn.UniqueName() != ":1.42" {
		t.Fatalf("unique name = %q", conn.UniqueName())
	}
}

func TestCallReturnsItsAnswerAndKeepsOtherMessagesForReceive(t *testing.T) {
	bus, conn := newFakeBus(t)
	done := make(chan *Message, 1)
	go func() {
		reply, err := conn.Call("org.x", "/x", "org.x.I", "Ask", "", nil)
		if err != nil {
			t.Error(err)
		}
		done <- reply
	}()
	call := bus.message()
	// A signal arrives before the answer; it belongs to Receive.
	bus.write((&Message{Type: TypeSignal, Serial: 5, Path: "/s", Interface: "org.s", Member: "Ping"}).encode())
	var body Encoder
	body.Uint32(99)
	bus.write((&Message{Type: TypeReturn, Serial: 6, ReplySerial: call.Serial, Signature: "u", Body: body.Bytes()}).encode())
	reply := <-done
	if reply == nil || reply.Args().Uint32() != 99 {
		t.Fatalf("reply = %+v", reply)
	}
	messages, err := conn.Receive()
	if err != nil || len(messages) != 1 || messages[0].Member != "Ping" {
		t.Fatalf("Receive = %v, %v", messages, err)
	}
}

func TestCallTurnsAnErrorReplyIntoAnError(t *testing.T) {
	bus, conn := newFakeBus(t)
	errs := make(chan error, 1)
	go func() {
		_, err := conn.Call("org.x", "/x", "org.x.I", "Fail", "", nil)
		errs <- err
	}()
	call := bus.message()
	var text Encoder
	text.String("no such thing")
	bus.write((&Message{Type: TypeError, Serial: 3, ReplySerial: call.Serial, ErrorName: "org.x.Error.Missing", Signature: "s", Body: text.Bytes()}).encode())
	err := <-errs
	if e, ok := err.(*Error); !ok || e.Name != "org.x.Error.Missing" || e.Text != "no such thing" {
		t.Fatalf("error = %#v", err)
	}
}

func TestReceiveAssemblesAMessageDeliveredOneByteAtATime(t *testing.T) {
	bus, conn := newFakeBus(t)
	var body Encoder
	body.String("payload")
	wire := (&Message{Type: TypeSignal, Serial: 4, Path: "/p", Interface: "org.p", Member: "Slow", Signature: "s", Body: body.Bytes()}).encode()
	for index, b := range wire {
		bus.write([]byte{b})
		messages, err := conn.Receive()
		if err != nil {
			t.Fatal(err)
		}
		if index < len(wire)-1 && len(messages) != 0 {
			t.Fatalf("a message appeared after %d of %d bytes", index+1, len(wire))
		}
		if index == len(wire)-1 {
			if len(messages) != 1 || messages[0].Args().String() != "payload" {
				t.Fatalf("final Receive = %v", messages)
			}
		}
		time.Sleep(0) // let the kernel move the byte
	}
}

func TestReceiveSplitsTwoMessagesThatArriveTogether(t *testing.T) {
	bus, conn := newFakeBus(t)
	first := (&Message{Type: TypeSignal, Serial: 1, Path: "/p", Interface: "org.p", Member: "A"}).encode()
	second := (&Message{Type: TypeSignal, Serial: 2, Path: "/p", Interface: "org.p", Member: "B"}).encode()
	bus.write(append(first, second...))
	var members []string
	for len(members) < 2 {
		messages, err := conn.Receive()
		if err != nil {
			t.Fatal(err)
		}
		for _, message := range messages {
			members = append(members, message.Member)
		}
		time.Sleep(time.Millisecond)
	}
	if members[0] != "A" || members[1] != "B" {
		t.Fatalf("members = %v", members)
	}
}

func TestReplyAndEmitReachThePeer(t *testing.T) {
	bus, conn := newFakeBus(t)
	var args Encoder
	args.Bool(true)
	if err := conn.Reply(&Message{Serial: 12, Sender: ":1.9"}, "b", args.Bytes()); err != nil {
		t.Fatal(err)
	}
	reply := bus.message()
	if reply.Type != TypeReturn || reply.ReplySerial != 12 || reply.Destination != ":1.9" || !reply.Args().Bool() {
		t.Fatalf("reply = %+v", reply)
	}
	if err := conn.ReplyError(&Message{Serial: 13, Sender: ":1.9"}, "org.x.Bad", "why"); err != nil {
		t.Fatal(err)
	}
	if failure := bus.message(); failure.Type != TypeError || failure.ErrorName != "org.x.Bad" || failure.Args().String() != "why" {
		t.Fatalf("error reply = %+v", failure)
	}
	if err := conn.Emit("/o", "org.o", "Changed", "", nil); err != nil {
		t.Fatal(err)
	}
	if signal := bus.message(); signal.Type != TypeSignal || signal.Member != "Changed" || !signal.NoReply() {
		t.Fatalf("signal = %+v", signal)
	}
	// A call that wants no answer gets none.
	if err := conn.Reply(&Message{Serial: 14, Flags: flagNoReply}, "", nil); err != nil {
		t.Fatal(err)
	}
}

func TestReceiveRefusesAMessageThatClaimsToBeHuge(t *testing.T) {
	bus, conn := newFakeBus(t)
	header := []byte{'l', TypeSignal, 0, 1, 0xff, 0xff, 0xff, 0x7f, 1, 0, 0, 0, 0, 0, 0, 0}
	bus.write(header)
	if _, err := conn.Receive(); err == nil {
		t.Fatal("a 2 GiB body was accepted")
	}
}

func FuzzDecodeMessage(f *testing.F) {
	var body Encoder
	body.String("x")
	f.Add((&Message{Type: TypeCall, Serial: 1, Path: "/a", Interface: "b.c", Member: "D", Signature: "s", Body: body.Bytes()}).encode())
	f.Add([]byte{})
	f.Add([]byte{'l', 1, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		total, err := messageLength(data)
		if err != nil || total == 0 || total > len(data) {
			return
		}
		message, err := decodeMessage(data[:total])
		if err != nil {
			return
		}
		message.Args().SkipAll(message.Signature)
	})
}

func FuzzSkip(f *testing.F) {
	f.Add("a{sv}", []byte{0, 0, 0, 0})
	f.Add("(ii)", make([]byte, 16))
	f.Add("a(iiay)", []byte{4, 0, 0, 0, 1, 2, 3, 4})
	f.Fuzz(func(t *testing.T, signature string, data []byte) {
		d := NewDecoder(data)
		d.Skip(signature)
		d.SkipAll(signature)
	})
}

func BenchmarkEncodeIconProperties(b *testing.B) {
	pixels := make([]byte, 24*24*4)
	b.ReportAllocs()
	for b.Loop() {
		var e Encoder
		outer := e.BeginArray(8)
		e.Align(8)
		e.String("IconPixmap")
		e.Variant("a(iiay)")
		array := e.BeginArray(8)
		e.Align(8)
		e.Int32(24)
		e.Int32(24)
		data := e.BeginArray(1)
		e.buf = append(e.buf, pixels...)
		e.EndArray(data)
		e.EndArray(array)
		e.EndArray(outer)
	}
}

func TestSendDoesNotWaitAndTheAnswerComesThroughReceive(t *testing.T) {
	bus, conn := newFakeBus(t)
	var body Encoder
	body.Uint32(7)
	serial, err := conn.Send("org.x", "/x", "org.x.I", "Ask", "u", body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	call := bus.message()
	if call.Serial != serial || call.Member != "Ask" || call.Signature != "u" || call.Args().Uint32() != 7 {
		t.Fatalf("the bus saw %+v for serial %d", call, serial)
	}
	if messages, err := conn.Receive(); err != nil || len(messages) != 0 {
		t.Fatalf("nothing has been answered yet, Receive = %v, %v", messages, err)
	}
	var answer Encoder
	answer.Bool(true)
	bus.write((&Message{Type: TypeReturn, Serial: 9, ReplySerial: serial, Signature: "b", Body: answer.Bytes()}).encode())
	var messages []*Message
	for len(messages) == 0 {
		if messages, err = conn.Receive(); err != nil {
			t.Fatal(err)
		}
	}
	if messages[0].Type != TypeReturn || messages[0].ReplySerial != serial || !messages[0].Args().Bool() {
		t.Fatalf("answer = %+v", messages[0])
	}
}

func TestDialRefusesAddressesItCannotUse(t *testing.T) {
	for _, address := range []string{"", "tcp:host=localhost,port=1", "unix:guid=1"} {
		if conn, err := Dial(address); err == nil {
			conn.Close()
			t.Errorf("Dial(%q) succeeded", address)
		}
	}
}
