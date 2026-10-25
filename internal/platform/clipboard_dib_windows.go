//go:build windows

// DIB-to-PNG conversion lives on Windows because only the Windows clipboard
// offers a packed DIB. Keeping it behind a build tag means macOS and Linux
// release builds do not link image/png, compress/zlib and compress/flate for a
// feature they cannot use.
package platform

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
)

const maxDIBPixels = 64 << 20

// DIB compression and header sizes (BITMAPINFOHEADER and its successors).
const (
	dibRGB            = 0
	dibRLE8           = 1
	dibRLE4           = 2
	dibBitfields      = 3
	dibAlphaBitfields = 6

	dibInfoHeader = 40
	dibV4Header   = 108
)

type channelMask struct {
	mask  uint32
	shift uint
	max   uint32 // the largest value the channel holds
}

func newChannelMask(mask uint32) channelMask {
	if mask == 0 {
		return channelMask{}
	}
	shift := uint(0)
	for mask>>shift&1 == 0 {
		shift++
	}
	return channelMask{mask: mask, shift: shift, max: mask >> shift}
}

// get returns the channel of pixel scaled to 0..255.
func (channel channelMask) get(pixel uint32) uint8 {
	value := (pixel & channel.mask) >> channel.shift
	if channel.max == 255 {
		return uint8(value)
	}
	return uint8((value*255 + channel.max/2) / channel.max)
}

// DIBToPNG converts a packed device-independent bitmap, as the Windows
// clipboard holds it in CF_DIB and CF_DIBV5, to PNG bytes. 1, 4 and 8 bit
// palette images and 16, 24 and 32 bit images are read, with BI_RGB or
// BI_BITFIELDS masks; an alpha mask (from a V4 or V5 header or
// BI_ALPHABITFIELDS) gives straight alpha, but a mask whose every pixel is zero
// reads as opaque, since many producers leave the byte unused. Rows may be
// bottom-up (positive height) or top-down. RLE and embedded JPEG/PNG
// bitmaps are refused.
func DIBToPNG(dib []byte) ([]byte, error) {
	if len(dib) > MaxClipboardRawBytes {
		return nil, ErrClipboardImageTooLarge
	}
	if len(dib) < dibInfoHeader {
		return nil, fmt.Errorf("clipboard bitmap is truncated")
	}
	headerSize := int(binary.LittleEndian.Uint32(dib))
	switch headerSize {
	case dibInfoHeader, 52, 56, dibV4Header, 124:
	default:
		return nil, fmt.Errorf("clipboard bitmap header size %d is unsupported", headerSize)
	}
	if len(dib) < headerSize {
		return nil, fmt.Errorf("clipboard bitmap is truncated")
	}
	width := int(int32(binary.LittleEndian.Uint32(dib[4:])))
	rawHeight := int(int32(binary.LittleEndian.Uint32(dib[8:])))
	planes := binary.LittleEndian.Uint16(dib[12:])
	bits := int(binary.LittleEndian.Uint16(dib[14:]))
	compression := binary.LittleEndian.Uint32(dib[16:])
	colors := int(binary.LittleEndian.Uint32(dib[32:]))
	topDown := rawHeight < 0
	height := rawHeight
	if topDown {
		height = -rawHeight
	}
	if planes != 1 || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("clipboard bitmap has invalid dimensions")
	}
	if width > maxDIBPixels || height > maxDIBPixels || width*height > maxDIBPixels {
		return nil, ErrClipboardImageTooLarge
	}
	switch bits {
	case 1, 4, 8, 16, 24, 32:
	default:
		return nil, fmt.Errorf("clipboard bitmap uses %d bits per pixel", bits)
	}

	offset := headerSize
	var masks [4]uint32 // red, green, blue, alpha
	switch compression {
	case dibRGB:
		switch bits {
		case 16:
			masks = [4]uint32{0x7c00, 0x03e0, 0x001f, 0}
		case 32:
			masks = [4]uint32{0x00ff0000, 0x0000ff00, 0x000000ff, 0}
		}
	case dibBitfields, dibAlphaBitfields:
		if bits != 16 && bits != 32 {
			return nil, fmt.Errorf("clipboard bitmap masks need 16 or 32 bits per pixel")
		}
		switch {
		case headerSize >= 52: // the masks are part of the header
			masks[0] = binary.LittleEndian.Uint32(dib[40:])
			masks[1] = binary.LittleEndian.Uint32(dib[44:])
			masks[2] = binary.LittleEndian.Uint32(dib[48:])
			if headerSize >= 56 {
				masks[3] = binary.LittleEndian.Uint32(dib[52:])
			}
		default: // they follow the header
			count := 3
			if compression == dibAlphaBitfields {
				count = 4
			}
			if len(dib) < offset+count*4 {
				return nil, fmt.Errorf("clipboard bitmap is truncated")
			}
			for index := range count {
				masks[index] = binary.LittleEndian.Uint32(dib[offset+index*4:])
			}
			offset += count * 4
		}
	case dibRLE8, dibRLE4:
		return nil, fmt.Errorf("clipboard bitmap is run-length encoded")
	default:
		return nil, fmt.Errorf("clipboard bitmap compression %d is unsupported", compression)
	}

	var palette [][4]uint8
	if bits <= 8 {
		count := 1 << bits
		if colors > 0 && colors < count {
			count = colors
		}
		if len(dib) < offset+count*4 {
			return nil, fmt.Errorf("clipboard bitmap is truncated")
		}
		palette = make([][4]uint8, 1<<bits)
		for index := range count {
			entry := dib[offset+index*4:]
			palette[index] = [4]uint8{entry[2], entry[1], entry[0], 255}
		}
		offset += count * 4
	} else if colors > 0 && colors <= 256 {
		offset += colors * 4 // an optimisation palette the pixels do not use
	}

	stride := (width*bits + 31) / 32 * 4
	if len(dib)-offset < stride*height {
		return nil, fmt.Errorf("clipboard bitmap pixels are truncated")
	}
	// Some writers (the .NET clipboard) put a second copy of the masks between
	// the header and the pixels. A recorded image size that matches the
	// geometry says the pixels are the last bytes, wherever they begin.
	if sizeImage := int(binary.LittleEndian.Uint32(dib[20:])); sizeImage == stride*height && len(dib)-offset > sizeImage {
		offset = len(dib) - sizeImage
	}
	pixels := dib[offset : offset+stride*height]

	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	var red, green, blue, alpha channelMask
	if bits >= 16 {
		red, green, blue, alpha = newChannelMask(masks[0]), newChannelMask(masks[1]), newChannelMask(masks[2]), newChannelMask(masks[3])
	}
	// bytes32 is the common case: 8-bit channels in a 32-bit word.
	bytes32 := bits == 32 && red.max == 255 && green.max == 255 && blue.max == 255 && (alpha.max == 255 || alpha.mask == 0)
	opaque := alpha.mask == 0
	anyAlpha := false
	for row := range height {
		source := pixels[row*stride : (row+1)*stride]
		line := row
		if !topDown {
			line = height - 1 - row
		}
		out := picture.Pix[line*picture.Stride : line*picture.Stride+width*4]
		switch {
		case bits == 32 && bytes32:
			for x := range width {
				pixel := binary.LittleEndian.Uint32(source[x*4:])
				out[x*4] = uint8(pixel >> red.shift)
				out[x*4+1] = uint8(pixel >> green.shift)
				out[x*4+2] = uint8(pixel >> blue.shift)
				if opaque {
					out[x*4+3] = 255
				} else {
					a := uint8(pixel >> alpha.shift)
					out[x*4+3] = a
					anyAlpha = anyAlpha || a != 0
				}
			}
		case bits == 32 || bits == 16:
			for x := range width {
				var pixel uint32
				if bits == 32 {
					pixel = binary.LittleEndian.Uint32(source[x*4:])
				} else {
					pixel = uint32(binary.LittleEndian.Uint16(source[x*2:]))
				}
				out[x*4], out[x*4+1], out[x*4+2] = red.get(pixel), green.get(pixel), blue.get(pixel)
				if opaque {
					out[x*4+3] = 255
				} else {
					a := alpha.get(pixel)
					out[x*4+3] = a
					anyAlpha = anyAlpha || a != 0
				}
			}
		case bits == 24:
			for x := range width {
				out[x*4], out[x*4+1], out[x*4+2], out[x*4+3] = source[x*3+2], source[x*3+1], source[x*3], 255
			}
		default: // palette
			for x := range width {
				var index uint8
				switch bits {
				case 8:
					index = source[x]
				case 4:
					index = source[x/2] >> (4 * (1 - uint(x)&1)) & 0x0f
				default:
					index = source[x/8] >> (7 - uint(x)&7) & 1
				}
				entry := palette[index]
				out[x*4], out[x*4+1], out[x*4+2], out[x*4+3] = entry[0], entry[1], entry[2], 255
			}
		}
	}
	if !opaque && !anyAlpha {
		for index := 3; index < len(picture.Pix); index += 4 {
			picture.Pix[index] = 255
		}
	}
	return EncodePNG(picture)
}

// EncodePNG encodes picture for the clipboard at the fastest compression level
// and applies the size cap.
func EncodePNG(picture image.Image) ([]byte, error) {
	var out bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := encoder.Encode(&out, picture); err != nil {
		return nil, err
	}
	if out.Len() > MaxClipboardImageBytes {
		return nil, ErrClipboardImageTooLarge
	}
	return out.Bytes(), nil
}
