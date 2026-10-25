package components

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

// A roving group has exactly one Tab stop: the selection (or first enabled
// item), and an explicitly focused item takes it over. The arrow keys can still
// focus every other item because they remain focusable.
func TestRovingGroupsKeepOneTabStop(t *testing.T) {
	check := func(t *testing.T, canvas *ui.Canvas, want ui.ID) {
		t.Helper()
		tabs := 0
		for _, hit := range canvas.Hits() {
			if hit.Focusable && !hit.SkipTab {
				tabs++
				if hit.ID != want {
					t.Fatalf("Tab stop is %v, want %v (hits %+v)", hit.ID, want, canvas.Hits())
				}
			}
		}
		if tabs != 1 {
			t.Fatalf("%d Tab stops, want one", tabs)
		}
	}

	t.Run("radio", func(t *testing.T) {
		var canvas ui.Canvas
		BeginRadioGroup(&canvas, ui.LightTheme(), ui.Rect{Width: 200, Height: 100}, []RadioOption{
			{ID: ui.Item(1), Label: "One"}, {ID: ui.Item(2), Label: "Two"}, {ID: ui.Item(3), Label: "Three"},
		}, ui.Item(2)).Paint()
		check(t, &canvas, ui.Item(2))
	})

	t.Run("tabs", func(t *testing.T) {
		var canvas ui.Canvas
		BeginTabs(&canvas, ui.LightTheme(), ui.Rect{Width: 300, Height: 32}, []TabOption{
			{ID: ui.Item(1), Label: "One"}, {ID: ui.Item(2), Label: "Two"}, {ID: ui.Item(3), Label: "Three"},
		}, ui.Item(2)).Paint()
		check(t, &canvas, ui.Item(2))
	})

	t.Run("toggle group", func(t *testing.T) {
		var canvas ui.Canvas
		BeginToggleGroup(&canvas, ui.LightTheme(), ui.Rect{Width: 300, Height: 32}, []ToggleGroupOption{
			{ID: ui.Item(1), Label: "One"}, {ID: ui.Item(2), Label: "Two", Active: true},
		}).Paint()
		check(t, &canvas, ui.Item(2))
	})

	t.Run("segmented", func(t *testing.T) {
		var canvas ui.Canvas
		BeginSegmentedControl(&canvas, ui.LightTheme(), ui.Rect{Width: 300, Height: 32}, []SegmentOption{
			{ID: ui.Item(1), Label: "One"}, {ID: ui.Item(2), Label: "Two"}, {ID: ui.Item(3), Label: "Three"},
		}, ui.Item(3)).Paint()
		check(t, &canvas, ui.Item(3))
	})

	t.Run("calendar", func(t *testing.T) {
		var canvas ui.Canvas
		days := make([]CalendarDay, 7)
		for index := range days {
			days[index] = CalendarDay{ID: ui.Item(index + 1), Label: "1", Selected: index == 4}
		}
		BeginCalendar(&canvas, ui.LightTheme(), ui.Rect{Width: 300, Height: 240}, CalendarOptions{MonthTitle: "May", Days: days}).Paint()
		check(t, &canvas, ui.Item(5))
	})

	t.Run("focused item takes the stop", func(t *testing.T) {
		var canvas ui.Canvas
		canvas.SetInteraction(ui.ID{}, ui.ID{}, ui.Item(3), true)
		BeginRadioGroup(&canvas, ui.LightTheme(), ui.Rect{Width: 200, Height: 100}, []RadioOption{
			{ID: ui.Item(1), Label: "One"}, {ID: ui.Item(2), Label: "Two"}, {ID: ui.Item(3), Label: "Three"},
		}, ui.Item(2)).Paint()
		check(t, &canvas, ui.Item(3))
	})
}
