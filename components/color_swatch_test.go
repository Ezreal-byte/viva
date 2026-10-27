package components

import (
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"
)

func TestColorSwatchSelectionKeyboardAndDisabled(t *testing.T) {
	var tree ui.Tree
	feature := tree.Feature("colors")
	red, blue := tree.ID(feature, "red"), tree.ID(feature, "blue")
	selected := red
	disabled := false
	driver, err := app.NewDriver(app.Config{Width: 180, Height: 80, Tree: &tree, Paint: func(c *ui.Canvas, size ui.Size) {
		if BeginColorSwatch(c, ui.LightTheme(), red, ui.Rect{X: 10, Y: 10, Width: 32, Height: 32}, ui.Color{Red: 1, Alpha: 1}).Label("Red").Selected(selected == red).Paint().Clicked() {
			selected = red
		}
		b := BeginColorSwatch(c, ui.LightTheme(), blue, ui.Rect{X: 52, Y: 10, Width: 32, Height: 32}, ui.Color{Blue: 1, Alpha: 1}).Label("Blue").Selected(selected == blue)
		if disabled {
			b = b.Disabled()
		}
		if b.Paint().Clicked() {
			selected = blue
		}
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if err := driver.Click(68, 26); err != nil {
		t.Fatal(err)
	}
	if selected != blue {
		t.Fatal("pointer did not select blue")
	}
	if err := driver.Click(26, 26); err != nil {
		t.Fatal(err)
	}
	if err := driver.Key(ui.KeyTab, 0); err != nil {
		t.Fatal(err)
	}
	if err := driver.Key(ui.KeySpace, 0); err != nil {
		t.Fatal(err)
	}
	if selected != blue {
		t.Fatal("keyboard did not select blue")
	}
	selected, disabled = red, true
	if err := driver.Resize(181, 80); err != nil {
		t.Fatal(err)
	}
	if err := driver.Click(68, 26); err != nil {
		t.Fatal(err)
	}
	if selected != red {
		t.Fatal("disabled swatch changed selection")
	}
	for _, h := range driver.Snapshot().Hits {
		if h.ID == blue && (!h.Disabled || h.Focusable) {
			t.Fatal("disabled swatch remains focusable")
		}
	}
}

func TestColorSwatchIgnoresBoundsTooSmallForItsDisc(t *testing.T) {
	var canvas ui.Canvas
	BeginColorSwatch(&canvas, ui.LightTheme(), ui.ID{}, ui.Rect{Width: 9, Height: 9}, ui.Color{}).Paint()
	if len(canvas.Rectangles()) != 0 || len(canvas.Hits()) != 0 {
		t.Fatal("undersized swatch declared negative-size geometry")
	}
}

func BenchmarkColorSwatch(b *testing.B) {
	var canvas ui.Canvas
	var tree ui.Tree
	id := tree.ID(tree.Feature("colors"), "color")
	paint := func() {
		canvas.Reset()
		BeginColorSwatch(&canvas, ui.LightTheme(), id, ui.Rect{Width: 32, Height: 32}, ui.Color{Red: 1, Alpha: 1}).Label("Red").Selected(true).Paint()
	}
	paint()
	b.ReportAllocs()
	for b.Loop() {
		paint()
	}
}
