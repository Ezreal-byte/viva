package text

import (
	"fmt"
	"math/rand"
	"testing"
	"unicode/utf8"
)

// fragments are what the operations below insert: ASCII, CJK, an emoji with a
// variation selector, a combining mark, a regional-indicator pair, a newline and
// bytes that are not valid UTF-8.
var fragments = []string{"a", "hello", "你好", "中文输入", "👍🏽", "é", "🇨🇳", "\n", "\xff\xfe", "", "12", "x9"}

// applyEditorOperation performs one operation chosen by code and argument.
func applyEditorOperation(editor *Editor, code, argument byte) {
	fragment := fragments[int(argument)%len(fragments)]
	span := int(argument >> 4)
	switch code % 12 {
	case 0:
		editor.Insert(fragment)
	case 1:
		editor.SetMarkedText(fragment, Range{Start: span % 4, End: span % 7})
	case 2:
		editor.Unmark()
	case 3:
		editor.DeleteBackward()
	case 4:
		editor.DeleteForward()
	case 5:
		editor.MoveLeft(argument&1 == 1)
	case 6:
		editor.MoveRight(argument&1 == 1)
	case 7:
		editor.SetSelection(Range{Start: span - 3, End: int(argument&0x0f) + 2})
	case 8:
		editor.SetOverwrite(argument&1 == 1)
	case 9:
		// A composition replaced by another, as an IME does while the user types.
		editor.SetMarkedText(fragment, Range{})
		editor.SetMarkedText(fragments[int(argument+1)%len(fragments)], Range{Start: 1, End: 1})
	case 10:
		// A native event reports positions in UTF-16 units.
		editor.SetSelection(Range{Start: editor.UTF16ToRuneIndex(span), End: editor.UTF16ToRuneIndex(int(argument & 0x0f))})
	case 11:
		editor.SetMarkedText("", Range{})
	}
}

// checkEditor reports a violated invariant of the editor's state.
func checkEditor(editor *Editor) error {
	count := editor.RuneCount()
	selection := editor.Selection()
	if selection.Start < 0 || selection.End < selection.Start || selection.End > count {
		return fmt.Errorf("selection %+v outside 0..%d", selection, count)
	}
	if composition, composing := editor.Composition(); composing {
		if composition.Start < 0 || composition.End < composition.Start || composition.End > count {
			return fmt.Errorf("composition %+v outside 0..%d", composition, count)
		}
	}
	value := editor.String()
	if !utf8.ValidString(value) {
		return fmt.Errorf("text %q is not valid UTF-8", value)
	}
	if utf8.RuneCountInString(value) != count {
		return fmt.Errorf("RuneCount %d disagrees with %q", count, value)
	}
	for index := 0; index <= count; index++ {
		if back := editor.UTF16ToRuneIndex(editor.RuneIndexToUTF16(index)); back != index {
			return fmt.Errorf("rune index %d came back as %d through UTF-16", index, back)
		}
	}
	return nil
}

func runEditorOperations(t testing.TB, restricted bool, operations []byte) {
	editor := NewEditor("")
	const limit = 6
	if restricted {
		editor.Restrict(func(r rune) bool { return r >= '0' && r <= '9' }, limit)
	}
	for index := 0; index+1 < len(operations); index += 2 {
		applyEditorOperation(editor, operations[index], operations[index+1])
		if err := checkEditor(editor); err != nil {
			t.Fatalf("after operation %d (%d, %d): %v", index/2, operations[index], operations[index+1], err)
		}
		if restricted {
			if editor.RuneCount() > limit {
				t.Fatalf("a restricted editor holds %d runes: %q", editor.RuneCount(), editor.String())
			}
			for _, r := range editor.String() {
				if r < '0' || r > '9' {
					t.Fatalf("a digit-only editor holds %q", editor.String())
				}
			}
		}
	}
}

func FuzzEditorOperations(f *testing.F) {
	f.Add(false, []byte{0, 2, 1, 2, 0, 5, 3, 0})
	f.Add(true, []byte{0, 10, 9, 3, 0, 10, 1, 11, 8, 1, 0, 10})
	f.Add(false, []byte{1, 2, 1, 6, 11, 0, 0, 3, 7, 0x35, 10, 0x24})
	f.Fuzz(func(t *testing.T, restricted bool, operations []byte) {
		runEditorOperations(t, restricted, operations)
	})
}

// The fuzzer explores; this keeps a fixed amount of that exploration in every
// ordinary test run.
func TestEditorRandomOperationsKeepTheirInvariants(t *testing.T) {
	random := rand.New(rand.NewSource(20261017))
	for sequence := 0; sequence < 3000; sequence++ {
		operations := make([]byte, 2*(5+random.Intn(60)))
		random.Read(operations)
		runEditorOperations(t, sequence%3 == 0, operations)
	}
}
