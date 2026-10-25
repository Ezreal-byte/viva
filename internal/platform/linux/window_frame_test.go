package linux

import "testing"

func TestFrameMarginKeepsWholePixelsAtEveryScale(t *testing.T) {
	for _, test := range []struct {
		scale float32
		want  uint32
	}{{1, 24}, {1.25, 30}, {1.5, 36}, {2, 48}, {0.5, 24}} {
		if got := frameMargin(test.scale); got != test.want {
			t.Fatalf("margin at %gx = %d, want %d", test.scale, got, test.want)
		}
	}
}

func TestContentSizeLeavesAtLeastAPixel(t *testing.T) {
	if got := contentSize(448, 24); got != 400 {
		t.Fatalf("content = %d", got)
	}
	if got := contentSize(40, 24); got != 1 {
		t.Fatalf("a window smaller than its margins has content %d", got)
	}
}

func TestFrameBandsTileTheMarginExactly(t *testing.T) {
	const width, height, margin = 448, 348, 24
	covered := make(map[[2]int32]int)
	for _, band := range frameBands(width, height, margin) {
		for y := band.y; y < band.y+band.height; y++ {
			for x := band.x; x < band.x+band.width; x++ {
				covered[[2]int32{x, y}]++
			}
		}
	}
	content := frameContent(width, height, margin)
	for y := int32(0); y < height; y++ {
		for x := int32(0); x < width; x++ {
			inside := x >= content.x && x < content.x+content.width && y >= content.y && y < content.y+content.height
			count := covered[[2]int32{x, y}]
			if inside && count != 0 || !inside && count != 1 {
				t.Fatalf("pixel %d,%d is in %d bands, inside the content: %v", x, y, count, inside)
			}
		}
	}
}

func TestFrameCornersSitInsideTheContent(t *testing.T) {
	const width, height, margin = 448, 348, 24
	content := frameContent(width, height, margin)
	for index, corner := range frameCorners(width, height, margin, 10) {
		if corner.width != 11 || corner.height != 11 {
			t.Fatalf("corner %d is %dx%d, want the radius and a pixel", index, corner.width, corner.height)
		}
		if corner.x < content.x || corner.y < content.y || corner.x+corner.width > content.x+content.width || corner.y+corner.height > content.y+content.height {
			t.Fatalf("corner %d %+v is outside the content %+v", index, corner, content)
		}
	}
}

func TestFrameInputReachesOnlyAGripIntoTheMargin(t *testing.T) {
	got := frameInput(448, 348, 24, 8)
	if got != (pixelBox{16, 16, 416, 316}) {
		t.Fatalf("input = %+v", got)
	}
	if got := frameInput(448, 348, 4, 8); got != (pixelBox{0, 0, 448, 348}) {
		t.Fatalf("a margin smaller than the grip: %+v", got)
	}
}
