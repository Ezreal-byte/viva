package platform

// ColorGlyphs reports whether value holds a character a color font draws in color: the
// pictographs and emoji sequences. A backend draws such a label grayscale, its glyphs in their own
// colors, since a color glyph has no coverage to blend per channel.
func ColorGlyphs(value string) bool {
	for _, r := range value {
		switch {
		case r >= 0x1F000 && r <= 0x1FAFF, r == 0xFE0F, r == 0x200D, r >= 0x1F1E6 && r <= 0x1F1FF:
			return true
		case r >= 0x2600 && r <= 0x27BF, r >= 0x2300 && r <= 0x23FF, r >= 0x2B00 && r <= 0x2BFF, r >= 0x25A0 && r <= 0x25FF:
			if emojiPresentation(r) {
				return true
			}
		}
	}
	return false
}

// emojiPresentation is the Emoji_Presentation set below U+2C00: pictographs that are
// drawn in color without a variation selector.
func emojiPresentation(r rune) bool {
	switch {
	case r == 0x231A, r == 0x231B, r >= 0x23E9 && r <= 0x23EC, r == 0x23F0, r == 0x23F3, r == 0x25FD, r == 0x25FE,
		r == 0x2614, r == 0x2615, r >= 0x2648 && r <= 0x2653, r == 0x267F, r == 0x2693, r == 0x26A1, r == 0x26AA, r == 0x26AB,
		r == 0x26BD, r == 0x26BE, r == 0x26C4, r == 0x26C5, r == 0x26CE, r == 0x26D4, r == 0x26EA, r == 0x26F2, r == 0x26F3,
		r == 0x26F5, r == 0x26FA, r == 0x26FD, r == 0x2705, r == 0x270A, r == 0x270B, r == 0x2728, r == 0x274C, r == 0x274E,
		r >= 0x2753 && r <= 0x2755, r == 0x2757, r >= 0x2795 && r <= 0x2797, r == 0x27B0, r == 0x27BF, r == 0x2B1B, r == 0x2B1C,
		r == 0x2B50, r == 0x2B55:
		return true
	}
	return false
}
