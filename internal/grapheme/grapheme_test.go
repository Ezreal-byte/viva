package grapheme

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

type vector struct {
	text   string
	runes  []rune
	breaks []int // byte offsets
	rbreak []int // rune offsets
	line   string
}

func loadVectors(t testing.TB) []vector {
	f, err := os.Open("testdata/GraphemeBreakTest.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []vector
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		var v vector
		v.line = strings.Join(fields, " ")
		var sb strings.Builder
		for _, f := range fields {
			switch f {
			case "÷", "×":
				if f == "÷" {
					v.breaks = append(v.breaks, sb.Len())
					v.rbreak = append(v.rbreak, len(v.runes))
				}
			default:
				cp, err := strconv.ParseUint(f, 16, 32)
				if err != nil {
					t.Fatalf("bad field %q", f)
				}
				sb.WriteRune(rune(cp))
				v.runes = append(v.runes, rune(cp))
			}
		}
		v.text = sb.String()
		out = append(out, v)
	}
	if len(out) < 500 {
		t.Fatalf("only %d vectors", len(out))
	}
	return out
}

func TestConformance(t *testing.T) {
	for _, v := range loadVectors(t) {
		isBreak := map[int]bool{}
		for _, b := range v.breaks {
			isBreak[b] = true
		}
		for i := 0; i <= len(v.text); i++ {
			if utf8.RuneStart(byte0(v.text, i)) && Boundary(v.text, i) != isBreak[i] {
				t.Fatalf("%s: Boundary(%d)=%v", v.line, i, !isBreak[i])
			}
		}
		for k := 0; k+1 < len(v.breaks); k++ {
			if got := Next(v.text, v.breaks[k]); got != v.breaks[k+1] {
				t.Fatalf("%s: Next(%d)=%d want %d", v.line, v.breaks[k], got, v.breaks[k+1])
			}
			if got := Prev(v.text, v.breaks[k+1]); got != v.breaks[k] {
				t.Fatalf("%s: Prev(%d)=%d want %d", v.line, v.breaks[k+1], got, v.breaks[k])
			}
		}
		rb := map[int]bool{}
		for _, b := range v.rbreak {
			rb[b] = true
		}
		for i := 0; i <= len(v.runes); i++ {
			if BoundaryRunes(v.runes, i) != rb[i] {
				t.Fatalf("%s: BoundaryRunes(%d)", v.line, i)
			}
		}
		for k := 0; k+1 < len(v.rbreak); k++ {
			if got := NextRunes(v.runes, v.rbreak[k]); got != v.rbreak[k+1] {
				t.Fatalf("%s: NextRunes(%d)=%d want %d", v.line, v.rbreak[k], got, v.rbreak[k+1])
			}
			if got := PrevRunes(v.runes, v.rbreak[k+1]); got != v.rbreak[k] {
				t.Fatalf("%s: PrevRunes(%d)=%d want %d", v.line, v.rbreak[k+1], got, v.rbreak[k])
			}
		}
	}
}

func byte0(s string, i int) byte {
	if i >= len(s) {
		return 0
	}
	return s[i]
}

func TestInvalidUTF8(t *testing.T) {
	s := "a\xffb\xe2\x82"
	var got []int
	for i := 0; i < len(s); i = Next(s, i) {
		got = append(got, i)
	}
	if want := []int{0, 1, 2, 3, 4}; !equal(got, want) {
		t.Fatalf("starts %v want %v", got, want)
	}
	if Boundary("€", 1) {
		t.Fatal("boundary inside a rune")
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func FuzzNext(f *testing.F) {
	for _, s := range []string{"", "abc", "\r\n", "🇯🇵🇺🇸🇫", "👨‍👩‍👧", "क्षि", "한", "é", "a\xff\xe2\x82"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		var starts []int
		for i := 0; i < len(s); {
			n := Next(s, i)
			if n <= i || n > len(s) {
				t.Fatalf("Next(%q,%d)=%d", s, i, n)
			}
			if got := Prev(s, n); got != i {
				t.Fatalf("Prev(%q,%d)=%d want %d", s, n, got, i)
			}
			starts = append(starts, i)
			i = n
		}
		isStart := map[int]bool{0: true, len(s): true}
		for _, i := range starts {
			isStart[i] = true
		}
		for i := 0; i <= len(s); i++ {
			if !utf8.ValidString(s) && i < len(s) && !utf8.RuneStart(s[i]) {
				continue
			}
			if Boundary(s, i) != isStart[i] {
				t.Fatalf("Boundary(%q,%d)=%v", s, i, !isStart[i])
			}
		}
		r := []rune(s)
		for i := 0; i < len(r); {
			n := NextRunes(r, i)
			if n <= i || PrevRunes(r, n) != i {
				t.Fatalf("runes %q at %d", s, i)
			}
			i = n
		}
	})
}

var sinkInt int

func benchText(b *testing.B, s string) {
	r := []rune(s)
	b.Run("string", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			for i := 0; i < len(s); i = Next(s, i) {
				sinkInt++
			}
		}
		b.SetBytes(int64(len(s)))
	})
	b.Run("runes", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			for i := 0; i < len(r); i = NextRunes(r, i) {
				sinkInt++
			}
		}
	})
	b.Run("prev", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			for i := len(s); i > 0; i = Prev(s, i) {
				sinkInt++
			}
		}
	})
}

func BenchmarkASCII(b *testing.B) {
	benchText(b, strings.Repeat("The quick brown fox jumps over the lazy dog. ", 20))
}

func BenchmarkMixed(b *testing.B) {
	benchText(b, strings.Repeat("🇯🇵🇺🇸 👨‍👩‍👧 क्षि 한 é ", 20))
}
