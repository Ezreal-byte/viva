package render

import (
	"testing"

	"github.com/vivaui/viva/internal/scene"
)

// Appending one shape to a large scene must stay a suffix update, not a full
// rebuild of every instance.
func BenchmarkStoreAppendTail(b *testing.B) {
	graph := scene.NewWithCapacity(10_001)
	for index := 0; index < 10000; index++ {
		graph.AddRectangle(graph.Root(), scene.Rect{X: float32(index)}, scene.Color{Alpha: 1})
	}
	var store Store
	if _, err := store.Apply(graph.DrainChanges(nil)); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		graph.AddRectangle(graph.Root(), scene.Rect{X: 1}, scene.Color{Alpha: 1})
		update, err := store.Apply(graph.DrainChanges(nil))
		if err != nil || update.Rebuild {
			b.Fatalf("append update = %+v, err %v", update, err)
		}
	}
}
