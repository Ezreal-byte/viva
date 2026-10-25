package platform

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Errors every platform's clipboard image reader returns; app re-exports them.
var (
	// ErrNoClipboardImage reports that the clipboard holds no image.
	ErrNoClipboardImage = errors.New("clipboard holds no image")
	// ErrClipboardImageTooLarge reports an image above MaxClipboardImageBytes
	// once encoded as PNG, or one whose pixels would not fit in memory.
	ErrClipboardImageTooLarge = errors.New("clipboard image is too large")
)

const (
	// MaxClipboardImageBytes is the largest PNG a clipboard read returns.
	MaxClipboardImageBytes = 32 << 20
	// MaxClipboardRawBytes bounds an uncompressed bitmap (a DIB or TIFF) that is
	// transcoded to PNG: 128 MiB holds a 5K screenshot at 32 bits per pixel.
	MaxClipboardRawBytes = 128 << 20
)

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// CheckClipboardPNG validates data a clipboard owner offered as PNG: it must be
// within the size cap and start with the PNG signature. The pixels are not
// decoded here.
func CheckClipboardPNG(data []byte) error {
	switch {
	case len(data) == 0:
		return ErrNoClipboardImage
	case len(data) > MaxClipboardImageBytes:
		return ErrClipboardImageTooLarge
	case !bytes.HasPrefix(data, pngSignature):
		return fmt.Errorf("clipboard image is not a PNG")
	}
	return nil
}

// TrimClipboardPNG returns data up to the end of its IEND chunk. A clipboard
// block is often larger than what the owner wrote, and the surplus is not
// part of the PNG.
func TrimClipboardPNG(data []byte) ([]byte, error) {
	if !bytes.HasPrefix(data, pngSignature) {
		return nil, fmt.Errorf("clipboard image is not a PNG")
	}
	offset := len(pngSignature)
	for offset+12 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[offset:]))
		end := offset + 12 + length
		if length < 0 || end < offset || end > len(data) {
			break
		}
		if string(data[offset+4:offset+8]) == "IEND" {
			return data[:end], nil
		}
		offset = end
	}
	return nil, fmt.Errorf("clipboard PNG is truncated")
}
