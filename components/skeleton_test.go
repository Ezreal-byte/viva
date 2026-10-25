package components

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestSkeletonPaintsStaticNeutralSurface(t *testing.T) {
	var canvas ui.Canvas
	theme := ui.LightTheme()
	BeginSkeleton(&canvas, theme, ui.Rect{X: 4, Y: 6, Width: 180, Height: 14}).Paint()
	if len(canvas.Rectangles()) != 1 || len(canvas.Hits()) != 0 {
		t.Fatalf("skeleton commands = %d rectangles, %d hits", len(canvas.Rectangles()), len(canvas.Hits()))
	}
	if canvas.Rectangles()[0].Radius != theme.Metrics.RadiusControl || canvas.Rectangles()[0].Color != theme.Muted {
		t.Fatalf("skeleton radius = %v", canvas.Rectangles()[0].Radius)
	}
}

func TestSkeletonCircleAndPaintAllocateNothingAfterWarmup(t *testing.T) {
	var canvas ui.Canvas
	theme := ui.DarkTheme()
	paint := func() {
		canvas.Reset()
		BeginSkeleton(&canvas, theme, ui.Rect{Width: 40, Height: 56}).Circle().Paint()
	}
	paint()
	if bounds := canvas.Rectangles()[0].Bounds; bounds.Width != 40 || bounds.Height != 40 || canvas.Rectangles()[0].Radius != 20 {
		t.Fatalf("circle skeleton = %+v", canvas.Rectangles()[0])
	}
	if allocations := testing.AllocsPerRun(100, paint); allocations != 0 {
		t.Fatalf("skeleton paint allocations = %v", allocations)
	}
}

func TestSkeletonPulseIsExplicitPhaseOnly(t *testing.T) {
	var low ui.Canvas
	BeginSkeleton(&low, ui.LightTheme(), ui.Rect{Width: 100, Height: 12}).Phase(0).Paint()
	var high ui.Canvas
	BeginSkeleton(&high, ui.LightTheme(), ui.Rect{Width: 100, Height: 12}).Phase(0.5).Paint()
	if low.Rectangles()[0].Color.Alpha <= high.Rectangles()[0].Color.Alpha {
		t.Fatalf("pulse alpha did not fall: low=%v high=%v", low.Rectangles()[0].Color.Alpha, high.Rectangles()[0].Color.Alpha)
	}
}
