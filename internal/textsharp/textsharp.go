// Package textsharp measures how crisp rasterized text is, so that the label rasterizers
// can be shown to be as sharp as the platform drawing the same string in one native call.
//
// A blurred or resampled glyph spreads each edge over more pixels: its coverage
// changes by less between neighbours and fewer pixels are solid. Both are measured on
// the coverage (alpha) of one rasterized line, independent of where the words sit.
package textsharp

import "fmt"

// Sharpness describes one rasterized line.
type Sharpness struct {
	// Ink is the summed coverage, in whole pixels of full ink: how heavy the text is.
	Ink float64
	// Edge is the total variation of the coverage between neighbouring pixels (both
	// directions) per pixel of ink. A stem of one pixel aligned to the grid scores 2 per
	// row (one step up and one down) for 1 of ink; spread over two pixels it scores 1.
	Edge float64
	// Solid is the share of the pixels that carry ink (coverage of at least 1/8) whose
	// coverage is at least 7/8: how much of the text is crisp stem and not edge haze.
	Solid float64
}

// Measure returns the sharpness of the coverage of a width x height line: coverage[i]
// is the alpha of pixel i, row by row, stride bytes per pixel and rowStride per row
// (a premultiplied BGRA bitmap is coverage[3:] with stride 4).
func Measure(coverage []byte, width, height, stride, rowStride int) Sharpness {
	at := func(x, y int) float64 { return float64(coverage[y*rowStride+x*stride]) / 255 }
	var ink, variation float64
	var inked, solid int
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := at(x, y)
			ink += c
			if c >= 1.0/8 {
				inked++
				if c >= 7.0/8 {
					solid++
				}
			}
			if x+1 < width {
				variation += abs(at(x+1, y) - c)
			}
			if y+1 < height {
				variation += abs(at(x, y+1) - c)
			}
		}
	}
	if ink == 0 || inked == 0 {
		return Sharpness{}
	}
	return Sharpness{Ink: ink, Edge: variation / ink, Solid: float64(solid) / float64(inked)}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// Specimen is a string the platform tests draw both ways.
type Specimen struct{ Name, Text string }

// Specimens cover the scripts a label holds: Latin, digits, CJK, and a mix.
var Specimens = []Specimen{
	{"latin", "The quick brown fox jumps over the lazy dog. Connect to host-01 as ada"},
	{"ui", "Connect  Cancel  Settings  Preferences  New connection"},
	{"digits", "0123456789 12:45:07 192.168.0.12:22"},
	{"cjk", "界面一致性测试。日本語のテキスト"},
	{"mixed", "Motty 连接 host-01 已连接 — 3 sessions"},
}

// Sizes and Weights are the styles drawn.
var (
	Sizes   = []float32{11, 13, 16}
	Weights = []float32{0, 0.4}
)

// Report formats the comparison of a label drawn as the product draws it and the
// reference drawn in one native call, for a log.
func Report(label string, produced, reference Sharpness) string {
	return fmt.Sprintf("%-28s edge %.3f / %.3f = %.3f   solid %.3f / %.3f = %.3f   ink %.3f", label,
		produced.Edge, reference.Edge, produced.Edge/reference.Edge,
		produced.Solid, reference.Solid, produced.Solid/reference.Solid, produced.Ink/reference.Ink)
}
