// Package mimeinfo writes the freedesktop.org shared-mime-info package file
// that teaches a Linux desktop a document type: which file names are of which
// media type. A .desktop entry names media types in MimeType, and the desktop
// only offers the application for a file when it can tell the file's type.
package mimeinfo

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

// Type is one media type and the file names that carry it.
type Type struct {
	// MIME is the media type, such as text/markdown.
	MIME string
	// Name is the human-readable description.
	Name string
	// Extensions are file name extensions without the dot.
	Extensions []string
}

// Types returns the media types, in order.
func Types(types []Type) []string {
	result := make([]string, len(types))
	for index, kind := range types {
		result[index] = kind.MIME
	}
	return result
}

// Package renders the file installed as /usr/share/mime/packages/<id>.xml. The
// same types give the same bytes.
func Package(types []Type) ([]byte, error) {
	if len(types) == 0 {
		return nil, errors.New("mimeinfo: there is no type to describe")
	}
	var buffer bytes.Buffer
	buffer.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buffer.WriteString(`<mime-info xmlns="http://www.freedesktop.org/standards/shared-mime-info">` + "\n")
	escape := func(text string) (string, error) {
		if strings.IndexFunc(text, func(r rune) bool { return r < ' ' || r == 0x7f }) >= 0 {
			return "", fmt.Errorf("mimeinfo: %q contains a control character", text)
		}
		var escaped bytes.Buffer
		if err := xml.EscapeText(&escaped, []byte(text)); err != nil {
			return "", err
		}
		return escaped.String(), nil
	}
	for _, kind := range types {
		if kind.MIME == "" || !strings.Contains(kind.MIME, "/") || len(kind.Extensions) == 0 {
			return nil, fmt.Errorf("mimeinfo: the type %q needs a media type and an extension", kind.MIME)
		}
		mime, err := escape(kind.MIME)
		if err != nil {
			return nil, err
		}
		name, err := escape(kind.Name)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&buffer, "\t<mime-type type=%q>\n", mime)
		if name != "" {
			fmt.Fprintf(&buffer, "\t\t<comment>%s</comment>\n", name)
		}
		for _, extension := range kind.Extensions {
			pattern, err := escape(extension)
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&buffer, "\t\t<glob pattern=%q/>\n", "*."+pattern)
		}
		buffer.WriteString("\t</mime-type>\n")
	}
	buffer.WriteString("</mime-info>\n")
	return buffer.Bytes(), nil
}
