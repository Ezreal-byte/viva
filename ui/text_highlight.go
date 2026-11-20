package ui

import "github.com/vivaui/viva/gfx"

// Highlight marks case-insensitive matches in Text or Paragraph while
// preserving the original label's shaping, wrapping and text direction.
func (b Box) Highlight(query string, color Color) Box {
	extra := b.f.extra(b.i)
	extra.highlightQuery = query
	extra.highlightColor = color
	return b
}

func (f *frame) highlightedLabel(n *node, value string, bounds Rect, style gfx.TextStyle, alignment gfx.Alignment) {
	extra := &f.exts[n.ext]
	f.canvas.LabelHighlighted(value, bounds, style, alignment, extra.highlightQuery, fade(extra.highlightColor, f.opacity))
}
