package components

import (
	"slices"
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestFieldReturnsStructuralControlSlotAndPaintsMetadata(t *testing.T) {
	theme := ui.LightTheme()
	var canvas ui.Canvas
	control := BeginField(&canvas, theme, ui.Rect{X: 4, Y: 8, Width: 220}).Required().Invalid().Paint("Email", "Used for receipts", "Enter a valid address")
	if control.Y <= 8 || control.Height != theme.Metrics.ControlHeight {
		t.Fatalf("control slot = %+v", control)
	}
	if len(canvas.Rectangles()) != 0 || len(canvas.Hits()) != 0 {
		t.Fatal("structural field must not impersonate a control")
	}
	if len(canvas.Texts()) != 4 || canvas.Texts()[0].Style.Color != theme.Danger || canvas.Texts()[3].Style.Color != theme.Danger {
		t.Fatalf("field texts = %+v", canvas.Texts())
	}
}

func TestFieldResponsiveAndSeparatorCommandsAreStable(t *testing.T) {
	var canvas ui.Canvas
	paint := func() ui.Rect {
		canvas.Reset()
		control := BeginField(&canvas, ui.DarkTheme(), ui.Rect{Width: 420, Height: 80}).Orientation(FieldResponsive).Disabled().Paint("Name", "Shown publicly", "")
		FieldSeparator(&canvas, ui.DarkTheme(), ui.Rect{Y: 70, Width: 420, Height: 20}, "or")
		return control
	}
	control := paint()
	if control.X <= 0 {
		t.Fatalf("responsive wide field stayed vertical: %+v", control)
	}
	rectangles := slices.Clone(canvas.Rectangles())
	texts := slices.Clone(canvas.Texts())
	paint()
	if !slices.Equal(rectangles, canvas.Rectangles()) || !slices.Equal(texts, canvas.Texts()) {
		t.Fatal("identical fields produced different command streams")
	}
}

// Field and PaintFormField are two recipes for the same thing; their rows must
// sit at the same offsets or a form built from one looks off beside the other.
func TestFieldAndFormFieldShareTheirVerticalRhythm(t *testing.T) {
	theme := ui.LightTheme()
	var field, form ui.Canvas
	control := BeginField(&field, theme, ui.Rect{Width: 200}).Paint("Email", "Used for receipts", "Enter a valid address")
	layout := PaintFormField(&form, theme, ui.Rect{Width: 200}, FormFieldSnapshot{Label: "Email", Description: "Used for receipts", Error: "Enter a valid address"})
	if control != layout.Control {
		t.Fatalf("field control %+v, form control %+v", control, layout.Control)
	}
	ft, gt := field.Texts(), form.Texts()
	if len(ft) != 3 || len(gt) != 3 {
		t.Fatalf("texts: field %d, form %d", len(ft), len(gt))
	}
	for index := range ft {
		if ft[index].Bounds != gt[index].Bounds || ft[index].Style.Size != gt[index].Style.Size {
			t.Errorf("text %d: field %+v/%v, form %+v/%v", index, ft[index].Bounds, ft[index].Style.Size, gt[index].Bounds, gt[index].Style.Size)
		}
	}
}

func TestFieldErrorListsDistinctMessages(t *testing.T) {
	theme := ui.LightTheme()
	var one, many, none ui.Canvas
	single := FieldError(&one, theme, ui.Rect{Width: 200}, []string{"Too short", "", "Too short"})
	list := FieldError(&many, theme, ui.Rect{Width: 200}, []string{"Too short", "No digit", "Too short", "No symbol"})
	empty := FieldError(&none, theme, ui.Rect{Width: 200}, []string{"", ""})
	if len(one.Texts()) != 1 || len(one.Rectangles()) != 0 {
		t.Fatalf("one distinct error painted %d texts and %d bullets", len(one.Texts()), len(one.Rectangles()))
	}
	if len(many.Texts()) != 3 || len(many.Rectangles()) != 3 {
		t.Fatalf("three distinct errors painted %d texts and %d bullets", len(many.Texts()), len(many.Rectangles()))
	}
	if list.Height <= single.Height*2 || empty.Height != 0 || len(none.Texts()) != 0 {
		t.Fatalf("heights: one %v, list %v, none %v", single.Height, list.Height, empty.Height)
	}
}

func TestFieldStackSpacesRowsAndCountsThem(t *testing.T) {
	theme := ui.LightTheme()
	set := BeginFieldSet(theme, ui.Rect{X: 4, Y: 10, Width: 300})
	first, second := set.Next(50), set.Next(50)
	if first.Y != 10 || second.Y <= first.Y+first.Height || second.Width != 300 || second.X != 4 {
		t.Fatalf("rows = %+v, %+v", first, second)
	}
	gap := second.Y - first.Y - first.Height
	group := BeginFieldGroup(theme, ui.Rect{Width: 300})
	group.Next(50)
	if after := group.Next(50); after.Y-50 <= gap {
		t.Fatalf("a group's rows are %v apart, a set's %v; a group is the looser one", after.Y-50, gap)
	}
	if set.Height() != 100+gap {
		t.Fatalf("height = %v, want the rows and the gap between them", set.Height())
	}
}

func TestFieldTitleLabelsItsControlOnlyWhenAskedTo(t *testing.T) {
	theme := ui.LightTheme()
	control := ui.Item(5)
	bounds := ui.Rect{X: 10, Y: 10, Width: 300, Height: 150}
	var plain ui.Canvas
	BeginField(&plain, theme, bounds).Paint("Email", "We never share it.", "")
	if len(plain.Hits()) != 0 {
		t.Fatalf("a field without For declared %d hits", len(plain.Hits()))
	}

	cases := map[string]func(canvas *ui.Canvas){
		"Field": func(canvas *ui.Canvas) { BeginField(canvas, theme, bounds).For(control).Paint("Email", "", "") },
		"horizontal": func(canvas *ui.Canvas) {
			BeginField(canvas, theme, bounds).For(control).Orientation(FieldHorizontal).Paint("Email", "", "")
		},
		"form": func(canvas *ui.Canvas) {
			PaintFormField(canvas, theme, bounds, FormFieldSnapshot{Label: "Email", ControlID: control})
		},
		"label": func(canvas *ui.Canvas) {
			BeginLabel(canvas, theme, ui.Rect{X: 10, Y: 10, Width: 300}).For(control).Paint("Email")
		},
		"title": func(canvas *ui.Canvas) {
			FieldTitle(canvas, theme, ui.Rect{X: 10, Y: 10, Width: 300, Height: 20}, "Email", control, false)
		},
	}
	for name, paint := range cases {
		var canvas ui.Canvas
		paint(&canvas)
		hits := canvas.Hits()
		if len(hits) != 1 || hits[0].ID != control || !hits[0].IsLabel || hits[0].Focusable || hits[0].Label != "Email" {
			t.Fatalf("%s: hits %+v, want one non-focusable label of the control", name, hits)
		}
		if hits[0].Bounds.Width >= 300 {
			t.Fatalf("%s: the label hit spans %v, the whole row, not just its words", name, hits[0].Bounds.Width)
		}
	}
}
