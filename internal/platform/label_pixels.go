package platform

import "math"

// DevicePosition is where, in whole device pixels, a label whose layout position is
// logical (the painter's rounded X plus the OriginX it took off) starts at a display
// scale. A label texture holds one texel per device pixel, so drawing it anywhere but
// on a whole device pixel makes the sampler blend neighbouring texels: at 125% or
// 150% that is most labels, and it blurs the text. Rounding the layout position
// moves the text by at most half a device pixel.
func DevicePosition(logical, origin, scale float32) int {
	return int(math.Floor(float64(logical+origin)*float64(scale) + 0.5))
}

// DeviceExtent is the width in whole device pixels of the span from logical to
// logical+length: both edges are rounded as DevicePosition rounds them, so labels
// that meet at a logical edge meet at a device edge, with no gap and no overlap.
func DeviceExtent(logical, length, scale float32) int {
	return DevicePosition(logical+length, 0, scale) - DevicePosition(logical, 0, scale)
}

// ResampleBGRA shrinks premultiplied 8-bit BGRA pixels from srcWidth by srcHeight to
// width by height by averaging the source area under each destination pixel, with
// fractional weights at the edges, so a bitmap drawn smaller than it is stored keeps
// its detail without the aliasing a bilinear sample of a few texels gives. A
// destination larger than the source in either direction is not supported: the
// result is nil, and the sampler enlarges such a bitmap.
func ResampleBGRA(pixels []byte, stride, srcWidth, srcHeight, width, height int) []byte {
	if width <= 0 || height <= 0 || width > srcWidth || height > srcHeight || stride < srcWidth*4 ||
		len(pixels) < (srcHeight-1)*stride+srcWidth*4 {
		return nil
	}
	// Horizontal pass into float rows, then vertical: the filter is separable.
	horizontal := make([]float32, width*4*srcHeight)
	weights := areaWeights(srcWidth, width)
	for row := range srcHeight {
		line := pixels[row*stride : row*stride+srcWidth*4]
		out := horizontal[row*width*4 : (row+1)*width*4]
		for column := range width {
			span := weights[column]
			var sum [4]float32
			for source := span.first; source <= span.last; source++ {
				weight := span.weight(source)
				base := source * 4
				sum[0] += float32(line[base]) * weight
				sum[1] += float32(line[base+1]) * weight
				sum[2] += float32(line[base+2]) * weight
				sum[3] += float32(line[base+3]) * weight
			}
			copy(out[column*4:], sum[:])
		}
	}
	result := make([]byte, width*height*4)
	rows := areaWeights(srcHeight, height)
	for row := range height {
		span := rows[row]
		out := result[row*width*4 : (row+1)*width*4]
		for index := range width * 4 {
			var sum float32
			for source := span.first; source <= span.last; source++ {
				sum += horizontal[source*width*4+index] * span.weight(source)
			}
			out[index] = byte(min(max(sum+0.5, 0), 255))
		}
	}
	return result
}

// areaSpan is the run of source pixels a destination pixel covers: whole pixels in
// between and a fraction of the first and last. Weights sum to one.
type areaSpan struct {
	first, last int
	head, tail  float32 // the share of the first and of the last source pixel
	scale       float32 // 1 over the number of source pixels one destination pixel covers
}

func (span areaSpan) weight(source int) float32 {
	switch source {
	case span.first:
		return span.head * span.scale
	case span.last:
		return span.tail * span.scale
	}
	return span.scale
}

func areaWeights(source, destination int) []areaSpan {
	ratio := float64(source) / float64(destination)
	spans := make([]areaSpan, destination)
	for index := range spans {
		start, end := float64(index)*ratio, float64(index+1)*ratio
		first, last := int(math.Floor(start)), int(math.Ceil(end))-1
		last = min(max(last, first), source-1)
		span := areaSpan{first: first, last: last, scale: float32(1 / ratio)}
		if first == last {
			span.head = float32(end - start)
		} else {
			span.head = float32(float64(first+1) - start)
			span.tail = float32(end - float64(last))
		}
		spans[index] = span
	}
	return spans
}
