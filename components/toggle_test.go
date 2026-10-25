package components

import (
	"slices"
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestToggleActiveUsesAccentAndDeclaresHit(t *testing.T) {
	theme := ui.LightTheme()
	var canvas ui.Canvas
	BeginToggle(&canvas, theme, ui.Item(41), ui.Rect{Width: 96}, true).Default("Bold")
	if len(canvas.Rectangles()) != 1 || canvas.Rectangles()[0].Color != theme.Muted {
		t.Fatalf("active toggle rectangles = %+v", canvas.Rectangles())
	}
	if len(canvas.Hits()) != 1 || canvas.Hits()[0].ID != ui.Item(41) || canvas.Hits()[0].Role != ui.RoleButton || !canvas.Hits()[0].Checked || !canvas.Hits()[0].Selected {
		t.Fatalf("toggle hits = %+v", canvas.Hits())
	}
	if canvas.Texts()[0].Style.Color != theme.Foreground {
		t.Fatalf("active foreground = %+v", canvas.Texts()[0].Style.Color)
	}
}

func TestToggleOutlineDisabledStableAndAllocationFree(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	paint := func() {
		canvas.Reset()
		BeginToggle(&canvas, theme, ui.Item(42), ui.Rect{Width: 88}, false).Disabled().Outline("Italic")
	}
	paint()
	rectangles := slices.Clone(canvas.Rectangles())
	texts := slices.Clone(canvas.Texts())
	paint()
	if len(canvas.Hits()) != 1 || !canvas.Hits()[0].Disabled || canvas.Hits()[0].Focusable || len(canvas.Rectangles()) != 2 {
		t.Fatalf("disabled outline toggle = %+v hits, %d rectangles", canvas.Hits(), len(canvas.Rectangles()))
	}
	if !slices.Equal(rectangles, canvas.Rectangles()) || !slices.Equal(texts, canvas.Texts()) {
		t.Fatal("identical toggles produced different command streams")
	}
	if allocations := testing.AllocsPerRun(100, paint); allocations != 0 {
		t.Fatalf("toggle paint allocations = %v", allocations)
	}
}

func TestToggleSizeAndInvalidVariants(t *testing.T) {
	var small ui.Canvas
	BeginToggle(&small, ui.LightTheme(), ui.Item(1), ui.Rect{Width: 80}, false).Small().Default("S")
	if small.Rectangles()[0].Bounds.Height != 28 {
		t.Fatalf("small toggle height = %v", small.Rectangles()[0].Bounds.Height)
	}
	var invalid ui.Canvas
	BeginToggle(&invalid, ui.LightTheme(), ui.Item(2), ui.Rect{Width: 80}, false).Invalid().Default("Invalid")
	if len(invalid.Rectangles()) != 3 || invalid.Rectangles()[1].Color != ui.LightTheme().Danger {
		t.Fatalf("invalid toggle rectangles = %+v", invalid.Rectangles())
	}
}

func TestToggleClickedReportsTheClick(t *testing.T) {
	var canvas ui.Canvas
	canvas.BeginEventPass(ui.Item(41))
	defer canvas.EndEventPass()
	if !BeginToggle(&canvas, ui.LightTheme(), ui.Item(41), ui.Rect{Width: 96}, false).Default("Bold").Clicked() {
		t.Fatal("toggle did not report its own click")
	}
	if BeginToggle(&canvas, ui.LightTheme(), ui.Item(42), ui.Rect{Width: 96}, false).Outline("Italic").Clicked() {
		t.Fatal("toggle reported a click on another control")
	}
}

func TestToggleSizeToFitWrapsTheLabelButStaysAsWideAsTall(t *testing.T) {
	theme := ui.LightTheme()
	var canvas ui.Canvas
	canvas.SetMeasurer(func(value string, size, weight float32) float32 { return float32(len(value)) * 8 })
	wide := BeginToggle(&canvas, theme, ui.Item(1), ui.Rect{}, false).SizeToFit("Underline")
	narrow := BeginToggle(&canvas, theme, ui.Item(2), ui.Rect{}, false).SizeToFit("B")
	if wide.BoundsWidth() != 9*8+20 || narrow.BoundsWidth() != 32 {
		t.Fatalf("fitted widths = %v and %v", wide.BoundsWidth(), narrow.BoundsWidth())
	}
}
