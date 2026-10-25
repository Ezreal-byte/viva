package textsharp

import (
	"image"
	"math"
	"sort"
)

// Plane is the coverage of a line of text on a background, row by row: 0 is the
// background, 1 is the text color, in the values that were blended (not light).
type Plane struct {
	Width, Height int
	// Pix is the luminance coverage of every pixel; Chroma the spread of the red,
	// green and blue coverages of the pixel (0 for grayscale smoothing, up to 1 for
	// a ClearType fringe).
	Pix, Chroma []float64
}

// Coverage turns the pixels of text drawn in the gray fg over the gray bg (both 0..255) into a
// Plane: the per-channel share of the distance from bg to fg that each pixel has
// travelled, the quantity every blend of text is defined by.
func Coverage(img image.Image, fg, bg float64) Plane {
	bounds := img.Bounds()
	plane := Plane{Width: bounds.Dx(), Height: bounds.Dy()}
	plane.Pix = make([]float64, plane.Width*plane.Height)
	plane.Chroma = make([]float64, plane.Width*plane.Height)
	span := fg - bg
	for y := 0; y < plane.Height; y++ {
		for x := 0; x < plane.Width; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// img is opaque in every use here: the values blended over the background.
			_ = a
			cr, cg, cb := (float64(r>>8)-bg)/span, (float64(g>>8)-bg)/span, (float64(b>>8)-bg)/span
			plane.Pix[y*plane.Width+x] = 0.299*cr + 0.587*cg + 0.114*cb
			plane.Chroma[y*plane.Width+x] = max(cr, cg, cb) - min(cr, cg, cb)
		}
	}
	return plane
}

func (p Plane) at(x, y int) float64 {
	if x < 0 || y < 0 || x >= p.Width || y >= p.Height {
		return 0
	}
	return p.Pix[y*p.Width+x]
}

// Ink is the summed coverage; Stem the mean coverage of the pixels that are at least half
// covered (how dark the stems are); Fringe the mean chroma per pixel of ink.
type Stats struct {
	Ink, Stem, Fringe float64
	Sharp             Sharpness
}

// Stats measures one plane.
func (p Plane) Stats() Stats {
	var ink, stem, fringe float64
	var stems int
	bytes := make([]byte, len(p.Pix))
	for i, c := range p.Pix {
		c = min(max(c, 0), 1)
		bytes[i] = byte(math.Round(c * 255))
		ink += c
		fringe += p.Chroma[i]
		if c >= 0.5 {
			stem += c
			stems++
		}
	}
	stats := Stats{Ink: ink, Sharp: Measure(bytes, p.Width, p.Height, 1, p.Width)}
	if stems > 0 {
		stats.Stem = stem / float64(stems)
	}
	if ink > 0 {
		stats.Fringe = fringe / ink
	}
	return stats
}

// Span is the horizontal extent of a run of ink between gaps wider than the gap given to Words.
type Span struct {
	// Left and Right are the edges of the ink with sub-pixel precision (the coverage
	// of the outermost column counts as a fraction of it), Center the ink's
	// coverage-weighted center.
	Left, Right, Center float64
}

// Spans cuts the plane into the words the ink forms: runs of columns with ink,
// separated by at least gap empty columns.
func (p Plane) Spans(gap int) []Span {
	columns := make([]float64, p.Width)
	for y := 0; y < p.Height; y++ {
		for x := 0; x < p.Width; x++ {
			columns[x] += max(p.Pix[y*p.Width+x], 0)
		}
	}
	const empty = 0.05
	var words []Span
	start, last := -1, -1
	flush := func() {
		if start < 0 {
			return
		}
		var total, moment float64
		for x := start; x <= last; x++ {
			total += columns[x]
			moment += columns[x] * (float64(x) + 0.5)
		}
		peak := 0.0
		for x := start; x <= last; x++ {
			peak = max(peak, columns[x])
		}
		word := Span{Center: moment / total}
		word.Left = float64(start) + 1 - min(1, columns[start]/(peak*0.5))
		word.Right = float64(last) + min(1, columns[last]/(peak*0.5))
		words = append(words, word)
		start = -1
	}
	for x := 0; x < p.Width; x++ {
		if columns[x] > empty {
			if start >= 0 && x-last > gap {
				flush()
			}
			if start < 0 {
				start = x
			}
			last = x
		}
	}
	flush()
	return words
}

// Vertical is the top and bottom edge, in rows, of the ink of capital letters with a
// flat foot: the baseline is Bottom. It measures the rows' ink against the median row
// of the ink, which the stems set.
func (p Plane) Vertical() (top, bottom float64) {
	rows := make([]float64, p.Height)
	var nonzero []float64
	for y := 0; y < p.Height; y++ {
		for x := 0; x < p.Width; x++ {
			rows[y] += max(p.Pix[y*p.Width+x], 0)
		}
		if rows[y] > 0.5 {
			nonzero = append(nonzero, rows[y])
		}
	}
	if len(nonzero) == 0 {
		return 0, 0
	}
	sort.Float64s(nonzero)
	median := nonzero[len(nonzero)/2]
	first, lastRow := -1, -1
	for y, v := range rows {
		if v >= 0.5*median {
			if first < 0 {
				first = y
			}
			lastRow = y
		}
	}
	top = float64(first)
	if first > 0 {
		top -= min(0.5, rows[first-1]/median)
	}
	bottom = float64(lastRow + 1)
	if lastRow+1 < p.Height {
		bottom += min(0.5, rows[lastRow+1]/median)
	}
	return top, bottom
}

// Fidelity compares text got with the reference ref of the same string, both as Planes.
type Fidelity struct {
	// Mass, Stem, Edge and Solid are got over ref: 1 is the same ink, stem darkness,
	// edge energy per ink and share of solid pixels.
	Mass, Stem, Edge, Solid float64
	// Fringe is the chroma of got and of ref per pixel of ink.
	Fringe, RefFringe float64
	// DX and DY are the shift of got against ref, in pixels, that matches best.
	DX, DY int
	// Residual is the mean absolute difference of the coverages after the shift, per
	// pixel of ink in ref: 0 for identical text, around 1 for text of another font.
	Residual float64
}

// Compare measures how close got is to ref: ink, stems, sharpness, fringes, and the
// pixel difference at the best shift of at most reachX pixels sideways and reachY up or down.
func Compare(got, ref Plane, reachX, reachY int) Fidelity {
	g, r := got.Stats(), ref.Stats()
	f := Fidelity{Fringe: g.Fringe, RefFringe: r.Fringe}
	if r.Ink == 0 || g.Ink == 0 {
		return f
	}
	f.Mass = g.Ink / r.Ink
	f.Stem = g.Stem / r.Stem
	f.Edge = g.Sharp.Edge / r.Sharp.Edge
	if r.Sharp.Solid > 0 {
		f.Solid = g.Sharp.Solid / r.Sharp.Solid
	}
	best := math.MaxFloat64
	for dy := -reachY; dy <= reachY; dy++ {
		for dx := -reachX; dx <= reachX; dx++ {
			sum := 0.0
			for y := 0; y < ref.Height; y++ {
				for x := 0; x < ref.Width; x++ {
					sum += math.Abs(got.at(x+dx, y+dy) - ref.Pix[y*ref.Width+x])
				}
			}
			if sum < best {
				best, f.DX, f.DY = sum, dx, dy
			}
		}
	}
	f.Residual = best / r.Ink
	return f
}

// Crop is the part of the plane between columns left and right (exclusive).
func (p Plane) Crop(left, right int) Plane {
	left, right = max(left, 0), min(right, p.Width)
	crop := Plane{Width: right - left, Height: p.Height}
	crop.Pix = make([]float64, crop.Width*crop.Height)
	crop.Chroma = make([]float64, crop.Width*crop.Height)
	for y := 0; y < p.Height; y++ {
		copy(crop.Pix[y*crop.Width:(y+1)*crop.Width], p.Pix[y*p.Width+left:y*p.Width+right])
		copy(crop.Chroma[y*crop.Width:(y+1)*crop.Width], p.Chroma[y*p.Width+left:y*p.Width+right])
	}
	return crop
}
