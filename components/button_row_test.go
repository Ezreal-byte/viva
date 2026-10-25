package components

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

func rowHits(canvas *ui.Canvas, ids ...uint32) []ui.Hit { return actionHits(canvas, ids...) }

// A row is not tied to a dialog: a card's footer is any rectangle, and the
// buttons are centered in it however tall it is.
func TestButtonRowSitsAtTheRightEdgeOfAnyAreaAndCentersVertically(t *testing.T) {
	theme := ui.LightTheme()
	var canvas ui.Canvas
	area := ui.Rect{X: 40, Y: 300, Width: 420, Height: 60}
	BeginButtonRow(&canvas, theme, area).Paint(
		ButtonAction{ID: ui.Item(1), Label: "Show log", Variant: ButtonGhost},
		ButtonAction{ID: ui.Item(2), Label: "Close", Variant: ButtonOutline},
		ButtonAction{ID: ui.Item(3), Label: "Try again"},
	)
	hits := rowHits(&canvas, 1, 2, 3)
	if len(hits) != 3 {
		t.Fatalf("hits = %+v", hits)
	}
	last := hits[2].Bounds
	if last.X+last.Width != area.X+area.Width {
		t.Errorf("the last button ends at %v, want %v", last.X+last.Width, area.X+area.Width)
	}
	for index, hit := range hits {
		if index > 0 {
			previous := hits[index-1].Bounds
			if gap := hit.Bounds.X - (previous.X + previous.Width); gap != theme.Metrics.GapItem {
				t.Errorf("gap before button %d = %v, want %v", index, gap, theme.Metrics.GapItem)
			}
		}
		if top, bottom := hit.Bounds.Y-area.Y, area.Y+area.Height-(hit.Bounds.Y+hit.Bounds.Height); top != bottom {
			t.Errorf("button %d sits %v from the top and %v from the bottom of its area", index, top, bottom)
		}
		if hit.Bounds.Height >= area.Height {
			t.Errorf("button %d is %v high in a %v area: it was stretched to fill it", index, hit.Bounds.Height, area.Height)
		}
	}
}

func TestButtonRowShrinksTogetherAndNeverLeavesItsArea(t *testing.T) {
	var canvas ui.Canvas
	area := ui.Rect{X: 10, Y: 10, Width: 120, Height: 40}
	BeginButtonRow(&canvas, ui.DarkTheme(), area).Paint(
		ButtonAction{ID: ui.Item(1), Label: "A rather long label"}, ButtonAction{ID: ui.Item(2), Label: "Another long label"})
	hits := rowHits(&canvas, 1, 2)
	if len(hits) != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].Bounds.X < area.X-0.01 || hits[1].Bounds.X+hits[1].Bounds.Width > area.X+area.Width+0.01 {
		t.Errorf("buttons %+v %+v leave the area %+v", hits[0].Bounds, hits[1].Bounds, area)
	}
}

func TestButtonRowDisabledAndEmptyCases(t *testing.T) {
	var canvas ui.Canvas
	BeginButtonRow(&canvas, ui.LightTheme(), ui.Rect{Width: 300, Height: 40}).Disabled().Paint(
		ButtonAction{ID: ui.Item(1), Label: "One"}, ButtonAction{ID: ui.Item(2), Label: "Two"})
	for _, hit := range rowHits(&canvas, 1, 2) {
		if !hit.Disabled {
			t.Errorf("a disabled row left %+v live", hit)
		}
	}
	var none ui.Canvas
	BeginButtonRow(&none, ui.LightTheme(), ui.Rect{Width: 300, Height: 40}).Paint()
	BeginButtonRow(&none, ui.LightTheme(), ui.Rect{}).Paint(ButtonAction{ID: ui.Item(1), Label: "x"})
	BeginButtonRow(nil, ui.LightTheme(), ui.Rect{Width: 300, Height: 40}).Paint(ButtonAction{ID: ui.Item(1), Label: "x"})
	if len(none.Hits()) != 0 {
		t.Errorf("empty rows painted %d hits", len(none.Hits()))
	}
}

func TestButtonRowAllocatesNothingOncePainted(t *testing.T) {
	var canvas ui.Canvas
	theme := ui.LightTheme()
	area := ui.Rect{Width: 420, Height: 60}
	paint := func() {
		canvas.Reset()
		BeginButtonRow(&canvas, theme, area).Paint(
			ButtonAction{ID: ui.Item(1), Label: "Show log", Variant: ButtonGhost}, ButtonAction{ID: ui.Item(2), Label: "Close", Variant: ButtonOutline})
	}
	paint()
	if allocations := testing.AllocsPerRun(100, paint); allocations != 0 {
		t.Fatalf("painting a button row allocated %v times", allocations)
	}
}

// A bar shorter than a footer takes a smaller button, still centered and still
// laid out against its right edge.
func TestButtonRowSizeFitsShorterBars(t *testing.T) {
	theme := ui.DarkTheme()
	bar := ui.Rect{X: 20, Y: 100, Width: 200, Height: 26}
	var small, xsmall ui.Canvas
	BeginButtonRow(&small, theme, bar).Paint(ButtonAction{ID: ui.Item(1), Label: "Show progress", Variant: ButtonGhost})
	BeginButtonRow(&xsmall, theme, bar).Size(ButtonSizeXSmall).Paint(ButtonAction{ID: ui.Item(1), Label: "Show progress", Variant: ButtonGhost})
	big, little := rowHits(&small, 1)[0].Bounds, rowHits(&xsmall, 1)[0].Bounds
	if big.Height <= bar.Height {
		t.Fatalf("the default row's button is %v high: this test needs a bar it does not fit", big.Height)
	}
	if little.Height >= bar.Height || little.Height >= big.Height {
		t.Fatalf("an XSmall button is %v high in a %v bar (default %v)", little.Height, bar.Height, big.Height)
	}
	if top, bottom := little.Y-bar.Y, bar.Y+bar.Height-(little.Y+little.Height); top != bottom {
		t.Errorf("the XSmall button sits %v from the top and %v from the bottom", top, bottom)
	}
	if little.X+little.Width != bar.X+bar.Width {
		t.Errorf("the XSmall button ends at %v, want the bar's right edge %v", little.X+little.Width, bar.X+bar.Width)
	}
}
