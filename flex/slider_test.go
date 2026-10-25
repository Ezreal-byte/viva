package flex

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

var volumeID = tree.ID(page, "volume")

// sliderPage is a slider 200 wide in a window of that width. changed collects what
// Changed said in each paint.
type sliderPage struct {
	view     View
	state    SliderState
	declare  func(Box) Box
	box      Box
	changed  int
	vertical bool
}

func (p *sliderPage) paint(canvas *ui.Canvas, size ui.Size) {
	p.view.Draw(canvas, ui.LightTheme(), ui.Rect{Width: 200, Height: 200}, func(f *Frame) {
		f.Column().Fill().Children(func() {
			b := f.Slider(volumeID, &p.state).Label("Volume")
			if p.vertical {
				b = b.Vertical().Height(160).Width(16)
			}
			if p.declare != nil {
				b = p.declare(b)
			}
			p.box = b
			if b.Changed() {
				p.changed++
			}
		})
	})
}

func (p *sliderPage) pointer(t *testing.T, canvas *ui.Canvas, kind ui.PointerEventKind, x, y float32) bool {
	t.Helper()
	return canvas.DeliverPointer(ui.PointerEvent{Kind: kind, Button: ui.PointerButtonPrimary, X: x, Y: y, Target: volumeID, Captured: kind != ui.PointerDown}, ui.Size{Width: 200, Height: 200}, p.paint)
}

// The track of a horizontal slider 200 wide runs between the thumb's two half widths.
const trackFrom, trackTo = float32(8), float32(192)

func TestSliderDragMovesTheValueAndStaysInRange(t *testing.T) {
	p := &sliderPage{}
	p.state.Value = 20
	canvas := testCanvas()
	p.paint(canvas, ui.Size{})
	y := p.box.Rect().Y + 8
	thumb := trackFrom + (trackTo-trackFrom)*0.2

	// Taking the thumb where it is changes nothing; moving drags it by the distance moved.
	if !p.pointer(t, canvas, ui.PointerDown, thumb, y) || p.state.Value != 20 || p.changed != 0 {
		t.Fatalf("grabbing the thumb made %v and %d changes, want it to stay 20 with none", p.state.Value, p.changed)
	}
	if !p.state.Dragging() {
		t.Fatal("a press on the thumb did not start a drag")
	}
	p.pointer(t, canvas, ui.PointerMove, thumb+(trackTo-trackFrom)*0.3, y)
	if v := p.state.Value; v < 49.9 || v > 50.1 || p.changed != 1 {
		t.Errorf("dragged a third of the track right: value %v after %d changes, want 50 after 1", v, p.changed)
	}
	p.pointer(t, canvas, ui.PointerMove, 5000, y)
	if p.state.Value != 100 {
		t.Errorf("dragged far right the value is %v, want 100", p.state.Value)
	}
	p.pointer(t, canvas, ui.PointerMove, -5000, y)
	if p.state.Value != 0 {
		t.Errorf("dragged far left the value is %v, want 0", p.state.Value)
	}
	p.pointer(t, canvas, ui.PointerUp, -5000, y)
	if p.state.Dragging() {
		t.Error("the release did not end the drag")
	}
	canvas.Reset()
	p.pointer(t, canvas, ui.PointerMove, 100, y) // after the release a move is nobody's
	if p.state.Value != 0 {
		t.Errorf("a move after the release changed the value to %v", p.state.Value)
	}
}

func TestSliderClickOnTheTrackJumpsThereAndSnapsToTheStep(t *testing.T) {
	p := &sliderPage{declare: func(b Box) Box { return b.Range(0, 1).Step(0.25) }}
	canvas := testCanvas()
	p.paint(canvas, ui.Size{})
	y := p.box.Rect().Y + 8
	click := func(fraction float32) {
		t.Helper()
		x := trackFrom + (trackTo-trackFrom)*fraction
		p.pointer(t, canvas, ui.PointerDown, x, y)
		p.pointer(t, canvas, ui.PointerUp, x, y)
	}
	click(0.9)
	if p.state.Value != 1 {
		t.Errorf("a click at 90%% of a 0..1 slider in quarters gives %v, want 1", p.state.Value)
	}
	click(0.4)
	if p.state.Value != 0.5 {
		t.Errorf("a click at 40%% gives %v, want the nearest quarter, 0.5", p.state.Value)
	}
	if p.changed != 2 {
		t.Errorf("%d changes reported for two clicks that moved it", p.changed)
	}
	click(0.4) // the same place again
	if p.changed != 2 {
		t.Errorf("a click that left the value as it was reported a change (%d)", p.changed)
	}
}

func TestSliderKeysMoveAFocusedSliderAndReportOnce(t *testing.T) {
	p := &sliderPage{declare: func(b Box) Box { return b.Range(0, 50).Step(5) }}
	p.state.Value = 20
	canvas := testCanvas()
	p.paint(canvas, ui.Size{})
	key := func(k ui.Key, target ui.ID) {
		p.state.Key(ui.KeyEvent{Key: k, Target: target})
	}
	for _, c := range []struct {
		key  ui.Key
		want float32
	}{{ui.KeyRight, 25}, {ui.KeyUp, 30}, {ui.KeyLeft, 25}, {ui.KeyDown, 20}, {ui.KeyPageUp, 30}, {ui.KeyPageDown, 20}, {ui.KeyEnd, 50}, {ui.KeyHome, 0}, {ui.KeyLeft, 0}} {
		key(c.key, volumeID)
		if p.state.Value != c.want {
			t.Fatalf("key %v: value %v, want %v", c.key, p.state.Value, c.want)
		}
	}
	p.state.Value = 20
	if r := p.state.Key(ui.KeyEvent{Key: ui.KeyRight, Target: omit}); r.Handled || p.state.Value != 20 {
		t.Error("an arrow key for another control moved the slider")
	}
	if r := p.state.Key(ui.KeyEvent{Key: ui.KeyEnter, Target: volumeID}); r.Handled {
		t.Error("a key the slider has no use for was handled")
	}
	before := p.changed
	key(ui.KeyRight, volumeID)
	canvas.Reset()
	p.paint(canvas, ui.Size{})
	canvas.Reset()
	p.paint(canvas, ui.Size{})
	if p.changed != before+1 {
		t.Errorf("a key was reported %d times, want once", p.changed-before)
	}
}

func TestADisabledSliderIgnoresThePointerAndTheKeyboard(t *testing.T) {
	p := &sliderPage{declare: func(b Box) Box { return b.Disabled(true) }}
	p.state.Value = 40
	canvas := testCanvas()
	p.paint(canvas, ui.Size{})
	y := p.box.Rect().Y + 8
	p.pointer(t, canvas, ui.PointerDown, trackTo, y)
	p.pointer(t, canvas, ui.PointerMove, trackFrom, y)
	p.state.Key(ui.KeyEvent{Key: ui.KeyEnd, Target: volumeID})
	if p.state.Value != 40 || p.state.Dragging() || p.changed != 0 {
		t.Errorf("a disabled slider went to %v (dragging %v, %d changes), want 40 untouched", p.state.Value, p.state.Dragging(), p.changed)
	}
	for _, hit := range canvas.Hits() {
		if hit.ID == volumeID && (!hit.Disabled || hit.Focusable) {
			t.Errorf("a disabled slider's hit is %+v, want disabled and not focusable", hit)
		}
	}
}

func TestSliderValueStaysInsideWhenTheRangeOrTheWindowShrinks(t *testing.T) {
	p := &sliderPage{}
	p.state.Value = 80
	canvas := testCanvas()
	p.paint(canvas, ui.Size{})
	p.declare = func(b Box) Box { return b.Range(0, 10) }
	p.paint(canvas, ui.Size{})
	if p.state.Value != 10 {
		t.Errorf("after the range shrank to 0..10 the value is %v, want 10", p.state.Value)
	}
	p.declare = func(b Box) Box { return b.Range(20, 30) }
	p.paint(canvas, ui.Size{})
	if p.state.Value != 20 {
		t.Errorf("after the range moved up to 20..30 the value is %v, want 20", p.state.Value)
	}
	// A window too narrow for a track still paints a slider and takes input.
	for _, hit := range canvas.Hits() {
		if hit.ID == volumeID && (hit.Bounds.Width <= 0 || hit.Bounds.X < 0 || hit.Bounds.X+hit.Bounds.Width > 200) {
			t.Errorf("the slider's hit %+v is outside the window", hit.Bounds)
		}
	}
	var view View
	var narrow Box
	view.Draw(canvas, ui.LightTheme(), ui.Rect{Width: 10, Height: 40}, func(f *Frame) {
		f.Column().Fill().Children(func() { narrow = f.Slider(volumeID, &p.state).Range(20, 30) })
	})
	if narrow.Rect().Width != 10 || p.state.Value < 20 || p.state.Value > 30 {
		t.Errorf("in a window 10 wide the slider is %v wide with value %v", narrow.Rect().Width, p.state.Value)
	}
}

func TestAVerticalSliderRaisesTheValueUpwardsAndSizesItself(t *testing.T) {
	p := &sliderPage{vertical: true}
	canvas := testCanvas()
	p.paint(canvas, ui.Size{})
	r := p.box.Rect()
	near(t, "vertical slider", r, ui.Rect{Width: 16, Height: 160})
	bottom, top := r.Y+r.Height-8, r.Y+8
	x := r.X + 8
	p.pointer(t, canvas, ui.PointerDown, x, bottom)
	if p.state.Value != 0 {
		t.Errorf("a press at the bottom gives %v, want the minimum", p.state.Value)
	}
	p.pointer(t, canvas, ui.PointerMove, x, (top+bottom)/2)
	if v := p.state.Value; v < 49.9 || v > 50.1 {
		t.Errorf("dragged to the middle the value is %v, want 50", v)
	}
	p.pointer(t, canvas, ui.PointerMove, x, -100)
	if p.state.Value != 100 {
		t.Errorf("dragged above the top the value is %v, want 100", p.state.Value)
	}
}

func TestSliderIsDefaultSizedAndNamedForAScreenReader(t *testing.T) {
	var view View
	var column, row Box
	canvas := testCanvas()
	var a, b SliderState
	a.Value = 25
	view.Draw(canvas, ui.LightTheme(), ui.Rect{Width: 300, Height: 100}, func(f *Frame) {
		f.Column().Pad(10).Gap(8).Children(func() {
			column = f.Slider(volumeID, &a).Label("Volume")
			f.Row().Children(func() { row = f.Slider(rowID, &b) })
		})
	})
	near(t, "in a column", column.Rect(), ui.Rect{X: 10, Y: 10, Width: 280, Height: 16})
	near(t, "in a row", row.Rect(), ui.Rect{X: 10, Y: 34, Width: 160, Height: 16})
	for _, hit := range canvas.Hits() {
		if hit.ID == volumeID && (hit.Role != ui.RoleSlider || hit.Label != "Volume" || hit.Value != "25" || !hit.Focusable) {
			t.Errorf("the slider's hit is %+v, want role slider, label Volume, value 25, focusable", hit)
		}
	}
}

func TestSlidersAllocateNothingOnceWarm(t *testing.T) {
	var view View
	canvas := testCanvas()
	var s SliderState
	s.Value = 30
	run := func() {
		canvas.Reset()
		view.Draw(canvas, ui.LightTheme(), ui.Rect{Width: 400, Height: 300}, func(f *Frame) {
			f.Column().Fill().Pad(8).Gap(8).Children(func() {
				f.Slider(volumeID, &s).Range(0, 1).Step(0.05).Label("Volume").Changed()
				f.Row().Height(100).Children(func() { f.Slider(rowID, &s).Vertical() })
			})
		})
	}
	run()
	if allocs := testing.AllocsPerRun(200, run); allocs != 0 {
		t.Fatalf("a warm page with sliders allocates %v times a frame, want 0", allocs)
	}
}

func BenchmarkSlider(b *testing.B) {
	var view View
	canvas := testCanvas()
	var s SliderState
	b.ReportAllocs()
	for b.Loop() {
		canvas.Reset()
		view.Draw(canvas, ui.LightTheme(), ui.Rect{Width: 400, Height: 300}, func(f *Frame) {
			f.Column().Fill().Children(func() { f.Slider(volumeID, &s).Label("Volume").Changed() })
		})
	}
}
