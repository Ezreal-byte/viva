package components

import "github.com/vivaui/viva/ui"

const (
	calendarColumnCount = 7
	calendarPadding     = float32(8)
	calendarSectionGap  = float32(16)
	calendarWeekGap     = float32(8)
)

// CalendarDay is one application-owned cell in display order. Label is the
// visible day text; AccessibilityLabel should contain the complete localized
// date when Label alone would be ambiguous. Date arithmetic and selection
// changes remain application responsibilities.
type CalendarDay struct {
	ID                 ui.ID
	Label              string
	AccessibilityLabel string
	Value              string
	Hint               string
	Selected           bool
	RangeStart         bool
	RangeMiddle        bool
	RangeEnd           bool
	Today              bool
	Outside            bool
	Disabled           bool
}

// CalendarOptions contains the complete application-owned month snapshot.
// Weekdays are always painted as seven columns and Days are consumed in display
// order without allocating or deriving dates during Paint.
type CalendarOptions struct {
	MonthTitle       string
	Weekdays         [calendarColumnCount]string
	Days             []CalendarDay
	PreviousID       ui.ID
	NextID           ui.ID
	PreviousLabel    string
	NextLabel        string
	PreviousDisabled bool
	NextDisabled     bool
}

// Calendar paints one source-owned month grid. It deliberately has no hidden
// navigation, locale, or date model; callers replace CalendarOptions when the
// visible month changes.
//
// It paints no background of its own, like shadcn's Calendar inside a popover
// or card: the surface it sits on (the page, a Popover, a Card) shows through,
// so it never becomes a darker rectangle on a raised surface.
type Calendar struct {
	canvas    *ui.Canvas
	theme     ui.Theme
	bounds    ui.Rect
	options   CalendarOptions
	cellSize  float32
	contentX  float32
	captionY  float32
	weekdaysY float32
	daysY     float32
	// tabStop is the one day Tab reaches; arrows move among the others.
	tabStop ui.ID
}

// BeginCalendar prepares a deterministic seven-column month grid.
func BeginCalendar(canvas *ui.Canvas, theme ui.Theme, bounds ui.Rect, options CalendarOptions) Calendar {
	cellSize := theme.Metrics.ControlHeight - 4
	if cellSize < 0 {
		cellSize = 0
	}
	availableWidth := bounds.Width - calendarPadding*2
	if availableWidth < cellSize*calendarColumnCount {
		cellSize = availableWidth / calendarColumnCount
		if cellSize < 0 {
			cellSize = 0
		}
	}
	rows := (len(options.Days) + calendarColumnCount - 1) / calendarColumnCount
	contentHeight := calendarPadding*2 + cellSize + calendarSectionGap + cellSize + float32(rows)*(calendarWeekGap+cellSize)
	if bounds.Height <= 0 {
		bounds.Height = contentHeight
	}
	captionY := bounds.Y + calendarPadding
	weekdaysY := captionY + cellSize + calendarSectionGap
	calendar := Calendar{
		canvas: canvas, theme: theme, bounds: bounds, options: options,
		cellSize: cellSize, contentX: bounds.X + calendarPadding,
		captionY: captionY, weekdaysY: weekdaysY,
		daysY: weekdaysY + cellSize + calendarWeekGap,
	}
	calendar.tabStop = calendar.rovingTabStop()
	return calendar
}

// rovingTabStop keeps one day in the Tab order: the focused day, else a
// selected or range day, else the first enabled day.
func (calendar Calendar) rovingTabStop() ui.ID {
	enabled := func(day CalendarDay) bool {
		return !day.Disabled && !day.ID.IsZero()
	}
	focus := ui.ID{}
	if calendar.canvas != nil {
		focus = calendar.canvas.FocusVisible()
	}
	for _, day := range calendar.options.Days {
		if day.ID == focus && enabled(day) {
			return focus
		}
	}
	for _, day := range calendar.options.Days {
		if (day.Selected || day.RangeStart || day.RangeMiddle || day.RangeEnd) && enabled(day) {
			return day.ID
		}
	}
	for _, day := range calendar.options.Days {
		if enabled(day) {
			return day.ID
		}
	}
	return ui.ID{}
}

// Bounds returns the normalized outer geometry. A missing height is derived
// from the number of caller-provided day cells.
func (calendar Calendar) Bounds() ui.Rect {
	return calendar.bounds
}

// DayBounds returns the deterministic geometry for a display-order day index.
func (calendar Calendar) DayBounds(index int) ui.Rect {
	if index < 0 || calendar.cellSize <= 0 {
		return ui.Rect{}
	}
	column := index % calendarColumnCount
	row := index / calendarColumnCount
	return ui.Rect{
		X:     calendar.contentX + float32(column)*calendar.cellSize,
		Y:     calendar.daysY + float32(row)*(calendar.cellSize+calendarWeekGap),
		Width: calendar.cellSize, Height: calendar.cellSize,
	}
}

// Paint emits the month caption, navigation, weekday labels, and day cells.
func (calendar Calendar) Paint() {
	if calendar.canvas == nil || calendar.cellSize <= 0 || calendar.bounds.Width <= 0 || calendar.bounds.Height <= 0 {
		return
	}
	rows := (len(calendar.options.Days) + calendarColumnCount - 1) / calendarColumnCount
	caption := ui.Rect{X: calendar.contentX + calendar.cellSize, Y: calendar.captionY, Width: max(calendar.bounds.Width-2*calendarPadding-2*calendar.cellSize, 0), Height: calendar.cellSize}
	calendar.canvas.ContainerWithOptions(caption, ui.RoleStatus, calendar.options.MonthTitle, ui.HitOptions{Live: ui.LivePolite})
	grid := ui.Rect{
		X: calendar.contentX, Y: calendar.weekdaysY,
		Width:  float32(calendarColumnCount) * calendar.cellSize,
		Height: calendar.daysY + float32(rows)*(calendar.cellSize+calendarWeekGap) - calendar.weekdaysY,
	}
	calendar.canvas.Container(grid, ui.RoleGrid, calendar.options.MonthTitle)
	calendar.paintNavigation()
	for column := 0; column < calendarColumnCount; column++ {
		bounds := ui.Rect{
			X: calendar.contentX + float32(column)*calendar.cellSize,
			Y: calendar.weekdaysY, Width: calendar.cellSize, Height: calendar.cellSize,
		}
		calendar.canvas.LabelCenter(calendar.options.Weekdays[column], bounds, ui.TextStyle{
			Size: calendar.theme.Metrics.FontSizeS, Weight: ui.WeightRegular,
			Color: calendar.theme.MutedForeground,
		})
	}
	for index := range calendar.options.Days {
		calendar.paintDay(calendar.DayBounds(index), calendar.options.Days[index])
	}
}

func (calendar Calendar) paintNavigation() {
	previousLabel := calendar.options.PreviousLabel
	if previousLabel == "" {
		previousLabel = "Previous month"
	}
	nextLabel := calendar.options.NextLabel
	if nextLabel == "" {
		nextLabel = "Next month"
	}
	previousBounds := ui.Rect{X: calendar.contentX, Y: calendar.captionY, Width: calendar.cellSize, Height: calendar.cellSize}
	nextBounds := ui.Rect{
		X: calendar.contentX + calendar.cellSize*(calendarColumnCount-1),
		Y: calendar.captionY, Width: calendar.cellSize, Height: calendar.cellSize,
	}
	calendar.paintNavigationButton(previousBounds, calendar.options.PreviousID, previousLabel, &calendarPreviousIcon, calendar.options.PreviousDisabled)
	calendar.paintNavigationButton(nextBounds, calendar.options.NextID, nextLabel, &chevronRightIcon, calendar.options.NextDisabled)
	calendar.canvas.LabelCenter(calendar.options.MonthTitle, ui.Rect{
		X:     calendar.contentX + calendar.cellSize,
		Y:     calendar.captionY,
		Width: calendar.cellSize * (calendarColumnCount - 2), Height: calendar.cellSize,
	}, ui.TextStyle{Size: calendar.theme.Metrics.FontSizeM, Weight: ui.WeightMedium, Color: calendar.theme.Foreground})
}

func (calendar Calendar) paintNavigationButton(bounds ui.Rect, id ui.ID, label string, icon *ui.Icon, disabled bool) {
	enabled := !id.IsZero() && !disabled
	background := calendar.theme.Background
	background.Alpha = 0
	if enabled && (calendar.canvas.Pressed() == id || calendar.canvas.Hovered() == id) {
		background = calendar.theme.Muted
	}
	if enabled && calendar.canvas.FocusVisible() == id {
		calendar.paintFocus(bounds)
	}
	calendar.canvas.Rounded(bounds, calendar.radius(), background)
	foreground := calendar.theme.Foreground
	if disabled {
		foreground.Alpha *= 0.50
	}
	paintChevron(calendar.canvas, icon, bounds, foreground)
	if !id.IsZero() {
		calendar.canvas.HitWithOptions(id, bounds, label, ui.HitOptions{
			Role: ui.RoleButton, Disabled: disabled,
		})
	}
}

func (calendar Calendar) paintDay(bounds ui.Rect, day CalendarDay) {
	selected := day.Selected || day.RangeStart || day.RangeMiddle || day.RangeEnd
	enabled := !day.ID.IsZero() && !day.Disabled
	if enabled && calendar.canvas.FocusVisible() == day.ID {
		calendar.paintFocus(bounds)
	}

	radius := calendar.radius()
	if day.RangeStart && !day.RangeEnd {
		calendar.canvas.Fill(ui.Rect{X: bounds.X + bounds.Width/2, Y: bounds.Y, Width: bounds.Width / 2, Height: bounds.Height}, calendar.theme.Muted)
	}
	if day.RangeEnd && !day.RangeStart {
		calendar.canvas.Fill(ui.Rect{X: bounds.X, Y: bounds.Y, Width: bounds.Width / 2, Height: bounds.Height}, calendar.theme.Muted)
	}

	background := calendar.theme.Background
	background.Alpha = 0
	foreground := calendar.theme.Foreground
	cellRadius := radius
	switch {
	case day.RangeStart || day.RangeEnd:
		background = calendar.theme.Primary
		foreground = calendar.theme.PrimaryForeground
	case day.RangeMiddle:
		background = calendar.theme.Muted
		cellRadius = 0
	case day.Selected:
		background = calendar.theme.Primary
		foreground = calendar.theme.PrimaryForeground
	case day.Today:
		background = calendar.theme.Muted
	case enabled && (calendar.canvas.Pressed() == day.ID || calendar.canvas.Hovered() == day.ID):
		// A ghost icon button: bg-muted under the pointer.
		background = calendar.theme.Muted
	}
	if day.Outside {
		foreground = calendar.theme.MutedForeground
	}
	if day.Disabled {
		foreground = calendar.theme.MutedForeground
		foreground.Alpha *= 0.50
		background.Alpha *= 0.50
	}
	calendar.canvas.Rounded(bounds, cellRadius, background)
	calendar.canvas.LabelCenter(day.Label, bounds, ui.TextStyle{
		Size: calendar.theme.Metrics.FontSizeM, Weight: ui.WeightRegular, Color: foreground,
	})

	if day.ID.IsZero() {
		return
	}
	accessibilityLabel := day.AccessibilityLabel
	if accessibilityLabel == "" {
		accessibilityLabel = day.Label
	}
	calendar.canvas.HitWithOptions(day.ID, bounds, accessibilityLabel, ui.HitOptions{
		Role: ui.RoleGridCell, Value: day.Value, Hint: day.Hint,
		Disabled: day.Disabled, Selected: selected,
		SkipTab: day.ID != calendar.tabStop,
	})
}

func (calendar Calendar) paintFocus(bounds ui.Rect) {
	ring := calendar.theme.Ring
	ring.Alpha *= 0.50
	calendar.canvas.Rounded(ui.Rect{
		X: bounds.X - 3, Y: bounds.Y - 3,
		Width: bounds.Width + 6, Height: bounds.Height + 6,
	}, calendar.radius()+3, ring)
}

func (calendar Calendar) radius() float32 {
	radius := calendar.theme.Metrics.RadiusControl
	if radius > calendar.cellSize/2 {
		return calendar.cellSize / 2
	}
	if radius < 0 {
		return 0
	}
	return radius
}

// calendarPreviousIcon is Lucide's chevron-left, the previous-month arrow; the
// next-month one is the chevron-right the tree uses. Like every other mark they
// are paths, not the "‹" and "›" of a font.
var calendarPreviousIcon = ui.Icon{ViewBox: 24, Stroke: 2, Cap: ui.CapRound, Join: ui.JoinRound, Commands: []ui.PathCommand{ui.MoveTo(15, 18), ui.LineTo(9, 12), ui.LineTo(15, 6)}}

// CalendarCommandKind is the intent of a navigation key, in the vocabulary of
// react-day-picker's keyboard model.
type CalendarCommandKind uint8

const (
	CalendarCommandNone CalendarCommandKind = iota
	// CalendarCommandMoveDays moves focus by Amount days (-1 or 1).
	CalendarCommandMoveDays
	// CalendarCommandMoveWeeks moves focus by Amount weeks (-1 or 1).
	CalendarCommandMoveWeeks
	// CalendarCommandMoveMonths moves focus by Amount months (-1 or 1), PageUp
	// and PageDown.
	CalendarCommandMoveMonths
	// CalendarCommandMoveYears moves focus by Amount years (-1 or 1),
	// Shift+PageUp and Shift+PageDown.
	CalendarCommandMoveYears
	// CalendarCommandWeekStart and CalendarCommandWeekEnd move focus to the
	// first and last day of the focused week, Home and End.
	CalendarCommandWeekStart
	CalendarCommandWeekEnd
)

// CalendarCommand is a request for the date the caller owns. The component has
// no date model, so it never applies one: the caller moves its focused date,
// changes the visible month if that leaves it, and replaces CalendarOptions.
type CalendarCommand struct {
	Kind   CalendarCommandKind
	Amount int
}

// CalendarCommandForKey maps the keys of a focused day to the move they ask
// for: the arrows move by day and week, PageUp and PageDown by month (by year
// with Shift), Home and End to the edges of the week. Left and Right follow
// the reading direction only if the caller flips them for right-to-left text.
func CalendarCommandForKey(key ui.Key, modifiers ui.Modifiers) (CalendarCommand, bool) {
	switch key {
	case ui.KeyLeft:
		return CalendarCommand{Kind: CalendarCommandMoveDays, Amount: -1}, true
	case ui.KeyRight:
		return CalendarCommand{Kind: CalendarCommandMoveDays, Amount: 1}, true
	case ui.KeyUp:
		return CalendarCommand{Kind: CalendarCommandMoveWeeks, Amount: -1}, true
	case ui.KeyDown:
		return CalendarCommand{Kind: CalendarCommandMoveWeeks, Amount: 1}, true
	case ui.KeyPageUp, ui.KeyPageDown:
		amount := -1
		if key == ui.KeyPageDown {
			amount = 1
		}
		if modifiers&ui.ModifierShift != 0 {
			return CalendarCommand{Kind: CalendarCommandMoveYears, Amount: amount}, true
		}
		return CalendarCommand{Kind: CalendarCommandMoveMonths, Amount: amount}, true
	case ui.KeyHome:
		return CalendarCommand{Kind: CalendarCommandWeekStart}, true
	case ui.KeyEnd:
		return CalendarCommand{Kind: CalendarCommandWeekEnd}, true
	default:
		return CalendarCommand{}, false
	}
}

// CalendarFocusForKey moves the focused day through a month snapshot: Left and
// Right by a day (reversed in RTL), Up and Down by a week, Home and End to the
// start and end of the current week. It clamps at the snapshot edges, skips
// disabled and zero days, and starts at the first enabled day when none is
// focused. PageUp/PageDown and Shift+PageUp/PageDown change the month and stay
// the caller's job, so this reports false for them.
func CalendarFocusForKey(key ui.Key, direction Direction, days []CalendarDay, focused ui.ID) (ui.ID, bool) {
	enabled := func(index int) bool {
		day := days[index]
		return !day.Disabled && !day.ID.IsZero()
	}
	index := -1
	for at := range days {
		if days[at].ID == focused && enabled(at) {
			index = at
			break
		}
	}
	if index < 0 {
		for at := range days {
			if enabled(at) {
				return days[at].ID, true
			}
		}
		return focused, false
	}
	step := 0
	switch key {
	case ui.KeyLeft:
		step = -1
		if direction.IsRTL() {
			step = 1
		}
	case ui.KeyRight:
		step = 1
		if direction.IsRTL() {
			step = -1
		}
	case ui.KeyUp:
		step = -7
	case ui.KeyDown:
		step = 7
	case ui.KeyHome, ui.KeyEnd:
		start := index - index%7
		end := min(start+6, len(days)-1)
		target := start
		if key == ui.KeyEnd {
			target = end
		}
		for target >= start && target <= end {
			if enabled(target) {
				return days[target].ID, true
			}
			if key == ui.KeyEnd {
				target--
			} else {
				target++
			}
		}
		return days[index].ID, true
	default:
		return focused, false
	}
	for at := index + step; at >= 0 && at < len(days); at += step {
		if enabled(at) {
			return days[at].ID, true
		}
	}
	return days[index].ID, true
}
