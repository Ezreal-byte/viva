package linux

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/vivaui/viva/internal/platform"
)

// TestClipboardImageFromX11Owner reads an image another program put on the real
// X11 clipboard. It needs a display and xclip, as the other XCB smoke test
// needs a display.
func TestClipboardImageFromX11Owner(t *testing.T) {
	if testing.Short() || os.Getenv("VIVA_LINUX_XCB_SMOKE") == "" {
		t.Skip("set VIVA_LINUX_XCB_SMOKE=1 with an X11 display and xclip")
	}
	xclip, err := exec.LookPath("xclip")
	if err != nil {
		t.Skip("xclip is not installed")
	}
	if ownsClipboardText() {
		t.Fatal("this process owns the clipboard")
	}
	directory := t.TempDir()
	// Noise does not compress: this PNG is about 120 KB, larger than one small
	// property and small enough for the owner to hand over in a single transfer
	// (xclip serves larger images in a way that stops answering other requesters
	// while a clipboard manager is also asking; TestClipboardImageStreamedWithINCR
	// covers large images with an owner that follows the protocol).
	picture := image.NewNRGBA(image.Rect(0, 0, 200, 200))
	state := uint32(3)
	for index := range picture.Pix {
		state = state*1664525 + 1013904223
		picture.Pix[index] = byte(state >> 24)
		if index%4 == 3 {
			picture.Pix[index] = 255
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(directory, "image.png")
	if err := os.WriteFile(imagePath, encoded.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	own := func(arguments ...string) {
		t.Helper()
		// xclip forks to serve the selection and keeps its inherited pipes, so
		// its output must go to files, not to a pipe this test would wait on.
		if err := exec.Command(xclip, append([]string{"-selection", "clipboard"}, arguments...)...).Run(); err != nil {
			t.Fatalf("xclip %v: %v", arguments, err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	own("-t", "image/png", "-i", imagePath)
	if has, err := clipboardOffersImage(); !has {
		t.Fatalf("clipboard offers no image with a PNG on it (error: %v)", err)
	}
	got, err := ClipboardImagePNG()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, encoded.Bytes()) {
		t.Fatalf("read %d bytes, want the %d bytes xclip owns", len(got), encoded.Len())
	}

	textPath := filepath.Join(directory, "text.txt")
	if err := os.WriteFile(textPath, []byte("plain text"), 0o600); err != nil {
		t.Fatal(err)
	}
	own("-i", textPath)
	if HasClipboardImage() {
		t.Fatal("HasClipboardImage = true with text on the clipboard")
	}
	if _, err := ClipboardImagePNG(); !errors.Is(err, platform.ErrNoClipboardImage) {
		t.Fatalf("error = %v, want ErrNoClipboardImage", err)
	}
}

// TestClipboardImageStreamedWithINCR reads a large image from a GTK owner. GTK
// serves an image that does not fit one property by streaming it with INCR (the
// ICCCM protocol every screenshot tool and toolkit uses for large selections),
// which is the path the xclip test above does not reach: xclip's own large-image
// serving does not answer other requesters while a clipboard manager is also asking.
func TestClipboardImageStreamedWithINCR(t *testing.T) {
	if testing.Short() || os.Getenv("VIVA_LINUX_XCB_SMOKE") == "" {
		t.Skip("set VIVA_LINUX_XCB_SMOKE=1 with an X11 display and python3 with PyGObject (GTK 3)")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}
	if err := exec.Command(python, "-c", "import gi; gi.require_version('Gtk', '3.0')").Run(); err != nil {
		t.Skip("PyGObject with GTK 3 is not installed")
	}
	if ownsClipboardText() {
		t.Fatal("this process owns the clipboard")
	}
	directory := t.TempDir()
	picture := image.NewNRGBA(image.Rect(0, 0, 600, 600))
	state := uint32(5)
	for index := range picture.Pix {
		state = state*1664525 + 1013904223
		picture.Pix[index] = byte(state >> 24)
		if index%4 == 3 {
			picture.Pix[index] = 255
		}
	}
	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&encoded, picture); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(directory, "image.png")
	if err := os.WriteFile(imagePath, encoded.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	script := `import gi, sys
gi.require_version('Gtk', '3.0')
from gi.repository import Gtk, Gdk, GdkPixbuf, GLib
Gtk.Clipboard.get(Gdk.SELECTION_CLIPBOARD).set_image(GdkPixbuf.Pixbuf.new_from_file(sys.argv[1]))
print("serving", flush=True)
GLib.timeout_add_seconds(20, Gtk.main_quit)
Gtk.main()`
	owner := exec.Command(python, "-c", script, imagePath)
	stdout, err := owner.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = owner.Process.Kill(); _ = owner.Wait() })
	ready := make([]byte, 7)
	if _, err := stdout.Read(ready); err != nil {
		t.Fatalf("the GTK owner did not start: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	if !HasClipboardImage() {
		t.Fatal("the clipboard offers no image while a GTK program owns a PNG")
	}
	got, err := ClipboardImagePNG()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("the streamed bytes are not a PNG: %v", err)
	}
	if bounds := decoded.Bounds(); bounds.Dx() != 600 || bounds.Dy() != 600 {
		t.Fatalf("got a %v image, want 600x600", bounds)
	}
	// Streaming reassembles the owner's bytes exactly: a pixel from each end
	// has to be the original's (GTK re-encodes the PNG, so the bytes differ, the pixels do not).
	for _, point := range []image.Point{{0, 0}, {599, 599}, {300, 17}} {
		if got, want := color.NRGBAModel.Convert(decoded.At(point.X, point.Y)), picture.At(point.X, point.Y); got != want {
			t.Fatalf("pixel %v is %v, want %v", point, got, want)
		}
	}
}
