package linux

// Dead keys and the Compose key.
//
// The X server hands a client the keysyms of a key, and a dead key's keysym
// (dead_acute) is not a character: it waits for the next key and puts the accent on
// it. The client has to do that, as GTK and Qt do, since the core keyboard
// protocol has no composition of its own. An input method that handles the key
// (IBus with a Chinese engine) never lets it get here; a key it passes back does,
// and is composed in the order it was typed.
//
// The tables are the Latin letters of the European languages, the part of the
// X.Org Compose files that a keyboard with dead keys needs. A pair they do not
// know types the accent and then the letter, as the X library does.

const (
	keysymDeadFirst = 0xfe50
	keysymDeadLast  = 0xfe8f
)

func isDeadKeysym(keysym uint32) bool { return keysym >= keysymDeadFirst && keysym <= keysymDeadLast }

// deadAccent is one dead key: its keysym, the character that stands for the
// accent on its own (what two presses of the key type), the ASCII character the
// key stands for on the keyboard where it is a letter's neighbor (what the key
// and a space type; zero when the accent is the same), and the letters it can
// sit on with the results.
type deadAccent struct {
	keysym  uint32
	spacing rune
	bare    rune
	base    string
	result  string
}

// withSpace is what the dead key and a space type.
func (accent deadAccent) withSpace() rune {
	if accent.bare != 0 {
		return accent.bare
	}
	return accent.spacing
}

var deadAccents = [...]deadAccent{
	{0xfe50, '`', 0, "AEIOUaeiou", "ÀÈÌÒÙàèìòù"},
	{0xfe51, '´', '\'', "AEIOUYaeiouyCcLlNnRrSsZz", "ÁÉÍÓÚÝáéíóúýĆćĹĺŃńŔŕŚśŹź"},
	{0xfe52, '^', 0, "AEIOUaeiouCcGgHhJjSsWwYy", "ÂÊÎÔÛâêîôûĈĉĜĝĤĥĴĵŜŝŴŵŶŷ"},
	{0xfe53, '~', 0, "ANOanoIiUuEeYy", "ÃÑÕãñõĨĩŨũẼẽỸỹ"},
	{0xfe54, '¯', 0, "AEIOUaeiouYy", "ĀĒĪŌŪāēīōūȲȳ"},
	{0xfe55, '˘', 0, "AEGIOUaegiou", "ĂĔĞĬŎŬăĕğĭŏŭ"},
	{0xfe56, '˙', 0, "CEGIZcegz", "ĊĖĠİŻċėġż"},
	{0xfe57, '¨', '"', "AEIOUaeiouyY", "ÄËÏÖÜäëïöüÿŸ"},
	{0xfe58, '˚', 0, "AaUu", "ÅåŮů"},
	{0xfe59, '˝', 0, "OoUu", "ŐőŰű"},
	{0xfe5a, 'ˇ', 0, "ACDEILNORSTUZacdeilnorstuz", "ǍČĎĚǏĽŇǑŘŠŤǓŽǎčďěǐľňǒřšťǔž"},
	{0xfe5b, '¸', 0, "CcGgKkLlNnRrSsTt", "ÇçĢģĶķĻļŅņŖŗŞşŢţ"},
}

func deadAccentFor(keysym uint32) (deadAccent, bool) {
	for _, accent := range deadAccents {
		if accent.keysym == keysym {
			return accent, true
		}
	}
	return deadAccent{}, false
}

// deadAccentForSpacing is the accent a typed character stands for after the
// Compose key (Compose, apostrophe, e).
func deadAccentForSpacing(character rune) (deadAccent, bool) {
	for _, accent := range deadAccents {
		if accent.spacing == character || accent.withSpace() == character {
			return accent, true
		}
	}
	return deadAccent{}, false
}

// accented is the letter with the accent on it.
func (accent deadAccent) accented(letter rune) (rune, bool) {
	index := 0
	for _, candidate := range accent.base {
		if candidate == letter {
			result := []rune(accent.result)
			return result[index], true
		}
		index++
	}
	return 0, false
}

// composeSequences are the two-key sequences that follow the Compose key and are
// not an accent and a letter, in the order X.Org's Compose file has them (the
// ones it lists both ways are listed both ways here).
var composeSequences = [...]struct {
	keys   string
	result rune
}{
	{"oc", '©'}, {"OC", '©'}, {"or", '®'}, {"OR", '®'}, {"tm", '™'}, {"TM", '™'},
	{"<<", '«'}, {">>", '»'}, {"ss", 'ß'}, {"ae", 'æ'}, {"AE", 'Æ'}, {"oe", 'œ'}, {"OE", 'Œ'},
	{"/o", 'ø'}, {"o/", 'ø'}, {"/O", 'Ø'}, {"O/", 'Ø'}, {"oa", 'å'}, {"oA", 'Å'},
	{"!!", '¡'}, {"??", '¿'}, {"12", '½'}, {"14", '¼'}, {"34", '¾'}, {"+-", '±'}, {"xx", '×'}, {"-:", '÷'},
	{"c/", '¢'}, {"/c", '¢'}, {"L-", '£'}, {"-L", '£'}, {"Y=", '¥'}, {"=Y", '¥'},
	{"e=", '€'}, {"=e", '€'}, {"E=", '€'}, {"=E", '€'}, {"C=", '€'}, {"=C", '€'},
	{"SO", '§'}, {"PP", '¶'}, {"mu", 'µ'}, {"oo", '°'}, {"-^", '¯'}, {"..", '…'},
}

func composeSequence(first, second rune) (rune, bool) {
	typed := string([]rune{first, second})
	for _, sequence := range composeSequences {
		if sequence.keys == typed {
			return sequence.result, true
		}
	}
	return 0, false
}

// composer follows the keys of one window through a dead key or a Compose
// sequence. Its zero value is idle.
type composer struct {
	state composeState
	// dead is the pending accent; first is the first key after Compose.
	dead  deadAccent
	known bool
	first rune
}

type composeState uint8

const (
	composeIdle composeState = iota
	composeDead
	composeAwaitFirst
	composeAwaitSecond
)

func (compose *composer) reset() {
	if compose != nil {
		*compose = composer{}
	}
}

// pending reports whether a sequence is half typed.
func (compose *composer) pending() bool { return compose != nil && compose.state != composeIdle }

// feed offers the keysym of a key that was pressed with no Control, Alt or Meta.
// handled says the key is part of a sequence: text is what it types (empty while
// the sequence goes on). A key that is not handled types as it always does.
func (compose *composer) feed(keysym uint32) (text string, handled bool) {
	key := composeClass(keysym)
	switch compose.state {
	case composeIdle:
		switch {
		case isDeadKeysym(keysym):
			accent, known := deadAccentFor(keysym)
			*compose = composer{state: composeDead, dead: accent, known: known}
			return "", true
		case keysym == keysymMultiKey:
			*compose = composer{state: composeAwaitFirst}
			return "", true
		}
		return "", false
	case composeDead:
		return compose.afterDead(keysym, key)
	case composeAwaitFirst:
		if keyCancels(key) {
			compose.reset()
			return "", key == composeCancel
		}
		if accent, ok := deadAccentFor(keysym); ok {
			*compose = composer{state: composeAwaitSecond, dead: accent, known: true}
			return "", true
		}
		character, ok := keysymRune(keysym)
		if !ok {
			compose.reset()
			return "", false
		}
		if accent, ok := deadAccentForSpacing(character); ok {
			*compose = composer{state: composeAwaitSecond, dead: accent, known: true, first: character}
			return "", true
		}
		*compose = composer{state: composeAwaitSecond, first: character}
		return "", true
	case composeAwaitSecond:
		defer compose.reset()
		if keyCancels(key) {
			return "", key == composeCancel
		}
		character, ok := keysymRune(keysym)
		if !ok {
			return "", true
		}
		if compose.known {
			if composed, ok := compose.dead.accented(character); ok {
				return string(composed), true
			}
		}
		if compose.first != 0 {
			if composed, ok := composeSequence(compose.first, character); ok {
				return string(composed), true
			}
		}
		return "", true // an unknown sequence types nothing
	}
	return "", false
}

// afterDead handles the key after a dead key.
func (compose *composer) afterDead(keysym uint32, key composeKey) (string, bool) {
	accent, known := compose.dead, compose.known
	if keyCancels(key) {
		compose.reset()
		return "", key == composeCancel
	}
	spacing := ""
	if known {
		spacing = string(accent.spacing)
	}
	if isDeadKeysym(keysym) {
		// The same dead key twice types the accent; a different one replaces it.
		next, nextKnown := deadAccentFor(keysym)
		if nextKnown == known && next.keysym == accent.keysym {
			compose.reset()
			return spacing, true
		}
		*compose = composer{state: composeDead, dead: next, known: nextKnown}
		return spacing, true
	}
	compose.reset()
	character, ok := keysymRune(keysym)
	if !ok {
		return "", false
	}
	if known {
		if character == ' ' {
			return string(accent.withSpace()), true
		}
		if composed, ok := accent.accented(character); ok {
			return string(composed), true
		}
	}
	return spacing + string(character), true
}

// composeKey says what a pressed key is for composing: text, or one that ends
// a sequence.
type composeKey uint8

const (
	composeText composeKey = iota
	// composeEnds is a key that is not text and goes on to do what it does,
	// ending the sequence (an arrow key, Enter).
	composeEnds
	// composeCancel ends the sequence and is consumed doing it (Escape,
	// Backspace).
	composeCancel
)

func keyCancels(key composeKey) bool { return key != composeText }

// composeClass classifies a keysym for feed.
func composeClass(keysym uint32) composeKey {
	switch keysym {
	case keysymEscape, keysymBackspace:
		return composeCancel
	}
	if _, ok := keysymRune(keysym); ok || isDeadKeysym(keysym) || keysym == keysymMultiKey {
		return composeText
	}
	return composeEnds
}
