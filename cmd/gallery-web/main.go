//go:build js && wasm

package main

import (
	"github.com/vivaui/viva/internal/gallery"
	"github.com/vivaui/viva/internal/platform/web"
)

func main() {
	showcase := gallery.New()
	showcase.EnableTextEditing()
	_ = web.Run(web.Config{
		Title:     "Viva · component gallery",
		Width:     gallery.DefaultWidth,
		Height:    gallery.DefaultHeight,
		Paint:     showcase.Paint,
		OnScroll:  showcase.Scroll,
		OnPointer: showcase.Pointer,
		OnKey:     showcase.Key,
		TextInput: web.EditorInputs(showcase.TextEditor),
	})
}
