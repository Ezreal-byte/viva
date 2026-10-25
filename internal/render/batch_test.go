package render

import (
	"testing"

	"github.com/vivaui/viva/internal/scene"
)

func TestGroupBatchesSplitsOnLayerAndClipAndKeepsPaintOrder(t *testing.T) {
	cut := scene.Rect{X: 10, Y: 10, Width: 100, Height: 50}
	other := scene.Rect{X: 0, Y: 0, Width: 5, Height: 5}
	instances := []Rectangle{
		{Layer: 1},
		{Layer: 1},
		{Layer: 1, Clipped: true, Clip: cut},
		{Layer: 1, Clipped: true, Clip: cut},
		{Layer: 1, Clipped: true, Clip: other},
		{Layer: 0},
		{Layer: 1},
	}
	got := GroupBatches(nil, instances)
	want := []Batch{
		{Layer: 0, First: 5, Count: 1},
		{Layer: 1, First: 0, Count: 2},
		{Layer: 1, Clipped: true, Clip: cut, First: 2, Count: 2},
		{Layer: 1, Clipped: true, Clip: other, First: 4, Count: 1},
		{Layer: 1, First: 6, Count: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("batches = %+v, want %+v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("batch %d = %+v, want %+v", index, got[index], want[index])
		}
	}
}

func TestGroupBatchesIgnoresStaleClipOnUnclippedInstances(t *testing.T) {
	stale := scene.Rect{X: 1, Y: 2, Width: 3, Height: 4}
	got := GroupBatches(nil, []Rectangle{{Clip: stale}, {}})
	if len(got) != 1 || got[0].Count != 2 || got[0].Clipped || got[0].Clip != (scene.Rect{}) {
		t.Fatalf("batches = %+v, want one unclipped run of 2", got)
	}
}

func TestGroupBatchesReusesItsBuffer(t *testing.T) {
	instances := []Rectangle{{Layer: 2}, {Layer: 1}, {Layer: 1, Clipped: true, Clip: scene.Rect{Width: 1, Height: 1}}}
	buffer := GroupBatches(nil, instances)
	if allocations := testing.AllocsPerRun(100, func() { buffer = GroupBatches(buffer, instances) }); allocations != 0 {
		t.Fatalf("GroupBatches allocated %v times with a large enough buffer", allocations)
	}
}

func TestScissorOfTakesTheNearerPixelBoundaryAndClamps(t *testing.T) {
	clip := scene.Rect{X: 10.4, Y: 20.6, Width: 30.2, Height: 10}
	got, visible := ScissorOf(clip, 1, 100, 100)
	if !visible || got != (Scissor{Left: 10, Top: 21, Right: 41, Bottom: 31}) {
		t.Fatalf("1x scissor = %+v visible %v", got, visible)
	}
	got, visible = ScissorOf(clip, 2, 100, 100)
	if !visible || got != (Scissor{Left: 21, Top: 41, Right: 81, Bottom: 61}) {
		t.Fatalf("2x scissor = %+v visible %v", got, visible)
	}
	got, visible = ScissorOf(scene.Rect{X: -20, Y: -5, Width: 500, Height: 500}, 1, 100, 80)
	if !visible || got != (Scissor{Left: 0, Top: 0, Right: 100, Bottom: 80}) {
		t.Fatalf("clamped scissor = %+v visible %v", got, visible)
	}
}

func TestScissorOfReportsNothingLeft(t *testing.T) {
	for name, clip := range map[string]scene.Rect{
		"beyond the target":    {X: 200, Y: 0, Width: 10, Height: 10},
		"thinner than a pixel": {X: 10.1, Y: 10, Width: 0.2, Height: 10},
		"empty":                {X: 5, Y: 5},
	} {
		if got, visible := ScissorOf(clip, 1, 100, 100); visible {
			t.Fatalf("%s: scissor %+v is visible", name, got)
		}
	}
}
