package components

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

// weightedMeasure is a measurer whose lines grow with their weight, as a real
// font's do: 10 per rune at regular, 14 at the weight of a medium label.
func weightedMeasure(value string, size, weight float32) float32 {
	return float32(len([]rune(value))) * 10 * (1 + weight*2.6)
}

// A label is as wide as the weight it is drawn at: fitting it at the regular
// weight clips its last letters where a heavier face is wider.
func TestFittedWidthsAreMeasuredAtTheWeightTheLabelIsDrawnWith(t *testing.T) {
	theme := ui.DarkTheme()
	regular := weightedMeasure("Download", 14, ui.WeightRegular)
	medium := weightedMeasure("Download", 14, ui.WeightMedium)
	if medium <= regular {
		t.Fatal("test measurer does not grow with weight")
	}

	var canvas ui.Canvas
	canvas.SetMeasurer(weightedMeasure)

	button := BeginButton(&canvas, theme, ui.ID{}, ui.Rect{}).SizeToFit("Download")
	_, padding, _, _ := button.geometry()
	if want := medium + padding*2; button.BoundsWidth() != want {
		t.Errorf("button fits %.1f, want %.1f (its label is medium weight)", button.BoundsWidth(), want)
	}

	badge := PaintBadge(&canvas, theme, ui.Rect{Width: 800, Height: 100}, "Download", BadgeSecondary)
	canvas.Reset()
	canvas.SetMeasurer(weightedMeasure)
	if got := badge.Width; got < medium {
		t.Errorf("badge is %.1f wide, narrower than its medium label (%.1f)", got, medium)
	}
}
