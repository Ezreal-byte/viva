package uniprops

import "testing"

func TestKnownProperties(t *testing.T) {
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"A line break", LineBreak('A'), LBAL},
		{"space line break", LineBreak(' '), LBSP},
		{"ideograph line break", LineBreak('中'), LBID},
		{"Thai line break", LineBreak('ก'), LBSA},
		{"hebrew bidi", Bidi('א'), BidiR},
		{"arabic bidi", Bidi('ب'), BidiAL},
		{"digit bidi", Bidi('1'), BidiEN},
		{"tag line break", LineBreak(0xE0041), LBCM},
		{"hangul LV", Grapheme('가'), GBLV},
		{"hangul LVT", Grapheme('각'), GBLVT},
		{"combining acute", Grapheme(0x301), GBExtend},
		{"emoji", ExtendedPictographic('😀'), true},
		{"virama", ConjunctBreak(0x94D), InCBLinker},
		{"devanagari ka", ConjunctBreak(0x915), InCBConsonant},
		{"beh joins both sides", Joining(0x628), JoinD},
		{"alef joins right", Joining(0x627), JoinR},
		{"fatha is transparent", Joining(0x64E), JoinT},
		{"ideograph is wide", Wide('中'), true},
		{"latin is not wide", Wide('a'), false},
		{"left quote is Pi", InitialQuote('“'), true},
		{"mark", Mark(0x301), true},
		{"mirror of (", Mirror('('), rune(')')},
		{"mirror of letter", Mirror('a'), rune('a')},
		{"private use plane", LineBreak(0x10FFFD), LBXX},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
	if b, ok := LookupBracket('['); !ok || b.Pair != ']' || !b.Open {
		t.Errorf("bracket [ = %+v %v", b, ok)
	}
	if _, ok := LookupBracket('a'); ok {
		t.Error("a is not a bracket")
	}
}
