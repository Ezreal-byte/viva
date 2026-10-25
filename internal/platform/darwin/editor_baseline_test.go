//go:build darwin && vivahead

package darwin

import (
	"testing"

	"github.com/vivaui/viva/internal/platform"
)

// inkRows is the first and last device row at or right of fromX that holds
// glyph pixels, or -1 for both when there are none.
func inkRows(pixels []byte, width, height, fromX int) (first, last int) {
	first, last = -1, -1
	for y := 0; y < height; y++ {
		for x := fromX; x < width; x++ {
			if b, g, r, _ := pixelAt(pixels, width, x, y); int(r)+int(g)+int(b) > 150 {
				if first < 0 {
					first = y
				}
				last = y
				break
			}
		}
	}
	return first, last
}

// A field's text must not move when focusing it hands the drawing from the
// label to the native editor: both stand on the baseline every label does,
// whatever the line metrics of the text's fallback fonts are.
func TestEditorTextStandsWhereTheLabelDoesInTheSameBox(t *testing.T) {
	white := platform.Color{Red: 1, Green: 1, Blue: 1, Alpha: 1}
	// The leading spaces keep the caret at the start of the editor out of the
	// measured columns.
	const value = "      主机Hxg 1212"
	for _, box := range []struct{ height, size float32 }{{36, 13}, {36, 14}, {28, 12}, {24, 14}, {40, 15}} {
		render := func(label *TextLabel, input *TextInput) (first, last int) {
			offscreen, err := NewOffscreen(160, 60, 2, Color{Alpha: 1})
			if err != nil {
				t.Skipf("no Metal device: %v", err)
			}
			defer offscreen.Close()
			var labels []TextLabel
			if label != nil {
				labels = []TextLabel{*label}
			}
			if err := offscreen.Set(nil, labels, input); err != nil {
				t.Fatal(err)
			}
			pixels, width, height, err := offscreen.Render()
			if err != nil {
				t.Fatal(err)
			}
			return inkRows(pixels, width, height, 2*24)
		}
		labelFirst, labelLast := render(&TextLabel{
			Value: value, X: 0, Y: 10, Width: 160, Height: box.height, FontSize: box.size, Color: white,
		}, nil)
		editorFirst, editorLast := render(nil, &TextInput{
			X: 0, Y: 10, Width: 160, Height: box.height, FontSize: box.size, Color: white,
			State:   TextState{Value: value},
			OnEvent: func(TextEvent) (TextState, error) { return TextState{}, nil },
		})
		if labelFirst < 0 || editorFirst < 0 {
			t.Fatalf("height %v size %v: no glyph pixels (label %d, editor %d)", box.height, box.size, labelFirst, editorFirst)
		}
		if labelFirst != editorFirst || labelLast != editorLast {
			t.Errorf("height %v size %v: the label's ink spans rows %d-%d, the editor's %d-%d", box.height, box.size, labelFirst, labelLast, editorFirst, editorLast)
		}
	}
}
