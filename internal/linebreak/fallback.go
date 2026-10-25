package linebreak

// saBreakBefore reports whether the fallback adds a break opportunity between prev and
// cur, two adjacent characters that UAX #14 keeps together, and rest is the text after
// cur.
//
// This is a dictionary-less approximation, not word segmentation. UAX #14 gives no
// break opportunities inside Thai, Lao, Khmer or Myanmar words because it cannot know
// where words end without a dictionary. The fallback instead breaks at likely syllable
// starts so that a long run can wrap. It will split some words in the middle (final
// consonants and glides after a vowel sign, such as the ย of เกี่ยว, look like syllable
// starts) and will not find every word boundary. It never breaks before a combining
// mark, a repetition sign (ๆ) or an abbreviation sign (ฯ), nor between characters of
// different scripts. Use a real dictionary segmenter when quality matters.
//
//   - Thai (U+0E01..U+0E5B) and Lao (U+0E81..U+0EDF): before a leading vowel (Thai
//     U+0E40..U+0E44, Lao U+0EC0..U+0EC4) that follows any character other than another
//     leading vowel; and before a consonant that follows a vowel sign or tone mark that
//     ends a syllable.
//   - Khmer (U+1780..U+17FF): before a consonant (U+1780..U+17A2) unless it follows
//     COENG (U+17D2), which subscripts it.
//   - Myanmar (U+1000..U+109F): before a consonant (U+1000..U+1021) unless it follows
//     the virama U+1039, or is followed by the virama or the asat U+103A.
func saBreakBefore(prev, cur rune, rest string) bool {
	switch {
	case cur >= 0xE00 && cur <= 0xEFF:
		thai := prev >= 0xE01 && prev <= 0xE5B && cur >= 0xE01 && cur <= 0xE5B
		lao := prev >= 0xE81 && prev <= 0xEDF && cur >= 0xE81 && cur <= 0xEDF
		switch {
		case thai:
			if cur >= 0xE40 && cur <= 0xE44 {
				return !(prev >= 0xE40 && prev <= 0xE44)
			}
			return cur <= 0xE2E && thaiSyllableEnd(prev)
		case lao:
			if cur >= 0xEC0 && cur <= 0xEC4 {
				return !(prev >= 0xEC0 && prev <= 0xEC4)
			}
			return cur >= 0xE81 && cur <= 0xEAE && laoSyllableEnd(prev)
		}
	case cur >= 0x1780 && cur <= 0x17FF:
		return prev >= 0x1780 && prev <= 0x17FF && cur <= 0x17A2 && prev != 0x17D2
	case cur >= 0x1000 && cur <= 0x109F:
		if prev < 0x1000 || prev > 0x109F || cur > 0x1021 || prev == 0x1039 {
			return false
		}
		if rest != "" {
			if next, _ := decode(rest, 0); next == 0x1039 || next == 0x103A {
				return false
			}
		}
		return true
	}
	return false
}

// thaiSyllableEnd reports whether r is a Thai vowel sign or tone mark that ends a
// syllable: sara a, aa, i-family long vowels, u, uu and the four tone marks.
func thaiSyllableEnd(r rune) bool {
	switch r {
	case 0xE30, 0xE32, 0xE35, 0xE37, 0xE38, 0xE39, 0xE48, 0xE49, 0xE4A, 0xE4B:
		return true
	}
	return false
}

// laoSyllableEnd is thaiSyllableEnd for Lao.
func laoSyllableEnd(r rune) bool {
	switch r {
	case 0xEB0, 0xEB2, 0xEB5, 0xEB7, 0xEB8, 0xEB9, 0xEC8, 0xEC9, 0xECA, 0xECB:
		return true
	}
	return false
}
