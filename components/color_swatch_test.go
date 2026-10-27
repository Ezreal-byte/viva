package components

import (
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"
)

func TestColorSwatchSelectionKeyboardAndDisabled(t *testing.T) {
	var tree ui.Tree
	feature := tree.Feature("colors")
	red, blue, green := tree.ID(feature, "red"), tree.ID(feature, "blue"), tree.ID(feature, "green")
	options := [...]RadioOption{{ID: red}, {ID: blue}, {ID: green, Disabled: true}}
	selected := red
	driver, err := app.NewDriver(app.Config{Width: 180, Height: 80, Tree: &tree, Paint: func(c *ui.Canvas, size ui.Size) {
		if BeginColorSwatch(c, ui.LightTheme(), red, ui.Rect{X: 10, Y: 10, Width: 32, Height: 32}, ui.Color{Red: 1, Alpha: 1}).Label("Red").Selected(selected == red).TabStop(selected == red).Paint().Clicked() {
			selected = red
		}
		if BeginColorSwatch(c, ui.LightTheme(), blue, ui.Rect{X: 52, Y: 10, Width: 32, Height: 32}, ui.Color{Blue: 1, Alpha: 1}).Label("Blue").Selected(selected == blue).TabStop(selected == blue).Paint().Clicked() {
			selected = blue
		}
		BeginColorSwatch(c, ui.LightTheme(), green, ui.Rect{X: 94, Y: 10, Width: 32, Height: 32}, ui.Color{Green: 1, Alpha: 1}).Label("Green").Selected(selected == green).TabStop(selected == green).Disabled().Paint()
	}, OnKey: func(event ui.KeyEvent) ui.EventResult {
		next, handled := RadioGroupKeyDown(event.Target, options[:], DirectionLTR, event.Key)
		if !handled {
			return ui.EventResult{}
		}
		selected = next
		return ui.EventResult{Focus: next, FocusVisible: true, Handled: true, Repaint: true}
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if err := driver.Key(ui.KeyTab, 0); err != nil {
		t.Fatal(err)
	}
	if err := driver.Key(ui.KeyRight, 0); err != nil {
		t.Fatal(err)
	}
	if selected != blue || driver.Snapshot().Focused != blue {
		t.Fatalf("right arrow selected %v with focus %v, want blue", selected, driver.Snapshot().Focused)
	}
	if err := driver.Key(ui.KeyRight, 0); err != nil {
		t.Fatal(err)
	}
	if selected != red || driver.Snapshot().Focused != red {
		t.Fatalf("right arrow did not skip disabled green: selected %v, focus %v", selected, driver.Snapshot().Focused)
	}
	if err := driver.Click(68, 26); err != nil {
		t.Fatal(err)
	}
	if selected != blue {
		t.Fatal("pointer did not select blue")
	}
	tabStops := 0
	for _, h := range driver.Snapshot().Hits {
		if h.Hint != "" {
			t.Fatalf("%v hint = %q, want empty", h.ID, h.Hint)
		}
		if h.Focusable && !h.SkipTab {
			tabStops++
		}
		if h.ID == green && (!h.Disabled || h.Focusable) {
			t.Fatal("disabled swatch remains focusable")
		}
	}
	if tabStops != 1 {
		t.Fatalf("tab stops = %d, want 1", tabStops)
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
