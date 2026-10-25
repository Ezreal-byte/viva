//go:build !(js && wasm)

package app

func nativeRequestClipboardImage(done func(png []byte, err error)) {
	png, err := ClipboardImagePNG()
	done(png, err)
}
