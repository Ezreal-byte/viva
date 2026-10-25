package web

import (
	"math"
	"testing"

	"github.com/vivaui/viva/ui"
)

// at evaluates the gradient shadeAxis describes at p, as Canvas 2D would.
func gradientAt(start, end ui.Point, from, to ui.Color, p ui.Point) ui.Color {
	dx, dy := end.X-start.X, end.Y-start.Y
	t := ((p.X-start.X)*dx + (p.Y-start.Y)*dy) / (dx*dx + dy*dy)
	t = min(max(t, 0), 1)
	return ui.Color{
		Red: from.Red + (to.Red-from.Red)*t, Green: from.Green + (to.Green-from.Green)*t,
		Blue: from.Blue + (to.Blue-from.Blue)*t, Alpha: from.Alpha + (to.Alpha-from.Alpha)*t,
	}
}

// The canvas's gradients have corner colors on one line, and then Canvas 2D's
// linear gradient reproduces the interpolation a GPU does, at every point.
func TestShadeAxisReproducesCornerColorInterpolation(t *testing.T) {
	red, blue := ui.Color{Red: 1, Alpha: 1}, ui.Color{Blue: 1, Alpha: 0.4}
	mix := func(u float32) ui.Color {
		return ui.Color{Red: red.Red + (blue.Red-red.Red)*u, Blue: red.Blue + (blue.Blue-red.Blue)*u, Alpha: red.Alpha + (blue.Alpha-red.Alpha)*u}
	}
	points := [3]ui.Point{{X: 10, Y: 10}, {X: 90, Y: 30}, {X: 40, Y: 80}}
	for name, colors := range map[string][3]ui.Color{
		"third farthest":  {red, mix(0.3), blue},
		"second farthest": {mix(0.2), blue, red},
		"first in middle": {mix(0.5), red, blue},
	} {
		start, end, from, to, ok := shadeAxis(points, colors)
		if !ok {
			t.Fatalf("%s: no gradient", name)
		}
		for _, w := range [][2]float32{{0, 0}, {1, 0}, {0, 1}, {0.25, 0.25}, {0.6, 0.3}, {0.1, 0.8}} {
			p := ui.Point{X: points[0].X + w[0]*(points[1].X-points[0].X) + w[1]*(points[2].X-points[0].X),
				Y: points[0].Y + w[0]*(points[1].Y-points[0].Y) + w[1]*(points[2].Y-points[0].Y)}
			want := ui.Color{}
			for index, weight := range [3]float32{1 - w[0] - w[1], w[0], w[1]} {
				want.Red += colors[index].Red * weight
				want.Green += colors[index].Green * weight
				want.Blue += colors[index].Blue * weight
				want.Alpha += colors[index].Alpha * weight
			}
			got := gradientAt(start, end, from, to, p)
			if math.Abs(float64(got.Red-want.Red)) > 1e-4 || math.Abs(float64(got.Blue-want.Blue)) > 1e-4 || math.Abs(float64(got.Alpha-want.Alpha)) > 1e-4 {
				t.Fatalf("%s: at %v gradient is %+v, interpolation %+v", name, p, got, want)
			}
		}
	}
}

func TestShadeAxisLeavesFlatAndDegenerateTrianglesToTheirColor(t *testing.T) {
	c := ui.Color{Red: 1, Alpha: 1}
	if _, _, _, _, ok := shadeAxis([3]ui.Point{{}, {X: 5}, {Y: 5}}, [3]ui.Color{c, c, c}); ok {
		t.Fatal("a flat triangle got a gradient")
	}
	line := [3]ui.Point{{}, {X: 5}, {X: 9}}
	if _, _, _, _, ok := shadeAxis(line, [3]ui.Color{c, {Blue: 1, Alpha: 1}, c}); ok {
		t.Fatal("a collinear triangle got a gradient")
	}
}
