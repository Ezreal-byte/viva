//go:build js && wasm

package web

import (
	"errors"
	"syscall/js"
	"testing"
	"time"

	"github.com/vivaui/viva/internal/platform"
)

// installClipboard replaces navigator.clipboard with a stub whose read() resolves
// to one item offering the given types, or rejects with the named error.
func installClipboard(t *testing.T, script string) {
	t.Helper()
	stub := js.Global().Get("Function").New(script).Invoke()
	navigator := js.Global().Get("Object").New()
	navigator.Set("clipboard", stub)
	js.Global().Get("Object").Call("defineProperty", js.Global(), "navigator",
		map[string]any{"value": navigator, "configurable": true, "writable": true})
}

func readImage(t *testing.T) ([]byte, error) {
	t.Helper()
	type result struct {
		png []byte
		err error
	}
	done := make(chan result, 1)
	ReadClipboardImagePNG(func(png []byte, err error) { done <- result{png, err} })
	select {
	case got := <-done:
		return got.png, got.err
	case <-time.After(5 * time.Second):
		t.Fatal("the clipboard read never completed")
		return nil, nil
	}
}

const pngHeader = "new Uint8Array([0x89,0x50,0x4e,0x47,0x0d,0x0a,0x1a,0x0a,1,2,3])"

func TestReadClipboardImagePNG(t *testing.T) {
	installClipboard(t, `return { read: () => Promise.resolve([
		{ types: ["text/plain"], getType: () => Promise.reject(new Error("text")) },
		{ types: ["image/png"], getType: () => Promise.resolve(new Blob([`+pngHeader+`])) },
	]) }`)
	png, err := readImage(t)
	if err != nil || len(png) != 11 || png[1] != 'P' {
		t.Fatalf("png = %d bytes, %v", len(png), err)
	}
}

func TestReadClipboardImageReportsEachFailure(t *testing.T) {
	installClipboard(t, `return { read: () => Promise.resolve([{ types: ["text/plain"] }]) }`)
	if _, err := readImage(t); !errors.Is(err, platform.ErrNoClipboardImage) {
		t.Fatalf("no image: error = %v", err)
	}
	installClipboard(t, `return { read: () => Promise.resolve([
		{ types: ["image/png"], getType: () => Promise.resolve(new Blob([new Uint8Array(12)])) }]) }`)
	if _, err := readImage(t); err == nil {
		t.Fatal("bytes that are not a PNG were accepted")
	}
	installClipboard(t, `return { read: () => Promise.reject(Object.assign(new Error("no"), { name: "NotAllowedError" })) }`)
	if _, err := readImage(t); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("denied: error = %v", err)
	}
	installClipboard(t, `return {}`)
	if _, err := readImage(t); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("no API: error = %v", err)
	}
}
