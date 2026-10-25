package iconpng

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestDecodeKeepsThePixelsAndRejectsWhatIsNotSquare(t *testing.T) {
	encode := func(width, height int) []byte {
		picture := image.NewNRGBA(image.Rect(0, 0, width, height))
		picture.SetNRGBA(1, 0, color.NRGBA{R: 9, G: 8, B: 7, A: 200})
		var buffer bytes.Buffer
		if err := png.Encode(&buffer, picture); err != nil {
			t.Fatal(err)
		}
		return buffer.Bytes()
	}
	icon, err := Decode(encode(4, 4))
	if err != nil || icon.Size != 4 {
		t.Fatalf("a square PNG gave size %d, err %v", icon.Size, err)
	}
	if got := icon.RGBA[4:8]; got[0] != 9 || got[1] != 8 || got[2] != 7 || got[3] != 200 {
		t.Errorf("the pixel came back as %v", got)
	}
	if _, err := Decode(encode(4, 2)); err == nil {
		t.Error("a PNG that is not square was accepted")
	}
	if _, err := Decode([]byte("not a png")); err == nil {
		t.Error("garbage was accepted")
	}
}
