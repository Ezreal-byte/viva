package componentdemo

import "github.com/vivaui/viva/ui"

// The demos draw a few Lucide-style icons of their own; Viva ships no icon set.

func lucide(commands ...ui.PathCommand) ui.Icon {
	return ui.Icon{ViewBox: 24, Stroke: 2, Cap: ui.CapRound, Join: ui.JoinRound, Commands: commands}
}

func circle(cx, cy, r float32) []ui.PathCommand {
	const k = 0.5523
	return []ui.PathCommand{
		ui.MoveTo(cx+r, cy),
		ui.CubicTo(cx+r, cy+r*k, cx+r*k, cy+r, cx, cy+r),
		ui.CubicTo(cx-r*k, cy+r, cx-r, cy+r*k, cx-r, cy),
		ui.CubicTo(cx-r, cy-r*k, cx-r*k, cy-r, cx, cy-r),
		ui.CubicTo(cx+r*k, cy-r, cx+r, cy-r*k, cx+r, cy),
		ui.ClosePath(),
	}
}

var (
	// infoIcon is a circled "i".
	infoIcon = lucide(append(circle(12, 12, 10),
		ui.MoveTo(12, 16), ui.LineTo(12, 12), ui.MoveTo(12, 8), ui.LineTo(12.01, 8))...)
	// alertIcon is a circled "!".
	alertIcon = lucide(append(circle(12, 12, 10),
		ui.MoveTo(12, 8), ui.LineTo(12, 12), ui.MoveTo(12, 16), ui.LineTo(12.01, 16))...)
	// folderIcon is an outlined folder.
	folderIcon = lucide(
		ui.MoveTo(4, 20), ui.LineTo(20, 20), ui.CubicTo(21.1, 20, 22, 19.1, 22, 18), ui.LineTo(22, 8),
		ui.CubicTo(22, 6.9, 21.1, 6, 20, 6), ui.LineTo(12.1, 6), ui.LineTo(10.5, 3.9),
		ui.CubicTo(10.1, 3.4, 9.6, 3.1, 9, 3.1), ui.LineTo(4, 3.1),
		ui.CubicTo(2.9, 3.1, 2, 4, 2, 5.1), ui.LineTo(2, 18), ui.CubicTo(2, 19.1, 2.9, 20, 4, 20), ui.ClosePath())
)

// moreIcon is three dots.
var moreIcon = lucide(
	ui.MoveTo(5, 12), ui.LineTo(5.01, 12), ui.MoveTo(12, 12), ui.LineTo(12.01, 12), ui.MoveTo(19, 12), ui.LineTo(19.01, 12))
