package bidi

// runeSpan maps the byte range [start, end) to rune indices [a, b); offsets inside a
// rune snap forward to the next rune boundary.
func (p Paragraph) runeSpan(start, end int) (a, b int) {
	n := len(p.levels)
	find := func(off int) int {
		lo, hi := 0, n
		for lo < hi {
			mid := int(uint(lo+hi) >> 1)
			if int(p.offs[mid]) < off {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		return lo
	}
	a, b = find(start), find(end)
	if b < a {
		b = a
	}
	return a, b
}

// lineLevels applies L1 to the runes [a, b) of one line, writing into dst[:b-a].
func (p Paragraph) lineLevels(a, b int, dst []uint8) {
	copy(dst, p.levels[a:b])
	trailing := true
	for i := b - 1; i >= a; i-- {
		switch c := p.orig[i]; c {
		case cB, cS:
			dst[i-a] = p.Level
			trailing = true
		case cWS, cLRI, cRLI, cFSI, cPDI, cBN, cLRE, cRLE, cLRO, cRLO, cPDF:
			if trailing {
				dst[i-a] = p.Level
			}
		default:
			trailing = false
		}
	}
}

// LineLevels applies L1 to the line [start, end) (byte offsets of the paragraph text)
// and returns the resolved level of each rune of the line in dst (reallocated only if too small).
func (r *Resolver) LineLevels(p Paragraph, start, end int, dst []uint8) []uint8 {
	a, b := p.runeSpan(start, end)
	dst = sized(dst, b-a)
	p.lineLevels(a, b, dst)
	return dst
}

// LineRuns applies L1 and L2 to the line [start, end) (byte offsets into the paragraph
// text, snapped forward to rune boundaries) and returns its runs in visual order, left
// to right, in dst[:0] extended as needed. Each run is a maximal range of equal level;
// an odd level means its characters are displayed in reverse logical order. Pass a
// dst with enough capacity to avoid allocation.
func (r *Resolver) LineRuns(p Paragraph, start, end int, dst []Run) []Run {
	dst = dst[:0]
	a, b := p.runeSpan(start, end)
	if a == b {
		return dst
	}
	r.line = sized(r.line, b-a)
	p.lineLevels(a, b, r.line)
	lv := r.line
	hi, lowOdd := uint8(0), uint8(255)
	from := 0
	for i := 1; i <= len(lv); i++ {
		if i < len(lv) && lv[i] == lv[from] {
			continue
		}
		l := lv[from]
		dst = append(dst, Run{int(p.offs[a+from]), int(p.offs[a+i]), l})
		hi = max(hi, l)
		if l&1 != 0 {
			lowOdd = min(lowOdd, l)
		}
		from = i
	}
	for l := hi; l >= lowOdd && l > 0; l-- { // L2
		for i := 0; i < len(dst); {
			if dst[i].Level < l {
				i++
				continue
			}
			j := i
			for j < len(dst) && dst[j].Level >= l {
				j++
			}
			for x, y := i, j-1; x < y; x, y = x+1, y-1 {
				dst[x], dst[y] = dst[y], dst[x]
			}
			i = j
		}
	}
	return dst
}
