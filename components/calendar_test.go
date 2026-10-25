package components

import (
	"slices"
	"testing"

	"github.com/vivaui/viva/ui"
)

var calendarTestWeekdays = [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}

var calendarTestDays = [...]CalendarDay{
	{ID: ui.Item(101), Label: "29", AccessibilityLabel: "September 29", Outside: true},
	{ID: ui.Item(102), Label: "30", AccessibilityLabel: "September 30", Outside: true, Disabled: true},
	{ID: ui.Item(103), Label: "1", AccessibilityLabel: "October 1", Today: true},
	{ID: ui.Item(104), Label: "2", AccessibilityLabel: "October 2", Selected: true},
	{ID: ui.Item(105), Label: "3", AccessibilityLabel: "October 3", RangeStart: true},
	{ID: ui.Item(106), Label: "4", AccessibilityLabel: "October 4", RangeMiddle: true},
	{ID: ui.Item(107), Label: "5", AccessibilityLabel: "October 5", RangeEnd: true},
	{ID: ui.Item(108), Label: "6", AccessibilityLabel: "October 6", Value: "2026-10-06", Hint: "Available"},
}

func calendarTestOptions() CalendarOptions {
	return CalendarOptions{
		MonthTitle: "October 2026", Weekdays: calendarTestWeekdays, Days: calendarTestDays[:],
		PreviousID: ui.Item(91), NextID: ui.Item(92), PreviousLabel: "Previous month", NextLabel: "Next month",
		PreviousDisabled: true,
	}
}

func TestCalendarUsesSevenColumnGeometryAndCallerOwnedSnapshot(t *testing.T) {
	theme := ui.LightTheme()
	var canvas ui.Canvas
	calendar := BeginCalendar(&canvas, theme, ui.Rect{X: 10, Y: 20, Width: 240}, calendarTestOptions())
	calendar.Paint()

	if got, want := calendar.Bounds(), (ui.Rect{X: 10, Y: 20, Width: 240, Height: 160}); got != want {
		t.Fatalf("calendar bounds = %+v, want %+v", got, want)
	}
	if got, want := calendar.DayBounds(0), (ui.Rect{X: 18, Y: 108, Width: 28, Height: 28}); got != want {
		t.Fatalf("first day bounds = %+v, want %+v", got, want)
	}
	if got, want := calendar.DayBounds(6), (ui.Rect{X: 186, Y: 108, Width: 28, Height: 28}); got != want {
		t.Fatalf("seventh day bounds = %+v, want %+v", got, want)
	}
	if got, want := calendar.DayBounds(7), (ui.Rect{X: 18, Y: 144, Width: 28, Height: 28}); got != want {
		t.Fatalf("eighth day bounds = %+v, want %+v", got, want)
	}
	// The navigation arrows are icons, not text: title, 7 weekdays, 8 days.
	if len(canvas.Texts()) != 16 {
		t.Fatalf("calendar texts = %d, want 16", len(canvas.Texts()))
	}
	if canvas.Texts()[0].Value != "October 2026" || canvas.Texts()[1].Value != "Su" || canvas.Texts()[8].Value != "29" {
		t.Fatalf("calendar text order = %+v", canvas.Texts())
	}
	if len(nonBlockHits(canvas.Hits())) != 10 {
		t.Fatalf("calendar hits = %d, want 10", len(nonBlockHits(canvas.Hits())))
	}
	if previous := nonBlockHits(canvas.Hits())[0]; previous.ID != ui.Item(91) || previous.Role != ui.RoleButton || !previous.Disabled || previous.Focusable {
		t.Fatalf("previous hit = %+v", previous)
	}
	if next := nonBlockHits(canvas.Hits())[1]; next.ID != ui.Item(92) || next.Role != ui.RoleButton || next.Disabled || !next.Focusable {
		t.Fatalf("next hit = %+v", next)
	}
	if disabledDay := nonBlockHits(canvas.Hits())[3]; disabledDay.Label != "September 30" || !disabledDay.Disabled || disabledDay.Focusable {
		t.Fatalf("disabled day hit = %+v", disabledDay)
	}
	if selected := nonBlockHits(canvas.Hits())[5]; !selected.Selected || selected.Role != ui.RoleGridCell {
		t.Fatalf("selected day hit = %+v", selected)
	}
	if rangeMiddle := nonBlockHits(canvas.Hits())[7]; !rangeMiddle.Selected {
		t.Fatalf("range-middle hit = %+v", rangeMiddle)
	}
	if value := nonBlockHits(canvas.Hits())[9]; value.Value != "2026-10-06" || value.Hint != "Available" {
		t.Fatalf("day value semantics = %+v", value)
	}
}

func TestCalendarPaintsOfficialSingleRangeTodayAndOutsideStates(t *testing.T) {
	theme := ui.LightTheme()
	var canvas ui.Canvas
	BeginCalendar(&canvas, theme, ui.Rect{Width: 240}, calendarTestOptions()).Paint()

	texts := canvas.Texts()
	if texts[8].Style.Color != theme.MutedForeground {
		t.Fatalf("outside foreground = %+v, want %+v", texts[8].Style.Color, theme.MutedForeground)
	}
	disabledForeground := theme.MutedForeground
	disabledForeground.Alpha *= 0.50
	if texts[9].Style.Color != disabledForeground {
		t.Fatalf("disabled foreground = %+v, want %+v", texts[9].Style.Color, disabledForeground)
	}
	if texts[10].Style.Color != theme.Foreground {
		t.Fatalf("today foreground = %+v, want %+v", texts[10].Style.Color, theme.Foreground)
	}
	if texts[11].Style.Color != theme.PrimaryForeground || texts[12].Style.Color != theme.PrimaryForeground || texts[14].Style.Color != theme.PrimaryForeground {
		t.Fatalf("selected endpoint foregrounds = %+v %+v %+v", texts[11].Style.Color, texts[12].Style.Color, texts[14].Style.Color)
	}
	if texts[13].Style.Color != theme.Foreground {
		t.Fatalf("range middle foreground = %+v, want %+v", texts[13].Style.Color, theme.Foreground)
	}

	dayRectangles := canvas.Rectangles()[2:]
	if dayRectangles[2].Color != theme.Muted || dayRectangles[2].Radius == 0 {
		t.Fatalf("today surface = %+v", dayRectangles[2])
	}
	if dayRectangles[3].Color != theme.Primary || dayRectangles[3].Radius == 0 {
		t.Fatalf("single selection surface = %+v", dayRectangles[3])
	}
	if dayRectangles[4].Bounds.Width != 14 || dayRectangles[4].Color != theme.Muted || dayRectangles[5].Color != theme.Primary {
		t.Fatalf("range start surfaces = %+v %+v", dayRectangles[4], dayRectangles[5])
	}
	if dayRectangles[6].Color != theme.Muted || dayRectangles[6].Radius != 0 {
		t.Fatalf("range middle surface = %+v", dayRectangles[6])
	}
	if dayRectangles[7].Bounds.Width != 14 || dayRectangles[7].Color != theme.Muted || dayRectangles[8].Color != theme.Primary {
		t.Fatalf("range end surfaces = %+v %+v", dayRectangles[7], dayRectangles[8])
	}
}

func TestCalendarFocusedDayUsesRingAndPreservesTypedSemantics(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	canvas.SetInteraction(ui.ID{}, ui.ID{}, ui.Item(106), true)
	calendar := BeginCalendar(&canvas, theme, ui.Rect{Width: 240}, calendarTestOptions())
	calendar.Paint()

	focusedBounds := calendar.DayBounds(5)
	wantRingBounds := ui.Rect{X: focusedBounds.X - 3, Y: focusedBounds.Y - 3, Width: 34, Height: 34}
	foundRing := false
	for _, rectangle := range canvas.Rectangles() {
		if rectangle.Bounds == wantRingBounds {
			wantColor := theme.Ring
			wantColor.Alpha *= 0.50
			if rectangle.Color != wantColor {
				t.Fatalf("focus ring color = %+v, want %+v", rectangle.Color, wantColor)
			}
			foundRing = true
			break
		}
	}
	if !foundRing {
		t.Fatalf("focused day ring with bounds %+v not found", wantRingBounds)
	}
	focusedHit := nonBlockHits(canvas.Hits())[7]
	if focusedHit.ID != ui.Item(106) || focusedHit.Role != ui.RoleGridCell || !focusedHit.Selected || !focusedHit.Focusable {
		t.Fatalf("focused range day semantics = %+v", focusedHit)
	}
}

// A calendar sits on whatever surface holds it. Painting the page background
// itself would make a darker rectangle inside a popover or card in dark mode.
func TestCalendarPaintsNoSurfaceOfItsOwn(t *testing.T) {
	for _, theme := range []ui.Theme{ui.LightTheme(), ui.DarkTheme()} {
		var canvas ui.Canvas
		calendar := BeginCalendar(&canvas, theme, ui.Rect{Width: 240}, calendarTestOptions())
		calendar.Paint()
		for _, rectangle := range canvas.Rectangles() {
			if rectangle.Bounds == calendar.Bounds() && rectangle.Color.Alpha != 0 {
				t.Fatalf("calendar filled its own bounds with %+v", rectangle.Color)
			}
		}
	}
}

func TestCalendarCommandForKeyMapsNavigationIntents(t *testing.T) {
	for _, test := range []struct {
		key       ui.Key
		modifiers ui.Modifiers
		want      CalendarCommand
		handled   bool
	}{
		{ui.KeyLeft, 0, CalendarCommand{Kind: CalendarCommandMoveDays, Amount: -1}, true},
		{ui.KeyRight, 0, CalendarCommand{Kind: CalendarCommandMoveDays, Amount: 1}, true},
		{ui.KeyUp, 0, CalendarCommand{Kind: CalendarCommandMoveWeeks, Amount: -1}, true},
		{ui.KeyDown, 0, CalendarCommand{Kind: CalendarCommandMoveWeeks, Amount: 1}, true},
		{ui.KeyPageUp, 0, CalendarCommand{Kind: CalendarCommandMoveMonths, Amount: -1}, true},
		{ui.KeyPageDown, 0, CalendarCommand{Kind: CalendarCommandMoveMonths, Amount: 1}, true},
		{ui.KeyPageUp, ui.ModifierShift, CalendarCommand{Kind: CalendarCommandMoveYears, Amount: -1}, true},
		{ui.KeyPageDown, ui.ModifierShift, CalendarCommand{Kind: CalendarCommandMoveYears, Amount: 1}, true},
		{ui.KeyHome, 0, CalendarCommand{Kind: CalendarCommandWeekStart}, true},
		{ui.KeyEnd, 0, CalendarCommand{Kind: CalendarCommandWeekEnd}, true},
		{ui.KeyEnter, 0, CalendarCommand{}, false},
	} {
		got, handled := CalendarCommandForKey(test.key, test.modifiers)
		if got != test.want || handled != test.handled {
			t.Errorf("key %v modifiers %v = (%+v, %v), want (%+v, %v)", test.key, test.modifiers, got, handled, test.want, test.handled)
		}
	}
}

func TestCalendarRepeatedPaintIsStableAndAllocationFree(t *testing.T) {
	theme := ui.DarkTheme()
	options := calendarTestOptions()
	var canvas ui.Canvas
	paint := func() {
		canvas.Reset()
		BeginCalendar(&canvas, theme, ui.Rect{Width: 240}, options).Paint()
	}
	paint()
	rectangles := slices.Clone(canvas.Rectangles())
	texts := slices.Clone(canvas.Texts())
	hits := slices.Clone(nonBlockHits(canvas.Hits()))
	paint()
	if !slices.Equal(rectangles, canvas.Rectangles()) || !slices.Equal(texts, canvas.Texts()) || !slices.Equal(hits, nonBlockHits(canvas.Hits())) {
		t.Fatal("identical calendar snapshot produced different command streams")
	}
	if allocations := testing.AllocsPerRun(100, paint); allocations != 0 {
		t.Fatalf("calendar paint allocations = %v", allocations)
	}
}

// Calendar focus movement skips disabled days, clamps at the snapshot edge, and
// reverses horizontal arrows in RTL.
func TestCalendarFocusForKeySkipsAndClamps(t *testing.T) {
	days := make([]CalendarDay, 14)
	for index := range days {
		days[index] = CalendarDay{ID: ui.Item(index + 1), Label: "day"}
	}
	days[1].Disabled = true
	if id, ok := CalendarFocusForKey(ui.KeyRight, LTR, days, ui.Item(1)); !ok || id != ui.Item(3) {
		t.Fatalf("Right over disabled = (%v, %v), want the next enabled day", id, ok)
	}
	if id, ok := CalendarFocusForKey(ui.KeyLeft, LTR, days, ui.Item(1)); !ok || id != ui.Item(1) {
		t.Fatalf("Left at the start = (%v, %v)", id, ok)
	}
	if id, ok := CalendarFocusForKey(ui.KeyRight, RTL, days, ui.Item(3)); !ok || id != ui.Item(1) {
		t.Fatalf("Right in RTL = (%v, %v), want the previous enabled day", id, ok)
	}
	if id, ok := CalendarFocusForKey(ui.KeyEnd, LTR, days, ui.Item(1)); !ok || id != ui.Item(7) {
		t.Fatalf("End = (%v, %v), want the week end", id, ok)
	}
	if id, ok := CalendarFocusForKey(ui.KeyDown, LTR, days, ui.Item(7)); !ok || id != ui.Item(14) {
		t.Fatalf("Down into the last week = (%v, %v)", id, ok)
	}
	if _, ok := CalendarFocusForKey(ui.KeyPageDown, LTR, days, ui.Item(1)); ok {
		t.Fatal("PageDown belongs to month navigation")
	}
}
