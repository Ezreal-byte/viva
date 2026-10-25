package ui

import "testing"

func TestLabelRunsCopiesItsRunsAndExposesThemPerText(t *testing.T) {
	var canvas Canvas
	runs := []TextRun{{Column: 0, Bytes: 2, Cells: 2, Color: Color{Red: 1, Alpha: 1}, Ligatures: true}}
	canvas.LabelRuns("hi", Rect{Width: 20, Height: 10}, TextStyle{Size: 12}, 6, runs)
	runs[0].Color = Color{Green: 1, Alpha: 1} // the caller reuses its buffer
	canvas.LabelRuns("yo", Rect{Y: 10, Width: 20, Height: 10}, TextStyle{Size: 12}, 6,
		[]TextRun{{Bytes: 1, Cells: 1}, {Column: 1, Bytes: 1, Cells: 1}})
	canvas.Label("plain", Rect{Y: 20, Width: 20, Height: 10}, TextStyle{Size: 12})

	texts := canvas.Texts()
	if len(texts) != 3 {
		t.Fatalf("texts = %d", len(texts))
	}
	if !texts[0].Style.Mono || texts[0].CellWidth != 6 {
		t.Errorf("a runs label is fixed-pitch: %+v", texts[0])
	}
	if got := canvas.TextRuns(texts[0]); len(got) != 1 || got[0].Color.Red != 1 || got[0].Color.Green != 0 || !got[0].Ligatures {
		t.Errorf("first runs = %+v, want the red ligature run as it was when added", got)
	}
	if got := canvas.TextRuns(texts[1]); len(got) != 2 || got[1].Column != 1 {
		t.Errorf("second runs = %+v", got)
	}
	if got := canvas.TextRuns(texts[2]); got != nil {
		t.Errorf("a plain label has runs: %+v", got)
	}
	if len(canvas.AllTextRuns()) != 3 {
		t.Errorf("run arena = %d", len(canvas.AllTextRuns()))
	}

	canvas.Reset()
	if len(canvas.AllTextRuns()) != 0 || len(canvas.Texts()) != 0 {
		t.Error("Reset kept runs")
	}
	if canvas.TextRuns(texts[0]) != nil {
		t.Error("stale text resolved to runs after Reset")
	}
}

func TestLabelRunsDoesNotAllocateOnceWarm(t *testing.T) {
	var canvas Canvas
	runs := make([]TextRun, 8)
	paint := func() {
		canvas.Reset()
		for row := 0; row < 40; row++ {
			canvas.LabelRuns("some text per row", Rect{Y: float32(row) * 15, Width: 700, Height: 15}, TextStyle{Size: 13}, 7, runs)
		}
	}
	paint()
	if allocs := testing.AllocsPerRun(20, paint); allocs != 0 {
		t.Errorf("a warm paint of 40 grid lines allocates %v times", allocs)
	}
}

// A font whose line box is taller than 1.18 sizes gets rows no taller than
// that, so a terminal is as dense on a platform whose mono font is loose as on one
// whose font is tight; a tighter font keeps its own height.
func TestCellHeightIsCappedForFontsWithTallLineBoxes(t *testing.T) {
	var canvas Canvas
	canvas.SetCellMeasurer(func(size float32, face MonoFace) (float32, float32) {
		if face == MonoClassic {
			return size * 0.6, size * 1.1
		}
		return size * 0.6, size * 1.45
	})
	if _, h := canvas.MonoCellSize(20, MonoSystem); h != 24 { // 28.9 capped to 23.6
		t.Errorf("loose font: height %v, want 24", h)
	}
	if _, h := canvas.MonoCellSize(20, MonoClassic); h != 22 {
		t.Errorf("tight font: height %v, want its own 22", h)
	}
}

func TestCellSizeUsesTheMeasurerAndRoundsHeightUp(t *testing.T) {
	var canvas Canvas
	if w, h := canvas.CellSize(0); w != 0 || h != 0 {
		t.Errorf("zero size = %v,%v", w, h)
	}
	if w, h := canvas.CellSize(10); w <= 0 || h < 10 {
		t.Errorf("estimate = %v,%v", w, h)
	}
	calls := 0
	canvas.SetCellMeasurer(func(size float32, face MonoFace) (float32, float32) {
		calls++
		if face == MonoClassic {
			return size * 0.62, size*1.3 + 0.1
		}
		return size * 0.6, size*1.2 + 0.1
	})
	w, h := canvas.CellSize(13)
	if w != 13*0.6 || h != 16 { // 15.7 rounds up to a whole pixel row
		t.Errorf("cell = %v,%v", w, h)
	}
	canvas.CellSize(13)
	canvas.CellSize(13)
	if calls != 1 {
		t.Errorf("the measurer ran %d times for one size", calls)
	}
	canvas.CellSize(14)
	if calls != 2 {
		t.Errorf("a new size did not measure: %d", calls)
	}
	classicWidth, classicHeight := canvas.MonoCellSize(14, MonoClassic)
	if classicWidth != 14*0.62 || classicHeight != 17 || calls != 3 { // 18.3 is capped at 1.18 sizes, 16.5, then rounded up
		t.Errorf("classic cell = %v,%v calls=%d", classicWidth, classicHeight, calls)
	}
	canvas.MonoCellSize(14, MonoClassic)
	if calls != 3 {
		t.Errorf("classic cell was not memoized: %d", calls)
	}
	canvas.SetCellMeasurer(func(float32, MonoFace) (float32, float32) { return 0, 0 })
	if w, h := canvas.CellSize(13); w <= 0 || h <= 0 {
		t.Errorf("a measurer that cannot say should fall back to the estimate: %v,%v", w, h)
	}
}
