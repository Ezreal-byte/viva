package windows

import (
	"bytes"
	"errors"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"testing"

	"github.com/vivaui/viva/internal/platform"
)

// TestClipboardImageFromWindowsClipboard reads an image another program (the
// .NET clipboard) put on the real clipboard. It replaces the clipboard, so it
// runs only on request, on a machine whose clipboard may be overwritten.
func TestClipboardImageFromWindowsClipboard(t *testing.T) {
	if os.Getenv("VIVA_WINDOWS_CLIPBOARD") == "" {
		t.Skip("set VIVA_WINDOWS_CLIPBOARD=1 to overwrite the real clipboard")
	}
	script := `Add-Type -AssemblyName System.Windows.Forms,System.Drawing
$b = New-Object System.Drawing.Bitmap 4,3
for ($x = 0; $x -lt 4; $x++) { for ($y = 0; $y -lt 3; $y++) { $b.SetPixel($x, $y, [System.Drawing.Color]::FromArgb(255, 0, 128, 0)) } }
$b.SetPixel(0, 0, [System.Drawing.Color]::FromArgb(255, 255, 0, 0))
$b.SetPixel(3, 2, [System.Drawing.Color]::FromArgb(255, 0, 0, 255))
[System.Windows.Forms.Clipboard]::SetImage($b)`
	if output, err := exec.Command("powershell", "-NoProfile", "-STA", "-Command", script).CombinedOutput(); err != nil {
		t.Fatalf("set clipboard image: %v: %s", err, output)
	}
	if !HasClipboardImage() {
		t.Fatal("HasClipboardImage = false with an image on the clipboard")
	}
	data, err := ClipboardImagePNG()
	if err != nil {
		t.Fatal(err)
	}
	picture, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if picture.Bounds().Dx() != 4 || picture.Bounds().Dy() != 3 {
		t.Fatalf("size = %v, want 4x3", picture.Bounds())
	}
	for _, check := range []struct {
		x, y int
		want color.NRGBA
	}{{0, 0, color.NRGBA{255, 0, 0, 255}}, {3, 2, color.NRGBA{0, 0, 255, 255}}, {1, 1, color.NRGBA{0, 128, 0, 255}}} {
		if got := color.NRGBAModel.Convert(picture.At(check.x, check.y)); got != check.want {
			t.Fatalf("pixel (%d,%d) = %v, want %v (rows upside down or channels swapped?)", check.x, check.y, got, check.want)
		}
	}

	// A program that offers the registered "PNG" format, as browsers do, hands
	// over the encoded file and Viva returns it untouched.
	script = `Add-Type -AssemblyName System.Windows.Forms,System.Drawing
$b = New-Object System.Drawing.Bitmap 2,2
$b.SetPixel(0, 0, [System.Drawing.Color]::FromArgb(128, 10, 20, 30))
$s = New-Object System.IO.MemoryStream
$b.Save($s, [System.Drawing.Imaging.ImageFormat]::Png)
$d = New-Object System.Windows.Forms.DataObject
$d.SetData("PNG", $s)
[System.Windows.Forms.Clipboard]::SetDataObject($d, $true)`
	if output, err := exec.Command("powershell", "-NoProfile", "-STA", "-Command", script).CombinedOutput(); err != nil {
		t.Fatalf("set PNG on the clipboard: %v: %s", err, output)
	}
	data, err = ClipboardImagePNG()
	if err != nil {
		t.Fatal(err)
	}
	if picture, err = png.Decode(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if got := color.NRGBAModel.Convert(picture.At(0, 0)).(color.NRGBA); got.A != 128 {
		t.Fatalf("alpha of the PNG pixel = %d, want 128", got.A)
	}

	if err := SetClipboardText("not an image"); err != nil {
		t.Fatal(err)
	}
	if HasClipboardImage() {
		t.Fatal("HasClipboardImage = true with text on the clipboard")
	}
	if _, err := ClipboardImagePNG(); !errors.Is(err, platform.ErrNoClipboardImage) {
		t.Fatalf("error = %v, want ErrNoClipboardImage", err)
	}
}
