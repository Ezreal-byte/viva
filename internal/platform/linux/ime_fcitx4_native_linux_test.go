package linux

import (
	"os"
	"os/exec"
	"testing"

	"github.com/vivaui/viva/internal/platform"
)

// Run under dbus-run-session and Xvfb with Fcitx 4 and its Pinyin engine installed.
func TestNativeFcitx4Frontend(t *testing.T) {
	if os.Getenv("VIVA_FCITX4_NATIVE") != "1" {
		t.Skip("requires an isolated Fcitx 4 session")
	}
	t.Setenv("XMODIFIERS", "@im=fcitx")
	app := &linuxApplication{}
	h := newIMEHarness(t)
	ime := app.newIME(&window{}, h.input, h.ime.router)
	if ime == nil {
		t.Fatal("Fcitx did not open an input context")
	}
	defer app.ibus.Close()
	defer ime.close()
	ime.origin = h.ime.origin
	h.ime = ime
	h.focus(t)
	remote := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("fcitx-remote", args...).CombinedOutput(); err != nil {
			t.Fatalf("fcitx-remote %v: %v: %s", args, err, out)
		}
	}
	remote("-s", "pinyin")
	remote("-o")
	var timestamp uint32
	press := func(symbol uint32, code uint8) {
		t.Helper()
		timestamp += 20
		event := &xcbInputEvent{Detail: code, Time: timestamp}
		key := platform.KeyEvent{Key: normalizedKeysym(symbol), Text: string(rune(symbol))}
		if !ime.processKey(symbol, event, key) {
			t.Fatal("key bypassed Fcitx")
		}
		nativeFrontendDrain(t, app.ibus, func() bool { _, waiting := app.ibus.Deadline(); return !waiting })
	}
	press('n', 57)
	press('i', 31)
	press('h', 43)
	press('a', 38)
	press('o', 32)
	if !ime.composing() || len(h.delivered) != 0 {
		t.Fatalf("Pinyin bypassed composing: text=%q keys=%v", h.editor.String(), h.delivered)
	}
	press(' ', 65)
	if got := h.editor.String(); got != "你好" || ime.composing() || len(h.delivered) != 0 {
		t.Fatalf("Pinyin commit was not consumed once: text=%q composing=%t keys=%v", got, ime.composing(), h.delivered)
	}
	remote("-c")
	press('x', 53)
	if got := h.editor.String(); got != "你好x" {
		t.Fatalf("plain fallback = %q, want 你好x", got)
	}
}
