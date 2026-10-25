// Command complexgen shapes text of the complex scripts with HarfBuzz (the pure Go port,
// github.com/go-text/typesetting) and Noto Sans fonts (SIL OFL 1.1), and writes the
// advance tables internal/textmetrics uses to lay out Arabic, Hebrew, Thai, Lao, Khmer,
// Myanmar, Armenian, Georgian and the Indic scripts.
//
// It is a separate module so that Viva itself depends on nothing it shapes with: the
// shaper runs here, once, and the results are checked in as tables.
//
//	go run . -fonts /path/to/noto -out ../../internal/textmetrics/complex_table.go \
//	    -samples ../../internal/textmetrics/testdata/shaped_samples.txt
//
// The directory holds NotoSans<Script>-Regular.ttf renamed NotoSans<Script>.ttf, from
// https://github.com/notofonts/notofonts.github.io/tree/main/fonts/NotoSans<Script>/hinted/ttf.
//
// What is stored, in font units scaled to 2048 per em:
//   - the advance of every code point of the script block as the font shapes it (a mark
//     is the advance it adds to a base, usually 0; a spacing vowel sign likewise);
//   - for Arabic, the advance of each letter in its initial, medial and final form, and
//     of the lam-alef ligatures, found by shaping the letter between zero width joiners;
//   - for the Indic scripts, per consonant the advance it keeps when it begins a conjunct
//     (the half form of Devanagari, Bengali, Gujarati) or when it is stacked under the
//     one before it (Telugu, Kannada, Gurmukhi, Khmer, Myanmar): the median over all
//     conjunct pairs of the font, which is also how the error of that model is reported.
//
// Calibration: the platforms' own fonts are not Noto Sans (macOS has Kohinoor, Tamil
// Sangam, Thonburi; Windows Nirmala UI, Leelawadee UI, Myanmar Text; Linux whatever
// fontconfig picks), and a word of Arabic, Thai, Khmer or an Indic script cannot be
// squeezed on Windows without taking its shaping apart (platform/windows draws it at its
// natural width), so it must not be wider there than layout says. The table of each such
// script is scaled so that the mean native width of the words on Windows is 0.97 of their
// layout width, measured with tools/visualparity (the scales in specs; Hebrew, Armenian and
// Georgian can be squeezed and keep Noto's own widths, as does a script without a
// specimen; Myanmar's Windows font is too far from the others' to be followed and the
// scale is the mean of macOS and Linux). Layout is the same everywhere either way; the
// price is gaps that grow where a platform's font is narrower (macOS Tamil is 0.86 of
// Windows').
//
// The program also writes real shaped words, the known answers of the package's tests.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/harfbuzz"

	"github.com/vivaui/viva/internal/uniprops"
)

const noAdvance = 0xFFFF

type spec struct {
	file     string
	name     string
	kind     string // simple, arabic, conjunct, stacked
	from, to rune
	linker   rune
	consFrom rune
	consTo   rune
	base     rune    // a base letter for measuring marks
	scale    float64 // calibration of the table, see above; 0 is 1
	samples  []string
}

var specs = []spec{
	{file: "NotoSansArmenian", name: "armenian", kind: "simple", from: 0x530, to: 0x58F, base: 0x531, samples: []string{"Հայերեն", "բարեւ", "շնորհակալություն", "Երեւան"}},
	{file: "NotoSansHebrew", name: "hebrew", kind: "simple", from: 0x590, to: 0x5FF, base: 0x5D0, samples: []string{"שלום", "עולם", "עברית", "ברוכים", "הבאים", "תודה", "רבה", "מחשב", "שָׁלוֹם", "ירושלים"}},
	{file: "NotoSansArabic", name: "arabic", scale: 1.04, kind: "arabic", from: 0x600, to: 0x6FF, base: 0x628, samples: []string{"مرحبا", "السلام", "عليكم", "ورحمة", "الله", "وبركاته", "اللغة", "العربية", "بسم", "الرحمن", "الرحيم", "محمد", "مكتبة", "الكتاب", "شكرا", "جزيلا", "أهلا", "وسهلا", "لا", "إلى", "سلام", "دنیا", "فارسی", "برنامه", "٠١٢٣٤٥", "مستخدم", "كلمة", "المرور"}},
	{file: "NotoSansArabic", name: "arabicsup", kind: "arabic", from: 0x750, to: 0x77F, base: 0x628},
	{file: "NotoSansArabic", name: "arabicext", kind: "arabic", from: 0x8A0, to: 0x8FF, base: 0x628},
	{file: "NotoSansArabic", name: "arabicforms", kind: "simple", from: 0xFE70, to: 0xFEFF, base: 0x628},
	{file: "NotoSansDevanagari", name: "devanagari", scale: 1.11, kind: "conjunct", from: 0x900, to: 0x97F, linker: 0x94D, consFrom: 0x915, consTo: 0x939, base: 0x915, samples: []string{"नमस्ते", "दुनिया", "हिन्दी", "भाषा", "क्षत्रिय", "विद्यालय", "प्रेम", "कृपया", "धन्यवाद", "संस्कृत", "ज्ञान", "श्रीमान", "द्वारा", "कर्म", "पुस्तक", "अंग्रेज़ी", "उपयोगकर्ता", "पासवर्ड"}},
	{file: "NotoSansBengali", name: "bengali", scale: 1.14, kind: "conjunct", from: 0x980, to: 0x9FF, linker: 0x9CD, consFrom: 0x995, consTo: 0x9B9, base: 0x995, samples: []string{"বাংলা", "ভাষা", "নমস্কার", "ধন্যবাদ", "বিশ্ব", "স্বাগতম", "কম্পিউটার", "ক্ষমা", "ব্যবহারকারী", "পাসওয়ার্ড"}},
	{file: "NotoSansGurmukhi", name: "gurmukhi", kind: "stacked", from: 0xA00, to: 0xA7F, linker: 0xA4D, consFrom: 0xA15, consTo: 0xA39, base: 0xA15, samples: []string{"ਪੰਜਾਬੀ", "ਭਾਸ਼ਾ", "ਸਤ", "ਸ੍ਰੀ", "ਅਕਾਲ", "ਧੰਨਵਾਦ", "ਵਰਤੋਂਕਾਰ"}},
	{file: "NotoSansGujarati", name: "gujarati", kind: "conjunct", from: 0xA80, to: 0xAFF, linker: 0xACD, consFrom: 0xA95, consTo: 0xAB9, base: 0xA95, samples: []string{"ગુજરાતી", "ભાષા", "નમસ્તે", "આભાર", "વિશ્વ", "સ્વાગત", "વપરાશકર્તા"}},
	{file: "NotoSansTamil", name: "tamil", scale: 1.09, kind: "simple", from: 0xB80, to: 0xBFF, base: 0xB95, samples: []string{"தமிழ்", "மொழி", "வணக்கம்", "நன்றி", "உலகம்", "கணினி", "கொண்டு", "பயனர்", "கடவுச்சொல்"}},
	{file: "NotoSansTelugu", name: "telugu", scale: 1.24, kind: "stacked", from: 0xC00, to: 0xC7F, linker: 0xC4D, consFrom: 0xC15, consTo: 0xC39, base: 0xC15, samples: []string{"తెలుగు", "భాష", "నమస్కారం", "ధన్యవాదాలు", "ప్రపంచం", "స్వాగతం", "వినియోగదారు"}},
	{file: "NotoSansKannada", name: "kannada", kind: "stacked", from: 0xC80, to: 0xCFF, linker: 0xCCD, consFrom: 0xC95, consTo: 0xCB9, base: 0xC95, samples: []string{"ಕನ್ನಡ", "ಭಾಷೆ", "ನಮಸ್ಕಾರ", "ಧನ್ಯವಾದ", "ಪ್ರಪಂಚ", "ಸ್ವಾಗತ", "ಬಳಕೆದಾರ"}},
	{file: "NotoSansMalayalam", name: "malayalam", kind: "simple", from: 0xD00, to: 0xD7F, base: 0xD15, samples: []string{"മലയാളം", "ഭാഷ", "നമസ്കാരം", "നന്ദി", "ലോകം", "സ്വാഗതം", "ഉപയോക്താവ്"}},
	{file: "NotoSansThai", name: "thai", scale: 0.99, kind: "simple", from: 0xE00, to: 0xE7F, base: 0xE01, samples: []string{"สวัสดี", "ครับ", "ประเทศไทย", "ภาษาไทย", "ขอบคุณ", "ที่นี่", "กรุงเทพมหานคร", "น้ำ", "เด็ก", "ผู้ใช้", "รหัสผ่าน", "ตั้งค่า"}},
	{file: "NotoSansLao", name: "lao", kind: "simple", from: 0xE80, to: 0xEFF, base: 0xE81, samples: []string{"ສະບາຍດີ", "ພາສາລາວ", "ຂອບໃຈ", "ລາວ"}},
	{file: "NotoSansMyanmar", name: "myanmar", scale: 0.97, kind: "stacked", from: 0x1000, to: 0x109F, linker: 0x1039, consFrom: 0x1000, consTo: 0x1021, base: 0x1000, samples: []string{"မြန်မာဘာသာ", "ကျေးဇူးတင်ပါတယ်", "မင်္ဂလာပါ", "စကားဝှက်"}},
	{file: "NotoSansGeorgian", name: "georgian", kind: "simple", from: 0x10A0, to: 0x10FF, base: 0x10D0, samples: []string{"ქართული", "გამარჯობა", "მადლობა", "პაროლი"}},
	{file: "NotoSansKhmer", name: "khmer", scale: 1.17, kind: "stacked", from: 0x1780, to: 0x17FF, linker: 0x17D2, consFrom: 0x1780, consTo: 0x17A2, base: 0x1780, samples: []string{"ភាសាខ្មែរ", "សួស្តី", "អរគុណ", "ប្រទេសកម្ពុជា", "ពាក្យសម្ងាត់"}},
}

type shaper struct {
	face  *font.Face
	hb    *harfbuzz.Font
	upem  float64
	buf   *harfbuzz.Buffer
	cache map[string]float64
	scale float64
}

func newShaper(path string) *shaper {
	data, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	faces, err := font.ParseTTC(bytes.NewReader(data))
	if err != nil {
		fatal(err)
	}
	return &shaper{face: faces[0], hb: harfbuzz.NewFont(faces[0]), upem: float64(faces[0].Upem()), buf: harfbuzz.NewBuffer(), cache: map[string]float64{}, scale: 1}
}

// width is the advance of text as HarfBuzz shapes it, in units of 2048 per em.
func (s *shaper) width(text string) float64 {
	if w, ok := s.cache[text]; ok {
		return w
	}
	s.buf.Clear()
	runes := []rune(text)
	s.buf.AddRunes(runes, 0, len(runes))
	s.buf.GuessSegmentProperties()
	s.buf.Shape(s.hb, nil)
	total := 0.0
	for _, p := range s.buf.Pos {
		total += float64(p.XAdvance)
	}
	w := total * 2048 / s.upem * s.scale
	s.cache[text] = w
	return w
}

func (s *shaper) has(r rune) bool {
	_, ok := s.face.NominalGlyph(r)
	return ok
}

func round(v float64) uint16 {
	if v < 0 {
		return 0
	}
	return uint16(math.Round(v))
}

func median(v []float64) float64 {
	sort.Float64s(v)
	if len(v) == 0 {
		return 0
	}
	return v[len(v)/2]
}

type out struct {
	buf bytes.Buffer
}

func (o *out) array(name, doc string, values []uint16) {
	fmt.Fprintf(&o.buf, "// %s\nvar %s = [%d]uint16{", doc, name, len(values))
	for i, v := range values {
		if i%16 == 0 {
			o.buf.WriteString("\n\t")
		} else {
			o.buf.WriteByte(' ')
		}
		fmt.Fprintf(&o.buf, "%d,", v)
	}
	o.buf.WriteString("\n}\n\n")
}

func (o *out) pairs(name, doc string, values [][2]uint16) {
	fmt.Fprintf(&o.buf, "// %s\nvar %s = [%d][2]uint16{", doc, name, len(values))
	for i, v := range values {
		if i%8 == 0 {
			o.buf.WriteString("\n\t")
		} else {
			o.buf.WriteByte(' ')
		}
		fmt.Fprintf(&o.buf, "{%d, %d},", v[0], v[1])
	}
	o.buf.WriteString("\n}\n\n")
}

func (o *out) forms(name, doc string, values [][3]uint16) {
	fmt.Fprintf(&o.buf, "// %s\nvar %s = [%d][3]uint16{", doc, name, len(values))
	for i, v := range values {
		if i%6 == 0 {
			o.buf.WriteString("\n\t")
		} else {
			o.buf.WriteByte(' ')
		}
		fmt.Fprintf(&o.buf, "{%d, %d, %d},", v[0], v[1], v[2])
	}
	o.buf.WriteString("\n}\n\n")
}

func main() {
	fonts := flag.String("fonts", "", "directory of NotoSans<Script>.ttf files")
	outPath := flag.String("out", "complex_table.go", "output Go file")
	samplePath := flag.String("samples", "", "output file of shaped sample words")
	flag.Parse()

	sort.Slice(specs, func(i, j int) bool { return specs[i].from < specs[j].from })
	var o out
	var entries strings.Builder
	var samples strings.Builder
	shapers := map[string]*shaper{}
	var lamAlef [2][4]uint16
	for _, sp := range specs {
		path := filepath.Join(*fonts, sp.file+".ttf")
		sh, ok := shapers[path]
		if !ok {
			sh = newShaper(path)
			shapers[path] = sh
		}
		if scale := max(sp.scale, 1e-9); sp.scale != 0 && sh.scale != scale || sp.scale == 0 && sh.scale != 1 {
			sh.cache = map[string]float64{}
			sh.scale = 1
			if sp.scale != 0 {
				sh.scale = scale
			}
		}
		n := int(sp.to-sp.from) + 1
		adv := make([]uint16, n)
		baseW := sh.width(string(sp.base))
		var letters []float64
		for i := range adv {
			r := sp.from + rune(i)
			adv[i] = noAdvance
			if !sh.has(r) && !uniprops.Mark(r) {
				continue
			}
			switch {
			case sp.kind == "arabic" && uniprops.Joining(r) == uniprops.JoinT:
				adv[i] = 0
			case uniprops.Mark(r) || isSpacingMark(r):
				// What the mark adds to a base letter: nothing, or a spacing sign's width.
				d := sh.width(string(sp.base)+string(r)) - baseW
				if d < 6 {
					d = 0
				}
				adv[i] = round(d)
			default:
				adv[i] = round(sh.width(string(r)))
				if uniprops.Bidi(r) != uniprops.BidiNSM && r != 0x200C && r != 0x200D && isLetter(r) && adv[i] > 0 {
					letters = append(letters, float64(adv[i]))
				}
			}
		}
		mean := round(median(letters))
		fmt.Fprintf(&entries, "\t{from: 0x%X, to: 0x%X, kind: %s, mean: %d, adv: %sAdv[:]", sp.from, sp.to, kindName(sp.kind), mean, sp.name)
		o.array(sp.name+"Adv", fmt.Sprintf("%sAdv is the advance of each code point of U+%04X to U+%04X, in 2048 units per em; 0xFFFF where the font has no glyph.", sp.name, sp.from, sp.to), adv)

		if sp.kind == "arabic" {
			forms := make([][3]uint16, n)
			for i := range forms {
				r := sp.from + rune(i)
				forms[i] = [3]uint16{noAdvance, noAdvance, noAdvance}
				j := uniprops.Joining(r)
				if adv[i] == noAdvance || (j != uniprops.JoinD && j != uniprops.JoinR && j != uniprops.JoinL && j != uniprops.JoinC) {
					continue
				}
				c := string(r)
				initial, medial, final := noAdvance, noAdvance, noAdvance
				if j == uniprops.JoinD || j == uniprops.JoinL || j == uniprops.JoinC {
					initial = int(round(sh.width(c + "‍")))
				}
				if j == uniprops.JoinD || j == uniprops.JoinC {
					medial = int(round(sh.width("‍" + c + "‍")))
				}
				if j == uniprops.JoinD || j == uniprops.JoinR || j == uniprops.JoinC {
					final = int(round(sh.width("‍" + c)))
				}
				forms[i] = [3]uint16{uint16(initial), uint16(medial), uint16(final)}
			}
			o.forms(sp.name+"Forms", fmt.Sprintf("%sForms is the initial, medial and final advance of each joining letter of %sAdv; 0xFFFF where the form does not exist.", sp.name, sp.name), forms)
			fmt.Fprintf(&entries, ", forms: %sForms[:]", sp.name)
			if sp.name == "arabic" {
				for k, alef := range []rune{0x622, 0x623, 0x625, 0x627} {
					lamAlef[0][k] = round(sh.width("ل" + string(alef)))
					lamAlef[1][k] = round(sh.width("‍ل" + string(alef)))
				}
			}
		}

		if sp.kind == "conjunct" || sp.kind == "stacked" {
			var cons []rune
			for r := sp.consFrom; r <= sp.consTo; r++ {
				if sh.has(r) {
					cons = append(cons, r)
				}
			}
			a := func(r rune) float64 { return sh.width(string(r)) }
			join := make([]uint16, int(sp.consTo-sp.consFrom)+1)
			for i := range join {
				join[i] = noAdvance
			}
			var errSum float64
			pairs := 0
			predict := map[rune]float64{}
			for _, c := range cons {
				var d []float64
				for _, other := range cons {
					if sp.kind == "conjunct" {
						d = append(d, sh.width(string([]rune{c, sp.linker, other}))-a(other))
					} else {
						d = append(d, sh.width(string([]rune{other, sp.linker, c}))-a(other))
					}
				}
				v := median(d)
				v = math.Max(0, math.Min(v, a(c)))
				predict[c] = v
				join[c-sp.consFrom] = round(v)
			}
			var exceptions [][2]uint16
			for _, c1 := range cons {
				for _, c2 := range cons {
					p := sh.width(string([]rune{c1, sp.linker, c2}))
					var m float64
					if sp.kind == "conjunct" {
						m = predict[c1] + a(c2)
					} else {
						m = a(c1) + predict[c2]
					}
					errSum += math.Abs(m-p) / p
					pairs++
					// A pair the font ligates into a glyph of its own (Devanagari ksha, jna)
					// is kept with its width when the model is more than 4 percent of an em off.
					if sp.kind == "conjunct" && math.Abs(m-p) > 80 {
						exceptions = append(exceptions, [2]uint16{uint16(c1-sp.consFrom)<<8 | uint16(c2-sp.consFrom), round(p)})
					}
				}
			}
			if sp.kind == "conjunct" {
				o.pairs(sp.name+"Pairs", fmt.Sprintf("%sPairs is the width of the conjuncts the font draws as a glyph of its own, sorted by the key (first consonant - U+%04X) << 8 | (second consonant - U+%04X).", sp.name, sp.consFrom, sp.consFrom), exceptions)
				fmt.Fprintf(os.Stderr, "%-11s %d ligated conjuncts of %d pairs\n", sp.name, len(exceptions), pairs)
				fmt.Fprintf(&entries, ", pairs: %sPairs[:]", sp.name)
			}
			fmt.Fprintf(os.Stderr, "%-11s %-8s %d consonants, conjunct pair model error %.1f%% mean\n", sp.name, sp.kind, len(cons), 100*errSum/float64(pairs))
			o.array(sp.name+"Join", fmt.Sprintf("%sJoin is, for each consonant U+%04X to U+%04X, its advance %s.", sp.name, sp.consFrom, sp.consTo, map[string]string{"conjunct": "when it begins a conjunct (its half form)", "stacked": "when it is stacked under the consonant before it"}[sp.kind]), join)
			fmt.Fprintf(&entries, ", linker: 0x%X, consFrom: 0x%X, consTo: 0x%X, join: %sJoin[:]", sp.linker, sp.consFrom, sp.consTo, sp.name)
		}
		entries.WriteString("},\n")

		for _, w := range sp.samples {
			fmt.Fprintf(&samples, "%s\t%s\t%d\n", sp.name, w, round(sh.width(w)))
		}
	}
	// The weights: the share by which Inter's bold, semibold and medium lowercase letters
	// are wider than the regular ones is applied to every script by the package.
	var src bytes.Buffer
	src.WriteString("// Code generated by tools/complexgen from Noto Sans (SIL OFL 1.1) shaped with HarfBuzz; DO NOT EDIT.\n\npackage textmetrics\n\n")
	fmt.Fprintf(&src, "// lamAlef is the advance of the lam-alef ligature, isolated and final, for alef with madda\n// above, hamza above, hamza below and plain alef.\nvar lamAlef = [2][4]uint16{{%d, %d, %d, %d}, {%d, %d, %d, %d}}\n\n", lamAlef[0][0], lamAlef[0][1], lamAlef[0][2], lamAlef[0][3], lamAlef[1][0], lamAlef[1][1], lamAlef[1][2], lamAlef[1][3])
	src.Write(o.buf.Bytes())
	src.WriteString("// scripts are the complex scripts, in code point order.\nvar scripts = [...]script{\n")
	src.WriteString(entries.String())
	src.WriteString("}\n")
	formatted, err := format.Source(src.Bytes())
	if err != nil {
		os.WriteFile(*outPath, src.Bytes(), 0o644)
		fatal(err)
	}
	if err := os.WriteFile(*outPath, formatted, 0o644); err != nil {
		fatal(err)
	}
	if *samplePath != "" {
		header := "# Words shaped by HarfBuzz with Noto Sans (tools/complexgen): script, word, advance in 2048 units per em.\n"
		if err := os.WriteFile(*samplePath, []byte(header+samples.String()), 0o644); err != nil {
			fatal(err)
		}
	}
}

func kindName(kind string) string {
	return map[string]string{"simple": "scriptSimple", "arabic": "scriptArabic", "conjunct": "scriptConjunct", "stacked": "scriptStacked"}[kind]
}

func isLetter(r rune) bool { return uniprops.Bidi(r) != uniprops.BidiON && !uniprops.Mark(r) }

// isSpacingMark is a vowel sign or other mark that is not a letter by itself: its
// advance is what it adds to a base letter. The Unicode database calls them Mc, and Mark
// holds those too, so this only has to cover the sign-like letters.
func isSpacingMark(r rune) bool { return r == 0xE33 || r == 0xEB3 }

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
