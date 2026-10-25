package textmetrics

import (
	"bufio"
	_ "embed"
	"math"
	"strconv"
	"strings"
	"testing"
)

// shapedSamples are words shaped by HarfBuzz with the Noto Sans fonts the tables were made
// from (tools/complexgen): the known answers of the complex script widths.
//
//go:embed testdata/shaped_samples.txt
var shapedSampleFile string

func shapedSamples(t *testing.T) (scripts []string, words []string, units []float64) {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(shapedSampleFile))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		u, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			t.Fatal(err)
		}
		scripts, words, units = append(scripts, fields[0]), append(words, fields[1]), append(units, u)
	}
	return
}

// The table rules reproduce what the shaper does to a word: the mean error of the sum of
// advances, joining forms and conjunct rules against the shaped width is small for each
// script, and no word is off by more than a third.
func TestComplexScriptWidthsFollowTheShaper(t *testing.T) {
	limits := map[string]float64{ // the mean error allowed per script
		"armenian": 0.01, "hebrew": 0.02, "arabic": 0.08, "thai": 0.01, "lao": 0.01, "georgian": 0.01,
		"devanagari": 0.08, "bengali": 0.15, "gujarati": 0.08, "gurmukhi": 0.08, "tamil": 0.05,
		"telugu": 0.08, "kannada": 0.08, "malayalam": 0.1, "khmer": 0.1, "myanmar": 0.12,
	}
	names, words, shaped := shapedSamples(t)
	sums, counts := map[string]float64{}, map[string]int{}
	for i, word := range words {
		got := float64(Width(word, 2048, 0))
		err := math.Abs(got-shaped[i]) / shaped[i]
		sums[names[i]] += err
		counts[names[i]]++
		if err > 0.34 {
			t.Errorf("%s %q: %.0f units, shaped %.0f (%.0f%% off)", names[i], word, got, shaped[i], 100*err)
		}
	}
	for name, limit := range limits {
		if counts[name] == 0 {
			t.Errorf("no samples for %s", name)
			continue
		}
		mean := sums[name] / float64(counts[name])
		t.Logf("%-10s %2d words, mean error %.1f%%", name, counts[name], 100*mean)
		if mean > limit {
			t.Errorf("%s: mean error %.1f%%, want at most %.0f%%", name, 100*mean, 100*limit)
		}
	}
}

func TestComplexScriptRules(t *testing.T) {
	w := func(s string) float32 { return Width(s, 2048, 0) }
	if w("مرحبا") < 3500 || w("مرحبا") > 5200 {
		t.Errorf("Arabic marhaba is %v units", w("مرحبا"))
	}
	if iso, joined := w("ب"), w("ببب"); joined >= 3*iso {
		t.Errorf("joined beh x3 %v is not narrower than three isolated %v", joined, 3*iso)
	}
	if w("لا") >= w("ل")+w("ا") {
		t.Error("lam-alef is not a ligature narrower than the two letters")
	}
	if w("كََ") != w("ك") {
		t.Error("marks above a letter add width")
	}
	if w("क्ष") >= w("क")+w("ष") {
		t.Error("a Devanagari conjunct is not narrower than its two consonants")
	}
	if w("กิ") != w("ก") {
		t.Error("a Thai vowel above adds width")
	}
	if got := w("한"); got != 2048 {
		t.Errorf("decomposed Hangul syllable %v units, want one em", got)
	}
	if got := w("한"); got != 2048 {
		t.Errorf("Hangul syllable %v units, want one em", got)
	}
	if w("אָ") != w("א") {
		t.Error("a niqqud adds width")
	}
	if w("م ب")-w("م")-w("ب") < 0 {
		t.Error("a space joined letters")
	}
}

// A weight makes the text of another script wider, as it does Latin.
func TestComplexScriptsGetWiderWithWeight(t *testing.T) {
	for _, word := range []string{"مرحبا", "שלום", "สวัสดี", "नमस्ते"} {
		if Width(word, 16, 0.4) <= Width(word, 16, 0) {
			t.Errorf("%q is not wider in bold", word)
		}
	}
}

func TestComplexWidthIsAllocationFree(t *testing.T) {
	if n := testing.AllocsPerRun(100, func() {
		Width("مرحبا بالعالم שלום สวัสดี नमस्ते 한국어", 14, 0)
	}); n != 0 {
		t.Errorf("Width allocates %v times", n)
	}
}

func BenchmarkWidthArabic(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		Width("مرحبا بالعالم، هذه جملة عربية للاختبار", 14, 0)
	}
}

func BenchmarkWidthHindi(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		Width("हिन्दी भाषा में यह एक परीक्षण वाक्य है", 14, 0)
	}
}

// A platform that cannot squeeze shaped words must be able to tell which they are.
