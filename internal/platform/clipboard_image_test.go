//go:build windows

package platform

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// dibBuilder assembles a packed DIB the way a Windows clipboard owner does.
type dibBuilder struct {
	headerSize  int
	width       int
	height      int // negative: top-down
	bits        int
	compression uint32
	colors      int
	masks       [4]uint32
	palette     [][3]uint8 // r, g, b
	pixels      []byte
}

func (b dibBuilder) bytes() []byte {
	header := make([]byte, b.headerSize)
	binary.LittleEndian.PutUint32(header[0:], uint32(b.headerSize))
	binary.LittleEndian.PutUint32(header[4:], uint32(int32(b.width)))
	binary.LittleEndian.PutUint32(header[8:], uint32(int32(b.height)))
	binary.LittleEndian.PutUint16(header[12:], 1)
	binary.LittleEndian.PutUint16(header[14:], uint16(b.bits))
	binary.LittleEndian.PutUint32(header[16:], b.compression)
	binary.LittleEndian.PutUint32(header[32:], uint32(b.colors))
	out := header
	if b.headerSize >= 52 {
		for index, mask := range b.masks {
			if 40+index*4+4 <= b.headerSize {
				binary.LittleEndian.PutUint32(out[40+index*4:], mask)
			}
		}
	} else if b.compression == dibBitfields || b.compression == dibAlphaBitfields {
		count := 3
		if b.compression == dibAlphaBitfields {
			count = 4
		}
		for index := range count {
			out = binary.LittleEndian.AppendUint32(out, b.masks[index])
		}
	}
	for _, entry := range b.palette {
		out = append(out, entry[2], entry[1], entry[0], 0)
	}
	return append(out, b.pixels...)
}

func decodePNG(t *testing.T, data []byte) *image.NRGBA {
	t.Helper()
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("result is not a PNG: %v", err)
	}
	picture := image.NewNRGBA(decoded.Bounds())
	for y := range picture.Bounds().Dy() {
		for x := range picture.Bounds().Dx() {
			picture.Set(x, y, decoded.At(x, y))
		}
	}
	return picture
}

func wantPixel(t *testing.T, picture *image.NRGBA, x, y int, want color.NRGBA) {
	t.Helper()
	if got := picture.NRGBAAt(x, y); got != want {
		t.Fatalf("pixel (%d,%d) = %v, want %v", x, y, got, want)
	}
}

func TestDIBBottomUpAndTopDown(t *testing.T) {
	// Two rows of 24-bit pixels, stored padded to 4 bytes: width 1 -> stride 4.
	red, blue := []byte{0, 0, 255, 0}, []byte{255, 0, 0, 0} // BGR + pad
	rows := append(append([]byte{}, red...), blue...)       // first stored row, then the second
	bottomUp := dibBuilder{headerSize: 40, width: 1, height: 2, bits: 24, pixels: rows}
	data, err := DIBToPNG(bottomUp.bytes())
	if err != nil {
		t.Fatal(err)
	}
	picture := decodePNG(t, data)
	wantPixel(t, picture, 0, 0, color.NRGBA{0, 0, 255, 255}) // the last stored row is the top
	wantPixel(t, picture, 0, 1, color.NRGBA{255, 0, 0, 255})

	topDown := dibBuilder{headerSize: 40, width: 1, height: -2, bits: 24, pixels: rows}
	if data, err = DIBToPNG(topDown.bytes()); err != nil {
		t.Fatal(err)
	}
	picture = decodePNG(t, data)
	wantPixel(t, picture, 0, 0, color.NRGBA{255, 0, 0, 255})
	wantPixel(t, picture, 0, 1, color.NRGBA{0, 0, 255, 255})
}

func TestDIB32BitIgnoresUnusedAlphaByte(t *testing.T) {
	// A BI_RGB screenshot leaves the fourth byte zero; it must not read as transparent.
	dib := dibBuilder{headerSize: 40, width: 2, height: 1, bits: 32, pixels: []byte{10, 20, 30, 0, 1, 2, 3, 0}}
	data, err := DIBToPNG(dib.bytes())
	if err != nil {
		t.Fatal(err)
	}
	picture := decodePNG(t, data)
	wantPixel(t, picture, 0, 0, color.NRGBA{30, 20, 10, 255})
	wantPixel(t, picture, 1, 0, color.NRGBA{3, 2, 1, 255})
}

func TestDIBV5AlphaMask(t *testing.T) {
	dib := dibBuilder{
		headerSize: 124, width: 2, height: -1, bits: 32, compression: dibBitfields,
		masks:  [4]uint32{0x00ff0000, 0x0000ff00, 0x000000ff, 0xff000000},
		pixels: []byte{10, 20, 30, 128, 1, 2, 3, 0},
	}
	data, err := DIBToPNG(dib.bytes())
	if err != nil {
		t.Fatal(err)
	}
	picture := decodePNG(t, data)
	wantPixel(t, picture, 0, 0, color.NRGBA{30, 20, 10, 128})
	wantPixel(t, picture, 1, 0, color.NRGBA{3, 2, 1, 0}) // a real alpha of zero stays transparent

	allZero := dib
	allZero.pixels = []byte{10, 20, 30, 0, 1, 2, 3, 0}
	if data, err = DIBToPNG(allZero.bytes()); err != nil {
		t.Fatal(err)
	}
	wantPixel(t, decodePNG(t, data), 0, 0, color.NRGBA{30, 20, 10, 255}) // an alpha plane that is all zero is unused
}

func TestDIBInfoHeaderBitfieldMasks(t *testing.T) {
	// 16-bit 5-6-5 with the masks after a 40-byte header.
	pixel := uint16(31<<11 | 0<<5 | 0) // pure red
	dib := dibBuilder{
		headerSize: 40, width: 1, height: 1, bits: 16, compression: dibBitfields,
		masks:  [4]uint32{0xf800, 0x07e0, 0x001f},
		pixels: []byte{byte(pixel), byte(pixel >> 8), 0, 0},
	}
	data, err := DIBToPNG(dib.bytes())
	if err != nil {
		t.Fatal(err)
	}
	wantPixel(t, decodePNG(t, data), 0, 0, color.NRGBA{255, 0, 0, 255})

	// 32-bit BGRA with ALPHABITFIELDS: four masks after the header.
	alpha := dibBuilder{
		headerSize: 40, width: 1, height: 1, bits: 32, compression: dibAlphaBitfields,
		masks:  [4]uint32{0x00ff0000, 0x0000ff00, 0x000000ff, 0xff000000},
		pixels: []byte{0, 0, 200, 100},
	}
	if data, err = DIBToPNG(alpha.bytes()); err != nil {
		t.Fatal(err)
	}
	wantPixel(t, decodePNG(t, data), 0, 0, color.NRGBA{200, 0, 0, 100})
}

func TestDIB16BitDefaultIs555(t *testing.T) {
	pixel := uint16(31 << 10) // red in 5-5-5
	dib := dibBuilder{headerSize: 40, width: 1, height: 1, bits: 16, pixels: []byte{byte(pixel), byte(pixel >> 8), 0, 0}}
	data, err := DIBToPNG(dib.bytes())
	if err != nil {
		t.Fatal(err)
	}
	wantPixel(t, decodePNG(t, data), 0, 0, color.NRGBA{255, 0, 0, 255})
}

func TestDIBPalettes(t *testing.T) {
	palette := [][3]uint8{{255, 0, 0}, {0, 255, 0}}
	// 1 bit, 3 pixels: 1,0,1 -> MSB first.
	one := dibBuilder{headerSize: 40, width: 3, height: 1, bits: 1, palette: palette, pixels: []byte{0b10100000, 0, 0, 0}}
	data, err := DIBToPNG(one.bytes())
	if err != nil {
		t.Fatal(err)
	}
	picture := decodePNG(t, data)
	wantPixel(t, picture, 0, 0, color.NRGBA{0, 255, 0, 255})
	wantPixel(t, picture, 1, 0, color.NRGBA{255, 0, 0, 255})
	wantPixel(t, picture, 2, 0, color.NRGBA{0, 255, 0, 255})

	// 4 bit, 3 pixels: indexes 1,0,1 -> high nibble first; colors given as 2.
	four := dibBuilder{headerSize: 40, width: 3, height: 1, bits: 4, colors: 2, palette: palette, pixels: []byte{0x10, 0x10, 0, 0}}
	if data, err = DIBToPNG(four.bytes()); err != nil {
		t.Fatal(err)
	}
	picture = decodePNG(t, data)
	wantPixel(t, picture, 0, 0, color.NRGBA{0, 255, 0, 255})
	wantPixel(t, picture, 1, 0, color.NRGBA{255, 0, 0, 255})
	wantPixel(t, picture, 2, 0, color.NRGBA{0, 255, 0, 255})

	// 8 bit.
	eight := dibBuilder{headerSize: 40, width: 2, height: 1, bits: 8, colors: 2, palette: palette, pixels: []byte{1, 0, 0, 0}}
	if data, err = DIBToPNG(eight.bytes()); err != nil {
		t.Fatal(err)
	}
	wantPixel(t, decodePNG(t, data), 0, 0, color.NRGBA{0, 255, 0, 255})
}

func TestDIBRejectsDamagedAndUnsupported(t *testing.T) {
	good := dibBuilder{headerSize: 40, width: 2, height: 2, bits: 24, pixels: make([]byte, 16)}
	cases := map[string][]byte{
		"empty":     nil,
		"truncated": good.bytes()[:50],
		"rle":       dibBuilder{headerSize: 40, width: 2, height: 2, bits: 8, compression: dibRLE8, pixels: make([]byte, 16)}.bytes(),
		"odd depth": dibBuilder{headerSize: 40, width: 2, height: 2, bits: 12, pixels: make([]byte, 16)}.bytes(),
		"no pixels": dibBuilder{headerSize: 40, width: 2, height: 2, bits: 24}.bytes(),
	}
	for name, dib := range cases {
		if _, err := DIBToPNG(dib); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
}

func TestDIBHugeDimensionsAreTooLarge(t *testing.T) {
	// A header that promises 60000x60000 pixels must not allocate them.
	dib := dibBuilder{headerSize: 40, width: 60000, height: 60000, bits: 24}.bytes()
	if _, err := DIBToPNG(dib); !errors.Is(err, ErrClipboardImageTooLarge) {
		t.Fatalf("error = %v, want ErrClipboardImageTooLarge", err)
	}
	if _, err := DIBToPNG(make([]byte, MaxClipboardRawBytes+1)); !errors.Is(err, ErrClipboardImageTooLarge) {
		t.Fatalf("oversize input error = %v, want ErrClipboardImageTooLarge", err)
	}
}

func TestEncodePNGEnforcesCap(t *testing.T) {
	// Noise does not compress: 4096x4096 RGB noise is about 48 MiB, over the cap.
	picture := image.NewNRGBA(image.Rect(0, 0, 4096, 4096))
	state := uint32(1)
	for index := range picture.Pix {
		state = state*1664525 + 1013904223
		picture.Pix[index] = byte(state >> 24)
	}
	if _, err := EncodePNG(picture); !errors.Is(err, ErrClipboardImageTooLarge) {
		t.Fatalf("error = %v, want ErrClipboardImageTooLarge", err)
	}
}

// screenshotDIB is a 1080p 32-bit bottom-up DIB with screenshot-like content:
// flat areas, gradients and a little noise.
func screenshotDIB(width, height int) []byte {
	pixels := make([]byte, width*height*4)
	state := uint32(7)
	for y := range height {
		for x := range width {
			index := (y*width + x) * 4
			state = state*1664525 + 1013904223
			noise := byte(0)
			if x%97 == 0 {
				noise = byte(state >> 28)
			}
			pixels[index] = byte(x/8) + noise
			pixels[index+1] = byte(y / 4)
			pixels[index+2] = 0x30
		}
	}
	return dibBuilder{headerSize: 40, width: width, height: height, bits: 32, pixels: pixels}.bytes()
}

func BenchmarkDIBToPNG1080p(b *testing.B) {
	dib := screenshotDIB(1920, 1080)
	b.SetBytes(int64(len(dib)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := DIBToPNG(dib); err != nil {
			b.Fatal(err)
		}
	}
}

// The .NET clipboard writes the bit masks twice; the pixels are the last bytes.
func TestDIBWithMasksRepeatedBeforeThePixels(t *testing.T) {
	header := make([]byte, 40)
	binary.LittleEndian.PutUint32(header[0:], 40)
	binary.LittleEndian.PutUint32(header[4:], 1)
	binary.LittleEndian.PutUint32(header[8:], 1)
	binary.LittleEndian.PutUint16(header[12:], 1)
	binary.LittleEndian.PutUint16(header[14:], 32)
	binary.LittleEndian.PutUint32(header[16:], 3)
	binary.LittleEndian.PutUint32(header[20:], 4)
	masks := []byte{0, 0, 0xff, 0, 0, 0xff, 0, 0, 0xff, 0, 0, 0}
	dib := append(append(append(header, masks...), masks...), 10, 20, 30, 255) // B G R A
	data, err := DIBToPNG(dib)
	if err != nil {
		t.Fatal(err)
	}
	picture, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got := color.NRGBAModel.Convert(picture.At(0, 0)); got != (color.NRGBA{30, 20, 10, 255}) {
		t.Fatalf("pixel = %v", got)
	}
}
