package components

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/vivaui/viva/markdown"
	"github.com/vivaui/viva/ui"
)

// markdownCanvas measures every rune as half its size wide, and every cell of
// fixed-pitch text as 6 wide at size 12, so widths are counts of characters.
func markdownCanvas() *ui.Canvas {
	canvas := &ui.Canvas{}
	canvas.SetMeasurer(func(value string, size, weight float32) float32 {
		return float32(utf8.RuneCountInString(value)) * size / 2
	})
	canvas.SetCellMeasurer(func(size float32, face ui.MonoFace) (float32, float32) { return size / 2, size * 1.25 })
	return canvas
}

func layoutMarkdown(t *testing.T, src string, width float32, style MarkdownStyle) (*MarkdownView, *markdown.Document) {
	t.Helper()
	var doc markdown.Document
	doc.Parse(src)
	view := &MarkdownView{}
	view.Layout(markdownCanvas(), style, &doc, width)
	return view, &doc
}

// words returns the non-blank characters of the pieces, in order.
func (v *MarkdownView) words() string {
	var out strings.Builder
	for _, p := range v.pieces {
		out.WriteString(strings.Join(strings.Fields(p.text), ""))
	}
	return out.String()
}

func TestMarkdownWrapsEveryParagraphIntoItsWidthWithoutLosingText(t *testing.T) {
	src := "A long paragraph with **bold words**, `inline code` and a [link to somewhere](https://example.com/x), 中文也要能换行，而且标点不能出现在行首。\n" +
		"- an item that is long enough to need a second line to be shown\n" +
		"> a quote that is long enough to wrap as well, inside its bar\n" +
		"Averyveryveryverylongwordthatdoesnotfitanywhereatall"
	const width = 200
	view, _ := layoutMarkdown(t, src, width, MarkdownStyle{Size: 12})
	want := strings.Join(strings.Fields(strings.NewReplacer("**", "", "`", "", "[link to somewhere](https://example.com/x)", "linktosomewhere", "- ", "", "> ", "").Replace(src)), "")
	if got := view.words(); strings.ReplaceAll(got, "•", "") != want {
		t.Errorf("text changed while wrapping:\n got  %q\n want %q", got, want)
	}
	for _, p := range view.pieces {
		if p.x < 0 || p.x+p.w > width+12 { // closing punctuation may hang a glyph past the edge
			t.Errorf("piece %q at %v..%v leaves the %d-wide column", p.text, p.x, p.x+p.w, width)
		}
	}
	// Pieces on one line never overlap.
	for i, a := range view.pieces {
		for _, b := range view.pieces[i+1:] {
			if a.y == b.y && a.x < b.x+b.w-0.5 && b.x < a.x+a.w-0.5 {
				t.Errorf("%q and %q overlap on one line", a.text, b.text)
			}
		}
	}
}

func TestMarkdownLinkRegionsFollowTheWrappedText(t *testing.T) {
	view, _ := layoutMarkdown(t, "see [the very long link text here](https://example.com/a) now", 150, MarkdownStyle{Size: 12})
	if len(view.links) < 2 {
		t.Fatalf("a link that wraps should have a region on each line, got %d", len(view.links))
	}
	var canvas ui.Canvas
	scope := ui.Scope(7)
	view.Paint(&canvas, MarkdownPaint{Look: MarkdownLook{}, X: 10, Y: 100, Links: scope})
	if len(canvas.Hits()) != len(view.links) {
		t.Fatalf("%d hit regions for %d link fragments", len(canvas.Hits()), len(view.links))
	}
	for _, hit := range canvas.Hits() {
		index, ok := hit.ID.In(scope)
		if !ok || view.Link(index) != "https://example.com/a" || hit.Bounds.X < 10 || hit.Bounds.Y < 100 {
			t.Errorf("hit %+v does not lead to the link", hit)
		}
	}
	// Scrolled out of view, a link has no region.
	canvas.Reset()
	view.Paint(&canvas, MarkdownPaint{X: 10, Y: 100, Links: scope, Clip: ui.Rect{Y: 1000, Width: 100, Height: 100}})
	if len(canvas.Hits()) != 0 || len(canvas.Texts()) != 0 {
		t.Errorf("painted %d texts and %d hits outside the clip", len(canvas.Texts()), len(canvas.Hits()))
	}
}

func TestMarkdownCodeBlocksAreReportedAndKeepRoomForAButton(t *testing.T) {
	src := "text\n```sh\nls -la /var/log/some/very/long/path/that/must/wrap\necho 你好\n```\n"
	view, _ := layoutMarkdown(t, src, 200, MarkdownStyle{Size: 12, CodeInset: 30})
	codes := view.Codes()
	if len(codes) != 1 || !strings.HasPrefix(codes[0].Text, "ls -la") {
		t.Fatalf("codes = %+v", codes)
	}
	box := codes[0].Bounds
	if box.Width != 200 || box.Y <= 0 {
		t.Errorf("block is at %+v", box)
	}
	var lines int
	for _, p := range view.pieces {
		if !p.mono {
			continue
		}
		lines++
		if p.x+p.w > box.X+box.Width-30 || p.y < box.Y || p.y+p.h > box.Y+box.Height {
			t.Errorf("code line %q at %v..%v is outside the text area of %+v", p.text, p.x, p.x+p.w, box)
		}
	}
	if lines < 3 {
		t.Errorf("the long line should have wrapped: %d lines", lines)
	}
}

func TestMarkdownTableFitsItsWidthAndWrapsItsCells(t *testing.T) {
	src := "| Name | Description |\n|:--|--:|\n| ssh | the secure shell protocol and the program that speaks it to remote hosts |\n| ftp | old |\n"
	view, doc := layoutMarkdown(t, src, 180, MarkdownStyle{Size: 12})
	var frame mdBox // the header band spans the table
	for _, box := range view.boxes {
		if box.kind == mdBoxHead {
			frame = box
		}
	}
	if frame.w == 0 || frame.w > 180.5 {
		t.Fatalf("table is %v wide in a 180 column", frame.w)
	}
	for _, p := range view.pieces {
		if p.x+p.w > frame.x+frame.w+0.5 {
			t.Errorf("%q sticks out of the table", p.text)
		}
	}
	if !strings.Contains(view.words(), "secureshellprotocol"[:6]) {
		t.Errorf("cell text is missing: %q", view.words())
	}
	_ = doc
	// The right-aligned column ends at its cell's right edge.
	var oldRight float32
	for _, p := range view.pieces {
		if p.text == "old" {
			oldRight = p.x + p.w
		}
	}
	if oldRight < frame.x+frame.w-markdownCellPadX-1 {
		t.Errorf("right-aligned cell ends at %v, the table's edge is %v", oldRight, frame.x+frame.w)
	}
}

func TestMarkdownQuoteBarRunsThroughConsecutiveQuotedBlocks(t *testing.T) {
	view, _ := layoutMarkdown(t, "> first\n> - second\n\nplain\n\n> again", 200, MarkdownStyle{Size: 12})
	var bars []mdBox
	for _, box := range view.boxes {
		if box.kind == mdBoxBar {
			bars = append(bars, box)
		}
	}
	if len(bars) != 2 {
		t.Fatalf("want one bar for the first quote and one for the second, got %d", len(bars))
	}
	if bars[0].y+bars[0].h > bars[1].y {
		t.Errorf("bars overlap: %+v %+v", bars[0], bars[1])
	}
}

func TestMarkdownLayoutAndPaintDoNotAllocateOnceWarm(t *testing.T) {
	src := strings.Repeat("## Title\nSome **bold** and `code` with [a link](https://example.com), 中文。\n1. one\n2. two\n> quote\n| a | b |\n|---|---|\n| 1 | 2 |\n\n```go\ncode\n```\n", 8)
	var doc markdown.Document
	doc.Parse(src)
	canvas := markdownCanvas()
	var view MarkdownView
	look := MarkdownLookFor(ui.DarkTheme())
	run := func() {
		canvas.Reset()
		view.Layout(canvas, MarkdownStyle{Size: 13}, &doc, 400)
		view.Paint(canvas, MarkdownPaint{Look: look, Links: ui.Scope(3)})
	}
	run()
	if allocs := testing.AllocsPerRun(10, run); allocs != 0 {
		t.Errorf("a warm layout and paint allocates %v times", allocs)
	}
}

func BenchmarkMarkdownLayout(b *testing.B) {
	src := strings.Repeat("## Title\nSome **bold** and `code` with [a link](https://example.com), 中文。\n- item one\n- item two\n> quote\n\n```go\ncode\n```\n", 20)
	var doc markdown.Document
	doc.Parse(src)
	canvas := markdownCanvas()
	var view MarkdownView
	b.ReportAllocs()
	for b.Loop() {
		view.Layout(canvas, MarkdownStyle{Size: 13}, &doc, 400)
	}
}

func TestHighlightedCodeIsDrawnInColoredRunsThatKeepTheText(t *testing.T) {
	// A toy highlighter: every "#" to the end of the line is a comment, and the
	// quoted words are strings.
	highlight := func(language, line string, dst []MarkdownCodeRun) []MarkdownCodeRun {
		if language != "sh" {
			return dst
		}
		if at := strings.Index(line, "#"); at >= 0 {
			dst = append(dst, MarkdownCodeRun{Start: at, End: len(line), Class: CodeComment})
		}
		return dst
	}
	src := "```sh\necho 你好 # 注释 and a very long comment that has to wrap somewhere\n```\n```text\nplain\n```\n"
	var doc markdown.Document
	doc.Parse(src)
	var view MarkdownView
	view.Layout(markdownCanvas(), MarkdownStyle{Size: 12, Highlight: highlight}, &doc, 220)
	var canvas ui.Canvas
	look := MarkdownLookFor(ui.DarkTheme())
	look.Code[CodeComment] = ui.Color{Red: 1, Alpha: 1}
	view.Paint(&canvas, MarkdownPaint{Look: look})

	var joined strings.Builder
	coloured := 0
	for _, text := range canvas.Texts() {
		runs := canvas.TextRuns(text)
		if len(runs) == 0 {
			continue
		}
		joined.WriteString(text.Value)
		bytes := 0
		for _, run := range runs {
			bytes += int(run.Bytes)
			if run.Color == look.Code[CodeComment] {
				coloured++
			}
		}
		if bytes != len(text.Value) {
			t.Errorf("runs cover %d of %d bytes of %q", bytes, len(text.Value), text.Value)
		}
	}
	if coloured == 0 || !strings.Contains(joined.String(), "echo") {
		t.Fatalf("no colored comment runs: coloured=%d text=%q", coloured, joined.String())
	}
	// A block the highlighter ignores is drawn as it was: one plain label.
	plain := 0
	for _, text := range canvas.Texts() {
		if text.Value == "plain" && len(canvas.TextRuns(text)) == 0 {
			plain++
		}
	}
	if plain != 1 {
		t.Errorf("the unhighlighted block is not drawn as plain text")
	}
}

func TestTabsInCodeBecomeBlanksSoColumnsAreTrue(t *testing.T) {
	var doc markdown.Document
	doc.Parse("```go\n\tfmt.Println(1)\n\t\tx\tend\n```\n")
	var view MarkdownView
	view.Layout(markdownCanvas(), MarkdownStyle{Size: 12}, &doc, 300)
	var lines []string
	for _, piece := range view.pieces {
		if piece.mono {
			lines = append(lines, piece.text)
		}
	}
	want := []string{"    fmt.Println(1)", "        x   end"}
	if len(lines) != 2 || lines[0] != want[0] || lines[1] != want[1] {
		t.Fatalf("code lines = %q, want %q", lines, want)
	}
	if codes := view.Codes(); len(codes) != 1 || !strings.Contains(codes[0].Text, "\t") {
		t.Errorf("the copied text lost its tabs: %+v", codes)
	}
}
