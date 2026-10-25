package platform

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"testing"
)

func TestCheckClipboardPNG(t *testing.T) {
	var small bytes.Buffer
	if err := png.Encode(&small, image.NewNRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := CheckClipboardPNG(small.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := CheckClipboardPNG(nil); !errors.Is(err, ErrNoClipboardImage) {
		t.Fatalf("empty = %v, want ErrNoClipboardImage", err)
	}
	big := append(append([]byte{}, pngSignature...), make([]byte, MaxClipboardImageBytes)...)
	if err := CheckClipboardPNG(big); !errors.Is(err, ErrClipboardImageTooLarge) {
		t.Fatalf("oversize = %v, want ErrClipboardImageTooLarge", err)
	}
	if err := CheckClipboardPNG([]byte("GIF89a...")); err == nil {
		t.Fatal("a GIF was accepted as PNG")
	}
}

func BenchmarkCheckClipboardPNG(b *testing.B) {
	data := append(append([]byte{}, pngSignature...), make([]byte, 1<<20)...)
	b.ReportAllocs()
	for b.Loop() {
		if err := CheckClipboardPNG(data); err != nil {
			b.Fatal(err)
		}
	}
}

func TestTrimClipboardPNGDropsBlockPadding(t *testing.T) {
	var small bytes.Buffer
	if err := png.Encode(&small, image.NewNRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	padded := append(append([]byte{}, small.Bytes()...), make([]byte, 100)...)
	trimmed, err := TrimClipboardPNG(padded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(trimmed, small.Bytes()) {
		t.Fatalf("trimmed to %d bytes, want %d", len(trimmed), small.Len())
	}
	if _, err := TrimClipboardPNG(small.Bytes()[:small.Len()-20]); err == nil {
		t.Fatal("a truncated PNG was accepted")
	}
}
