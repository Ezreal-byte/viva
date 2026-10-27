//go:build darwin || windows || linux

package headless

import (
	"bytes"
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/text"
	"github.com/vivaui/viva/ui"
)

// A text field raised onto a layer, as one inside a popup or dialog is, must
// still show the text the platform editor draws: the offscreen render places
// that text above the layer's own paint.
func TestLayeredTextFieldDrawsItsNativeText(t *testing.T) {
	if !rendersTextEditor {
		t.Skip("this platform's headless renderer does not draw the editor text")
	}
	var editor text.Editor
	id := ui.Item(3)
	field := ui.Rect{X: 40, Y: 40, Width: 200, Height: 32}
	theme := ui.LightTheme()
	session, err := Open(app.Config{
		Title: "layered field", Width: 320, Height: 140,
		ClearColor: app.Color{Red: 1, Green: 1, Blue: 1, Alpha: 1},
		Paint: func(canvas *ui.Canvas, size ui.Size) {
			canvas.Fill(ui.Rect{Width: size.Width, Height: size.Height}, ui.Color{Red: 1, Green: 1, Blue: 1, Alpha: 1})
			canvas.PushLayer(3)
			defer canvas.PopLayer()
			canvas.Rounded(field, theme.Metrics.RadiusItem(), theme.Muted)
			canvas.HitWithOptions(id, field, "Name", ui.HitOptions{
				Role: ui.RoleTextField, Value: editor.String(), TextBounds: field,
				TextSize: theme.Metrics.FontSizeM, TextColor: theme.Foreground,
			})
		},
		TextInput: func(app.TextInputContext) *app.TextInput { return &app.TextInput{Editor: &editor} },
	}, nil, Options{Scale: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(session.Close)

	before, err := session.Image()
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Click(field.X+field.Width/2, field.Y+field.Height/2); err != nil {
		t.Fatal(err)
	}
	if err := session.Type("abc"); err != nil {
		t.Fatal(err)
	}
	after, err := session.Image()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before.Pix, after.Pix) {
		t.Fatal("typing into a layered field did not change the picture")
	}
}
