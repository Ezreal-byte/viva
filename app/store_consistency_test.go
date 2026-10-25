package app

import (
	"math/rand"
	"testing"

	"github.com/vivaui/viva/internal/scene"
	"github.com/vivaui/viva/ui"
)

// storeMatchesCanvas checks the retained render store against what the canvas
// painted: every rectangle and every triangle with the content the canvas gave
// it, and the shapes of both kinds in the order they were painted.
func storeMatchesCanvas(t *testing.T, painter *painter, painted []ui.Color, step int) {
	t.Helper()
	var rects, triangles int
	var order []ui.Color
	for _, primitive := range painter.renderer.Rectangles() {
		var color ui.Color
		if primitive.Kind == scene.KindTriangle {
			if triangles >= len(painter.clippedPaths) {
				t.Fatalf("step %d: the store holds more triangles than the canvas painted (%d)", step, len(painter.clippedPaths))
			}
			want := sceneTriangle(painter.clippedPaths[triangles])
			if primitive.Points != want.Points || primitive.Color != want.Color {
				t.Fatalf("step %d: stored triangle %d = %+v, canvas painted %+v", step, triangles, primitive.Points, want.Points)
			}
			triangles++
			color = painter.clippedPaths[triangles-1].Color
		} else {
			if rects >= len(painter.clipped) {
				t.Fatalf("step %d: the store holds more rectangles than the canvas painted (%d)", step, len(painter.clipped))
			}
			want := sceneRectangle(painter.clipped[rects])
			if primitive.Bounds != want.Bounds || primitive.Color != want.Color || primitive.Radius != want.Radius {
				t.Fatalf("step %d: stored rectangle %d = %+v, canvas painted %+v", step, rects, primitive.Bounds, want.Bounds)
			}
			rects++
			color = painter.clipped[rects-1].Color
		}
		if len(order) == 0 || order[len(order)-1] != color {
			order = append(order, color)
		}
	}
	if rects != len(painter.clipped) || triangles != len(painter.clippedPaths) {
		t.Fatalf("step %d: store has %d rectangles and %d triangles, canvas painted %d and %d",
			step, rects, triangles, len(painter.clipped), len(painter.clippedPaths))
	}
	// Every shape was painted in a color of its own, so the colors in store
	// order, one per run, are the shapes in store order.
	if len(order) != len(painted) {
		t.Fatalf("step %d: the store paints %d shapes in order, the canvas painted %d", step, len(order), len(painted))
	}
	for index := range order {
		if order[index] != painted[index] {
			t.Fatalf("step %d: shape %d of the store is %+v, the canvas painted %+v there (a shape drawn over what it should be under)",
				step, index, order[index], painted[index])
		}
	}
}

// A page whose set of shapes changes from one paint to the next (a hover
// highlight appears, a close button's icon shows up) must leave the render
// store equal to the canvas, wherever in paint order the change is, and the
// shapes must still stack as painted: the icon of a nav item stays above that
// item's highlight even though the highlight is a rectangle that was created
// long after the icon's triangles.
func TestStoreStaysEqualToTheCanvasAsShapesAppearAndDisappear(t *testing.T) {
	random := rand.New(rand.NewSource(3))
	shapes := func(variant int) (rectangles, icons int) {
		return variant & 7, (variant >> 3) & 7
	}
	variant := 0
	var painted []ui.Color
	icon := ui.Icon{ViewBox: 24, Stroke: 2, Cap: ui.CapRound, Join: ui.JoinRound, Commands: []ui.PathCommand{
		ui.MoveTo(4, 4), ui.LineTo(20, 4), ui.LineTo(20, 20), ui.LineTo(4, 20), ui.LineTo(4, 4),
	}}
	var canvas ui.Canvas
	config := Config{Title: "consistency", Width: 400, Height: 300}
	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		painted = painted[:0]
		next := func() ui.Color {
			// Distinct per shape within a pass: 1/255 apart in red, then green.
			index := len(painted) + 1
			color := ui.Color{Red: float32(index%250) / 255, Green: float32(index/250) / 255, Alpha: 1}
			painted = append(painted, color)
			return color
		}
		rectangles, icons := shapes(variant)
		// A fixed skeleton, with the variable parts in the middle: a nav item
		// whose icon must stay above its highlight, between a bar that grows
		// and content that follows.
		canvas.Fill(ui.Rect{Width: 400, Height: 36}, next())
		for index := 0; index < rectangles; index++ {
			canvas.Rounded(ui.Rect{X: float32(10 + index*30), Y: 4, Width: 24, Height: 24}, 5, next())
		}
		for index := 0; index < icons; index++ {
			canvas.DrawIcon(icon, ui.Rect{X: float32(10 + index*30), Y: 4, Width: 16, Height: 16}, next())
		}
		canvas.Rounded(ui.Rect{X: 9, Y: 105, Width: 180, Height: 36}, 7, next())
		canvas.DrawIcon(icon, ui.Rect{X: 24, Y: 115, Width: 16, Height: 16}, next())
		canvas.Rounded(ui.Rect{X: 200, Y: 60, Width: 100, Height: 100}, 8, next())
		canvas.DrawIcon(icon, ui.Rect{X: 220, Y: 80, Width: 16, Height: 16}, next())
	}
	painter := newTestPainter(t, &canvas, config)
	storeMatchesCanvas(t, painter, painted, 0)
	for step := 1; step <= 400; step++ {
		variant = random.Intn(64)
		if _, err := painter.resize(size(400+float32(step%2), 300)); err != nil {
			t.Fatal(err)
		}
		storeMatchesCanvas(t, painter, painted, step)
	}
}

// A page of icons whose hover highlight comes and goes: the same repaint a
// toolbar does on every pointer move. It must reach the renderer as a few
// changed nodes without allocating.
func BenchmarkPainterIconPageHoverToggle(b *testing.B) {
	icon := ui.Icon{ViewBox: 24, Stroke: 2, Cap: ui.CapRound, Join: ui.JoinRound, Commands: []ui.PathCommand{
		ui.MoveTo(4, 4), ui.LineTo(20, 4), ui.LineTo(20, 20), ui.LineTo(4, 20), ui.LineTo(4, 4),
	}}
	hovered := false
	var canvas ui.Canvas
	config := Config{Title: "icons", Width: 800, Height: 600}
	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		canvas.Fill(ui.Rect{Width: 800, Height: 600}, ui.Color{Alpha: 1})
		for index := 0; index < 60; index++ {
			x, y := float32(index%10*70), float32(index/10*50)
			if hovered && index == 7 {
				canvas.Rounded(ui.Rect{X: x, Y: y, Width: 64, Height: 44}, 6, ui.Color{Red: 0.3, Alpha: 1})
			}
			canvas.DrawIcon(icon, ui.Rect{X: x + 8, Y: y + 8, Width: 20, Height: 20}, ui.Color{Green: 1, Alpha: 1})
			canvas.Rounded(ui.Rect{X: x + 34, Y: y + 12, Width: 24, Height: 20}, 4, ui.Color{Blue: 1, Alpha: 1})
		}
	}
	painter := newTestPainter(b, &canvas, config)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		hovered = !hovered
		if _, err := painter.resize(size(800+float32(index%2), 600)); err != nil {
			b.Fatal(err)
		}
	}
}

// A highlight appearing in the middle of a page of icons costs the scene one
// new node, not a rewrite of everything painted after it, and what the
// renderer is told is that one node.
func TestInsertingAShapeMidPageTouchesOnlyThatShape(t *testing.T) {
	icon := ui.Icon{ViewBox: 24, Stroke: 2, Cap: ui.CapRound, Join: ui.JoinRound, Commands: []ui.PathCommand{
		ui.MoveTo(4, 4), ui.LineTo(20, 4), ui.LineTo(20, 20), ui.LineTo(4, 20), ui.LineTo(4, 4),
	}}
	highlighted := false
	var canvas ui.Canvas
	config := Config{Title: "icons", Width: 800, Height: 600}
	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		for index := 0; index < 30; index++ {
			x, y := float32(index%10*70), float32(index/10*50)
			if highlighted && index == 12 {
				canvas.Rounded(ui.Rect{X: x, Y: y, Width: 64, Height: 44}, 6, ui.Color{Red: 0.3, Alpha: 1})
			}
			canvas.DrawIcon(icon, ui.Rect{X: x + 8, Y: y + 8, Width: 20, Height: 20}, ui.Color{Green: 1, Alpha: 1})
			canvas.Rounded(ui.Rect{X: x + 34, Y: y + 12, Width: 24, Height: 20}, 4, ui.Color{Blue: 1, Alpha: 1})
		}
	}
	painter := newTestPainter(t, &canvas, config)
	shapes := len(painter.order)

	for _, want := range []bool{true, false, true} {
		highlighted = want
		if _, err := painter.resize(size(800, 600)); err != nil {
			t.Fatal(err)
		}
		if len(painter.changes) != 1 {
			t.Fatalf("highlighted=%v: %d scene changes for one shape among %d", want, len(painter.changes), shapes)
		}
		total := shapes
		if highlighted {
			total++
		}
		if len(painter.renderer.Rectangles()) != total {
			t.Fatalf("highlighted=%v: renderer holds %d shapes, want %d", highlighted, len(painter.renderer.Rectangles()), total)
		}
	}
}
