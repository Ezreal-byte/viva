package ui

import "slices"

// Painting an icon means flattening its curves and triangulating its strokes and
// fills, and an interface paints the same few icons at the same few sizes every
// frame. The tessellation depends on the icon, the size and whether the stroke is
// opaque, and on nothing else: where it is and what color it is are applied when it
// is replayed. So the canvas tessellates an icon once, at the origin, and every
// later paint is a translation and a color.
//
// The result is the same triangles the direct path emits (see the test), so no
// renderer changes; what is saved is the flattening, the stroke geometry and the
// ear clipping, which dominated a repaint of the Viva SSH workbench.

// iconCacheLimit bounds the cache: when it is full it starts over, so an
// application that paints many sizes cannot grow it without end.
const iconCacheLimit = 512

type iconKey struct {
	commands *PathCommand
	layers   *IconLayer
	viewBox  float32
	viewHigh float32
	stroke   float32
	fit      float32
	scale    float32 // the pixel scale the strokes were rounded for
	cap      StrokeCap
	join     StrokeJoin
	opaque   bool
}

// iconTriangle is one tessellated triangle, relative to the icon's origin. layer is
// -1 for the icon's own strokes and fills, and the index of the layer otherwise.
type iconTriangle struct {
	points [3]Point
	edges  uint8
	layer  int16
	// t is the gradient position at each corner of a triangle of a gradient layer.
	t [3]float32
}

type iconCache struct {
	entries map[iconKey][]iconTriangle
}

// iconPaintKey identifies a fully materialized icon: the tessellation, where it
// is, how it is tinted, and the canvas state its triangles carry. A repeated
// paint can then be one slice copy instead of one Triangle fill per triangle.
type iconPaintKey struct {
	icon        iconKey
	origin      Point
	tint        Color
	layer       uint16
	clip        Rect
	hasClip     bool
	rectsBefore uint32
}

type iconPaintCache struct {
	entries  map[iconPaintKey][]Triangle
	previous map[iconPaintKey][]Triangle
}

const iconPaintCacheLimit = 512

func (canvas *Canvas) iconPaints() *iconPaintCache {
	if canvas.iconPaintCache == nil {
		canvas.iconPaintCache = new(iconPaintCache)
	}
	return canvas.iconPaintCache
}

func (cache *iconPaintCache) get(key iconPaintKey) ([]Triangle, bool) {
	if triangles, ok := cache.entries[key]; ok {
		return triangles, true
	}
	triangles, ok := cache.previous[key]
	if !ok {
		return nil, false
	}
	delete(cache.previous, key)
	cache.put(key, triangles)
	return triangles, true
}

func (cache *iconPaintCache) put(key iconPaintKey, triangles []Triangle) {
	if cache.entries == nil {
		cache.entries = make(map[iconPaintKey][]Triangle, 64)
	}
	if len(cache.entries) >= iconPaintCacheLimit {
		cache.previous = cache.entries
		cache.entries = make(map[iconPaintKey][]Triangle, 64)
	}
	cache.entries[key] = triangles
}

func (canvas *Canvas) icons() *iconCache {
	if canvas.iconCache == nil {
		canvas.iconCache = new(iconCache)
	}
	return canvas.iconCache
}

func newIconKey(icon Icon, fit, scale float32, tint Color) iconKey {
	key := iconKey{viewBox: icon.ViewBox, viewHigh: icon.height(), stroke: icon.Stroke, fit: fit, scale: scale, cap: icon.Cap, join: icon.Join, opaque: tint.Alpha >= 0.999}
	if len(icon.Commands) > 0 {
		key.commands = &icon.Commands[0]
	}
	if len(icon.Layers) > 0 {
		key.layers = &icon.Layers[0]
	}
	return key
}

func (canvas *Canvas) drawIconCached(icon Icon, fit float32, origin Point, tint Color) {
	key := newIconKey(icon, fit, canvas.PixelScale(), tint)
	// The layer, the clip and the rectangle count are the same for every triangle
	// of the icon: read them once.
	layer := canvas.currentLayer()
	clip, hasClip := canvas.clipSnapshot()
	rectsBefore := uint32(len(canvas.rectangles))
	paintKey := iconPaintKey{icon: key, origin: origin, tint: tint, layer: layer, clip: clip, hasClip: hasClip, rectsBefore: rectsBefore}
	paints := canvas.iconPaints()
	if cached, ok := paints.get(paintKey); ok {
		canvas.triangles = append(canvas.triangles, cached...)
		return
	}
	cache := canvas.icons()
	triangles, ok := cache.entries[key]
	if !ok {
		triangles = canvas.tessellateIcon(icon, fit, tint)
		if cache.entries == nil || len(cache.entries) >= iconCacheLimit {
			cache.entries = make(map[iconKey][]iconTriangle, 64)
		}
		cache.entries[key] = triangles
	}
	start := len(canvas.triangles)
	canvas.triangles = slices.Grow(canvas.triangles, len(triangles))
	for index := range triangles {
		cached := &triangles[index]
		triangle := Triangle{
			Points: [3]Point{
				{X: cached.points[0].X + origin.X, Y: cached.points[0].Y + origin.Y},
				{X: cached.points[1].X + origin.X, Y: cached.points[1].Y + origin.Y},
				{X: cached.points[2].X + origin.X, Y: cached.points[2].Y + origin.Y},
			},
			Color: tint, Layer: layer, EdgeMask: cached.edges,
			ClipOf: clip, HasClip: hasClip, RectsBefore: rectsBefore,
		}
		if cached.layer >= 0 {
			source := &icon.Layers[cached.layer]
			if source.Gradient != nil {
				colorize(&triangle, source.Gradient, cached.t, tint.Alpha)
			} else {
				triangle.Color = fade(source.Fill, tint.Alpha)
			}
			if !triangle.visible() {
				continue
			}
		}
		canvas.triangles = append(canvas.triangles, triangle)
	}
	paints.put(paintKey, append([]Triangle(nil), canvas.triangles[start:]...))
}

// tessellateIcon paints the icon at the origin with the direct path and takes the
// triangles it appended back out of the canvas, recording which layer made each.
// Colors are placeholders of the right opacity: geometry does not depend on them.
func (canvas *Canvas) tessellateIcon(icon Icon, fit float32, tint Color) []iconTriangle {
	probe := Color{Red: 1, Green: 1, Blue: 1, Alpha: 1}
	if tint.Alpha < 0.999 {
		probe.Alpha = 0.5
	}
	start, rectangles := len(canvas.triangles), len(canvas.rectangles)
	var out []iconTriangle
	canvas.paintIconShapes(icon, Rect{Width: fit, Height: fit * icon.height() / icon.ViewBox}, probe, func(layer int) {
		if layer >= 0 && icon.Layers[layer].Gradient != nil {
			axis := newGradientAxis(*icon.Layers[layer].Gradient, Rect{Width: fit, Height: fit * icon.height() / icon.ViewBox}, fit/icon.ViewBox, fit/icon.ViewBox)
			for _, triangle := range canvas.triangles[start:] {
				axis.each(triangle.Points, triangle.EdgeMask, func(points [3]Point, edges uint8) {
					out = append(out, iconTriangle{points: points, edges: edges, layer: int16(layer), t: axis.vertexT(points)})
				})
			}
		} else {
			for _, triangle := range canvas.triangles[start:] {
				out = append(out, iconTriangle{points: triangle.Points, edges: triangle.EdgeMask, layer: int16(layer)})
			}
		}
		canvas.triangles = canvas.triangles[:start]
	})
	canvas.rectangles = canvas.rectangles[:rectangles]
	return out
}
