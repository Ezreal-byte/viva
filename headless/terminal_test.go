//go:build darwin || windows || linux

package headless

import (
	"reflect"
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"
)

func TestSessionDrivesATerminalWithRawKeysChordsTextAndPaste(t *testing.T) {
	var keys []ui.KeyEvent
	var typed []string
	config := app.Config{Title: "term", Width: 300, Height: 120}
	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		canvas.HitWithOptions(ui.Item(1), ui.Rect{Width: size.Width, Height: size.Height}, "Terminal", ui.HitOptions{Role: ui.RoleTerminal})
	}
	config.OnKey = func(event ui.KeyEvent) ui.EventResult {
		keys = append(keys, event)
		result := ui.EventResult{Handled: true}
		if event.Modifiers == ui.ModifierMeta && event.Text == "c" {
			result.ClipboardText = "selected"
		}
		return result
	}
	config.TextInput = func(app.TextInputContext) *app.TextInput {
		return &app.TextInput{
			Bounds: ui.Rect{X: 4, Y: 4, Width: 8, Height: 16}, FontSize: 13,
			AltSendsEscape: true,
			OnEvent: func(event app.TextEvent) (app.TextState, error) {
				typed = append(typed, event.Text)
				return app.TextState{}, nil
			},
		}
	}
	session, err := Open(config, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	if err := session.Keys("ctrl+c"); err == nil {
		t.Fatal("a Control chord with no terminal focused was accepted")
	}
	if err := session.ClickLabel("Terminal"); err != nil {
		t.Fatal(err)
	}
	if err := session.Keys("cmd+c"); err != nil {
		t.Fatal(err)
	}
	if got := session.Driver().Clipboard(); got != "selected" {
		t.Fatalf("private clipboard = %q", got)
	}
	for _, spec := range []string{"ctrl+c", "ctrl+D", "alt+x", "up", "pageup", "f5", "f12", "insert", "shift+tab", "alt+left", "enter", "esc"} {
		if err := session.Keys(spec); err != nil {
			t.Fatalf("%s: %v", spec, err)
		}
	}
	if err := session.Type("ls -la"); err != nil {
		t.Fatal(err)
	}
	session.Driver().SetClipboard("a\nb")
	if err := session.Keys("cmd+v"); err != nil {
		t.Fatal(err)
	}
	want := []ui.KeyEvent{
		{Modifiers: ui.ModifierMeta, Text: "c"},
		{Modifiers: ui.ModifierControl, Text: "c"}, {Modifiers: ui.ModifierControl, Text: "d"},
		{Modifiers: ui.ModifierAlt, Text: "x"},
		{Key: ui.KeyUp}, {Key: ui.KeyPageUp}, {Key: ui.KeyF5}, {Key: ui.KeyF12}, {Key: ui.KeyInsert},
		{Key: ui.KeyTab, Modifiers: ui.ModifierShift}, {Key: ui.KeyLeft, Modifiers: ui.ModifierAlt},
		{Key: ui.KeyEnter}, {Key: ui.KeyEscape},
	}
	for index := range want {
		want[index].Target = ui.Item(1)
	}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("raw keys =\n%+v\nwant\n%+v", keys, want)
	}
	if !reflect.DeepEqual(typed, []string{"ls -la", "a\nb"}) {
		t.Fatalf("text = %q", typed)
	}
}
