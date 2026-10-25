package appbundle

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

// icnsKind is one icon representation in an .icns file. Every kind here holds a
// PNG; Pixels is its width and height.
type icnsKind struct {
	Type   string
	Pixels int
}

// icnsKinds is what iconutil writes for a full iconset, as PNG: each size once
// as the plain image and once as the @2x of the half size. The 16 and 32 pixel
// plain entries (icp4, icp5) are left out on purpose: IconServices reads their
// payload as the old ARGB format even when it is a PNG, and Finder's list view,
// Spotlight and Get Info then draw the small icon as colored noise. macOS scales
// 16 px down from ic11 and 32 px from ic12 (the same pixels) instead.
var icnsKinds = []icnsKind{
	{"ic11", 32}, {"ic12", 64}, {"ic07", 128},
	{"ic13", 256}, {"ic08", 256}, {"ic14", 512}, {"ic09", 512}, {"ic10", 1024},
}

// MinIconPixels is the smallest source image accepted.
const MinIconPixels = 128

// RecommendedIconPixels is the size at which every representation is
// downscaled from the source rather than missing.
const RecommendedIconPixels = 1024

// EncodeICNS builds an .icns file from a square image. Representations larger
// than the source are left out instead of being upscaled, so a smaller source
// gives a smaller file, and the largest size actually used is returned.
func EncodeICNS(source image.Image) (data []byte, largest int, err error) {
	bounds := source.Bounds()
	if bounds.Dx() != bounds.Dy() {
		return nil, 0, fmt.Errorf("appbundle: the icon must be square, not %dx%d", bounds.Dx(), bounds.Dy())
	}
	if bounds.Dx() < MinIconPixels {
		return nil, 0, fmt.Errorf("appbundle: the icon is %d px; at least %d px is required (%d recommended)", bounds.Dx(), MinIconPixels, RecommendedIconPixels)
	}

	var body bytes.Buffer
	scaled := map[int][]byte{}
	for _, kind := range icnsKinds {
		if kind.Pixels > bounds.Dx() {
			continue
		}
		encoded, ok := scaled[kind.Pixels]
		if !ok {
			var buffer bytes.Buffer
			encoder := png.Encoder{CompressionLevel: png.BestCompression}
			if err := encoder.Encode(&buffer, Downscale(source, kind.Pixels)); err != nil {
				return nil, 0, err
			}
			encoded = buffer.Bytes()
			scaled[kind.Pixels] = encoded
		}
		body.WriteString(kind.Type)
		binary.Write(&body, binary.BigEndian, uint32(8+len(encoded)))
		body.Write(encoded)
		largest = max(largest, kind.Pixels)
	}

	var file bytes.Buffer
	file.WriteString("icns")
	binary.Write(&file, binary.BigEndian, uint32(8+body.Len()))
	file.Write(body.Bytes())
	return file.Bytes(), largest, nil
}

// Downscale shrinks a square image to size by averaging the source pixels each
// destination pixel covers (a box filter), weighting partial pixels by area.
// The average is taken on premultiplied color so a transparent edge does not
// darken the icon.
func Downscale(source image.Image, size int) *image.NRGBA {
	bounds := source.Bounds()
	if bounds.Dx() == size {
		out := image.NewNRGBA(image.Rect(0, 0, size, size))
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				out.Set(x, y, color.NRGBAModel.Convert(source.At(bounds.Min.X+x, bounds.Min.Y+y)))
			}
		}
		return out
	}
	scale := float64(bounds.Dx()) / float64(size)
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		y0, y1 := float64(y)*scale, float64(y+1)*scale
		for x := 0; x < size; x++ {
			x0, x1 := float64(x)*scale, float64(x+1)*scale
			var r, g, b, a, weight float64
			for sy := int(y0); float64(sy) < y1; sy++ {
				wy := min(float64(sy+1), y1) - max(float64(sy), y0)
				for sx := int(x0); float64(sx) < x1; sx++ {
					w := wy * (min(float64(sx+1), x1) - max(float64(sx), x0))
					pr, pg, pb, pa := source.At(bounds.Min.X+sx, bounds.Min.Y+sy).RGBA()
					r += w * float64(pr)
					g += w * float64(pg)
					b += w * float64(pb)
					a += w * float64(pa)
					weight += w
				}
			}
			offset := out.PixOffset(x, y)
			if a == 0 {
				continue
			}
			// r, g, b, a are premultiplied sums in 0..65535; undo the alpha.
			out.Pix[offset+0] = uint8(min(255, r/a*255+0.5))
			out.Pix[offset+1] = uint8(min(255, g/a*255+0.5))
			out.Pix[offset+2] = uint8(min(255, b/a*255+0.5))
			out.Pix[offset+3] = uint8(min(255, a/weight/65535*255+0.5))
		}
	}
	return out
}
