package components

import (
	"slices"
	"testing"

	"github.com/vivaui/viva/internal/input"
	"github.com/vivaui/viva/ui"
)

var nativeSelectTestOptions = [...]NativeSelectOption{
	{Label: "Comfortable", Value: "comfortable"},
	{Label: "Compact", Value: "compact", Disabled: true},
	{Label: "System", Value: "system", Hint: "Follow the OS"},
}

func TestNativeSelectSizesAndComboboxSemantics(t *testing.T) {
	var canvas ui.Canvas
	bounds := PaintNativeSelect(&canvas, ui.LightTheme(), ui.Rect{X: 4, Y: 5, Width: 180}, NativeSelectSnapshot{
		ID: ui.Item(1), Label: "Density", Value: "Compact", Hint: "UI density", Open: true, Focused: true,
	})
	if bounds.Height != 32 {
		t.Fatalf("default height = %v", bounds.Height)
	}
	hit := canvas.Hits()[0]
	if hit.Role != ui.RoleCombobox || hit.Value != "Compact" || hit.Hint != "UI density" || !hit.Expanded {
		t.Fatalf("hit = %+v", hit)
	}
	canvas.Reset()
	small := PaintNativeSelect(&canvas, ui.LightTheme(), ui.Rect{Width: 140}, NativeSelectSnapshot{Size: NativeSelectSmall, Placeholder: "Choose"})
	if small.Height != 28 || canvas.Texts()[0].Value != "Choose" {
		t.Fatalf("small=%+v texts=%+v", small, canvas.Texts())
	}
}

func TestNativeSelectDisabledReadOnlyState(t *testing.T) {
	var canvas ui.Canvas
	PaintNativeSelect(&canvas, ui.DarkTheme(), ui.Rect{Width: 150}, NativeSelectSnapshot{
		ID: ui.Item(4), Label: "Language", Value: "Go", Disabled: true, ReadOnly: true, Invalid: true,
	})
	if hit := canvas.Hits()[0]; !hit.Disabled || !hit.ReadOnly || hit.Focusable || hit.Role != ui.RoleCombobox {
		t.Fatalf("hit = %+v", hit)
	}
	if canvas.Texts()[0].Style.Color.Alpha >= ui.DarkTheme().Foreground.Alpha {
		t.Fatalf("disabled text = %+v", canvas.Texts()[0])
	}
}

func TestNativeSelectCallerOwnedSelectionAndSemanticPaint(t *testing.T) {
	snapshot := NativeSelectSnapshot{
		ID: ui.Item(11), Label: "Density", Hint: "UI density", Options: nativeSelectTestOptions[:],
		Selection: NativeSelectSelection{Index: 2, Valid: true}, Open: true,
		Focused: true, Invalid: true,
	}
	option, ok := snapshot.SelectedOption()
	if !ok || option != nativeSelectTestOptions[2] {
		t.Fatalf("selected option = %+v, %v", option, ok)
	}
	want := NativeSelectSemantics{
		ID: ui.Item(11), Label: "Density", Value: "system", Hint: "UI density",
		Selection: NativeSelectSelection{Index: 2, Valid: true}, OptionCount: 3,
		Focused: true, Open: true, Invalid: true,
	}
	if got := snapshot.Semantics(); got != want {
		t.Fatalf("semantics = %+v, want %+v", got, want)
	}
	var canvas ui.Canvas
	PaintNativeSelect(&canvas, ui.LightTheme(), ui.Rect{Width: 180}, snapshot)
	if text := canvas.Texts()[0].Value; text != "System" {
		t.Fatalf("display = %q", text)
	}
	if hit := canvas.Hits()[0]; hit.Value != "system" || !hit.Expanded {
		t.Fatalf("hit = %+v", hit)
	}
	canvas.Reset()
	PaintNativeSelectSemantics(&canvas, ui.Rect{Width: 180, Height: 32}, snapshot)
	if len(canvas.Rectangles()) != 0 || len(canvas.Texts()) != 0 || len(canvas.Hits()) != 1 {
		t.Fatalf("semantic-only stream: rectangles=%+v texts=%+v hits=%+v", canvas.Rectangles(), canvas.Texts(), canvas.Hits())
	}
}

func TestNativeSelectCommandHelperSkipsDisabledOptions(t *testing.T) {
	snapshot := NativeSelectSnapshot{
		Options: nativeSelectTestOptions[:], Selection: NativeSelectSelection{Index: 0, Valid: true},
	}
	if command, handled := NativeSelectCommandForKey(snapshot, input.KeyDown); !handled || command != (NativeSelectCommand{
		Kind: NativeSelectCommandSelect, Selection: NativeSelectSelection{Index: 2, Valid: true},
	}) {
		t.Fatalf("down = (%+v, %v)", command, handled)
	}
	if command, handled := NativeSelectCommandForKey(snapshot, input.KeyEnter); !handled || command.Kind != NativeSelectCommandOpen {
		t.Fatalf("enter = (%+v, %v)", command, handled)
	}
	snapshot.Open = true
	if command, handled := NativeSelectCommandForKey(snapshot, input.KeyEscape); !handled || command.Kind != NativeSelectCommandClose {
		t.Fatalf("escape = (%+v, %v)", command, handled)
	}
	for _, inert := range []NativeSelectSnapshot{{Disabled: true}, {ReadOnly: true}} {
		if command, handled := NativeSelectCommandForKey(inert, input.KeyEnter); handled || command.Kind != NativeSelectCommandNone {
			t.Fatalf("inert enter = (%+v, %v)", command, handled)
		}
	}
	if allocations := testing.AllocsPerRun(100, func() {
		_ = snapshot.Semantics()
		_, _ = snapshot.SelectedOption()
		NativeSelectCommandForKey(snapshot, input.KeyDown)
	}); allocations != 0 {
		t.Fatalf("native-select helpers allocations = %v", allocations)
	}
}

func TestNativeSelectInvalidExplicitSelectionUsesPlaceholder(t *testing.T) {
	var canvas ui.Canvas
	snapshot := NativeSelectSnapshot{
		ID: ui.Item(2), Label: "Density", Options: nativeSelectTestOptions[:],
		Selection: NativeSelectSelection{Index: 99, Valid: true}, Value: "legacy", Display: "Legacy", Placeholder: "Choose",
	}
	PaintNativeSelect(&canvas, ui.LightTheme(), ui.Rect{Width: 150}, snapshot)
	if text := canvas.Texts()[0].Value; text != "Choose" {
		t.Fatalf("display = %q", text)
	}
	if semantics := snapshot.Semantics(); semantics.Selection.Valid || semantics.Value != "" {
		t.Fatalf("semantics = %+v", semantics)
	}
}

func TestNativeSelectPaintIsDeterministicAndAllocationFreeAfterWarmup(t *testing.T) {
	var canvas ui.Canvas
	paint := func() {
		canvas.Reset()
		PaintNativeSelect(&canvas, ui.LightTheme(), ui.Rect{X: 1, Y: 2, Width: 160}, NativeSelectSnapshot{
			ID: ui.Item(3), Label: "Framework", Options: nativeSelectTestOptions[:],
			Selection: NativeSelectSelection{Index: 0, Valid: true}, Focused: true, Open: true,
		})
	}
	paint()
	rectangles := slices.Clone(canvas.Rectangles())
	texts := slices.Clone(canvas.Texts())
	hits := slices.Clone(canvas.Hits())
	paint()
	if !slices.Equal(rectangles, canvas.Rectangles()) || !slices.Equal(texts, canvas.Texts()) || !slices.Equal(hits, canvas.Hits()) {
		t.Fatal("identical native-select snapshots produced different streams")
	}
	if allocations := testing.AllocsPerRun(100, paint); allocations != 0 {
		t.Fatalf("native-select paint allocations = %v", allocations)
	}
}
