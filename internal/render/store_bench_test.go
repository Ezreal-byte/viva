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

// Deleting one instance from a large page still rebuilds every instance,
// because the backends keep a store sized to the count. Measure that fallback
// so an incremental delete can be compared against it.
func BenchmarkStoreDeleteRebuild(b *testing.B) {
	graph := scene.NewWithCapacity(10_001)
	var id scene.ID
	for index := 0; index < 10000; index++ {
		next, _ := graph.AddRectangle(graph.Root(), scene.Rect{X: float32(index)}, scene.Color{Alpha: 1})
		if index == 5000 {
			id = next
		}
	}
	var store Store
	if _, err := store.Apply(graph.DrainChanges(nil)); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if !graph.Remove(id) {
			b.Fatal("remove failed")
		}
		next, ok := graph.AddRectangle(graph.Root(), scene.Rect{X: 1}, scene.Color{Alpha: 1})
		if !ok {
			b.Fatal("add failed")
		}
		id = next
		update, err := store.Apply(graph.DrainChanges(nil))
		if err != nil || !update.Rebuild {
			b.Fatalf("delete update = %+v, err %v", update, err)
		}
	}
}
