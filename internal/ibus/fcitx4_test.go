//go:build unix

package ibus

import (
	"reflect"
	"testing"

	"github.com/vivaui/viva/internal/dbus"
)

func TestFcitx4DeliversCompositionBeforeTheKeyAnswer(t *testing.T) {
	const owner = ":1.8"
	transport := &fakeTransport{}
	bus := newInputBus(transport, owner, fcitx4Context)
	defer bus.Close()
	var output []string
	context, err := bus.NewContext("viva", Handler{
		Preedit: func(event PreeditEvent) {
			if !event.Visible || event.Cursor != 1 {
				t.Errorf("preedit visibility or UTF-8 cursor: %+v", event)
			}
			output = append(output, event.Text)
		},
		Commit: func(text string) { output = append(output, text) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := context.ProcessKeyAt('i', 23, 1|ReleaseMask, 42, func(handled bool) {
		if !handled {
			t.Error("the key that committed text would also be typed")
		}
		output = append(output, "answered")
	}); err != nil {
		t.Fatal(err)
	}
	call := transport.calls("ProcessKeyEvent")[0]
	args := dbus.NewDecoder(call.body)
	if call.signature != "uuuiu" || call.path != "/inputcontext_3" || args.Uint32() != 'i' || args.Uint32() != 31 ||
		args.Uint32() != 1 || args.Int32() != 1 || args.Uint32() != 42 || args.Err() != nil {
		t.Fatalf("invalid Fcitx key release: %+v", call)
	}
	var preedit, commit, answer dbus.Encoder
	array := preedit.BeginArray(8)
	preedit.String("你")
	preedit.Int32(0)
	preedit.Align(8)
	preedit.String("hao")
	preedit.Int32(0)
	preedit.EndArray(array)
	preedit.Int32(3)
	commit.String("你好")
	answer.Int32(1)
	transport.incoming = []*dbus.Message{
		{Type: dbus.TypeSignal, Sender: ":1.9", Path: call.path, Interface: fcitx4Context, Member: "CommitString", Signature: "s", Body: commit.Bytes()},
		{Type: dbus.TypeSignal, Sender: owner, Path: call.path, Interface: fcitx4Context, Member: "UpdateFormattedPreedit", Signature: "a(si)i", Body: preedit.Bytes()},
		{Type: dbus.TypeSignal, Sender: owner, Path: call.path, Interface: fcitx4Context, Member: "CommitString", Signature: "s", Body: commit.Bytes()},
		{Type: dbus.TypeReturn, Sender: owner, ReplySerial: call.serial, Signature: "i", Body: answer.Bytes()},
	}
	if err := bus.Dispatch(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"你hao", "你好", "answered"}; !reflect.DeepEqual(output, want) {
		t.Fatalf("Fcitx events = %v, want %v", output, want)
	}
}
