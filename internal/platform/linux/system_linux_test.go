package linux

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/vivaui/viva/internal/platform"
)

func TestUnavailableSystemIntegrationsAreDetectable(t *testing.T) {
	activeWindow.Store(nil)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/bus")
	if err := SetClipboardText("text"); err == nil || !strings.Contains(err.Error(), "running window") {
		t.Fatalf("SetClipboardText error = %v", err)
	}
	if _, _, err := OpenFiles(OpenFileOptions{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("OpenFiles error = %v", err)
	}
	if _, _, err := SaveFile(SaveFileOptions{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SaveFile error = %v", err)
	}
	if err := OpenURL("https://example.com"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("OpenURL error = %v", err)
	}
	if err := PlaySystemSound(0); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("PlaySystemSound error = %v", err)
	}
	if err := SendNotification("title", "body"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SendNotification error = %v", err)
	}
}

func TestDesktopCommandsPreserveArguments(t *testing.T) {
	directory := t.TempDir()
	capture := filepath.Join(directory, "arguments")
	t.Setenv("PATH", directory)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/bus")
	t.Setenv("VIVA_CAPTURE_FILE", capture)
	writeCommandCapture(t, directory, "xdg-open")
	writeCommandCapture(t, directory, "notify-send")

	target := "https://example.com/a b?option=--help"
	if err := OpenURL(target); err != nil {
		t.Fatal(err)
	}
	if arguments := readCommandCapture(t, capture); !reflect.DeepEqual(arguments, []string{target}) {
		t.Fatalf("xdg-open arguments = %#v", arguments)
	}

	if err := SendNotification("Build --complete", "Body with spaces"); err != nil {
		t.Fatal(err)
	}
	want := []string{"--app-name=" + notificationAppName(), "--", "Build --complete", "Body with spaces"}
	if arguments := readCommandCapture(t, capture); !reflect.DeepEqual(arguments, want) {
		t.Fatalf("notify-send arguments = %#v", arguments)
	}
}

func TestFileDialogArguments(t *testing.T) {
	filters := []FileFilter{{Name: "Text", Pattern: "*.txt; *.md"}}
	zenity, separator, err := openDialogArguments("zenity", OpenFileOptions{
		Title: "Open", Filters: filters, AllowMultiple: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantZenity := []string{"--file-selection", "--title=Open", "--multiple", "--separator=" + dialogSeparator, "--file-filter=Text | *.txt *.md"}
	if !reflect.DeepEqual(zenity, wantZenity) || separator != dialogSeparator {
		t.Fatalf("zenity = %#v, separator = %q", zenity, separator)
	}
	kdialog, separator, err := openDialogArguments("kdialog", OpenFileOptions{
		Title: "Open", Filters: filters, AllowMultiple: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantKDialog := []string{"--getopenfilename", "", "*.txt *.md|Text", "--multiple", "--separate-output", "--title", "Open"}
	if !reflect.DeepEqual(kdialog, wantKDialog) || separator != "\n" {
		t.Fatalf("kdialog = %#v, separator = %q", kdialog, separator)
	}
	save, err := saveDialogArguments("zenity", SaveFileOptions{
		Title: "Save", SuggestedName: "notes", DefaultExtension: ".txt", Filters: filters,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(save, []string{"--file-selection", "--save", "--confirm-overwrite", "--title=Save", "--filename=notes.txt", "--file-filter=Text | *.txt *.md"}) {
		t.Fatalf("save arguments = %#v", save)
	}
	zenityDirectory, err := openDirectoryDialogArguments("zenity", OpenDirectoryOptions{Title: "Choose folder"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(zenityDirectory, []string{"--file-selection", "--directory", "--title=Choose folder"}) {
		t.Fatalf("zenity directory arguments = %#v", zenityDirectory)
	}
	kdialogDirectory, err := openDirectoryDialogArguments("kdialog", OpenDirectoryOptions{Title: "Choose folder"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(kdialogDirectory, []string{"--getexistingdirectory", "", "--title", "Choose folder"}) {
		t.Fatalf("kdialog directory arguments = %#v", kdialogDirectory)
	}
}

func TestFileDialogCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dialog")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	output, canceled, err := runDialog("test dialog", path, nil)
	if err != nil || !canceled || output != nil {
		t.Fatalf("runDialog = %q, %t, %v", output, canceled, err)
	}
}

func TestSetClipboardTextClaimsSelection(t *testing.T) {
	reply := &xcbSelectionOwnerReply{Owner: 22}
	var owner, selection uint32
	api := &xcbAPI{
		setSelectionOwner: func(_ uintptr, nextOwner, nextSelection, _ uint32) xcbVoidCookie {
			owner, selection = nextOwner, nextSelection
			return xcbVoidCookie{}
		},
		getSelectionOwner: func(uintptr, uint32) xcbSelectionOwnerCookie {
			return xcbSelectionOwnerCookie{}
		},
		selectionOwnerReply: func(uintptr, xcbSelectionOwnerCookie, *unsafe.Pointer) *xcbSelectionOwnerReply {
			return reply
		},
		flush:              func(uintptr) int32 { return 1 },
		connectionHasError: func(uintptr) int32 { return 0 },
		free:               func(unsafe.Pointer) {},
	}
	win := &window{api: api, connection: 11, id: 22, clipboardAtom: 33, utf8Atom: 34, textAtom: 35, targetsAtom: 36}
	activeWindow.Store(win)
	t.Cleanup(func() { activeWindow.Store(nil) })
	if err := SetClipboardText("你好"); err != nil {
		t.Fatal(err)
	}
	if owner != 22 || selection != 33 || !win.ownsClipboard || win.clipboard != "你好" {
		t.Fatalf("clipboard state = owner %d selection %d owned %t value %q", owner, selection, win.ownsClipboard, win.clipboard)
	}
	value, err := ClipboardText()
	if err != nil || value != "你好" {
		t.Fatalf("ClipboardText = %q, %v", value, err)
	}
}

func TestSelectionRequestReturnsUTF8(t *testing.T) {
	var propertyType uint32
	var propertyValue string
	var sent xcbSelectionNotifyWire
	api := &xcbAPI{
		changeProperty: func(_ uintptr, _ uint8, _, _, nextType uint32, format uint8, length uint32, data unsafe.Pointer) xcbVoidCookie {
			if format != 8 {
				t.Fatalf("property format = %d", format)
			}
			propertyType = nextType
			propertyValue = string(unsafe.Slice((*byte)(data), int(length)))
			return xcbVoidCookie{}
		},
		sendEvent: func(_ uintptr, _ uint8, _ uint32, _ uint32, data *byte) xcbVoidCookie {
			copy(unsafe.Slice((*byte)(unsafe.Pointer(&sent)), 32), unsafe.Slice(data, 32))
			return xcbVoidCookie{}
		},
		flush:              func(uintptr) int32 { return 1 },
		connectionHasError: func(uintptr) int32 { return 0 },
	}
	win := &window{
		api: api, connection: 1, id: 2, clipboard: "你好", ownsClipboard: true,
		clipboardAtom: 10, utf8Atom: 11, textAtom: 12, targetsAtom: 13,
	}
	request := xcbSelectionRequestEvent{Time: 20, Owner: 2, Requestor: 3, Selection: 10, Target: 11, Property: 14}
	if err := win.handleSelectionRequest(&request); err != nil {
		t.Fatal(err)
	}
	if propertyType != 11 || propertyValue != "你好" {
		t.Fatalf("property = type %d, value %q", propertyType, propertyValue)
	}
	if sent.ResponseType != xcbSelectionNotify || sent.Time != 20 || sent.Requestor != 3 || sent.Selection != 10 || sent.Target != 11 || sent.Property != 14 {
		t.Fatalf("selection notify = %+v", sent)
	}
}

func TestSelectionClearDropsOwnedText(t *testing.T) {
	win := &window{clipboard: "value", clipboardAtom: 10, ownsClipboard: true}
	win.handleSelectionClear(&xcbSelectionClearEvent{Selection: 10})
	if win.ownsClipboard || win.clipboard != "" {
		t.Fatalf("clipboard remained owned: %t %q", win.ownsClipboard, win.clipboard)
	}
}

func TestClipboardEncodingHelpers(t *testing.T) {
	if got, want := string(latin1Clipboard("Aé中")), "A\xe9?"; got != want {
		t.Fatalf("latin1 clipboard = %q, want %q", got, want)
	}
	if got, want := decodeLatin1("A\xe9"), "Aé"; got != want {
		t.Fatalf("decoded latin1 = %q, want %q", got, want)
	}
	activeWindow.Store(nil)
	if err := SetClipboardText("bad\xff"); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
}

func TestSetCursorShapeAppliesAndReleasesCursor(t *testing.T) {
	var nextID uint32 = 40
	var fontName string
	var sourceGlyph, maskGlyph uint16
	var cursorValue uint32
	var freed []uint32
	requests := 0
	api := &xcbAPI{
		generateID: func(uintptr) uint32 {
			nextID++
			return nextID
		},
		openFont: func(_ uintptr, _ uint32, length uint16, name *byte) xcbVoidCookie {
			fontName = string(unsafe.Slice(name, int(length)))
			return xcbVoidCookie{Sequence: 1}
		},
		closeFont: func(uintptr, uint32) xcbVoidCookie { return xcbVoidCookie{} },
		createGlyphCursor: func(_ uintptr, _ uint32, _, _ uint32, source, mask, _, _, _, _, _, _ uint16) xcbVoidCookie {
			sourceGlyph, maskGlyph = source, mask
			return xcbVoidCookie{Sequence: 2}
		},
		changeWindow: func(_ uintptr, _ uint32, mask uint32, values *uint32) xcbVoidCookie {
			if mask != xcbCWCursor {
				t.Fatalf("window mask = %#x", mask)
			}
			cursorValue = *values
			return xcbVoidCookie{Sequence: 3}
		},
		freeCursor: func(_ uintptr, cursor uint32) xcbVoidCookie {
			freed = append(freed, cursor)
			return xcbVoidCookie{}
		},
		requestCheck: func(_ uintptr, cookie xcbVoidCookie) *xcbGenericError {
			if cookie.Sequence == 0 {
				t.Fatal("checked an empty request")
			}
			requests++
			return nil
		},
		flush:              func(uintptr) int32 { return 1 },
		connectionHasError: func(uintptr) int32 { return 0 },
	}
	win := &window{api: api, connection: 1, id: 2, cursor: 9}
	activeWindow.Store(win)
	t.Cleanup(func() { activeWindow.Store(nil) })

	if err := SetCursorShape(2); err != nil {
		t.Fatal(err)
	}
	if fontName != "cursor" || sourceGlyph != 60 || maskGlyph != 61 {
		t.Fatalf("cursor font = %q, glyphs = %d/%d", fontName, sourceGlyph, maskGlyph)
	}
	if requests != 3 || cursorValue != 42 || win.cursor != 42 {
		t.Fatalf("requests = %d, applied = %d, retained = %d", requests, cursorValue, win.cursor)
	}
	if !reflect.DeepEqual(freed, []uint32{9}) {
		t.Fatalf("freed cursors = %v", freed)
	}
}

func TestSetCursorShapeRequiresWindow(t *testing.T) {
	activeWindow.Store(nil)
	if err := SetCursorShape(0); err == nil || !strings.Contains(err.Error(), "running window") {
		t.Fatalf("SetCursorShape error = %v", err)
	}
	if err := SetCursorShape(255); err == nil || !strings.Contains(err.Error(), "invalid cursor") {
		t.Fatalf("invalid cursor error = %v", err)
	}
}

func TestXCBSystemIntegrationSmoke(t *testing.T) {
	if testing.Short() || os.Getenv("VIVA_LINUX_XCB_SMOKE") == "" {
		t.Skip("set VIVA_LINUX_XCB_SMOKE=1 with an X11 display")
	}
	t.Setenv("PATH", t.TempDir())
	win, err := openWindow(platform.Config{
		Title:      "Viva Linux system integration smoke",
		Width:      160,
		Height:     100,
		WindowMode: platform.WindowTitled,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer win.close()
	if !activeWindow.CompareAndSwap(nil, win) {
		t.Fatal("an active window already exists")
	}
	defer activeWindow.Store(nil)
	for shape := uint8(0); shape < 10; shape++ {
		if err := SetCursorShape(shape); err != nil {
			t.Fatalf("cursor %d: %v", shape, err)
		}
	}
	if err := PlaySystemSound(0); err != nil {
		t.Fatal(err)
	}
	if err := SetClipboardText("Viva 你好"); err != nil {
		t.Fatal(err)
	}
	activeWindow.Store(nil)
	// A clipboard manager on the display asks for the new clipboard first, so the window
	// answers every request until the reader has its text.
	stop := pumpSelectionEvents(win)
	defer stop()
	value, err := ClipboardText()
	if err != nil {
		t.Fatal(err)
	}
	if value != "Viva 你好" {
		t.Fatalf("ClipboardText = %q", value)
	}
}

func writeCommandCapture(t *testing.T, directory, name string) {
	t.Helper()
	path := filepath.Join(directory, name)
	contents := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$VIVA_CAPTURE_FILE\"\n"
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readCommandCapture(t *testing.T, path string) []string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(contents), "\n"), "\n")
}

func TestDarkAppearanceAnswersOnThisMachine(t *testing.T) {
	dark, known := DarkAppearance()
	if dark && !known {
		t.Fatal("a desktop that does not say cannot be dark")
	}
	t.Logf("portal: dark=%v known=%v", dark, known)
}
