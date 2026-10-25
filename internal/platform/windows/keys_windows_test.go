//go:build windows

package windows

import (
	"errors"
	"testing"

	"github.com/vivaui/viva/internal/input"
	"github.com/vivaui/viva/internal/platform"
)

// Input injected by virtual key alone has no scan code in its message; the
// press and the release must still name the same physical key.
func TestKeyCodeFallsBackToTheLayoutWhenTheMessageHasNoScanCode(t *testing.T) {
	const vkA = 0x41
	if code := keyCode(vkA, 0); code == 0 {
		t.Fatal("a key injected without a scan code has no code")
	}
	if keyCode(vkA, 0) != keyCode(vkA, 0x1e<<16) {
		t.Fatalf("injected %#x differs from the keyboard's %#x", keyCode(vkA, 0), keyCode(vkA, 0x1e<<16))
	}
}

// A terminal has no text model of its own: the editing keys a text field would
// consume, and the control chords it would undo with, are the program's keys.
// They must reach the application instead of becoming text commands, which is
// what Backspace, Delete and the arrows did before.
func TestATerminalGetsEditingKeysAndChordsAsRawKeys(t *testing.T) {
	state := &windowState{textEnabled: true}
	state.textInput.Terminal = true
	var events []platform.KeyEvent
	state.config.OnKey = func(event platform.KeyEvent) (platform.Frame, error) {
		events = append(events, event)
		return platform.Frame{}, errors.New("stop before the renderer")
	}
	for _, test := range []struct {
		name string
		vk   uintptr
		mods input.Modifiers
		key  input.Key
		text string
	}{
		{"Backspace", 8, 0, input.KeyBackspace, ""},
		{"Delete", 46, 0, input.KeyDelete, ""},
		{"Left", 37, 0, input.KeyLeft, ""},
		{"Shift+Right", 39, input.ModifierShift, input.KeyRight, ""},
		{"Enter", 13, 0, input.KeyEnter, ""},
		{"Ctrl+R", 'R', input.ModifierControl, input.KeyUnknown, "r"},
	} {
		events = events[:0]
		if !state.textKey(test.vk, 0, test.mods) {
			t.Fatalf("%s was not consumed", test.name)
		}
		if len(events) != 1 || events[0].Key != test.key || events[0].Text != test.text || events[0].Modifiers != test.mods {
			t.Fatalf("%s = %+v", test.name, events)
		}
	}
	// A printable key is text, inserted when its WM_CHAR arrives, and AltGr
	// (Control+Alt) types a character too.
	for _, test := range []struct {
		name string
		vk   uintptr
		mods input.Modifiers
	}{
		{"a", 'A', 0},
		{"AltGr+a", 'A', input.ModifierControl | input.ModifierAlt},
	} {
		events = events[:0]
		if state.textKey(test.vk, 0, test.mods) {
			t.Fatalf("%s was consumed as a key", test.name)
		}
		if len(events) != 0 {
			t.Fatalf("%s sent %+v", test.name, events)
		}
	}
}

// A text field keeps taking the editing commands, and never sees them as keys.
func TestATextFieldStillGetsEditingCommands(t *testing.T) {
	state := &windowState{textEnabled: true}
	var commands []platform.TextCommand
	state.textInput.OnEvent = func(event platform.TextEvent) (platform.TextState, error) {
		commands = append(commands, event.Command)
		return platform.TextState{}, errors.New("stop before the renderer")
	}
	state.config.OnKey = func(platform.KeyEvent) (platform.Frame, error) {
		t.Fatal("a text field sent its editing key to the application")
		return platform.Frame{}, nil
	}
	if !state.textKey(8, 0, 0) {
		t.Fatal("Backspace in a text field was not consumed")
	}
	if len(commands) != 1 || commands[0] != platform.TextDeleteBackward {
		t.Fatalf("text field commands = %v", commands)
	}
}

// A win32-input-mode packet needs the console record's own fields, not a
// reconstruction from the normalized key.
func TestWin32KeyRecordCarriesTheConsoleFields(t *testing.T) {
	key := win32Key('A', 0x1e<<16|3, 'a', 3)
	if key.VirtualKey != 'A' || key.ScanCode != 0x1e || key.Unicode != 'a' || key.Repeat != 3 {
		t.Fatalf("win32 key = %+v", key)
	}
}

// Alt+printable arrives as WM_SYSCHAR, not WM_CHAR. A terminal that asked for
// releases must still see it as a raw key rather than losing it.
func TestAltPrintableIsSentAsARawKey(t *testing.T) {
	state := &windowState{}
	var events []platform.KeyEvent
	state.config.OnKey = func(event platform.KeyEvent) (platform.Frame, error) {
		events = append(events, event)
		return platform.Frame{}, nil
	}
	if !state.keyDown('A', 0x1e<<16, input.ModifierAlt) {
		t.Fatal("Alt+A was not consumed")
	}
	if len(events) != 1 || events[0].Text != "a" || events[0].Modifiers != input.ModifierAlt || events[0].Win32.VirtualKey != 'A' || events[0].Win32.ScanCode != 0x1e {
		t.Fatalf("Alt+A = %+v", events)
	}
}

// WM_CHAR completes a printable key's press with the typed code unit.
func TestCharCompletesTheConsoleRecord(t *testing.T) {
	state := &windowState{}
	var events []platform.KeyEvent
	state.config.OnKey = func(event platform.KeyEvent) (platform.Frame, error) {
		events = append(events, event)
		return platform.Frame{}, nil
	}
	state.pendingKey = pendingKey{set: true, vk: 'A', code: 0x1e}
	if !state.character('a', 0x1e<<16) {
		t.Fatal("the character did not complete the key")
	}
	if len(events) != 1 || events[0].Win32.Unicode != 'a' || events[0].Win32.VirtualKey != 'A' || events[0].Win32.ScanCode != 0x1e {
		t.Fatalf("character = %+v", events)
	}
}
