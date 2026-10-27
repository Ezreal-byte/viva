//go:build darwin || windows || linux

package app

import (
	"testing"

	"github.com/vivaui/viva/text"
	"github.com/vivaui/viva/ui"
)

func TestEditableComboboxUsesNativeQueryGeometry(t *testing.T) {
	var editor text.Editor
	id := ui.Item(1)
	query := ui.Rect{X: 120, Y: 10, Width: 150, Height: 30}
	driver, err := NewDriver(Config{Width: 300, Height: 100, Paint: func(canvas *ui.Canvas, _ ui.Size) {
		canvas.HitWithOptions(id, query, "Tags", ui.HitOptions{Role: ui.RoleCombobox, Value: editor.String(), Expanded: true, TextBounds: query, TextSize: 14})
	}, TextInput: func(c TextInputContext) *TextInput {
		if c.ID != id || c.TextBounds != query {
			t.Fatal("combobox lost its query editor geometry", c)
		}
		return &TextInput{Editor: &editor}
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if err := driver.Click(140, 20); err != nil {
		t.Fatal(err)
	}
	if err := driver.Type("开发"); err != nil {
		t.Fatal(err)
	}
	if editor.String() != "开发" {
		t.Fatal("combobox did not receive native text", editor.String())
	}
}
