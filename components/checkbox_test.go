package components

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestCheckboxCheckedPaintAndTypedState(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	BeginCheckbox(&canvas, theme, ui.Item(1), ui.Rect{X: 20, Y: 20, Width: 16, Height: 16}, true).Label("Accept terms").Paint()
	rectangles := canvas.Rectangles()
	if len(rectangles) != 2 || rectangles[1].Color != theme.Primary {
		t.Fatalf("checked checkbox rectangles = %+v", rectangles)
	}
	if len(canvas.Texts()) != 0 || len(canvas.Triangles()) == 0 {
		t.Fatalf("the check mark is a path, not a character: %d texts, %d triangles", len(canvas.Texts()), len(canvas.Triangles()))
	}
	hit := canvas.Hits()[0]
	if hit.Role != ui.RoleCheckbox || !hit.Checked || hit.Label != "Accept terms" || !hit.Focusable {
		t.Fatalf("typed checkbox hit = %+v", hit)
	}
	if hit.Bounds != (ui.Rect{X: 8, Y: 12, Width: 40, Height: 32}) {
		t.Fatalf("checkbox hit bounds = %+v, want official expanded target", hit.Bounds)
	}
}

func TestCheckboxOfficialDefaultGeometryAndUncheckedSurfaces(t *testing.T) {
	for _, theme := range []ui.Theme{ui.LightTheme(), ui.DarkTheme()} {
		var canvas ui.Canvas
		BeginCheckbox(&canvas, theme, ui.ID{}, ui.Rect{X: 3, Y: 5, Width: 40, Height: 28}, false).Paint()
		box := canvas.Rectangles()[0]
		if box.Bounds != (ui.Rect{X: 3, Y: 5, Width: 16, Height: 16}) || box.Radius != 4 {
			t.Fatalf("default checkbox geometry = %+v", box)
		}
		// An unchecked box is the page's color inside its border (or Input/30 over
		// it in a dark theme), never the border color showing through.
		want := theme.Background
		if theme.Background.Red < 0.5 {
			want = scaledAlpha(theme.Input, 0.30).Over(theme.Background)
		}
		if fill := canvas.Rectangles()[1].Color; fill != want {
			t.Fatalf("unchecked fill = %+v, want %+v", fill, want)
		}
		if len(canvas.Hits()) != 0 {
			t.Fatalf("preview checkbox declared %d hits", len(canvas.Hits()))
		}
	}
}

func TestCheckboxDisabledRetainsSemanticHitAtHalfOpacity(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	BeginCheckbox(&canvas, theme, ui.Item(1), ui.Rect{Width: 16, Height: 16}, true).Disabled().Paint()
	if len(canvas.Hits()) != 1 || !canvas.Hits()[0].Disabled || canvas.Hits()[0].Focusable || !canvas.Hits()[0].Checked {
		t.Fatalf("disabled checkbox hit = %+v", canvas.Hits())
	}
	want := withAlpha(theme.Primary, 0.50).Over(theme.Background)
	if fill := canvas.Rectangles()[1].Color; fill != want {
		t.Fatalf("disabled checked fill = %+v, want %+v", fill, want)
	}
}

func TestCheckboxFocusAndInvalidUseOfficialRings(t *testing.T) {
	theme := ui.DarkTheme()
	var focused, invalid ui.Canvas
	focused.SetInteraction(ui.ID{}, ui.ID{}, ui.Item(1), true)
	BeginCheckbox(&focused, theme, ui.Item(1), ui.Rect{X: 4, Y: 4, Width: 16, Height: 16}, false).Paint()
	BeginCheckbox(&invalid, theme, ui.Item(2), ui.Rect{X: 4, Y: 4, Width: 16, Height: 16}, false).Invalid().Paint()

	wantFocus := theme.Ring
	wantFocus.Alpha *= 0.50
	if ring := focused.Rectangles()[0]; ring.Color != wantFocus || ring.Bounds != (ui.Rect{X: 1, Y: 1, Width: 22, Height: 22}) {
		t.Fatalf("focus ring = %+v", ring)
	}
	wantInvalid := withAlpha(theme.Danger, 0.40)
	wantBorder := invalidBorder(theme).Over(theme.Background)
	if ring := invalid.Rectangles()[0]; ring.Color != wantInvalid || invalid.Rectangles()[1].Color != wantBorder {
		t.Fatalf("invalid ring/border = %+v / %+v", ring, invalid.Rectangles()[1])
	}
}

func TestSeparatorExplicitOrientationAndThickness(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	PaintSeparator(&canvas, theme, ui.Rect{Width: 120, Height: 9}, SeparatorHorizontal)
	PaintSeparator(&canvas, theme, ui.Rect{X: 130, Width: 9, Height: 120}, SeparatorVertical)
	lines := canvas.Rectangles()
	if len(lines) != 2 || lines[0].Bounds.Height != 1 || lines[0].Bounds.Width != 120 {
		t.Fatalf("horizontal separator = %+v", lines)
	}
	if lines[1].Bounds.Width != 1 || lines[1].Bounds.Height != 120 {
		t.Fatalf("vertical separator = %+v", lines[1].Bounds)
	}
}

func TestCheckboxRepeatedPaintReusesCanvasStorage(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	BeginCheckbox(&canvas, theme, ui.Item(1), ui.Rect{Width: 16, Height: 16}, true).Paint()
	allocations := testing.AllocsPerRun(100, func() {
		canvas.Reset()
		BeginCheckbox(&canvas, theme, ui.Item(1), ui.Rect{Width: 16, Height: 16}, true).Paint()
	})
	if allocations != 0 {
		t.Fatalf("repeated checkbox paint allocated %v times", allocations)
	}
}

// The check and the cross are paths centered in the box they are given, so they
// look the same on every platform and sit where the box says; no character is
// involved.
func TestMarksAreVectorsCenteredInTheirBounds(t *testing.T) {
	bounds := ui.Rect{X: 100, Y: 40, Width: 28, Height: 28}
	for name, paint := range map[string]func(*ui.Canvas, ui.Rect, float32, ui.Color){"check": paintCheck, "cross": paintCross} {
		var canvas ui.Canvas
		paint(&canvas, bounds, 16, ui.Color{Red: 1, Alpha: 1})
		if len(canvas.Texts()) != 0 || len(canvas.Triangles()) == 0 {
			t.Fatalf("%s: %d texts, %d triangles", name, len(canvas.Texts()), len(canvas.Triangles()))
		}
		least, greatest := ui.Point{X: 1e9, Y: 1e9}, ui.Point{X: -1e9, Y: -1e9}
		for _, triangle := range canvas.Triangles() {
			for _, point := range triangle.Points {
				least.X, least.Y = min(least.X, point.X), min(least.Y, point.Y)
				greatest.X, greatest.Y = max(greatest.X, point.X), max(greatest.Y, point.Y)
			}
		}
		if centerX, centerY := (least.X+greatest.X)/2, (least.Y+greatest.Y)/2; centerX < 113 || centerX > 115 || centerY < 53 || centerY > 55 {
			t.Errorf("%s is centered on (%v, %v), not on the box's (114, 54)", name, centerX, centerY)
		}
	}
}

// The indeterminate state is a primary box with a dash and a mixed
// accessibility value, not a checked box with a tick.
func TestCheckboxIndeterminatePaintAndSemantics(t *testing.T) {
	theme := ui.LightTheme()
	var mixed, checked ui.Canvas
	mixedHit := BeginCheckbox(&mixed, theme, ui.Item(9), ui.Rect{Width: 16, Height: 16}, true).Indeterminate().Paint()
	BeginCheckbox(&checked, theme, ui.Item(9), ui.Rect{Width: 16, Height: 16}, true).Paint()
	if hit := mixed.Hits()[0]; !hit.Mixed || hit.Checked || hit.Role != ui.RoleCheckbox {
		t.Fatalf("mixed hit = %+v", hit)
	}
	if hit := checked.Hits()[0]; hit.Mixed || !hit.Checked {
		t.Fatalf("checked hit = %+v", hit)
	}
	if len(mixed.Rectangles()) == len(checked.Rectangles()) {
		t.Fatal("the dash did not add a rectangle the tick does not have")
	}
	if mixedHit.Clicked() || !mixed.Hits()[0].Focusable {
		t.Fatal("mixed checkbox lost its hit")
	}
}
