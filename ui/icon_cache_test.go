package ui

import (
	"math"
	"testing"
)

var testStrokeIcon = Icon{
	ViewBox: 24, Stroke: 2, Cap: CapRound, Join: JoinRound,
	Commands: []PathCommand{MoveTo(4, 12), LineTo(20, 12), MoveTo(14, 6), QuadTo(20, 12, 14, 18), MoveTo(12, 3), CubicTo(17, 3, 21, 7, 21, 12), ClosePath()},
}

var testLayeredIcon = Icon{
	ViewBox: 24,
	Layers: []IconLayer{
		{Fill: Color{Red: 0.9, Green: 0.2, Blue: 0.2, Alpha: 1}, Commands: []PathCommand{MoveTo(3, 3), LineTo(21, 3), LineTo(21, 21), LineTo(3, 21), ClosePath()}},
		{Fill: Color{Red: 1, Green: 1, Blue: 1, Alpha: 0.8}, FillRule: PathFillEvenOdd, Commands: []PathCommand{MoveTo(7, 7), LineTo(17, 7), LineTo(12, 17), ClosePath()}},
	},
}

// testWideIcon is a 4:1 horizontal mark: a filled bar and a stroked line.
var testWideIcon = Icon{
	ViewBox: 40, ViewHeight: 10, Stroke: 2, Cap: CapRound, Join: JoinRound,
	Commands: []PathCommand{MoveTo(24, 5), LineTo(36, 5)},
	Layers: []IconLayer{
		{Fill: Color{Red: 0.2, Green: 0.8, Blue: 0.4, Alpha: 1}, Commands: []PathCommand{MoveTo(0, 0), LineTo(20, 0), LineTo(20, 10), LineTo(0, 10), ClosePath()}},
	},
}

// A wide icon is scaled to fit and centered, not stretched to the bounds or
// reduced to the shorter side.
func TestWideIconKeepsItsAspectRatio(t *testing.T) {
	var canvas Canvas
	canvas.DrawIcon(testWideIcon, Rect{X: 100, Y: 200, Width: 80, Height: 80}, Color{Alpha: 1})
	minX, minY, maxX, maxY := float32(math.Inf(1)), float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.Inf(-1))
	for _, triangle := range canvas.Triangles() {
		for _, point := range triangle.Points {
			minX, minY = min(minX, point.X), min(minY, point.Y)
			maxX, maxY = max(maxX, point.X), max(maxY, point.Y)
		}
	}
	// 80 wide fits 40 units at 2 px each, so the 10 units are 20 px high, centered
	// in the 80 px slot: the shapes span y 230..250, from x 100 to the line's round
	// cap one unit past its end at 36.
	if math.Abs(float64(minX-100)) > 0.01 || math.Abs(float64(maxX-174)) > 0.5 || math.Abs(float64(minY-230)) > 0.01 || math.Abs(float64(maxY-250)) > 0.01 {
		t.Fatalf("the wide icon covers x %.2f..%.2f, y %.2f..%.2f; want x 100..174 and y 230..250", minX, maxX, minY, maxY)
	}
	canvas.Reset()
	canvas.DrawIcon(testWideIcon, Rect{X: 0, Y: 0, Width: 400, Height: 5}, Color{Alpha: 1})
	minX, minY, maxX, maxY = float32(math.Inf(1)), float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.Inf(-1))
	for _, triangle := range canvas.Triangles() {
		for _, point := range triangle.Points {
			minX, minY = min(minX, point.X), min(minY, point.Y)
			maxX, maxY = max(maxX, point.X), max(maxY, point.Y)
		}
	}
	// The 5 px height limits the scale to half a pixel per unit: the shapes fit a
	// 20 x 5 box, centered in the 400 px slot, not stretched across it.
	if maxY-minY > 5.01 || maxX-minX > 18.6 || minX < 189.9 {
		t.Fatalf("a 5 px high slot must shrink the wide icon, got x %.2f..%.2f, y %.2f..%.2f", minX, maxX, minY, maxY)
	}
}

// sameTriangles checks that two tessellations of one icon paint the same
// shape. The cache tessellates at the origin and the direct path at the icon's
// position, so the ear clipper sees polygons whose vertices differ in the last
// bits; where the outline has near-collinear vertices (the join arcs of a
// translucent stroke) it may then pick another of the equally valid
// triangulations, and which one depends on the CPU's float32 rounding (fused
// multiply-add on arm64, none on amd64). Both are the same polygon: the test
// accepts identical triangles, and otherwise requires identical covered area and
// identical coverage at a grid of sample points.
func sameTriangles(t *testing.T, got, want []Triangle) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("cached icon made %d triangles, direct made %d", len(got), len(want))
	}
	const tolerance = 2e-3
	identical := true
	for i := range got {
		if !colorsClose(got[i].Color, want[i].Color) || got[i].Shaded != want[i].Shaded ||
			got[i].Shaded && (!colorsClose(got[i].Shade[0], want[i].Shade[0]) || !colorsClose(got[i].Shade[1], want[i].Shade[1])) || got[i].Layer != want[i].Layer ||
			got[i].RectsBefore != want[i].RectsBefore || got[i].HasClip != want[i].HasClip || got[i].ClipOf != want[i].ClipOf {
			t.Fatalf("triangle %d differs apart from position: %+v vs %+v", i, got[i], want[i])
		}
		for p := range got[i].Points {
			if math.Abs(float64(got[i].Points[p].X-want[i].Points[p].X)) > tolerance || math.Abs(float64(got[i].Points[p].Y-want[i].Points[p].Y)) > tolerance {
				identical = false
			}
		}
		if identical && got[i].EdgeMask != want[i].EdgeMask {
			identical = false
		}
	}
	if identical {
		return
	}
	areaGot, areaWant := triangleArea(got), triangleArea(want)
	if math.Abs(areaGot-areaWant) > 1e-3*math.Max(areaGot, 1) {
		t.Fatalf("cached icon covers area %.4f, direct %.4f", areaGot, areaWant)
	}
	var minX, minY, maxX, maxY = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, triangle := range want {
		for _, point := range triangle.Points {
			minX, maxX = math.Min(minX, float64(point.X)), math.Max(maxX, float64(point.X))
			minY, maxY = math.Min(minY, float64(point.Y)), math.Max(maxY, float64(point.Y))
		}
	}
	for x := minX; x < maxX; x += 0.0371 {
		for y := minY; y < maxY; y += 0.0371 {
			if a, b := triangleCoverage(got, x, y), triangleCoverage(want, x, y); a != b {
				t.Fatalf("at (%.3f, %.3f) the cached icon is covered %d times, the direct one %d", x, y, a, b)
			}
		}
	}
}

func colorsClose(a, b Color) bool {
	return math.Abs(float64(a.Red-b.Red)) < 1e-5 && math.Abs(float64(a.Green-b.Green)) < 1e-5 &&
		math.Abs(float64(a.Blue-b.Blue)) < 1e-5 && math.Abs(float64(a.Alpha-b.Alpha)) < 1e-5
}

func triangleArea(triangles []Triangle) float64 {
	total := 0.0
	for _, triangle := range triangles {
		p := triangle.Points
		total += math.Abs(float64(p[1].X-p[0].X)*float64(p[2].Y-p[0].Y)-float64(p[2].X-p[0].X)*float64(p[1].Y-p[0].Y)) / 2
	}
	return total
}

// triangleCoverage counts the triangles that contain the point.
func triangleCoverage(triangles []Triangle, x, y float64) int {
	count := 0
	for _, triangle := range triangles {
		p := triangle.Points
		side := func(a, b Point) float64 {
			return (float64(b.X)-float64(a.X))*(y-float64(a.Y)) - (float64(b.Y)-float64(a.Y))*(x-float64(a.X))
		}
		d1, d2, d3 := side(p[0], p[1]), side(p[1], p[2]), side(p[2], p[0])
		if d1 >= 0 && d2 >= 0 && d3 >= 0 || d1 <= 0 && d2 <= 0 && d3 <= 0 {
			count++
		}
	}
	return count
}

// DrawIcon replays a cached tessellation; it must produce what tessellating from
// scratch would, wherever the icon is, however large, in any tint.
func TestCachedIconsPaintLikeTheDirectPath(t *testing.T) {
	var cached, direct Canvas
	tints := []Color{{Red: 0.1, Green: 0.2, Blue: 0.3, Alpha: 1}, {Red: 0.1, Green: 0.2, Blue: 0.3, Alpha: 0.4}}
	for _, icon := range []Icon{testStrokeIcon, testLayeredIcon, testWideIcon} {
		for _, bounds := range []Rect{
			{X: 10, Y: 20, Width: 16, Height: 16}, {X: 333.5, Y: 701, Width: 16, Height: 16},
			{X: 5, Y: 5, Width: 24, Height: 40}, {X: 0.3, Y: 0.7, Width: 13, Height: 13},
		} {
			for _, tint := range tints {
				for pass := 0; pass < 2; pass++ { // the second is a cache hit
					cached.Reset()
					direct.Reset()
					cached.DrawIcon(icon, bounds, tint)
					direct.drawIconDirect(icon, bounds, tint)
					sameTriangles(t, cached.Triangles(), direct.Triangles())
				}
			}
		}
	}
}

func TestIconCacheIsBounded(t *testing.T) {
	var canvas Canvas
	for size := 8; size < 8+3*iconCacheLimit; size++ {
		canvas.DrawIcon(testStrokeIcon, Rect{Width: float32(size), Height: float32(size)}, Color{Alpha: 1})
	}
	if got := len(canvas.icons().entries); got > iconCacheLimit {
		t.Fatalf("the icon cache holds %d entries, limit %d", got, iconCacheLimit)
	}
}

func BenchmarkDrawIconCached(b *testing.B) {
	var canvas Canvas
	bounds := Rect{X: 40, Y: 40, Width: 16, Height: 16}
	b.ReportAllocs()
	for b.Loop() {
		canvas.Reset()
		canvas.DrawIcon(testStrokeIcon, bounds, Color{Alpha: 1})
	}
}

func BenchmarkDrawIconDirect(b *testing.B) {
	var canvas Canvas
	bounds := Rect{X: 40, Y: 40, Width: 16, Height: 16}
	b.ReportAllocs()
	for b.Loop() {
		canvas.Reset()
		canvas.drawIconDirect(testStrokeIcon, bounds, Color{Alpha: 1})
	}
}

// The materialized cache is keyed by the canvas state its triangles carry, so a
// paint after a rectangle, or inside a clip, must not reuse the plain one.
func TestIconPaintCacheKeepsCanvasState(t *testing.T) {
	var canvas Canvas
	bounds := Rect{X: 20, Y: 20, Width: 16, Height: 16}
	tint := Color{Red: 0.2, Green: 0.4, Blue: 0.6, Alpha: 1}

	canvas.Fill(Rect{Width: 40, Height: 40}, Color{Alpha: 1})
	canvas.DrawIcon(testStrokeIcon, bounds, tint)
	withRect := append([]Triangle(nil), canvas.Triangles()...)
	if len(withRect) == 0 || withRect[0].RectsBefore != 1 {
		t.Fatalf("the icon after a rectangle carries RectsBefore %d", withRect[0].RectsBefore)
	}
	canvas.Reset()
	canvas.DrawIcon(testStrokeIcon, bounds, tint)
	plain := canvas.Triangles()
	if len(plain) != len(withRect) || plain[0].RectsBefore != 0 {
		t.Fatalf("the plain icon reused RectsBefore %d", plain[0].RectsBefore)
	}
	canvas.Reset()
	canvas.Clip(Rect{Width: 30, Height: 30})
	canvas.DrawIcon(testStrokeIcon, bounds, tint)
	if !canvas.Triangles()[0].HasClip {
		t.Fatal("the clipped icon reused an unclipped cache entry")
	}
}

func TestIconPaintCacheIsBounded(t *testing.T) {
	var canvas Canvas
	for size := 8; size < 8+3*iconPaintCacheLimit; size++ {
		canvas.Reset()
		canvas.DrawIcon(testStrokeIcon, Rect{Width: float32(size), Height: float32(size)}, Color{Alpha: 1})
	}
	if got := len(canvas.iconPaints().entries); got > iconPaintCacheLimit {
		t.Fatalf("the paint cache holds %d entries, limit %d", got, iconPaintCacheLimit)
	}
}
