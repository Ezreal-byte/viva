package linuxicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func square(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 40, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestRenderSizes(t *testing.T) {
	for _, test := range []struct {
		source int
		want   []int
	}{
		{128, []int{48, 64, 128}}, {200, []int{48, 64, 128}}, {256, []int{48, 64, 128, 256}}, {1024, []int{48, 64, 128, 256}},
	} {
		renditions, largest, err := Render(square(t, test.source, test.source))
		if err != nil {
			t.Fatal(err)
		}
		if largest != test.source {
			t.Errorf("largest = %d", largest)
		}
		var got []int
		for _, rendition := range renditions {
			got = append(got, rendition.Pixels)
			decoded, err := png.Decode(bytes.NewReader(rendition.PNG))
			if err != nil || decoded.Bounds().Dx() != rendition.Pixels || decoded.Bounds().Dy() != rendition.Pixels {
				t.Errorf("%d: the PNG is not %d px square (%v)", test.source, rendition.Pixels, err)
			}
		}
		if len(got) != len(test.want) {
			t.Fatalf("%d px source gave %v, want %v", test.source, got, test.want)
		}
		for index := range got {
			if got[index] != test.want[index] {
				t.Errorf("%d px source gave %v, want %v", test.source, got, test.want)
			}
		}
		if Largest(renditions).Pixels != test.want[len(test.want)-1] {
			t.Error("Largest is not the last")
		}
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	source := square(t, 300, 300)
	first, _, _ := Render(source)
	second, _, _ := Render(source)
	for index := range first {
		if !bytes.Equal(first[index].PNG, second[index].PNG) {
			t.Errorf("size %d differs between runs", first[index].Pixels)
		}
	}
}

func TestRenderRejects(t *testing.T) {
	for _, test := range []struct {
		name   string
		source []byte
		want   string
	}{
		{"small", square(t, 64, 64), "at least 128"},
		{"not square", square(t, 256, 200), "square"},
		{"not a png", []byte("hello"), "not a PNG"},
	} {
		if _, _, err := Render(test.source); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("%s: %v", test.name, err)
		}
	}
}
