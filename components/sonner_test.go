package components

import (
	"slices"
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestSonnerPaintsCallerOwnedStackNewestOnTop(t *testing.T) {
	var canvas ui.Canvas
	toasts := []SonnerToast{
		{Snapshot: ToastSnapshot{Title: "Newest", Index: 99}, Scope: ui.Item(10), DismissID: ui.Item(1)},
		{Snapshot: ToastSnapshot{Title: "Middle", Index: 99}, Scope: ui.Item(11), DismissID: ui.Item(2)},
		{Snapshot: ToastSnapshot{Title: "Oldest", Index: 99}, Scope: ui.Item(12), DismissID: ui.Item(3)},
	}
	layout := PaintSonner(&canvas, ui.LightTheme(), ui.Rect{Width: 1000, Height: 700}, toasts, SonnerOptions{Layer: 40})
	if layout != (SonnerLayout{Bounds: ui.Rect{X: 600, Y: 596, Width: 384, Height: 88}, Painted: 3}) {
		t.Fatalf("stack layout = %+v", layout)
	}
	if canvas.EscapeTarget() != ui.Item(1) {
		t.Fatalf("escape target = %d", canvas.EscapeTarget())
	}
	if len(canvas.Hits()) != 3 || canvas.Hits()[0].ID != ui.Item(3) || canvas.Hits()[2].ID != ui.Item(1) {
		t.Fatalf("stack hits = %+v", canvas.Hits())
	}
	seenLayer := [3]bool{}
	for _, rectangle := range canvas.Rectangles() {
		if rectangle.Layer < 40 || rectangle.Layer > 42 {
			t.Fatalf("rectangle layer = %d", rectangle.Layer)
		}
		seenLayer[rectangle.Layer-40] = true
	}
	if seenLayer != [3]bool{true, true, true} {
		t.Fatalf("layers = %v", seenLayer)
	}
	for _, toast := range toasts {
		if toast.Snapshot.Index != 99 {
			t.Fatal("caller snapshot was mutated")
		}
	}
}

func TestSonnerExpandedGeometryAndLimitedEntry(t *testing.T) {
	var canvas ui.Canvas
	layout := PaintSonner(&canvas, ui.DarkTheme(), ui.Rect{X: 2, Y: 3, Width: 800, Height: 600}, []SonnerToast{
		{Snapshot: ToastSnapshot{Title: "One", Description: "Details"}, DismissID: ui.Item(1)},
		{Snapshot: ToastSnapshot{Title: "Hidden", Limited: true}, DismissID: ui.Item(2)},
		{Snapshot: ToastSnapshot{Title: "Three", Description: "Details"}, DismissID: ui.Item(3)},
	}, SonnerOptions{Layer: 8, Expanded: true})
	if layout.Painted != 2 {
		t.Fatalf("painted = %d", layout.Painted)
	}
	if layout.Bounds != (ui.Rect{X: 402, Y: 419, Width: 384, Height: 168}) {
		t.Fatalf("expanded bounds = %+v", layout.Bounds)
	}
}

func TestSonnerPaintIsDeterministicAndAllocationFreeAfterWarmup(t *testing.T) {
	var canvas ui.Canvas
	toasts := []SonnerToast{
		{Snapshot: ToastSnapshot{Title: "Saved", Type: ToastSuccess}, Scope: ui.Item(20), DismissID: ui.Item(1)},
		{Snapshot: ToastSnapshot{Title: "Saving", Type: ToastLoading, SwipeX: 4}, Scope: ui.Item(21), DismissID: ui.Item(2), ActionID: ui.Item(3), ActionLabel: "Cancel"},
	}
	paint := func() {
		canvas.Reset()
		PaintSonner(&canvas, ui.LightTheme(), ui.Rect{Width: 900, Height: 640}, toasts, SonnerOptions{Layer: 16})
	}
	paint()
	rectangles := slices.Clone(canvas.Rectangles())
	texts := slices.Clone(canvas.Texts())
	hits := slices.Clone(canvas.Hits())
	paint()
	if !slices.Equal(rectangles, canvas.Rectangles()) || !slices.Equal(texts, canvas.Texts()) || !slices.Equal(hits, canvas.Hits()) {
		t.Fatal("sonner stream changed")
	}
	if allocations := testing.AllocsPerRun(100, paint); allocations != 0 {
		t.Fatalf("sonner allocations = %v", allocations)
	}
}

func TestSonnerShowsOnlyTheNewestLimitToasts(t *testing.T) {
	var canvas ui.Canvas
	toasts := make([]SonnerToast, 5)
	for index := range toasts {
		toasts[index] = SonnerToast{Snapshot: ToastSnapshot{Title: "Toast"}, DismissID: ui.Item(index + 1)}
	}
	if layout := PaintSonner(&canvas, ui.LightTheme(), ui.Rect{Width: 1000, Height: 700}, toasts, SonnerOptions{Layer: 1}); layout.Painted != 3 {
		t.Fatalf("painted = %d, want Base UI's default limit of 3", layout.Painted)
	}
	canvas.Reset()
	if layout := PaintSonner(&canvas, ui.LightTheme(), ui.Rect{Width: 1000, Height: 700}, toasts, SonnerOptions{Layer: 1, Limit: 5}); layout.Painted != 5 {
		t.Fatalf("painted = %d with Limit 5", layout.Painted)
	}
}
