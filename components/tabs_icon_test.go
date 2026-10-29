package components

import (
	"github.com/vivaui/viva/ui"
	"math"
	"testing"
)

func TestTabsIconsCenterWithLabelAndFit(t *testing.T) {
	icons := []ui.Icon{
		{ViewBox: 24, Stroke: 2, Commands: []ui.PathCommand{ui.MoveTo(2, 12), ui.LineTo(22, 12)}},
		{ViewBox: 24, Layers: []ui.IconLayer{{Fill: ui.RGB(200, 0, 0), Commands: []ui.PathCommand{ui.MoveTo(2, 2), ui.LineTo(22, 2), ui.LineTo(12, 22), ui.ClosePath()}}}},
	}
	for _, icon := range icons {
		for _, width := range []float32{240, 300, 600} {
			var canvas ui.Canvas
			theme := ui.LightTheme()
			theme.Metrics.FontSizeM = 12
			canvas.SetMeasurer(chartTestMeasure)
			options := []TabOption{{ID: ui.Item(1), Label: "Overview", Icon: icon}, {ID: ui.Item(2), Label: "Disk", Icon: icon}, {ID: ui.Item(3), Label: "Network", Icon: icon}}
			paint := func() {
				canvas.Reset()
				BeginTabs(&canvas, theme, ui.Rect{X: 10, Y: 20, Width: width, Height: 36}, options, ui.Item(1)).Paint()
			}
			paint()
			paint()
			for i, text := range canvas.Texts() {
				trigger := canvas.Hits()[i].Bounds
				size, gap := float32(12), float32(6)
				groupLeft := text.Bounds.X - size - gap
				center := (groupLeft + text.Bounds.X + text.Bounds.Width) / 2
				if math.Abs(float64(center-(trigger.X+trigger.Width/2))) > .001 || text.Bounds.Y != trigger.Y || text.Bounds.Height != trigger.Height {
					t.Fatalf("tab %d not centered: text=%+v trigger=%+v", i, text.Bounds, trigger)
				}
			}
			if len(canvas.Triangles()) == 0 {
				t.Fatal("tab icon not painted")
			}
			if allocations := testing.AllocsPerRun(30, paint); allocations != 0 {
				t.Fatalf("icon tabs allocations=%v", allocations)
			}
			canvas.Reset()
			BeginTabs(&canvas, theme, ui.Rect{}, options, ui.Item(1)).Fit().Paint()
			want := canvas.TextWidthWeight(options[0].Label, 12, ui.WeightMedium) + 12 + 6 + tabsTriggerPad*2
			if got := canvas.Hits()[0].Bounds.Width; got != want {
				t.Fatalf("Fit icon width=%v want%v", got, want)
			}
		}
	}
}
