package textsharp

import (
	"fmt"
	"hash/fnv"
	"math"
	"unicode/utf8"

	"github.com/vivaui/viva/internal/textmetrics"
)

// Case is one line of text of the fidelity matrix: a string, a size, a weight and a
// theme, drawn at a display scale. Gray values are 0..255.
type Case struct {
	Name, Text string
	Size       float32 // logical pixels
	Bold       bool
	Dark       bool // light text on a dark background
	Scale      float32
	Width      int // logical pixels
	Height     int
}

// Texts are the strings of the matrix: Latin words and digits, CJK, a mix, and capitals
// whose flat feet show the baseline.
var Texts = []struct{ Name, Text string }{
	{"latin", "Hamburgefonstiv Connect settings 0123"},
	{"cjk", "界面一致性测试。日本語のテキスト"},
	{"mixed", "Motty 连接 host-01 已连接 — 3"},
	{"caps", "HHHH HEHE"},
}

// FidelitySizes are the logical sizes compared, 11 to 24.
var FidelitySizes = []float32{11, 13, 16, 20, 24}

// Cases builds the matrix: every text at every size, regular and bold, dark on light and
// light on dark, at every scale.
func Cases(scales, sizes []float32) []Case {
	var cases []Case
	for _, scale := range scales {
		for _, text := range Texts {
			for _, size := range sizes {
				for _, bold := range []bool{false, true} {
					for _, dark := range []bool{false, true} {
						c := Case{Text: text.Text, Size: size, Bold: bold, Dark: dark, Scale: scale, Height: 28}
						c.Name = fmt.Sprintf("%s-%g-%s-%s-%gx", text.Name, size, map[bool]string{false: "regular", true: "bold"}[bold], map[bool]string{false: "light", true: "dark"}[dark], scale)
						c.Width = int(math.Ceil(float64(textmetrics.Width(text.Text, size, c.Weight())))) + 40
						cases = append(cases, c)
					}
				}
			}
		}
	}
	return cases
}

// Weight is the weight a label is given (0 regular, 0.4 bold).
func (c Case) Weight() float32 {
	if c.Bold {
		return 0.4
	}
	return 0
}

// CSSWeight is the weight as the reference toolkits name it.
func (c Case) CSSWeight() string {
	if c.Bold {
		return "700"
	}
	return "400"
}

// Foreground and Background are the gray levels of the text and of what it is drawn on.
func (c Case) Foreground() float64 {
	if c.Dark {
		return 0xe8
	}
	return 0x1a
}

func (c Case) Background() float64 {
	if c.Dark {
		return 0x1e
	}
	return 0xff
}

// HexForeground and HexBackground are the same colors as #rrggbb.
func (c Case) HexForeground() string { return hex(c.Foreground()) }
func (c Case) HexBackground() string { return hex(c.Background()) }

func hex(gray float64) string { return fmt.Sprintf("#%02x%02x%02x", int(gray), int(gray), int(gray)) }

// DeviceSize is the size of the image in device pixels.
func (c Case) DeviceSize() (width, height int) {
	return int(math.Ceil(float64(c.Width) * float64(c.Scale))), int(math.Ceil(float64(c.Height) * float64(c.Scale)))
}

// Word is a piece of a line as textmetrics.NextRun cuts it: a word between spaces, or a
// full-width character with its marks.
type Word struct {
	Text string
}

// Words are the pieces of the case, in order. Each is drawn alone by the platform's own
// toolkit as the reference for the same piece inside the line.
func (c Case) Words() []Word {
	var words []Word
	for from := 0; ; {
		start, end, ok := textmetrics.NextRun(c.Text, from)
		if !ok {
			return words
		}
		words = append(words, Word{Text: c.Text[start:end]})
		from = end
	}
}

// Margin is the origin of a reference word drawn on its own, in device pixels from the left
// of its image: room for the ink that begins left of the origin.
const Margin = 6

// Summary accumulates fidelities and reports their mean and worst values.
type Summary struct {
	N                                         int
	Mass, Stem, Edge, Solid                   float64
	Residual, WorstResidual                   float64
	LeastMass, LeastStem, LeastEdge, LeastSol float64
}

// Add folds one comparison in.
func (s *Summary) Add(f Fidelity) {
	if f.Mass == 0 {
		return
	}
	if s.N == 0 {
		s.LeastMass, s.LeastStem, s.LeastEdge, s.LeastSol = math.Inf(1), math.Inf(1), math.Inf(1), math.Inf(1)
	}
	s.N++
	s.Mass += f.Mass
	s.Stem += f.Stem
	s.Edge += f.Edge
	s.Solid += f.Solid
	s.Residual += f.Residual
	s.WorstResidual = max(s.WorstResidual, f.Residual)
	s.LeastMass, s.LeastStem, s.LeastEdge, s.LeastSol = min(s.LeastMass, f.Mass), min(s.LeastStem, f.Stem), min(s.LeastEdge, f.Edge), min(s.LeastSol, f.Solid)
}

func (s Summary) String() string {
	if s.N == 0 {
		return "no comparisons"
	}
	n := float64(s.N)
	return fmt.Sprintf("n=%d mass %.3f (least %.3f) stem %.3f (least %.3f) edge %.3f (least %.3f) solid %.3f (least %.3f) residual %.3f (worst %.3f)",
		s.N, s.Mass/n, s.LeastMass, s.Stem/n, s.LeastStem, s.Edge/n, s.LeastEdge, s.Solid/n, s.LeastSol, s.Residual/n, s.WorstResidual)
}

// WordRequest is a word of a case drawn on its own as a reference, in the style of
// the case (the Width of its image is the room for the word and its Margin).
type WordRequest struct {
	Name string
	Case Case
}

// WordName is the name of the reference image of the word in the style of c.
func WordName(c Case, w Word) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s|%g|%t|%t|%g", w.Text, c.Size, c.Bold, c.Dark, c.Scale)
	return fmt.Sprintf("word-%016x", h.Sum64())
}

// WordRequests lists every distinct word of the cases once. Each request's Case carries the
// text of the word, the style and the scale of the cases it comes from.
func WordRequests(cases []Case) []WordRequest {
	seen := map[string]bool{}
	var requests []WordRequest
	for _, c := range cases {
		for _, w := range c.Words() {
			name := WordName(c, w)
			if seen[name] {
				continue
			}
			seen[name] = true
			word := c
			word.Name, word.Text = name, w.Text
			word.Width = int(math.Ceil(float64(c.Size)*float64(utf8.RuneCountInString(w.Text))*1.4)) + 2*Margin + 8
			requests = append(requests, WordRequest{Name: name, Case: word})
		}
	}
	return requests
}
