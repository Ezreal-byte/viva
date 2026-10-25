package bidi

import (
	"slices"
	"unicode/utf8"

	"github.com/vivaui/viva/internal/uniprops"
)

const retainMax = 1 << 14 // runes of scratch kept across calls

type override uint8

const (
	ovNone override = iota
	ovL
	ovR
)

type status struct {
	level    uint8
	override override
	isolate  bool
}

type pair struct{ open, close int32 }

type opener struct {
	closer rune
	pos    int32
}

// Resolver resolves paragraphs and lines. The zero value is ready to use.
type Resolver struct {
	runes    []rune
	offs     []int32
	orig     []class
	types    []class
	levels   []uint8
	elev     []uint8 // explicit levels (X1-X8)
	match    []int32 // isolate initiator -> PDI (n if none), PDI -> initiator (-1 if none)
	pos      []int32 // rune index -> position in idx
	idx      []int32 // rune indices that survive X9
	runStart []int32 // start positions in idx of the level runs, plus len(idx)
	runOf    []int32 // position in idx -> level run
	seq      []int32 // positions in idx of the current isolating run sequence
	stk      []int32
	st       []class
	pairs    []pair
	line     []uint8
	stack    [MaxDepth + 3]status
}

func sized[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}

// Resolve runs the algorithm up to I2 on text as one paragraph. The result aliases the
// Resolver until its next call.
func (r *Resolver) Resolve(text string, base Direction) Paragraph {
	if len(text) > maxText {
		text = text[:maxText]
	}
	if cap(r.runes) > retainMax && len(text) <= retainMax {
		*r = Resolver{}
	}
	r.runes, r.offs, r.orig = r.runes[:0], r.offs[:0], r.orig[:0]
	needs := false
	for i := 0; i < len(text); {
		var ru rune
		w := 1
		if c := text[i]; c < utf8.RuneSelf {
			ru = rune(c)
		} else {
			ru, w = utf8.DecodeRuneInString(text[i:])
		}
		c := uniprops.Bidi(ru)
		if c == cR || c == cAL || c == cAN || c >= cLRE {
			needs = true
		}
		r.runes = append(r.runes, ru)
		r.offs = append(r.offs, int32(i))
		r.orig = append(r.orig, c)
		i += w
	}
	n := len(r.runes)
	r.offs = append(r.offs, int32(len(text)))
	r.levels = sized(r.levels, n)
	p := Paragraph{levels: r.levels, offs: r.offs, orig: r.orig}
	if !needs && base != RTL {
		clear(r.levels)
		return p
	}
	r.match = sized(r.match, n)
	r.matchIsolates(n)
	switch base {
	case RTL:
		p.Level = 1
	case Auto:
		if r.firstStrong(0, n) == 1 {
			p.Level = 1
		}
	}
	r.explicit(n, p.Level)
	r.sequences(n, p.Level)
	for i := 0; i < n; i++ { // removed characters and separators
		switch {
		case r.orig[i] == cB:
			r.levels[i] = p.Level
		case r.types[i] == cBN:
			if i == 0 {
				r.levels[i] = p.Level
			} else {
				r.levels[i] = r.levels[i-1]
			}
		}
	}
	return p
}

func isInitiator(c class) bool { return c == cLRI || c == cRLI || c == cFSI }

// matchIsolates pairs isolate initiators with their PDIs (BD9).
func (r *Resolver) matchIsolates(n int) {
	r.stk = r.stk[:0]
	for i := 0; i < n; i++ {
		switch c := r.orig[i]; {
		case isInitiator(c):
			r.match[i] = int32(n)
			r.stk = append(r.stk, int32(i))
		case c == cPDI:
			r.match[i] = -1
			if k := len(r.stk); k > 0 {
				j := r.stk[k-1]
				r.stk = r.stk[:k-1]
				r.match[j], r.match[i] = int32(i), j
			}
		}
	}
}

// firstStrong applies P2 to runes [from, to): 0 for L, 1 for R or AL, -1 for none.
func (r *Resolver) firstStrong(from, to int) int {
	for i := from; i < to; i++ {
		switch c := r.orig[i]; {
		case c == cL:
			return 0
		case c == cR || c == cAL:
			return 1
		case isInitiator(c):
			i = int(r.match[i]) // continue after the matching PDI
		}
	}
	return -1
}

// explicit applies X1-X8, filling elev and types. Characters removed by X9 get cBN.
func (r *Resolver) explicit(n int, para uint8) {
	r.types = sized(r.types, n)
	r.elev = sized(r.elev, n)
	stack := &r.stack
	sp := 0
	stack[0] = status{level: para}
	overflowIso, overflowEmb, validIso := 0, 0, 0
	apply := func(t class, o override) class {
		switch o {
		case ovL:
			return cL
		case ovR:
			return cR
		}
		return t
	}
	for i := 0; i < n; i++ {
		t := r.orig[i]
		top := stack[sp]
		switch t {
		case cRLE, cLRE, cRLO, cLRO:
			r.types[i], r.elev[i] = cBN, top.level
			var nl uint8
			if t == cRLE || t == cRLO {
				nl = (top.level + 1) | 1
			} else {
				nl = (top.level + 2) &^ 1
			}
			if nl <= MaxDepth && overflowIso == 0 && overflowEmb == 0 {
				ov := ovNone
				if t == cRLO {
					ov = ovR
				} else if t == cLRO {
					ov = ovL
				}
				sp++
				stack[sp] = status{nl, ov, false}
			} else if overflowIso == 0 {
				overflowEmb++
			}
		case cRLI, cLRI, cFSI:
			r.types[i], r.elev[i] = apply(t, top.override), top.level
			rtl := t == cRLI
			if t == cFSI {
				rtl = r.firstStrong(i+1, int(r.match[i])) == 1
			}
			var nl uint8
			if rtl {
				nl = (top.level + 1) | 1
			} else {
				nl = (top.level + 2) &^ 1
			}
			if nl <= MaxDepth && overflowIso == 0 && overflowEmb == 0 {
				validIso++
				sp++
				stack[sp] = status{nl, ovNone, true}
			} else {
				overflowIso++
			}
		case cPDI:
			if overflowIso > 0 {
				overflowIso--
			} else if validIso > 0 {
				overflowEmb = 0
				for !stack[sp].isolate {
					sp--
				}
				sp--
				validIso--
			}
			top = stack[sp]
			r.types[i], r.elev[i] = apply(t, top.override), top.level
		case cPDF:
			r.types[i], r.elev[i] = cBN, top.level
			if overflowIso > 0 {
			} else if overflowEmb > 0 {
				overflowEmb--
			} else if !top.isolate && sp >= 1 {
				sp--
			}
		case cB:
			r.types[i], r.elev[i] = cB, para
		case cBN:
			r.types[i], r.elev[i] = cBN, top.level
		default:
			r.types[i], r.elev[i] = apply(t, top.override), top.level
		}
	}
}

// sequences builds the isolating run sequences (X10) and resolves each of them.
func (r *Resolver) sequences(n int, para uint8) {
	r.pos = sized(r.pos, n)
	r.idx = r.idx[:0]
	r.runStart = r.runStart[:0]
	r.runOf = r.runOf[:0]
	for i := 0; i < n; i++ {
		if r.types[i] == cBN {
			continue
		}
		p := len(r.idx)
		if p == 0 || r.elev[i] != r.elev[r.idx[p-1]] {
			r.runStart = append(r.runStart, int32(p))
		}
		r.pos[i] = int32(p)
		r.idx = append(r.idx, int32(i))
		r.runOf = append(r.runOf, int32(len(r.runStart)-1))
	}
	nruns := len(r.runStart)
	r.runStart = append(r.runStart, int32(len(r.idx)))
	for k := 0; k < nruns; k++ {
		first := r.idx[r.runStart[k]]
		if r.orig[first] == cPDI && r.match[first] >= 0 {
			continue // continues the sequence of its initiator
		}
		r.seq = r.seq[:0]
		for cur := k; ; {
			for p := r.runStart[cur]; p < r.runStart[cur+1]; p++ {
				r.seq = append(r.seq, p)
			}
			last := r.idx[r.runStart[cur+1]-1]
			if isInitiator(r.orig[last]) && int(r.match[last]) < n {
				cur = int(r.runOf[r.pos[r.match[last]]])
				continue
			}
			break
		}
		r.resolveSequence(para)
	}
}

func dirOfLevel(l uint8) class {
	if l&1 != 0 {
		return cR
	}
	return cL
}

// resolveSequence applies W1-W7, N0-N2 and I1-I2 to r.seq.
func (r *Resolver) resolveSequence(para uint8) {
	seq := r.seq
	m := len(seq)
	level := r.elev[r.idx[seq[0]]]
	prevLevel, nextLevel := para, para
	if p := seq[0]; p > 0 {
		prevLevel = r.elev[r.idx[p-1]]
	}
	if p := seq[m-1]; int(p)+1 < len(r.idx) && !isInitiator(r.orig[r.idx[p]]) {
		nextLevel = r.elev[r.idx[p+1]]
	}
	sos, eos := dirOfLevel(max(level, prevLevel)), dirOfLevel(max(level, nextLevel))

	r.st = r.st[:0]
	for _, p := range seq {
		r.st = append(r.st, r.types[r.idx[p]])
	}
	st := r.st

	prev := sos // W1
	for k, t := range st {
		if t == cNSM {
			if isInitiator(prev) || prev == cPDI {
				t = cON
			} else {
				t = prev
			}
			st[k] = t
		}
		prev = t
	}
	last := sos // W2, W3
	for k, t := range st {
		switch t {
		case cL, cR:
			last = t
		case cAL:
			last = cAL
			st[k] = cR
		case cEN:
			if last == cAL {
				st[k] = cAN
			}
		}
	}
	for k := 1; k+1 < m; k++ { // W4
		switch t := st[k]; t {
		case cES:
			if st[k-1] == cEN && st[k+1] == cEN {
				st[k] = cEN
			}
		case cCS:
			if p := st[k-1]; (p == cEN || p == cAN) && st[k+1] == p {
				st[k] = p
			}
		}
	}
	for k := 0; k < m; { // W5
		if st[k] != cET {
			k++
			continue
		}
		e := k
		for e < m && st[e] == cET {
			e++
		}
		if (k > 0 && st[k-1] == cEN) || (e < m && st[e] == cEN) {
			for j := k; j < e; j++ {
				st[j] = cEN
			}
		}
		k = e
	}
	last = sos
	for k, t := range st { // W6, W7
		switch t {
		case cES, cET, cCS:
			st[k] = cON
		case cL, cR:
			last = t
		case cEN:
			if last == cL {
				st[k] = cL
			}
		}
	}

	r.resolveBrackets(level, sos)

	emb := dirOfLevel(level)
	for k := 0; k < m; { // N1, N2
		if !isNI(st[k]) {
			k++
			continue
		}
		e := k
		for e < m && isNI(st[e]) {
			e++
		}
		lead, trail := sos, eos
		if k > 0 {
			lead = strongOf(st[k-1])
		}
		if e < m {
			trail = strongOf(st[e])
		}
		d := emb
		if lead == trail {
			d = lead
		}
		for j := k; j < e; j++ {
			st[j] = d
		}
		k = e
	}
	for k, t := range st { // I1, I2
		lv := level
		if lv&1 == 0 {
			switch t {
			case cR:
				lv++
			case cAN, cEN:
				lv += 2
			}
		} else if t != cR {
			lv++
		}
		r.levels[r.idx[seq[k]]] = lv
	}
}

func isNI(t class) bool {
	switch t {
	case cB, cS, cWS, cON, cFSI, cLRI, cRLI, cPDI:
		return true
	}
	return false
}

// strongOf maps a resolved class to L or R, EN and AN counting as R; 0xFF otherwise.
func strongOf(t class) class {
	switch t {
	case cL:
		return cL
	case cR, cEN, cAN:
		return cR
	}
	return 0xFF
}

func canonicalBracket(r rune) rune {
	switch r {
	case 0x2329:
		return 0x3008
	case 0x232A:
		return 0x3009
	}
	return r
}

// resolveBrackets is N0 over r.st.
func (r *Resolver) resolveBrackets(level uint8, sos class) {
	st, seq := r.st, r.seq
	const maxPairing = 63
	var stk [maxPairing]opener
	nst := 0
	r.pairs = r.pairs[:0]
scan:
	for k, t := range st {
		if t != cON {
			continue
		}
		ru := r.runes[r.idx[seq[k]]]
		b, ok := uniprops.LookupBracket(ru)
		if !ok {
			continue
		}
		if b.Open {
			if nst == maxPairing {
				break scan
			}
			stk[nst] = opener{canonicalBracket(b.Pair), int32(k)}
			nst++
			continue
		}
		c := canonicalBracket(ru)
		for s := nst - 1; s >= 0; s-- {
			if stk[s].closer == c {
				r.pairs = append(r.pairs, pair{stk[s].pos, int32(k)})
				nst = s
				break
			}
		}
	}
	if len(r.pairs) == 0 {
		return
	}
	if len(r.pairs) > 1 {
		slices.SortFunc(r.pairs, func(a, b pair) int { return int(a.open) - int(b.open) })
	}
	emb := dirOfLevel(level)
	for _, pr := range r.pairs {
		o, c := int(pr.open), int(pr.close)
		foundE, foundO := false, false
		for k := o + 1; k < c; k++ {
			switch s := strongOf(st[k]); {
			case s == emb:
				foundE = true
			case s != 0xFF:
				foundO = true
			}
			if foundE {
				break
			}
		}
		var d class
		switch {
		case foundE:
			d = emb
		case foundO:
			ctx := sos
			for k := o - 1; k >= 0; k-- {
				if s := strongOf(st[k]); s != 0xFF {
					ctx = s
					break
				}
			}
			if ctx != emb {
				d = ctx
			} else {
				d = emb
			}
		default:
			continue
		}
		st[o], st[c] = d, d
		for k := o + 1; k < len(st) && r.orig[r.idx[seq[k]]] == cNSM; k++ {
			st[k] = d
		}
		for k := c + 1; k < len(st) && r.orig[r.idx[seq[k]]] == cNSM; k++ {
			st[k] = d
		}
	}
}
