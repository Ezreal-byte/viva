package gallery

import "github.com/vivaui/viva/ui"

func paintMark(canvas *ui.Canvas, theme ui.Theme, bounds ui.Rect) {
	// Two cells and the gap between them span the full bounds.
	unit := bounds.Width / 2.3
	gap := unit * 0.3
	canvas.Rounded(ui.Rect{X: bounds.X, Y: bounds.Y, Width: unit, Height: unit}, 2, theme.Foreground)
	canvas.Rounded(ui.Rect{X: bounds.X + unit + gap, Y: bounds.Y, Width: unit, Height: unit}, 2, theme.Foreground)
	canvas.Rounded(ui.Rect{X: bounds.X, Y: bounds.Y + unit + gap, Width: unit, Height: unit}, 2, theme.Foreground)
	canvas.Rounded(ui.Rect{X: bounds.X + unit + gap, Y: bounds.Y + unit + gap, Width: unit, Height: unit}, unit/2, theme.Primary)
}

// Lucide chevron-left and chevron-right, hand-compiled to the ui.Icon form.
var (
	iconChevronLeft = ui.Icon{
		ViewBox: 24, Stroke: 2, Cap: ui.CapRound, Join: ui.JoinRound,
		Commands: []ui.PathCommand{ui.MoveTo(15, 18), ui.LineTo(9, 12), ui.LineTo(15, 6)},
	}
	iconChevronRight = ui.Icon{
		ViewBox: 24, Stroke: 2, Cap: ui.CapRound, Join: ui.JoinRound,
		Commands: []ui.PathCommand{ui.MoveTo(9, 18), ui.LineTo(15, 12), ui.LineTo(9, 6)},
	}
)

// paintIconButton paints a ghost icon button: the hover surface, the icon
// centered exactly in its bounds, and the labelled hit region. A text glyph
// would sit wherever the font puts it; an icon is centered by construction. It
// reports whether this paint is delivering a click on the button.
func paintIconButton(canvas *ui.Canvas, theme ui.Theme, id ui.ID, bounds ui.Rect, icon ui.Icon, label string) bool {
	if canvas.Hovered() == id || canvas.Pressed() == id {
		canvas.Rounded(bounds, theme.Metrics.RadiusControl, theme.Accent)
	}
	const size = float32(16)
	canvas.DrawIcon(icon, ui.Rect{
		X: bounds.X + (bounds.Width-size)/2, Y: bounds.Y + (bounds.Height-size)/2, Width: size, Height: size,
	}, theme.Foreground)
	canvas.HitWithOptions(id, bounds, label, ui.HitOptions{Role: ui.RoleButton})
	return canvas.Clicked(id)
}
