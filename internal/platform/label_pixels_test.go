package platform

import "testing"

// A layout position half a device pixel either way snaps to the nearest pixel, and
// the remainder the painter took off is part of it.
func TestDevicePositionUsesTheLayoutOrigin(t *testing.T) {
	if got := DevicePosition(13, 0, 1.5); got != 20 { // 19.5 device pixels
		t.Errorf("13 at 1.5x: %d, want 20", got)
	}
	if got := DevicePosition(13, -0.4, 1.5); got != 19 { // 18.9
		t.Errorf("13 less 0.4 at 1.5x: %d, want 19", got)
	}
}

// Shrinking a bitmap keeps its average color, including where the ratio is not whole,
// and an even split of black and white comes out gray, not one of the two.
func TestResampleBGRAAveragesTheAreaUnderEachPixel(t *testing.T) {
	uniform := make([]byte, 7*5*4)
	for index := range uniform {
		uniform[index] = 200
	}
	for _, size := range [][2]int{{3, 2}, {4, 3}, {7, 5}, {1, 1}} {
		out := ResampleBGRA(uniform, 7*4, 7, 5, size[0], size[1])
		if len(out) != size[0]*size[1]*4 {
			t.Fatalf("size %v: %d bytes", size, len(out))
		}
		for index, value := range out {
			if value != 200 {
				t.Fatalf("size %v: byte %d is %d, want 200", size, index, value)
			}
		}
	}
	split := make([]byte, 4*1*4) // black, black, white, white
	for pixel := 2; pixel < 4; pixel++ {
		copy(split[pixel*4:], []byte{255, 255, 255, 255})
	}
	out := ResampleBGRA(split, 16, 4, 1, 1, 1)
	if out[0] != 128 && out[0] != 127 {
		t.Errorf("average of black and white is %d, want 127 or 128", out[0])
	}
}

func TestResampleBGRARefusesToEnlarge(t *testing.T) {
	if ResampleBGRA(make([]byte, 16), 8, 2, 2, 3, 2) != nil {
		t.Error("enlarging a bitmap returned pixels")
	}
}
