package components

import (
	"slices"
	"strings"
	"testing"

	"github.com/vivaui/viva/ui"
)

func TestTableUsesHorizontalRulesAndOfficialRowStates(t *testing.T) {
	theme := ui.LightTheme()
	columns := []TableColumn{{Width: 120}, {Width: 90, Alignment: TableAlignCenter}, {Alignment: TableAlignRight}}
	var canvas ui.Canvas
	table := BeginTable(&canvas, theme, ui.Rect{Width: 320, Height: 140}, columns)
	table.Header([]string{"Name", "State", "Total"})
	table.RowState(ui.Item(7), []string{"Ada", "Active", "$42"}, TableRowState{Expanded: true})
	table.Footer([]string{"", "Total", "$42"})
	table.Caption("Current members")
	if len(canvas.Hits()) != 1 || canvas.Hits()[0].Label != "Ada" {
		t.Fatalf("table hits = %+v", canvas.Hits())
	}
	if canvas.Texts()[5].Bounds.X <= 210 || !canvas.Texts()[0].HasClip {
		t.Fatalf("table cell placement = %+v", canvas.Texts())
	}
	for _, rectangle := range canvas.Rectangles() {
		if rectangle.Bounds.Width == 1 && rectangle.Bounds.Height > 1 {
			t.Fatal("table emitted a vertical cell border")
		}
	}
	if canvas.Texts()[len(canvas.Texts())-1].Value != "Current members" {
		t.Fatal("table caption missing")
	}
}

func TestTableCommandStreamIsStable(t *testing.T) {
	columns := []TableColumn{{Width: 100}, {Width: 0}}
	var canvas ui.Canvas
	paint := func() {
		canvas.Reset()
		table := BeginTable(&canvas, ui.DarkTheme(), ui.Rect{Width: 240, Height: 100}, columns)
		table.Header([]string{"Key", "Value"})
		table.Row(ui.Item(1), []string{"Region", "Europe"}, true)
	}
	paint()
	rectangles := slices.Clone(canvas.Rectangles())
	texts := slices.Clone(canvas.Texts())
	hits := slices.Clone(canvas.Hits())
	paint()
	if !slices.Equal(rectangles, canvas.Rectangles()) || !slices.Equal(texts, canvas.Texts()) || !slices.Equal(hits, canvas.Hits()) {
		t.Fatal("identical tables produced different command streams")
	}
}

func TestTableRowLayoutReturnsReusableCellGeometry(t *testing.T) {
	columns := []TableColumn{{Width: 80}, {Width: 120, Inset: 22}, {Width: 0}}
	storage := make([]ui.Rect, 0, len(columns))
	var canvas ui.Canvas
	table := BeginTable(&canvas, ui.DarkTheme(), ui.Rect{X: 10, Y: 20, Width: 300, Height: 80}, columns)
	table.Header([]string{"", "Name", "Size"})
	layout := table.RowLayout(ui.Item(7), []string{"", "notes.txt", "12 B"}, TableRowState{Selected: true}, storage)
	if layout.Bounds != (ui.Rect{X: 10, Y: 20 + TableHeaderHeight(ui.DarkTheme()), Width: 300, Height: TableRowHeight(ui.DarkTheme())}) || len(layout.Cells) != 3 {
		t.Fatalf("row layout = %+v", layout)
	}
	if layout.Cells[0].Width != 80 || layout.Cells[1].X != 90 || layout.Cells[1].Width != 120 || layout.Cells[2].Width != 100 {
		t.Fatalf("cell geometry = %+v", layout.Cells)
	}
	// A cell without text paints nothing, so the name is the last text of the row.
	texts := canvas.Texts()
	if name := texts[len(texts)-2]; name.Value != "notes.txt" || name.Bounds.X != 90+8+22 {
		t.Fatalf("inset name = %+v", name)
	}
	if &layout.Cells[:1][0] != &storage[:1][0] {
		t.Fatal("row layout did not reuse caller storage")
	}
}

func TestTableEndsCellsThatDoNotFitInAnEllipsis(t *testing.T) {
	theme := ui.DarkTheme()
	columns := []TableColumn{{Width: 70}, {Width: 70, Alignment: TableAlignRight}, {Width: 0}}
	var canvas ui.Canvas
	table := BeginTable(&canvas, theme, ui.Rect{Width: 300, Height: 120}, columns)
	table.Header([]string{"Marker / hash", "Key", "Comment"})
	table.Row(ui.Item(1), []string{"@cert-authority", "ssh-ed25519", "short"}, false)
	texts := canvas.Texts()
	if len(texts) != 6 {
		t.Fatalf("texts = %+v", texts)
	}
	room := columns[0].Width - theme.Metrics.GapItem*2
	for _, index := range []int{0, 3, 4} {
		text := texts[index]
		if !strings.HasSuffix(text.Value, "…") || canvas.TextWidthWeight(text.Value, text.Style.Size, text.Style.Weight) > text.Bounds.Width {
			t.Errorf("cell %d = %q (width %v), want it ended in an ellipsis within its cell", index, text.Value, canvas.TextWidthWeight(text.Value, text.Style.Size, text.Style.Weight))
		}
	}
	if texts[0].Bounds.Width != room {
		t.Errorf("ellipsized cell bounds = %v, want %v", texts[0].Bounds.Width, room)
	}
	if texts[1].Value != "Key" || texts[2].Value != "Comment" || texts[5].Value != "short" {
		t.Errorf("a cell that fits was changed: %+v", texts)
	}
}

func TestTablePaintsCellsThatFitWithoutAllocating(t *testing.T) {
	columns := []TableColumn{{Width: 120}, {Width: 0}}
	var canvas ui.Canvas
	paint := func() {
		canvas.Reset()
		table := BeginTable(&canvas, ui.DarkTheme(), ui.Rect{Width: 300, Height: 120}, columns)
		table.Header([]string{"Name", "Value"})
		table.Row(ui.Item(1), []string{"Region", "Europe"}, false)
	}
	if allocs := testing.AllocsPerRun(100, paint); allocs != 0 {
		t.Fatalf("table of fitting cells allocated %v times per paint", allocs)
	}
}

func TestFitTableHeadingsWidensOnlyColumnsNarrowerThanTheirHeading(t *testing.T) {
	theme := ui.DarkTheme()
	var canvas ui.Canvas
	headings := []string{"", "Marker / hash", "Key type", "Comment", "Fingerprint"}
	columns := []TableColumn{{Width: 42}, {Width: 20}, {Width: 400}, {Width: 20, Inset: 10}, {Width: 0}}
	FitTableHeadings(&canvas, theme, columns, headings)
	pad := theme.Metrics.GapItem * 2
	fitted := canvas.TextWidthWeight("Marker / hash", theme.Metrics.FontSizeM, ui.WeightMedium)
	if columns[0].Width != 42 || columns[2].Width != 400 || columns[4].Width != 0 {
		t.Fatalf("columns that needed no change = %+v", columns)
	}
	if columns[1].Width < fitted+pad || columns[1].Width > fitted+pad+1 {
		t.Fatalf("heading column = %v, want %v for its text and padding", columns[1].Width, fitted+pad)
	}
	if columns[3].Width <= canvas.TextWidthWeight("Comment", theme.Metrics.FontSizeM, ui.WeightMedium)+pad+10-1 {
		t.Fatalf("inset column = %v does not hold the heading after its inset", columns[3].Width)
	}

	canvas.Reset()
	table := BeginTable(&canvas, theme, ui.Rect{Width: 800, Height: 80}, columns)
	table.Header(headings)
	for _, text := range canvas.Texts() {
		if strings.Contains(text.Value, "…") {
			t.Fatalf("fitted heading %q was ellipsized", text.Value)
		}
	}
}

func BenchmarkTablePaint(b *testing.B) {
	for _, bench := range []struct {
		name  string
		width float32
	}{{"fitting", 400}, {"ellipsized", 60}} {
		b.Run(bench.name, func(b *testing.B) {
			var canvas ui.Canvas
			columns := []TableColumn{{Width: bench.width}, {Width: bench.width}, {Width: 0}}
			rows := [][]string{
				{"known_hosts", "ssh-ed25519", "SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY8"},
				{"@cert-authority", "ecdsa-sha2-nistp256", "SHA256:uNiVztksCsDhcc0u9e8BujQXVUpKZIDTMczCvj3tD2s"},
			}
			b.ReportAllocs()
			for b.Loop() {
				canvas.Reset()
				table := BeginTable(&canvas, ui.DarkTheme(), ui.Rect{Width: 1000, Height: 200}, columns)
				table.Header([]string{"Marker / hash", "Key type", "Fingerprint"})
				for _, row := range rows {
					table.Row(ui.Item(1), row, false)
				}
			}
		})
	}
}
