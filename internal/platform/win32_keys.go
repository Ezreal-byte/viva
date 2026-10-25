package platform

// IsAltGr reports whether the modifier keys held are AltGr. Windows delivers
// AltGr as a left Control and a right Alt together, which is why Control+Alt
// reads as a chord while it is only how a key of the layout's third level is
// typed. A right Control held as well is a real chord.
func IsAltGr(rightAlt, leftControl, rightControl bool) bool {
	return rightAlt && leftControl && !rightControl
}

// ShortcutPunctuationKey reports whether a Windows virtual key is one of the
// layout's punctuation keys (VK_OEM_*), whose character a shortcut such as
// Control+, or Control+= is named by.
func ShortcutPunctuationKey(vk uintptr) bool {
	return vk >= 0xba && vk <= 0xc0 || vk >= 0xdb && vk <= 0xdf || vk == 0xe2
}
