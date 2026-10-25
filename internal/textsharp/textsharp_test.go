package textsharp

import "testing"

// A stem on the pixel grid is sharper than the same stem spread over two pixels, and
// the measure says so for the quantities it is used for.
func TestASpreadStemScoresLowerThanAnAlignedOne(t *testing.T) {
	const width, height = 6, 8
	line := func(coverage ...byte) Sharpness {
		pixels := make([]byte, width*height)
		for y := 0; y < height; y++ {
			copy(pixels[y*width+2:], coverage)
		}
		return Measure(pixels, width, height, 1, width)
	}
	aligned, spread := line(255), line(128, 127)
	if aligned.Edge <= spread.Edge*1.5 || aligned.Solid != 1 || spread.Solid != 0 {
		t.Errorf("aligned %+v, spread %+v", aligned, spread)
	}
	if d := aligned.Ink - spread.Ink; d < -0.1*aligned.Ink || d > 0.1*aligned.Ink {
		t.Errorf("the same stem has ink %.1f aligned and %.1f spread", aligned.Ink, spread.Ink)
	}
}

func stemPlane(width, height int, columns ...int) Plane {
	plane := Plane{Width: width, Height: height, Pix: make([]float64, width*height), Chroma: make([]float64, width*height)}
	for y := 2; y < height-2; y++ {
		for _, x := range columns {
			plane.Pix[y*width+x] = 1
		}
	}
	return plane
}

// The comparison finds the shift that matches two renderings of the same stems and reports
// no difference once it is applied, and a real difference when the stems are not the
// same.
func TestCompareFindsTheShiftAndTheResidual(t *testing.T) {
	ref := stemPlane(20, 12, 4, 9)
	shifted := stemPlane(20, 12, 5, 10)
	f := Compare(shifted, ref, 2, 2)
	if f.DX != 1 || f.Residual != 0 || f.Mass != 1 {
		t.Errorf("shifted stems: %+v", f)
	}
	other := stemPlane(20, 12, 4, 12)
	if f := Compare(other, ref, 2, 2); f.Residual < 0.4 {
		t.Errorf("different stems matched: %+v", f)
	}
}
