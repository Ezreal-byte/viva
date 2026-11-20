package base

import "github.com/vivaui/viva/gfx"

// Highlight marks case-insensitive matches in Text or Paragraph while
// preserving the original label's shaping, wrapping and text direction.
func (b Box) Highlight(query string, color Color) Box {
	extra := b.f.Extra(b.i)
	extra.highlightQuery = query
	extra.highlightColor = color
	return b
}

func (f *Frame) highlightedLabel(n *Node, value string, bounds Rect, style gfx.TextStyle, alignment gfx.Alignment) {
	extra := &f.Exts[n.Ext]
	f.Surface.LabelHighlighted(value, bounds, style, alignment, extra.highlightQuery, fade(extra.highlightColor, f.Opacity))
}
