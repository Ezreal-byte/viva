package windows

import (
	"bytes"
	"testing"

	"github.com/vivaui/viva/internal/platform"
	"github.com/vivaui/viva/ui"
)

// TestFixedPitchLigatureRunShapesTheFont guards the Direct2D shaped-run path:
// the same cells must produce different pixels when the run may ligate. The
// Windows environments use Consolas, whose programming ligatures cover these
// sequences.
func TestFixedPitchLigatureRunShapesTheFont(t *testing.T) {
	render := func(ligatures bool) []byte {
		label := platform.TextLabel{
			Value: "fi -> => != <=", Width: 240, Height: 36, FontSize: 22, CellWidth: 13,
			Mono: true, Color: platform.Color{Red: 1, Green: 1, Blue: 1, Alpha: 1},
			Runs: []ui.TextRun{{Bytes: 14, Cells: 14, Color: ui.Color{Red: 1, Green: 1, Blue: 1, Alpha: 1}, Ligatures: ligatures}},
		}
		pixels := renderScene(t, newOffscreen(t, 250, 40, 1), nil, []platform.TextLabel{label})
		return pixels.Pix
	}
	if bytes.Equal(render(false), render(true)) {
		t.Fatal("the fixed-pitch renderer drew no ligature for Consolas")
	}
}
