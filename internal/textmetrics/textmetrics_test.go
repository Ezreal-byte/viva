package textmetrics

import (
	"strings"
	"testing"
)

// The point of the package: these numbers are the same on every platform, so the
// test runs on all three with the same constants.
func TestWidthIsTheSameEverywhere(t *testing.T) {
	for _, c := range []struct {
		value        string
		size, weight float32
		want         float32
	}{
		{"Connect", 14, 0.15, 54.38},
		{"The quick brown fox jumps over the lazy dog.", 13, 0, 270.23},
		{"界面一致性测试", 14, 0, 98},
		{"Save", 12, 0.4, 28.0},
	} {
		got := Width(c.value, c.size, c.weight)
		if d := got - c.want; d < -0.3 || d > 0.3 {
			t.Errorf("Width(%q, %g, %g) = %.2f, want %.1f", c.value, c.size, c.weight, got, c.want)
		}
	}
}

// Caret and selection positions are widths of prefixes: they must add up, or the
// caret moves when a character is typed after it.
func TestWidthAddsUpAcrossASplit(t *testing.T) {
	for _, value := range []string{"Hello, world", "Écrire à Zoë — 日本語 ok", "a  b", "x😀y", "ß ñ Ж Ω"} {
		for _, weight := range []float32{0, 0.15, 0.23, 0.4} {
			whole := Width(value, 14, weight)
			for at := range value {
				sum := Width(value[:at], 14, weight) + Width(value[at:], 14, weight)
				if d := whole - sum; d < -0.01 || d > 0.01 {
					t.Fatalf("%q split at %d, weight %g: %.3f whole, %.3f in parts", value, at, weight, whole, sum)
				}
			}
		}
	}
}

func TestWidthRules(t *testing.T) {
	const size = 20
	approx := func(name string, got, want float32) {
		t.Helper()
		if d := got - want; d < -0.05 || d > 0.05 {
			t.Errorf("%s: %.3f, want %.3f", name, got, want)
		}
	}
	approx("CJK is one em a character", Width("漢字かなカナ", size, 0), 6*size)
	approx("fullwidth form", Width("ＡＢ", size, 0), 2*size)
	approx("half-width katakana is half an em", Width("ｱｲ", size, 0), size)
	approx("combining accent adds nothing", Width("é", size, 0), Width("e", size, 0))
	approx("variation selector adds nothing", Width("❤️", size, 0), Width("❤", size, 0))
	approx("control characters have no width", Width("a\tb\n", size, 0), Width("ab", size, 0))
	approx("an emoji with a skin tone is one emoji", Width("👍🏽", size, 0), Width("👍", size, 0))
	approx("a joined sequence is one emoji", Width("👨‍👩‍👧", size, 0), Width("👨", size, 0))
	approx("an unknown script has a class width", Width("አማርኛ", size, 0), 4*0.6*size)
	if Width("Hamburgefonstiv", size, 0.23) <= Width("Hamburgefonstiv", size, 0) {
		t.Error("semibold is not wider than regular")
	}
	approx("width is linear in the size", Width("Settings", 2*size, 0), 2*Width("Settings", size, 0))
}

// Every full-width character is a piece of its own, with its marks: it is anchored at its
// layout x, so a CJK font whose advance is not one em never moves the characters after it.
func TestEveryFullWidthCharacterIsAPiece(t *testing.T) {
	const value = "a字\u0301b 日本x"
	var got []string
	for from := 0; ; {
		start, end, ok := NextRun(value, from)
		if !ok {
			break
		}
		got = append(got, value[start:end])
		from = end
	}
	if want := []string{"a", "字\u0301", "b", "日", "本", "x"}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("pieces %q, want %q", got, want)
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func BenchmarkWidth(b *testing.B) {
	for _, c := range []struct{ name, value string }{
		{"ascii", "The quick brown fox jumps over the lazy dog."},
		{"cjk", "界面一致性测试界面一致性测试"},
		{"long", strings.Repeat("Lorem ipsum dolor sit amet, ", 20)},
	} {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			var sink float32
			for b.Loop() {
				sink += Width(c.value, 14, 0.15)
			}
			_ = sink
		})
	}
}
