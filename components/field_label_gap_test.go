package components

import (
	"math"
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestFieldLabelGapIsOptIn(t *testing.T) {
	for _, density := range []ui.Density{ui.DensityRegular, ui.DensityCompact} {
		theme := ui.LightTheme()
		theme.Metrics = theme.Metrics.WithDensity(density)
		var c ui.Canvas
		bounds := ui.Rect{X: 10, Y: 20, Width: 200, Height: 100}
		old := BeginField(&c, theme, bounds).Paint("Name", "", "")
		for _, gap := range []float32{0, -1, float32(math.NaN()), float32(math.Inf(1)), 8} {
			got := BeginField(&c, theme, bounds).LabelGap(gap).Paint("Name", "", "")
			want := old.Y
			if gap == 8 {
				want += 8 - fieldGap(theme.Metrics)
			}
			if got.Y != want || got.Height != old.Height {
				t.Fatal(gap, old, got)
			}
		}
	}
}
