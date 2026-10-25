package render

import (
	"math"
	"slices"

	"github.com/vivaui/viva/internal/scene"
)

// Batch is a run of consecutive instances drawn together: they share a layer
// and a clip, so a renderer issues one draw call with one scissor rectangle.
type Batch struct {
	Layer   uint16
	Clipped bool
	Clip    scene.Rect
	First   int
	Count   int
}

// GroupBatches splits instances, kept in paint order, into runs of equal layer
// and clip, then orders the runs by layer. The sort is stable, so within a layer
// the runs stay in paint order, and the instance list itself is never reordered.
// It appends to dst[:0] and allocates only when dst is too small.
func GroupBatches(dst []Batch, instances []Rectangle) []Batch {
	dst = dst[:0]
	for first := 0; first < len(instances); {
		head := &instances[first]
		last := first + 1
		for last < len(instances) && sameBatch(head, &instances[last]) {
			last++
		}
		batch := Batch{Layer: head.Layer, First: first, Count: last - first}
		if head.Clipped {
			batch.Clipped, batch.Clip = true, head.Clip
		}
		dst = append(dst, batch)
		first = last
	}
	slices.SortStableFunc(dst, func(left, right Batch) int {
		return int(left.Layer) - int(right.Layer)
	})
	return dst
}

func sameBatch(left, right *Rectangle) bool {
	if left.Layer != right.Layer || left.Clipped != right.Clipped {
		return false
	}
	return !left.Clipped || left.Clip == right.Clip
}

// Scissor is a rectangle of whole target pixels: Left and Top inclusive, Right
// and Bottom exclusive.
type Scissor struct {
	Left, Top, Right, Bottom int
}

// ScissorOf converts a clip in logical pixels to the target's pixels. A pixel is
// inside when its centre is inside the clip, so an edge falling between two pixel
// boundaries moves to the nearer one, which is the coverage a hard-edged shape
// had before it was clipped by the GPU instead of the CPU. The result is clamped
// to the width x height target; visible is false when nothing of it is left.
func ScissorOf(clip scene.Rect, scale float32, width, height int) (scissor Scissor, visible bool) {
	scissor = Scissor{
		Left:   nearestPixel(clip.X*scale, width),
		Top:    nearestPixel(clip.Y*scale, height),
		Right:  nearestPixel((clip.X+clip.Width)*scale, width),
		Bottom: nearestPixel((clip.Y+clip.Height)*scale, height),
	}
	return scissor, scissor.Right > scissor.Left && scissor.Bottom > scissor.Top
}

func nearestPixel(value float32, limit int) int {
	pixel := math.Floor(float64(value) + 0.5)
	return int(min(max(pixel, 0), float64(limit)))
}
