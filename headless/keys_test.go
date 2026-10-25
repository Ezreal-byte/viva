//go:build darwin || windows || linux

package headless

import (
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"
)

func TestControlKeydownKeyupHoldKeysAndModifiers(t *testing.T) {
	var keys []ui.KeyEvent
	var pointers []ui.PointerEvent
	config := app.Config{Title: "hold", Width: 300, Height: 120}
	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		canvas.HitWithOptions(ui.Item(1), ui.Rect{Width: size.Width, Height: size.Height}, "Terminal", ui.HitOptions{Role: ui.RoleTerminal})
	}
	config.OnKey = func(event ui.KeyEvent) ui.EventResult {
		keys = append(keys, event)
		return ui.EventResult{Handled: true}
	}
	config.OnPointer = func(event ui.PointerEvent) ui.EventResult {
		pointers = append(pointers, event)
		return ui.EventResult{}
	}
	config.TextInput = func(app.TextInputContext) *app.TextInput {
		return &app.TextInput{Bounds: ui.Rect{Width: 8, Height: 16}, FontSize: 13, KeyReleases: true}
	}
	session, err := Open(config, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.ClickLabel("Terminal"); err != nil {
		t.Fatal(err)
	}
	keys, pointers = nil, nil
	reply := func(line string) string { r, _ := execute(session, line); return r }
	for _, line := range []string{"keydown shift", "click 10 10", "keydown left", "keydown alt", "blur", "keyup left"} {
		if got := reply(line); !strings.HasPrefix(got, "ok") {
			t.Fatalf("%s: %s", line, got)
		}
	}
	var shiftClicks int
	for _, event := range pointers {
		if event.Kind == ui.PointerDown && event.Modifiers == ui.ModifierShift {
			shiftClicks++
		}
	}
	if shiftClicks != 1 {
		t.Fatalf("shift-click not seen: %+v", pointers)
	}
	var down, up int
	for _, event := range keys {
		if event.Release {
			up++
		} else {
			down++
		}
	}
	if down != 3 || up != 3 {
		t.Fatalf("down=%d up=%d (blur must release all, keyup of released key must not repeat): %+v", down, up, keys)
	}
	if got := reply("keydown bogus"); !strings.HasPrefix(got, "error") {
		t.Fatalf("unknown key accepted: %s", got)
	}
}
