// Package linebreak finds line break opportunities in UTF-8 text following
// Unicode Standard Annex #14 (Unicode 18.0.0).
//
// A [Scanner] walks a string once, left to right, and reports every byte offset at
// which a line may end. The scanner is a small value type: it needs no allocation,
// starts no goroutine and keeps no tables of its own; character properties come from
// package uniprops.
//
// # Algorithm
//
// Rules LB2 to LB31 (including LB8a, LB9, LB10, LB15a to LB15d, LB19a, LB20a, LB21a,
// LB21b, LB25, LB27, LB28a, LB30, LB30a and LB30b) are evaluated in their published
// order for each boundary between two characters. The rules that look at several
// characters are carried as a few bytes of state: the effective class of the previous
// character (after LB9 has attached combining marks and ZWJ to their base), the class
// before a run of spaces, the number state of LB25, the regional indicator run and the
// Brahmic syllable flags. A handful of rules (LB15b, LB15c, LB19a, LB25, LB28a) need to
// see one or two characters ahead, which the scanner does by decoding forward without
// moving. There is no pair table.
//
// LB1 resolves the classes UAX #14 leaves to the implementation: AI, SG and XX become
// AL, CJ becomes NS, and SA becomes CM for Mn and Mc and AL otherwise. The end of the
// text is a mandatory break (LB3); hard line breaks (LB4, LB5) are reported as
// [Mandatory].
//
// # Limits
//
//   - There is no dictionary segmentation. Thai, Lao, Khmer and Myanmar text is one
//     unbreakable run per UAX #14 unless [Scanner.Fallback] is set, which adds
//     approximate opportunities; see fallback.go.
//   - There is no locale tailoring: the default behaviour of every tailorable rule is
//     used, and CB is not resolved (LB20 breaks around it).
//   - Invalid UTF-8 is not rejected: every invalid byte is one character of class AL.
//   - East Asian Width F, W and H are recognized for LB19a and LB30; ambiguous width is
//     treated as narrow.
package linebreak

import (
	"unicode/utf8"

	"github.com/vivaui/viva/internal/uniprops"
)

// Kind tells whether a line break at an offset is optional or required.
type Kind uint8

const (
	// Allowed is a break opportunity: a line may end here.
	Allowed Kind = iota
	// Mandatory is a hard break (LB4, LB5) or the end of the text (LB3).
	Mandatory
)

type class = uniprops.LineBreakClass

const (
	clAK  = uniprops.LBAK
	clAL  = uniprops.LBAL
	clAP  = uniprops.LBAP
	clAS  = uniprops.LBAS
	clB2  = uniprops.LBB2
	clBA  = uniprops.LBBA
	clBB  = uniprops.LBBB
	clBK  = uniprops.LBBK
	clCB  = uniprops.LBCB
	clCL  = uniprops.LBCL
	clCM  = uniprops.LBCM
	clCP  = uniprops.LBCP
	clCR  = uniprops.LBCR
	clEB  = uniprops.LBEB
	clEM  = uniprops.LBEM
	clEX  = uniprops.LBEX
	clGL  = uniprops.LBGL
	clH2  = uniprops.LBH2
	clH3  = uniprops.LBH3
	clHH  = uniprops.LBHH
	clHL  = uniprops.LBHL
	clHY  = uniprops.LBHY
	clID  = uniprops.LBID
	clIN  = uniprops.LBIN
	clIS  = uniprops.LBIS
	clJL  = uniprops.LBJL
	clJT  = uniprops.LBJT
	clJV  = uniprops.LBJV
	clLF  = uniprops.LBLF
	clNL  = uniprops.LBNL
	clNS  = uniprops.LBNS
	clNU  = uniprops.LBNU
	clOP  = uniprops.LBOP
	clPO  = uniprops.LBPO
	clPR  = uniprops.LBPR
	clQU  = uniprops.LBQU
	clRI  = uniprops.LBRI
	clSP  = uniprops.LBSP
	clSY  = uniprops.LBSY
	clVF  = uniprops.LBVF
	clVI  = uniprops.LBVI
	clWJ  = uniprops.LBWJ
	clZW  = uniprops.LBZW
	clZWJ = uniprops.LBZWJ

	// clNone is the class of "no character": before the text (sot) and after it (eot).
	clNone class = 255
)

const dottedCircle = 0x25CC

// Scanner reports the break opportunities of one text in order. The zero value is
// ready for Init. Set Fallback before Init.
type Scanner struct {
	// Fallback adds approximate break opportunities inside Thai, Lao, Khmer and
	// Myanmar text on top of UAX #14. See fallback.go for what it does and does not do.
	Fallback bool

	text string
	pos  int // offset of the next undecoded character

	p     class // effective class of the previous character (LB9 applied), clNone at sot
	bs    class // effective class of the last non-SP character, clNone at sot
	pRune rune  // the previous character's code point ('A' for a CM resolved by LB10)

	prevRaw rune // the previous character as written, for the fallback

	num uint8 // LB25: 1 after NU (SY|IS)*, 2 after NU (SY|IS)* (CL|CP)
	ri  uint8 // length of the run of regional indicators ending at the previous character

	started  bool
	plainAL  bool // the state is exactly that after an AL with no other flag set
	done     bool
	zwj      bool // the previous character is a ZWJ (LB8a)
	pending  bool // LB15a: an initial quote and spaces precede
	pPf      bool // the previous character is a final-punctuation quote (LB19)
	quPrior  bool // the previous quote follows sot or a non East Asian character (LB19a)
	hyCtx    bool // the previous hyphen starts a word (LB20a)
	hyHL     bool // the previous hyphen follows a Hebrew letter (LB21a)
	pAK, pAS bool // the previous character is AK or U+25CC, or AS (LB28a)
	viOK     bool // the previous character is VI after AK, U+25CC or AS (LB28a)
}

// Init prepares s to scan text and discards any earlier state. Fallback is kept.
func (s *Scanner) Init(text string) {
	*s = Scanner{Fallback: s.Fallback, text: text, p: clNone, bs: clNone}
}

// Next returns the offset of the next break opportunity: a line may end just before
// text[end:]. Offsets increase strictly. The last one is len(text), always
// Mandatory (LB3); after it, and for an empty text, ok is false.
func (s *Scanner) Next() (end int, kind Kind, ok bool) {
	if s.done {
		return 0, 0, false
	}
	for s.pos < len(s.text) {
		i := s.pos
		if s.plainAL {
			// A letter after a plain AL leaves the state as it is and never breaks.
			j := i
			for j < len(s.text) && isASCIILetter(s.text[j]) {
				j++
			}
			if j > i {
				s.pos, s.pRune, s.prevRaw = j, rune(s.text[j-1]), rune(s.text[j-1])
				continue
			}
		}
		r, n := decode(s.text, i)
		s.pos = i + n
		if b := s.step(r, i, s.pos); b != 0 && i > 0 {
			return i, Kind(b - 1), true
		}
	}
	s.done = true
	if len(s.text) == 0 {
		return 0, 0, false
	}
	return len(s.text), Mandatory, true
}

// Breaks appends every break opportunity of text, mandatory ones included, to dst.
func Breaks(dst []int, text string) []int {
	var s Scanner
	s.Init(text)
	for {
		end, _, ok := s.Next()
		if !ok {
			return dst
		}
		dst = append(dst, end)
	}
}

func isASCIILetter(c byte) bool { return c|0x20 >= 'a' && c|0x20 <= 'z' }

// asciiClass holds the resolved line break class of every ASCII character.
var asciiClass = func() (t [128]class) {
	for r := range t {
		t[r] = resolve(uniprops.LineBreak(rune(r)), rune(r))
	}
	return
}()

func decode(s string, i int) (rune, int) {
	if c := s[i]; c < utf8.RuneSelf {
		return rune(c), 1
	}
	return utf8.DecodeRuneInString(s[i:])
}

// resolve applies LB1.
func resolve(c class, r rune) class {
	switch c {
	case uniprops.LBAI, uniprops.LBSG, uniprops.LBXX:
		return clAL
	case uniprops.LBCJ:
		return clNS
	case uniprops.LBSA:
		if uniprops.Mark(r) {
			return clCM
		}
		return clAL
	}
	return c
}

// peek returns the effective class and code point of the first character at or after
// offset j that is not a combining mark or ZWJ, and the offset after it. The marks
// belong to the character before j, so this is only valid when that character is not a
// hard break, space or ZW. At the end of the text it returns clNone.
func (s *Scanner) peek(j int) (class, rune, int) {
	for j < len(s.text) {
		r, n := decode(s.text, j)
		j += n
		c := resolve(uniprops.LineBreak(r), r)
		if c == clCM || c == clZWJ {
			continue
		}
		return c, r, j
	}
	return clNone, 0, j
}

// eaw reports East Asian Width F, W or H, the set $EastAsian of UAX #14.
// uniprops knows F and W; the halfwidth ranges are the ones in EastAsianWidth.txt.
func eaw(r rune) bool {
	if r < 0x1100 {
		return false
	}
	if uniprops.Wide(r) {
		return true
	}
	switch {
	case r == 0x20A9:
		return true
	case r >= 0xFF61 && r <= 0xFFBE, r >= 0xFFC2 && r <= 0xFFC7, r >= 0xFFCA && r <= 0xFFCF,
		r >= 0xFFD2 && r <= 0xFFD7, r >= 0xFFDA && r <= 0xFFDC, r >= 0xFFE8 && r <= 0xFFEE:
		return true
	}
	return false
}

// step consumes the character r at [i, after) and returns 0 when there is no break
// before it, 1 for an allowed break and 2 for a mandatory one.
func (s *Scanner) step(r rune, i, after int) uint8 {
	var raw, c class
	if r < utf8.RuneSelf {
		raw = asciiClass[r]
		c = raw
	} else {
		raw = uniprops.LineBreak(r)
		c = resolve(raw, r)
	}
	cr := r
	var brk uint8
	if !s.started {
		s.started = true
		if c == clCM || c == clZWJ {
			c, cr = clAL, 'A'
		}
	} else {
		if c == clCM || c == clZWJ {
			switch s.p {
			case clBK, clCR, clLF, clNL, clSP, clZW:
				c, cr = clAL, 'A' // LB10
			default:
				// LB9: the mark joins the preceding character; nothing else changes.
				s.zwj = raw == clZWJ
				s.prevRaw = r
				return 0
			}
		}
		brk = s.decide(c, cr, after)
		if brk == 0 && s.Fallback && r >= 0xE00 && r <= 0x17FF && saBreakBefore(s.prevRaw, r, s.text[after:]) {
			brk = 1
		}
	}
	s.push(c, cr, raw)
	s.prevRaw = r
	return brk
}

// push records c, the effective class of the character just consumed, as the previous
// character.
func (s *Scanner) push(c class, cr rune, raw class) {
	prior := s.p
	s.pending = (c == clSP && s.pending) ||
		(c == clQU && uniprops.InitialQuote(cr) && isOneOf(prior, clNone, clBK, clCR, clLF, clNL, clOP, clQU, clGL, clSP, clZW))
	s.hyCtx, s.hyHL = false, false
	if c == clHY || c == clHH {
		s.hyCtx = isOneOf(prior, clNone, clBK, clCR, clLF, clNL, clSP, clZW, clCB, clGL)
		s.hyHL = prior == clHL
	}
	s.pPf, s.quPrior = false, false
	if c == clQU {
		s.pPf = uniprops.FinalQuote(cr)
		s.quPrior = prior == clNone || !eaw(s.pRune)
	}
	s.viOK = c == clVI && (s.pAK || s.pAS)
	s.pAK = c == clAK || cr == dottedCircle
	s.pAS = c == clAS
	switch {
	case c == clNU:
		s.num = 1
	case (c == clSY || c == clIS) && s.num == 1:
	case (c == clCL || c == clCP) && s.num == 1:
		s.num = 2
	default:
		s.num = 0
	}
	if c == clRI {
		if prior == clRI {
			s.ri++
		} else {
			s.ri = 1
		}
	} else {
		s.ri = 0
	}
	if c != clSP {
		s.bs = c
	}
	s.zwj = raw == clZWJ
	s.p = c
	s.pRune = cr
	s.plainAL = c == clAL && !s.zwj && !s.pAK
}

func isOneOf(c class, set ...class) bool {
	for _, x := range set {
		if c == x {
			return true
		}
	}
	return false
}

// decide evaluates LB4 to LB31 for the boundary before a character of effective class c
// (code point cr, ending at offset after) that is not absorbed by LB9. It returns 0, 1
// or 2 as step does.
func (s *Scanner) decide(c class, cr rune, after int) uint8 {
	p := s.p
	// Hot paths: letters after letters, and a letter after a single space.
	if c == clAL {
		if p == clAL {
			return 0
		}
		if p == clSP && !s.pending && s.bs != clZW && s.bs != clOP && !s.zwj {
			return 1
		}
	}
	// LB4, LB5.
	switch p {
	case clBK, clLF, clNL:
		return 2
	case clCR:
		if c == clLF {
			return 0
		}
		return 2
	}
	// LB6, LB7.
	switch c {
	case clBK, clCR, clLF, clNL, clSP, clZW:
		return 0
	}
	// LB8, LB8a.
	if s.bs == clZW {
		return 1
	}
	if s.zwj {
		return 0
	}
	// LB11, LB12, LB12a.
	if c == clWJ || p == clWJ || p == clGL {
		return 0
	}
	if c == clGL && p != clSP && p != clHY && p != clHH {
		return 0
	}
	// LB13, LB14.
	switch c {
	case clCL, clCP, clEX, clSY:
		return 0
	}
	if s.bs == clOP {
		return 0
	}
	// LB15a to LB15d.
	if s.pending {
		return 0
	}
	if c == clQU && uniprops.FinalQuote(cr) {
		switch nc, _, _ := s.peek(after); nc {
		case clSP, clGL, clWJ, clCL, clQU, clCP, clEX, clIS, clSY, clBK, clCR, clLF, clNL, clZW, clNone:
			return 0
		}
	}
	if c == clIS {
		if p == clSP {
			if nc, _, _ := s.peek(after); nc == clNU {
				return 1
			}
		}
		return 0
	}
	// LB16, LB17.
	if (s.bs == clCL || s.bs == clCP) && c == clNS {
		return 0
	}
	if s.bs == clB2 && c == clB2 {
		return 0
	}
	// LB18.
	if p == clSP {
		return 1
	}
	// LB19, LB19a.
	if c == clQU && !uniprops.InitialQuote(cr) {
		return 0
	}
	if p == clQU && !s.pPf {
		return 0
	}
	if c == clQU {
		if !eaw(s.pRune) {
			return 0
		}
		if nc, nr, _ := s.peek(after); nc == clNone || !eaw(nr) {
			return 0
		}
	}
	if p == clQU && (!eaw(cr) || s.quPrior) {
		return 0
	}
	// LB20, LB20a.
	if c == clCB || p == clCB {
		return 1
	}
	if s.hyCtx && (c == clAL || c == clHL) {
		return 0
	}
	// LB21, LB21a, LB21b, LB22.
	if c == clBA || c == clHH || c == clHY || c == clNS || p == clBB {
		return 0
	}
	if s.hyHL && c != clHL {
		return 0
	}
	if p == clSY && c == clHL {
		return 0
	}
	if c == clIN {
		return 0
	}
	// LB23, LB23a, LB24.
	if (p == clAL || p == clHL) && c == clNU || p == clNU && (c == clAL || c == clHL) {
		return 0
	}
	if p == clPR && (c == clID || c == clEB || c == clEM) || (p == clID || p == clEB || p == clEM) && c == clPO {
		return 0
	}
	if (p == clPR || p == clPO) && (c == clAL || c == clHL) || (p == clAL || p == clHL) && (c == clPR || c == clPO) {
		return 0
	}
	// LB25.
	if s.num != 0 && (c == clPO || c == clPR) || s.num == 1 && c == clNU {
		return 0
	}
	if p == clPO || p == clPR {
		if c == clNU {
			return 0
		}
		if c == clOP {
			nc, _, j := s.peek(after)
			if nc == clIS {
				nc, _, _ = s.peek(j)
			}
			if nc == clNU {
				return 0
			}
		}
	}
	if (p == clHY || p == clIS) && c == clNU {
		return 0
	}
	// LB26, LB27.
	switch p {
	case clJL:
		if c == clJL || c == clJV || c == clH2 || c == clH3 {
			return 0
		}
	case clJV, clH2:
		if c == clJV || c == clJT {
			return 0
		}
	case clJT, clH3:
		if c == clJT {
			return 0
		}
	}
	if c == clPO && (p == clJL || p == clJV || p == clJT || p == clH2 || p == clH3) {
		return 0
	}
	if p == clPR && (c == clJL || c == clJV || c == clJT || c == clH2 || c == clH3) {
		return 0
	}
	// LB28, LB28a.
	if (p == clAL || p == clHL) && (c == clAL || c == clHL) {
		return 0
	}
	cAK := c == clAK || cr == dottedCircle
	if p == clAP && (cAK || c == clAS) {
		return 0
	}
	if (s.pAK || s.pAS) && (c == clVF || c == clVI) {
		return 0
	}
	if s.viOK && cAK {
		return 0
	}
	if (s.pAK || s.pAS) && (cAK || c == clAS) {
		if nc, _, _ := s.peek(after); nc == clVF {
			return 0
		}
	}
	// LB29, LB30.
	if p == clIS && (c == clAL || c == clHL) {
		return 0
	}
	if (p == clAL || p == clHL || p == clNU) && c == clOP && !eaw(cr) {
		return 0
	}
	if p == clCP && (c == clAL || c == clHL || c == clNU) && !eaw(s.pRune) {
		return 0
	}
	// LB30a, LB30b.
	if p == clRI && c == clRI && s.ri%2 == 1 {
		return 0
	}
	if c == clEM && (p == clEB || uniprops.ExtendedPictographic(s.pRune) && uniprops.Unassigned(s.pRune)) {
		return 0
	}
	// LB31.
	return 1
}
