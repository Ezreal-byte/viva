package components

import (
	"slices"
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestBreadcrumbPaintsLinksDisabledAndCurrentItems(t *testing.T) {
	theme := ui.LightTheme()
	var canvas ui.Canvas
	canvas.SetInteraction(ui.Item(11), ui.ID{}, ui.ID{}, false)
	breadcrumb := BeginBreadcrumb(&canvas, theme, ui.Rect{Width: 320})
	breadcrumb.AddLink(ui.Item(11), "Projects")
	breadcrumb.AddDisabled("Archive")
	breadcrumb.AddCurrent("October")
	if len(canvas.Texts()) != 3 || len(canvas.Triangles()) == 0 {
		t.Fatalf("breadcrumb texts = %d, want the 3 items (separators are icons)", len(canvas.Texts()))
	}
	if len(canvas.Hits()) != 1 || canvas.Hits()[0].ID != ui.Item(11) || canvas.Hits()[0].Label != "Projects" {
		t.Fatalf("breadcrumb hits = %+v", canvas.Hits())
	}
	if canvas.Texts()[0].Style.Color != theme.Foreground {
		t.Fatalf("hovered link color = %+v, want Foreground", canvas.Texts()[0].Style.Color)
	}
	if canvas.Hits()[0].Role != ui.RoleLink || !canvas.Hits()[0].Focusable {
		t.Fatalf("breadcrumb link semantics = %+v", canvas.Hits()[0])
	}
	if canvas.Texts()[2].Style.Weight != ui.WeightRegular || canvas.Texts()[2].Style.Color != theme.Foreground {
		t.Fatalf("current item style = %+v", canvas.Texts()[4].Style)
	}
}

func TestBreadcrumbEllipsisIsAButtonOnlyWithAnID(t *testing.T) {
	paint := func(id ui.ID) *ui.Canvas {
		canvas := new(ui.Canvas)
		breadcrumb := BeginBreadcrumb(canvas, ui.LightTheme(), ui.Rect{Width: 200})
		breadcrumb.AddLink(ui.Item(1), "Home")
		breadcrumb.AddEllipsis(id)
		breadcrumb.AddCurrent("Page")
		return canvas
	}
	if canvas := paint(ui.ID{}); len(canvas.Hits()) != 1 || len(canvas.Texts()) != 2 {
		t.Fatalf("presentational ellipsis: %d hits, %d texts", len(canvas.Hits()), len(canvas.Texts()))
	}
	canvas := paint(ui.Item(2))
	if len(canvas.Hits()) != 2 || canvas.Hits()[1].ID != ui.Item(2) || canvas.Hits()[1].Role != ui.RoleButton {
		t.Fatalf("ellipsis button hits = %+v", canvas.Hits())
	}
	if home, more, page := canvas.Hits()[0].Bounds, canvas.Hits()[1].Bounds, canvas.Texts()[1].Bounds; more.X < home.X+home.Width || page.X < more.X+more.Width {
		t.Fatalf("ellipsis %+v is not between Home %+v and Page %+v", more, home, page)
	}
}

func TestBreadcrumbRepeatedPaintIsStableAndAllocationFree(t *testing.T) {
	var canvas ui.Canvas
	paint := func() {
		canvas.Reset()
		breadcrumb := BeginBreadcrumb(&canvas, ui.DarkTheme(), ui.Rect{Width: 320})
		breadcrumb.AddLink(ui.Item(1), "Home")
		breadcrumb.AddLink(ui.Item(2), "Components")
		breadcrumb.AddCurrent("Breadcrumb")
	}
	paint()
	texts := slices.Clone(canvas.Texts())
	hits := slices.Clone(canvas.Hits())
	paint()
	if !slices.Equal(texts, canvas.Texts()) || !slices.Equal(hits, canvas.Hits()) {
		t.Fatal("identical breadcrumbs produced different command streams")
	}
	if allocations := testing.AllocsPerRun(100, paint); allocations != 0 {
		t.Fatalf("breadcrumb paint allocations = %v, want 0 after warm-up", allocations)
	}
}

func TestBreadcrumbDoesNotLeaveDanglingSeparator(t *testing.T) {
	var canvas ui.Canvas
	breadcrumb := BeginBreadcrumb(&canvas, ui.LightTheme(), ui.Rect{Width: 34})
	breadcrumb.AddCurrent("Home")
	breadcrumb.AddCurrent("Hidden")
	if len(canvas.Texts()) != 1 || canvas.Texts()[0].Value != "Home" || len(canvas.Triangles()) != 0 {
		t.Fatalf("overflow breadcrumb texts = %+v", canvas.Texts())
	}
}

func TestBreadcrumbFontSizeOverridesTheBodyDefault(t *testing.T) {
	theme := ui.DarkTheme()
	paint := func(size float32) []ui.Text {
		var canvas ui.Canvas
		crumbs := BeginBreadcrumb(&canvas, theme, ui.Rect{Width: 300, Height: 24})
		if size > 0 {
			crumbs = crumbs.FontSize(size)
		}
		crumbs.AddLink(ui.Item(1), "home")
		crumbs.AddCurrent("root")
		return canvas.Texts()
	}
	for _, text := range paint(0) {
		if text.Style.Size != theme.Metrics.FontSizeM {
			t.Fatalf("default size = %v, want the theme's body size", text.Style.Size)
		}
	}
	texts := paint(11)
	if len(texts) != 2 {
		t.Fatalf("texts = %+v", texts)
	}
	for _, text := range texts {
		if text.Style.Size != 11 {
			t.Fatalf("text %q has size %v, want 11", text.Value, text.Style.Size)
		}
	}
}
