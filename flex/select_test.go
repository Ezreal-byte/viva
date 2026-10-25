package flex

import (
	"fmt"
	"testing"
	"time"

	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/ui"
)

var (
	selScope  = tree.Sub(page, tree.Name("select"))
	selLabels = []string{"Small", "Medium", "Large"}
)

// A Select looks exactly like the components.Select it is painted by, closed and
// open, with a value and with a placeholder.
func TestSelectIsPaintedLikeTheComponent(t *testing.T) {
	theme := ui.LightTheme()
	for _, test := range []struct {
		open     bool
		selected int
		hovered  int
	}{{false, 1, -1}, {false, -1, 3}, {true, 1, 2}, {true, -1, -1}} {
		name := fmt.Sprintf("open=%v selected=%d", test.open, test.selected)
		flexed, direct := testCanvas(), testCanvas()
		if test.hovered >= 0 {
			flexed.SetInteraction(selScope.At(test.hovered), ui.ID{}, ui.ID{}, false)
			direct.SetInteraction(selScope.At(test.hovered), ui.ID{}, ui.ID{}, false)
		}
		state := SelectState{Open: test.open}
		viewport := ui.Rect{Width: 400, Height: 300}
		var view View
		view.Draw(flexed, theme, viewport, func(f *Frame) {
			f.Column().Pad(10).Children(func() {
				f.Select(selScope, &state, selLabels, test.selected).Placeholder("Size").Width(200)
			})
		})
		options := components.SelectOptions{
			TriggerID: selScope.At(3), Label: "Size", Placeholder: "Size", Open: test.open,
			Layer: 1, Scope: selScope.At(5), DismissID: selScope.At(4), Viewport: viewport,
		}
		if test.selected >= 0 {
			options.Value = selLabels[test.selected]
		}
		if test.open {
			options.Content = ui.Rect{X: 10, Y: 10 + 32 + 4, Width: 200, Height: 8 + 32*3}
		}
		control := components.BeginSelect(direct, theme, ui.Rect{X: 10, Y: 10, Width: 200}, options)
		if test.open {
			for i, label := range selLabels {
				control.AddItem(components.SelectItem{ID: selScope.At(i), Label: label, Selected: i == test.selected})
			}
		}
		sameCanvas(t, name, flexed, direct)
	}
}

func TestSelectOpensChoosesAndClosesThroughItsClicks(t *testing.T) {
	theme := ui.LightTheme()
	var state SelectState
	selected := 0
	draw := func(canvas *ui.Canvas) {
		var view View
		view.Draw(canvas, theme, ui.Rect{Width: 300, Height: 300}, func(f *Frame) {
			f.Column().Pad(10).Children(func() {
				if i, ok := f.Select(selScope, &state, selLabels, selected).Picked(); ok {
					selected = i
				}
			})
		})
	}
	click := func(index int) {
		canvas := testCanvas()
		canvas.BeginEventPass(selScope.At(index))
		draw(canvas)
		canvas.EndEventPass()
	}
	click(3) // the trigger
	if !state.Open {
		t.Fatal("a click on the trigger did not open the list")
	}
	canvas := testCanvas()
	draw(canvas)
	var items, dismiss int
	for _, hit := range canvas.Hits() {
		if i, ok := hit.ID.In(selScope); ok {
			if i < 3 {
				items++
			}
			if i == 4 {
				dismiss++
			}
		}
	}
	if items != 3 || dismiss != 1 {
		t.Errorf("an open list declares %d options and %d dismiss targets, want 3 and 1", items, dismiss)
	}
	click(2)
	if state.Open || selected != 2 {
		t.Errorf("choosing the third option left open=%v selected=%d", state.Open, selected)
	}
	click(3)
	click(4) // outside, or Escape
	if state.Open || selected != 2 {
		t.Errorf("dismissing left open=%v selected=%d", state.Open, selected)
	}
}

// A list too long for the frame shows a window of options, with arrows that move it.
func TestSelectTooLongForTheFrameScrollsItsWindow(t *testing.T) {
	theme := ui.LightTheme()
	labels := make([]string, 40)
	for i := range labels {
		labels[i] = fmt.Sprintf("option %d", i)
	}
	state := SelectState{Open: true}
	draw := func(canvas *ui.Canvas) (first, last, arrows int) {
		first, last = -1, -1
		var view View
		view.Draw(canvas, theme, ui.Rect{Width: 300, Height: 300}, func(f *Frame) {
			f.Column().Pad(10).Children(func() {
				f.Select(selScope, &state, labels, 20).Picked()
			})
		})
		for _, hit := range canvas.Hits() {
			if i, ok := hit.ID.In(selScope); ok {
				switch {
				case i < 40:
					if first < 0 {
						first = i
					}
					last = i
				case i == 43 || i == 44:
					arrows++
				}
			}
		}
		return
	}
	first, last, arrows := draw(testCanvas())
	if arrows != 2 || last-first+1 > 8 || last-first+1 < 4 {
		t.Fatalf("a window of options %d..%d with %d arrows", first, last, arrows)
	}
	canvas := testCanvas()
	canvas.BeginEventPass(selScope.At(44)) // scroll down
	draw(canvas)
	canvas.EndEventPass()
	if state.First <= first {
		t.Errorf("the down arrow left the window at %d", state.First)
	}
}

func TestSelectAllocatesNothingOnceWarm(t *testing.T) {
	var view View
	canvas := testCanvas()
	theme := ui.LightTheme()
	state := SelectState{Open: true}
	closed := SelectState{}
	run := func() {
		canvas.Reset()
		view.Draw(canvas, theme, ui.Rect{Width: 400, Height: 300}, func(f *Frame) {
			f.Column().Pad(10).Gap(8).Children(func() {
				f.Select(selScope, &state, selLabels, 1)
				f.Select(selScope, &closed, selLabels, -1).Placeholder("Pick").Small()
			})
		})
	}
	run()
	if allocs := testing.AllocsPerRun(200, run); allocs != 0 {
		t.Fatalf("warm selects allocate %v times a frame, want 0", allocs)
	}
}

func BenchmarkOpenSelectAndTabs(b *testing.B) {
	var view View
	canvas := testCanvas()
	theme := ui.LightTheme()
	state := SelectState{Open: true}
	b.ReportAllocs()
	for b.Loop() {
		canvas.Reset()
		view.Draw(canvas, theme, ui.Rect{Width: 400, Height: 300}, func(f *Frame) {
			f.Column().Pad(10).Gap(8).Children(func() {
				f.Tabs(tabsScope, tabLabels, 1).Line()
				f.Select(selScope, &state, selLabels, 1)
			})
		})
	}
}

var fruit = []string{"Apple", "Avocado", "Banana", "Blueberry", "Cherry", "Date"}

func drawFruit(state *SelectState, labels []string, height float32) {
	var view View
	view.Draw(testCanvas(), ui.LightTheme(), ui.Rect{Width: 300, Height: height}, func(f *Frame) {
		f.Column().Pad(10).Children(func() { f.Select(selScope, state, labels, 0).Picked() })
	})
}

// Typing jumps to the option that starts with the letters: in an open list the match
// takes the focus, on the focused trigger of a closed select the list opens on it; a
// repeated letter cycles, a pause starts a new word, and other focus is left alone.
func TestSelectTypeAheadJumpsToTheMatchingOption(t *testing.T) {
	now := time.Unix(1000, 0)
	clock = func() time.Time { return now }
	defer func() { clock = time.Now }()
	var state SelectState
	drawFruit(&state, fruit, 400)
	trigger := selScope.At(len(fruit))
	typed := func(target ui.ID, text string) ui.EventResult {
		return state.Key(ui.KeyEvent{Text: text, Target: target})
	}

	if got := typed(omit, "b"); got.Handled || state.Open {
		t.Fatal("typing with another control focused opened the select")
	}
	got := typed(trigger, "b")
	if !state.Open || got.Focus != selScope.At(2) {
		t.Fatalf("typing b on the closed select: open=%v focus=%v, want it open on Banana", state.Open, got.Focus)
	}
	now = now.Add(200 * time.Millisecond)
	if got := typed(selScope.At(2), "l"); got.Focus != selScope.At(3) {
		t.Errorf("b then l focuses %v, want Blueberry", got.Focus)
	}
	now = now.Add(2 * time.Second)
	if got := typed(selScope.At(3), "a"); got.Focus != selScope.At(0) {
		t.Errorf("after a pause, a focuses %v, want Apple", got.Focus)
	}
	now = now.Add(100 * time.Millisecond)
	if got := typed(selScope.At(0), "a"); got.Focus != selScope.At(1) {
		t.Errorf("a pressed twice focuses %v, want to cycle on to Avocado", got.Focus)
	}
	now = now.Add(2 * time.Second)
	if got := typed(selScope.At(1), "z"); got.Focus.IsZero() == false || !got.Handled {
		t.Errorf("a letter nothing starts with gave %+v, want it swallowed without moving focus", got)
	}
	if got := state.Key(ui.KeyEvent{Text: "c", Modifiers: ui.ModifierControl, Target: selScope.At(1)}); got.Handled {
		t.Error("a Control chord was taken as type-ahead")
	}
}

func TestSelectTypeAheadScrollsALongListToTheMatch(t *testing.T) {
	labels := make([]string, 60)
	for i := range labels {
		labels[i] = fmt.Sprintf("item %02d", i)
	}
	labels[50] = "zebra"
	state := SelectState{Open: true}
	drawFruit(&state, labels, 300)
	if state.visible >= len(labels) || state.visible < 2 {
		t.Fatalf("the window shows %d of %d options", state.visible, len(labels))
	}
	got := state.Key(ui.KeyEvent{Text: "z", Target: selScope.At(0)})
	if got.Focus != selScope.At(50) || state.First > 50 || state.First+state.visible <= 50 {
		t.Errorf("z: focus %v, window starts at %d showing %d, want option 50 inside it", got.Focus, state.First, state.visible)
	}
}

// The wheel scrolls an open list a row at a time and stays in range; a closed select
// leaves the event to the page.
func TestSelectWheelScrollsTheOpenList(t *testing.T) {
	labels := make([]string, 40)
	for i := range labels {
		labels[i] = fmt.Sprintf("option %d", i)
	}
	state := SelectState{}
	drawFruit(&state, labels, 300)
	if got := state.Wheel(ui.ScrollEvent{DeltaY: 10}); got.Handled {
		t.Error("a closed select took the wheel")
	}
	state.Open = true
	drawFruit(&state, labels, 300)
	start := state.First
	if got := state.Wheel(ui.ScrollEvent{DeltaY: 10}); !got.Handled || !got.Repaint || state.First != start+1 {
		t.Errorf("a notch moved the window from %d to %d, want one row", start, state.First)
	}
	state.Wheel(ui.ScrollEvent{DeltaY: 64, Precise: true})
	if state.First != start+3 {
		t.Errorf("64 pixels of trackpad moved the window to %d, want two more rows (%d)", state.First, start+3)
	}
	state.Wheel(ui.ScrollEvent{DeltaY: 1e6})
	if want := len(labels) - state.visible; state.First != want {
		t.Errorf("scrolled far down the window starts at %d, want the last full one %d", state.First, want)
	}
	state.Wheel(ui.ScrollEvent{DeltaY: -1e6})
	if state.First != 0 {
		t.Errorf("scrolled far up the window starts at %d, want 0", state.First)
	}
}
