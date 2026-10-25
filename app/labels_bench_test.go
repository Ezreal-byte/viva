package app

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

// A labelled page used to cost O(controls*hits) in nameFromLabels. This keeps
// the linear implementation honest; the page has one label per control.
func BenchmarkPainterLabelledPage(b *testing.B) {
	for _, controls := range []int{1000, 4000} {
		b.Run(map[int]string{1000: "1k", 4000: "4k"}[controls], func(b *testing.B) {
			var canvas ui.Canvas
			config := Config{Title: "labels", Width: 1200, Height: 1200}
			config.Paint = func(c *ui.Canvas, _ ui.Size) {
				for index := 0; index < controls; index++ {
					id := ui.Item(index + 1)
					bounds := ui.Rect{X: float32(index%100) * 10, Y: float32(index/100) * 10, Width: 8, Height: 8}
					c.HitWithLabel(id, bounds, "control")
					c.HitLabel(id, ui.Rect{X: bounds.X, Y: bounds.Y + 2, Width: 4, Height: 4}, "label", false)
				}
			}
			painter := newTestPainter(b, &canvas, config)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if _, err := painter.repaint(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
