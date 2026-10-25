package linebreak

import (
	"bufio"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// conformance runs every case of a UCD LineBreakTest.txt file.
func conformance(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	cases, failed := 0, 0
	for line := 1; sc.Scan(); line++ {
		text, _, _ := strings.Cut(sc.Text(), "#")
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		var b strings.Builder
		var want []int
		for _, tok := range fields {
			switch tok {
			case "×":
			case "÷":
				if b.Len() > 0 {
					want = append(want, b.Len())
				}
			default:
				v, err := strconv.ParseUint(tok, 16, 32)
				if err != nil {
					t.Fatalf("%s:%d: bad token %q", path, line, tok)
				}
				b.WriteRune(rune(v))
			}
		}
		cases++
		if got := Breaks(nil, b.String()); !slices.Equal(got, want) {
			failed++
			if failed <= 20 {
				t.Errorf("%s:%d: %q\n got %v\nwant %v\n%s", path, line, b.String(), got, want, sc.Text())
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if cases == 0 {
		t.Fatalf("%s holds no cases", path)
	}
	t.Logf("%s: %d cases, %d failed", path, cases, failed)
	if failed > 0 {
		t.Errorf("%d of %d cases failed", failed, cases)
	}
}

func TestConformanceSubset(t *testing.T) {
	conformance(t, "testdata/LineBreakTest.subset.txt")
}

// TestConformanceFull runs the complete LineBreakTest.txt named by LINEBREAK_TEST.
func TestConformanceFull(t *testing.T) {
	path := os.Getenv("LINEBREAK_TEST")
	if path == "" {
		t.Skip("LINEBREAK_TEST not set")
	}
	conformance(t, path)
}

func TestKinds(t *testing.T) {
	var s Scanner
	s.Init("a b\nc\r\nd")
	type br struct {
		end  int
		kind Kind
	}
	var got []br
	for {
		end, kind, ok := s.Next()
		if !ok {
			break
		}
		got = append(got, br{end, kind})
	}
	want := []br{{2, Allowed}, {4, Mandatory}, {7, Mandatory}, {8, Mandatory}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	s.Init("")
	if _, _, ok := s.Next(); ok {
		t.Fatal("empty text has no break")
	}
}

func TestInitResets(t *testing.T) {
	var s Scanner
	s.Init("a b")
	s.Next()
	s.Init("x y z")
	var got []int
	for {
		end, _, ok := s.Next()
		if !ok {
			break
		}
		got = append(got, end)
	}
	if want := []int{2, 4, 5}; !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

const (
	thai    = "ประเทศไทยสวัสดีครับ"
	khmer   = "ភាសាខ្មែរ"
	myanmar = "မြန်မာဘာသာ"
)

func fallbackBreaks(text string) []int {
	var s Scanner
	s.Fallback = true
	s.Init(text)
	var out []int
	for {
		end, _, ok := s.Next()
		if !ok {
			return out
		}
		out = append(out, end)
	}
}

func TestFallbackThai(t *testing.T) {
	if got := Breaks(nil, thai); !slices.Equal(got, []int{len(thai)}) {
		t.Fatalf("UAX #14 alone must not split Thai: %v", got)
	}
	got := fallbackBreaks(thai)
	for _, before := range []string{"เทศ", "ไทย", "ครับ"} {
		if !slices.Contains(got, strings.Index(thai, before)) {
			t.Errorf("no break before %s: %v", before, got)
		}
	}
	if slices.Contains(got, strings.Index(thai, "ร")+3) {
		t.Errorf("break between ป ร ะ: %v", got)
	}
	checkBoundaries(t, thai, got)
}

func TestFallbackNeverBeforeMarksOrRepeat(t *testing.T) {
	for _, s := range []string{"กี่", "กๆ", "สวัสดีๆ", "กัข"[:6]} {
		for _, b := range fallbackBreaks(s) {
			if b < len(s) {
				t.Errorf("%q breaks at %d", s, b)
			}
		}
	}
}

func TestFallbackKhmer(t *testing.T) {
	got := fallbackBreaks(khmer)
	if len(got) < 2 {
		t.Fatalf("Khmer not split: %v", got)
	}
	// ខ្មែរ: ខ COENG ម; the ម after the COENG must stay with it.
	if slices.Contains(got, strings.Index(khmer, "្")+3) {
		t.Errorf("break after COENG: %v", got)
	}
	checkBoundaries(t, khmer, got)
}

func TestFallbackMyanmar(t *testing.T) {
	got := fallbackBreaks(myanmar)
	if len(got) < 2 {
		t.Fatalf("Myanmar not split: %v", got)
	}
	// န်: a consonant followed by the asat is a final consonant.
	if slices.Contains(got, strings.Index(myanmar, "န")) {
		t.Errorf("break before a consonant with asat: %v", got)
	}
	const stacked = "ဗုဒ္ဓ" // ဒ virama ဓ
	if slices.Contains(fallbackBreaks(stacked), strings.Index(stacked, "ဓ")) ||
		slices.Contains(fallbackBreaks(stacked), strings.Index(stacked, "ဒ")) {
		t.Errorf("break inside stacked consonants: %v", fallbackBreaks(stacked))
	}
	checkBoundaries(t, myanmar, got)
}

func TestFallbackOffMatchesPlain(t *testing.T) {
	texts := []string{thai, khmer, myanmar, "Hello, world (test) 12.5% “q” 日本語、です。\n", "ຂ້ອຍ ເປັນ"}
	for _, text := range texts {
		var s Scanner
		s.Init(text)
		var got []int
		for {
			end, _, ok := s.Next()
			if !ok {
				break
			}
			got = append(got, end)
		}
		if want := Breaks(nil, text); !slices.Equal(got, want) {
			t.Errorf("%q: %v vs %v", text, got, want)
		}
	}
	// The fallback only adds opportunities.
	for _, text := range texts {
		plain := Breaks(nil, text)
		with := fallbackBreaks(text)
		for _, b := range plain {
			if !slices.Contains(with, b) {
				t.Errorf("%q: fallback lost break %d", text, b)
			}
		}
	}
}

func TestFallbackKeepsFieldAcrossInit(t *testing.T) {
	var s Scanner
	s.Fallback = true
	s.Init("a")
	s.Init(thai)
	n := 0
	for {
		if _, _, ok := s.Next(); !ok {
			break
		}
		n++
	}
	if n < 3 {
		t.Fatalf("Fallback lost by Init: %d breaks", n)
	}
}

// checkBoundaries verifies offsets are strictly increasing rune boundaries ending at len.
func checkBoundaries(t *testing.T, text string, got []int) {
	t.Helper()
	if len(got) == 0 || got[len(got)-1] != len(text) {
		t.Fatalf("%q: last break is not len: %v", text, got)
	}
	prev := 0
	for _, b := range got {
		if b <= prev || b > len(text) {
			t.Fatalf("%q: offsets not increasing: %v", text, got)
		}
		if b < len(text) && !utf8.RuneStart(text[b]) {
			t.Fatalf("%q: offset %d inside a character", text, b)
		}
		prev = b
	}
}

func FuzzScanner(f *testing.F) {
	for _, s := range []string{"", "a b", "A\r\nB", thai, khmer, myanmar, "\xff\xfe a", "1,234.5 (x)", "🇦🇧🇨 👍🏽", "“a” «b»", "क्ष कि", "‍́ ́"} {
		f.Add(s, false)
		f.Add(s, true)
	}
	f.Fuzz(func(t *testing.T, text string, fallback bool) {
		var s Scanner
		s.Fallback = fallback
		s.Init(text)
		prev := 0
		for {
			end, kind, ok := s.Next()
			if !ok {
				break
			}
			if end <= prev || end > len(text) {
				t.Fatalf("offset %d after %d in %d bytes", end, prev, len(text))
			}
			if utf8.ValidString(text) && end < len(text) && !utf8.RuneStart(text[end]) {
				t.Fatalf("offset %d inside a character", end)
			}
			if end == len(text) && kind != Mandatory {
				t.Fatal("the end of the text must be mandatory")
			}
			prev = end
		}
		if len(text) > 0 && prev != len(text) {
			t.Fatalf("last offset %d, text is %d bytes", prev, len(text))
		}
	})
}

const english = "The quick brown fox jumps over the lazy dog, and then (quite suddenly) it was 12.50% faster than before. " +
	"Nobody expected that, least of all the well-known author who wrote \"every word\" twice; see e.g. chapter 3 for details. " +
	"After lunch we walked to the old harbour and watched the boats."

func BenchmarkEnglishParagraph(b *testing.B) {
	b.Logf("%d characters", len(english))
	b.ReportAllocs()
	var s Scanner
	for b.Loop() {
		s.Init(english)
		for {
			if _, _, ok := s.Next(); !ok {
				break
			}
		}
	}
}

func BenchmarkMixedParagraph(b *testing.B) {
	text := strings.Repeat("日本語の文章、です。Hello, world! 1,234.5円 “quoted” ", 6)
	b.ReportAllocs()
	var s Scanner
	for b.Loop() {
		s.Init(text)
		for {
			if _, _, ok := s.Next(); !ok {
				break
			}
		}
	}
}
