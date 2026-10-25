package components

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestButtonOfficialVariants(t *testing.T) {
	theme := ui.LightTheme()
	tests := [...]struct {
		variant    ButtonVariant
		background ui.Color
		foreground ui.Color
		rectangles int
	}{
		{ButtonDefault, theme.Primary, theme.PrimaryForeground, 1},
		{ButtonDestructive, withAlpha(theme.Danger, 0.10), theme.Danger, 1},
		{ButtonSecondary, theme.Secondary, theme.SecondaryForeground, 1},
		{ButtonOutline, theme.Background, theme.Foreground, 2},
		{ButtonGhost, ui.Color{}, theme.Foreground, 0},
		{ButtonLink, ui.Color{}, theme.Primary, 0},
		{ButtonDestructiveOutline, theme.Background, theme.Danger, 2},
	}
	for _, test := range tests {
		var canvas ui.Canvas
		button := BeginButton(&canvas, theme, ui.Item(1), ui.Rect{Width: 120})
		button.paint("Action", test.variant)
		if len(canvas.Rectangles()) != test.rectangles {
			t.Fatalf("variant %d painted %d rectangles, want %d", test.variant, len(canvas.Rectangles()), test.rectangles)
		}
		if test.rectangles != 0 {
			fillIndex := len(canvas.Rectangles()) - 1
			if got := canvas.Rectangles()[fillIndex].Color; got != test.background {
				t.Fatalf("variant %d fill = %+v, want %+v", test.variant, got, test.background)
			}
		}
		if got := canvas.Texts()[0].Style.Color; got != test.foreground {
			t.Fatalf("variant %d foreground = %+v, want %+v", test.variant, got, test.foreground)
		}
	}
}

func TestButtonOfficialSizes(t *testing.T) {
	theme := ui.DarkTheme()
	tests := [...]struct {
		size   ButtonSize
		height float32
		icon   bool
	}{
		{ButtonSizeDefault, 32, false},
		{ButtonSizeXSmall, 24, false},
		{ButtonSizeSmall, 28, false},
		{ButtonSizeLarge, 36, false},
		{ButtonSizeIcon, 32, true},
		{ButtonSizeIconXSmall, 24, true},
		{ButtonSizeIconSmall, 28, true},
		{ButtonSizeIconLarge, 36, true},
	}
	for _, test := range tests {
		var canvas ui.Canvas
		BeginButton(&canvas, theme, ui.ID{}, ui.Rect{Width: 100}).Size(test.size).Default("A")
		bounds := canvas.Rectangles()[0].Bounds
		if bounds.Height != test.height {
			t.Fatalf("size %d height = %v, want %v", test.size, bounds.Height, test.height)
		}
		if test.icon && bounds.Width != test.height {
			t.Fatalf("icon size %d width = %v, want %v", test.size, bounds.Width, test.height)
		}
	}
}

func TestButtonTypedHitAndZeroIDPreview(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	BeginButton(&canvas, theme, ui.Item(7), ui.Rect{Width: 100}).Default("Save")
	BeginButton(&canvas, theme, ui.ID{}, ui.Rect{X: 120, Width: 100}).Default("Preview")
	if len(canvas.Hits()) != 1 {
		t.Fatalf("buttons declared %d hits, want only live declaration", len(canvas.Hits()))
	}
	hit := canvas.Hits()[0]
	if hit.ID != ui.Item(7) || hit.Label != "Save" || hit.Role != ui.RoleButton || !hit.Focusable || hit.Disabled {
		t.Fatalf("typed button hit = %+v", hit)
	}
}

func TestButtonDisabledKeepsTypedSemanticsAtHalfOpacity(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	BeginButton(&canvas, theme, ui.Item(9), ui.Rect{Width: 100}).Disabled().Default("Unavailable")
	if len(canvas.Hits()) != 1 || !canvas.Hits()[0].Disabled || canvas.Hits()[0].Focusable {
		t.Fatalf("disabled hit = %+v", canvas.Hits())
	}
	want := theme.Primary
	want.Alpha *= 0.50
	if got := canvas.Rectangles()[0].Color; got != want {
		t.Fatalf("disabled fill = %+v, want %+v", got, want)
	}
}

func TestButtonFocusAndInvalidRingsUseOfficialTokensInDark(t *testing.T) {
	theme := ui.DarkTheme()
	var focused, invalid ui.Canvas
	focused.SetInteraction(ui.ID{}, ui.ID{}, ui.Item(1), true)
	BeginButton(&focused, theme, ui.Item(1), ui.Rect{X: 4, Y: 4, Width: 100}).Default("Save")
	BeginButton(&invalid, theme, ui.Item(2), ui.Rect{X: 4, Y: 4, Width: 100}).Invalid().Outline("Save")

	focusRing := focused.Rectangles()[0]
	wantFocus := theme.Ring
	wantFocus.Alpha *= 0.50
	if focusRing.Color != wantFocus || focusRing.Bounds != (ui.Rect{X: 1, Y: 1, Width: 106, Height: 38}) {
		t.Fatalf("focus ring = %+v, want Ring/50 expanded 3px", focusRing)
	}
	invalidRing := invalid.Rectangles()[0]
	wantInvalid := withAlpha(theme.Danger, 0.40)
	wantBorder := invalidBorder(theme).Over(theme.Background)
	if invalidRing.Color != wantInvalid || invalid.Rectangles()[1].Color != wantBorder {
		t.Fatalf("invalid ring/border = %+v / %+v", invalidRing, invalid.Rectangles()[1])
	}
}

func TestButtonSizeToFitUsesSelectedPadding(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	defaultButton := BeginButton(&canvas, theme, ui.ID{}, ui.Rect{}).SizeToFit("Download")
	xsmallButton := BeginButton(&canvas, theme, ui.ID{}, ui.Rect{}).XSmall().SizeToFit("Download")
	wantDefault := canvas.TextWidthWeight("Download", theme.Metrics.FontSizeM, ui.WeightMedium) + 20
	wantXSmall := canvas.TextWidthWeight("Download", theme.Metrics.FontSizeS, ui.WeightMedium) + 16
	if defaultButton.BoundsWidth() != wantDefault || xsmallButton.BoundsWidth() != wantXSmall {
		t.Fatalf("fitted widths = %v and %v, want %v and %v", defaultButton.BoundsWidth(), xsmallButton.BoundsWidth(), wantDefault, wantXSmall)
	}
}

func TestButtonCentersInItsArea(t *testing.T) {
	var canvas ui.Canvas
	area := ui.Rect{X: 100, Y: 50, Width: 300, Height: 100}
	button := BeginButton(&canvas, ui.DarkTheme(), ui.ID{}, ui.Rect{}).
		SizeToFit("Download").CenterIn(area)
	if want := area.X + (area.Width-button.bounds.Width)/2; button.bounds.X != want {
		t.Fatalf("centered X = %v, want %v", button.bounds.X, want)
	}
	if want := area.Y + (area.Height-button.bounds.Height)/2; button.bounds.Y != want {
		t.Fatalf("centered Y = %v, want %v", button.bounds.Y, want)
	}
}

func TestButtonRepeatedPaintReusesCanvasStorage(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	BeginButton(&canvas, theme, ui.Item(1), ui.Rect{Width: 100}).Default("Save")
	allocations := testing.AllocsPerRun(100, func() {
		canvas.Reset()
		BeginButton(&canvas, theme, ui.Item(1), ui.Rect{Width: 100}).Default("Save")
	})
	if allocations != 0 {
		t.Fatalf("repeated button paint allocated %v times", allocations)
	}
}
