package linux

import (
	"testing"

	"github.com/vivaui/viva/internal/platform"
)

func TestNetWMIconOffersEverySizeTheSourceCanFill(t *testing.T) {
	const side = 64
	pixels := make([]byte, side*side*4)
	for index := 0; index < len(pixels); index += 4 {
		copy(pixels[index:], []byte{0x10, 0x20, 0x30, 0x80})
	}
	data := netWMIcon(platform.WindowIcon{Size: side, RGBA: pixels})

	var sizes []uint32
	for offset := 0; offset < len(data); {
		width, height := data[offset], data[offset+1]
		if width != height {
			t.Fatalf("an image of %dx%d at element %d is not square", width, height, offset)
		}
		middle := data[offset+2+int(width*width)/2+int(width)/2]
		if middle != 0x80102030 {
			t.Fatalf("a pixel of 0x80/0x10/0x20/0x30 is packed as %#x, want 0x80102030 (alpha, red, green, blue)", middle)
		}
		if corner := data[offset+2]; corner>>24 != 0 {
			t.Fatalf("the corner of the %d px image is opaque (%#x): the square was not rounded", width, corner)
		}
		sizes = append(sizes, width)
		offset += 2 + int(width*height)
	}
	if len(sizes) != 4 || sizes[0] != 16 || sizes[3] != 64 {
		t.Fatalf("a 64 px source is offered at %v, want 16, 32, 48 and 64", sizes)
	}
}

func TestNetWMIconOfATinySourceIsOfferedAsItIs(t *testing.T) {
	data := netWMIcon(platform.WindowIcon{Size: 8, RGBA: make([]byte, 8*8*4)})
	if len(data) != 2+64 || data[0] != 8 {
		t.Fatalf("an 8 px source gives %d elements starting %d, want one 8x8 image", len(data), data[0])
	}
	if netWMIcon(platform.WindowIcon{}) != nil {
		t.Fatal("a window without an icon must not set the property")
	}
}
