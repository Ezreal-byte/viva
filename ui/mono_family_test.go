package ui

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestMonoFamilyIsStableAndBounded(t *testing.T) {
	monoFamilies.Store(nil)
	first := MonoFamily("JetBrains Mono")
	if first != 2 || MonoFamily(" JetBrains Mono ") != first {
		t.Fatalf("first face = %d", first)
	}
	if second := MonoFamily("Menlo"); second != 3 || MonoFamilyName(second) != "Menlo" || MonoFamilyName(first) != "JetBrains Mono" {
		t.Fatalf("second face = %d", second)
	}
	for _, bad := range []string{"", "   ", "a\x00b", "line\nbreak", strings.Repeat("x", 97)} {
		if MonoFamily(bad) != MonoSystem {
			t.Errorf("%q was registered", bad)
		}
	}
	for index := 0; index < MaxMonoFamilies+5; index++ {
		MonoFamily(fmt.Sprintf("Family %d", index))
	}
	if MonoFamilyCount() != MaxMonoFamilies || MonoFamily("One too many") != MonoSystem {
		t.Fatalf("registry holds %d", MonoFamilyCount())
	}
	if MonoFamilyName(MonoSystem) != "" || MonoFamilyName(MonoClassic) != "" || MonoFamilyName(255) != "" {
		t.Fatal("a built-in or unknown face has a name")
	}
}

func TestMonoFamilyConcurrentRegistrationAgreesOnFaces(t *testing.T) {
	monoFamilies.Store(nil)
	var wait sync.WaitGroup
	faces := make([]MonoFace, 16)
	for index := range faces {
		wait.Add(1)
		go func() {
			defer wait.Done()
			faces[index] = MonoFamily("Shared Family")
		}()
	}
	wait.Wait()
	for _, face := range faces {
		if face != faces[0] || MonoFamilyName(face) != "Shared Family" {
			t.Fatalf("faces = %v", faces)
		}
	}
	if MonoFamilyCount() != 1 {
		t.Fatalf("count = %d", MonoFamilyCount())
	}
}

func TestACanvasMeasuresARegisteredFamilyAndTreatsAnUnknownFaceAsTheSystemOne(t *testing.T) {
	monoFamilies.Store(nil)
	var canvas Canvas
	asked := map[MonoFace]int{}
	canvas.SetCellMeasurer(func(size float32, face MonoFace) (float32, float32) {
		asked[face]++
		return 7, 15
	})
	face := MonoFamily("Test Mono")
	canvas.MonoCellSize(13, face)
	canvas.MonoCellSize(13, 9) // not registered
	if asked[face] != 1 || asked[MonoSystem] != 1 {
		t.Fatalf("measurer asked %v", asked)
	}
}
