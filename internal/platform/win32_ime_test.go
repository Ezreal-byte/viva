package platform

import (
	"testing"
	"unicode/utf16"

	"github.com/vivaui/viva/text"
)

func TestReconvertContextIsTheLineAroundTheSelection(t *testing.T) {
	units, start, length, ok := ReconvertContext("first\nhello 世界 world\nlast", text.Range{Start: 12, End: 14}, 64)
	if !ok {
		t.Fatal("a selection reconverts")
	}
	if got := string(utf16.Decode(units)); got != "hello 世界 world" {
		t.Fatalf("context = %q", got)
	}
	if got := string(utf16.Decode(units[start : start+length])); got != "世界" {
		t.Fatalf("selection in the context = %q", got)
	}
}

func TestReconvertContextCountsUTF16Units(t *testing.T) {
	// 😀 is two units; the selection is the letter after it.
	units, start, length, ok := ReconvertContext("😀a", text.Range{Start: 1, End: 2}, 64)
	if !ok || start != 2 || length != 1 || len(units) != 3 {
		t.Fatalf("units %d, start %d, length %d, ok %v", len(units), start, length, ok)
	}
}

func TestReconvertContextStaysWithinItsLimit(t *testing.T) {
	long := ""
	for range 100 {
		long += "abcdefghij"
	}
	units, start, length, ok := ReconvertContext(long, text.Range{Start: 500, End: 505}, 64)
	if !ok || len(units) > 64 || length != 5 || start < 0 || start+length > len(units) {
		t.Fatalf("units %d, start %d, length %d, ok %v", len(units), start, length, ok)
	}
	if string(utf16.Decode(units[start:start+length])) != long[500:505] {
		t.Fatalf("the selection moved: %q", string(utf16.Decode(units[start:start+length])))
	}
	if _, _, _, ok := ReconvertContext(long, text.Range{Start: 0, End: 200}, 64); ok {
		t.Fatal("a selection longer than the limit cannot be reconverted")
	}
}

func TestReconvertContextNeedsASelection(t *testing.T) {
	if _, _, _, ok := ReconvertContext("hello", text.Range{Start: 2, End: 2}, 64); ok {
		t.Fatal("a caret alone reconverts nothing")
	}
}

func TestCompositionTargetIsTheClauseBeingConverted(t *testing.T) {
	// "ab" converted, "cd" the target, "e" not yet converted.
	attrs := []byte{imeAttrConverted, imeAttrConverted, imeAttrTargetConverted, imeAttrTargetConverted, imeAttrInput}
	target, ok := CompositionTarget(attrs, "abcde")
	if !ok || target != (text.Range{Start: 2, End: 4}) {
		t.Fatalf("target = %+v, %v", target, ok)
	}
	if _, ok := CompositionTarget([]byte{0, 0, 0}, "abc"); ok {
		t.Fatal("typing before conversion has no target")
	}
	// A surrogate pair is one attribute pair and one rune.
	pair := []byte{imeAttrInput, imeAttrInput, imeAttrTargetNotConverted}
	if target, ok := CompositionTarget(pair, "😀a"); !ok || target != (text.Range{Start: 1, End: 2}) {
		t.Fatalf("target = %+v, %v", target, ok)
	}
	if target, ok := CompositionTarget([]byte{imeAttrTargetConverted}, "ab"); !ok || target != (text.Range{Start: 0, End: 1}) {
		t.Fatalf("short attributes: %+v, %v", target, ok)
	}
}
