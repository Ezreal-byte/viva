//go:build js && wasm

package app

import webplatform "github.com/vivaui/viva/internal/platform/web"

func nativeRequestClipboardImage(done func(png []byte, err error)) {
	webplatform.ReadClipboardImagePNG(func(png []byte, err error) {
		done(png, webSystemError("read clipboard image", err))
	})
}
