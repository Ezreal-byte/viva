package headless

import (
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"
)

// A screenshot must draw fixed-pitch text in the face the application asked for,
// the named families included.
func TestSceneryKeepsTheFixedPitchFaceOfALabel(t *testing.T) {
	face := ui.MonoFamily("Convert Test Mono")
	_, labels, _ := scenery(app.Snapshot{Labels: []app.Label{
		{Value: "a", Mono: true, MonoFace: ui.MonoClassic},
		{Value: "b", Mono: true, MonoFace: face},
	}})
	if len(labels) != 2 || labels[0].MonoFace != ui.MonoClassic || labels[1].MonoFace != face {
		t.Fatalf("labels = %+v", labels)
	}
}
