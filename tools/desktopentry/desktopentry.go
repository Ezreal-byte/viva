// Package desktopentry writes freedesktop.org Desktop Entry files (.desktop),
// the menu entries of Linux desktops, following the Desktop Entry
// Specification's escaping rules.
package desktopentry

import (
	"errors"
	"fmt"
	"strings"
)

// Entry is an application's menu entry.
type Entry struct {
	Name    string
	Comment string
	// Exec is the command line as the spec defines it, already quoted: use
	// QuoteExec for a program path.
	Exec string
	// Icon is an icon theme name or an absolute path; empty leaves it out.
	Icon       string
	Categories []string
	Keywords   []string
	// StartupWMClass is the window class the desktop matches windows to the
	// entry with; empty leaves it out.
	StartupWMClass string
	// URLSchemes are the link schemes the application opens (ssh, for ssh://...).
	// They add the %u field code to Exec, so the desktop passes the link as an
	// argument, and the matching MimeType entries.
	URLSchemes []string
	// MimeTypes are the media types of the documents the application opens.
	// They add the %F field code to Exec, so the desktop passes the files as
	// arguments, and the MimeType entries.
	MimeTypes []string
}

// ExecField is the field code that ends an Exec line for an application that
// opens documents, links or both: the desktop replaces it with the file paths
// (%F), the link (%u) or, for both, a list of locations (%U, where a file is a
// file:// address). It is empty for an application that opens neither.
func ExecField(documents, links bool) string {
	switch {
	case documents && links:
		return "%U"
	case documents:
		return "%F"
	case links:
		return "%u"
	}
	return ""
}

// Bytes is the file, ending in a newline. Values are escaped, so any text is
// safe except control characters other than newline, tab and carriage return,
// which are an error.
func (entry Entry) Bytes() ([]byte, error) {
	if entry.Name == "" || entry.Exec == "" {
		return nil, errors.New("desktopentry: a menu entry needs a name and a command")
	}
	var builder strings.Builder
	builder.WriteString("[Desktop Entry]\nType=Application\nVersion=1.0\n")
	var failure error
	write := func(key, value string) {
		if value == "" {
			return
		}
		if index := strings.IndexFunc(value, func(r rune) bool {
			return (r < ' ' && r != '\n' && r != '\t' && r != '\r') || r == 0x7f
		}); index >= 0 && failure == nil {
			failure = fmt.Errorf("desktopentry: %s contains a control character", key)
		}
		builder.WriteString(key + "=" + value + "\n")
	}
	write("Name", Escape(entry.Name))
	write("Comment", Escape(entry.Comment))
	exec := entry.Exec
	if field := ExecField(len(entry.MimeTypes) > 0, len(entry.URLSchemes) > 0); field != "" {
		exec += " " + field
	}
	write("Exec", Escape(exec))
	write("Icon", Escape(entry.Icon))
	builder.WriteString("Terminal=false\n")
	write("Categories", EscapeList(entry.Categories))
	write("Keywords", EscapeList(entry.Keywords))
	write("StartupWMClass", Escape(entry.StartupWMClass))
	if types := append(append([]string(nil), entry.MimeTypes...), schemeHandlers(entry.URLSchemes)...); len(types) > 0 {
		write("MimeType", EscapeList(types))
	}
	if failure != nil {
		return nil, failure
	}
	return []byte(builder.String()), nil
}

func schemeHandlers(schemes []string) []string {
	types := make([]string, len(schemes))
	for index, scheme := range schemes {
		types[index] = "x-scheme-handler/" + scheme
	}
	return types
}

// Escape encodes a string value: backslash, newline, tab and carriage return
// become \\, \n, \t and \r, and a leading space becomes \s.
func Escape(text string) string {
	var builder strings.Builder
	for index, r := range text {
		switch {
		case r == '\\':
			builder.WriteString(`\\`)
		case r == '\n':
			builder.WriteString(`\n`)
		case r == '\t':
			builder.WriteString(`\t`)
		case r == '\r':
			builder.WriteString(`\r`)
		case r == ' ' && index == 0:
			builder.WriteString(`\s`)
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// EscapeList encodes a list value: each item is escaped, a semicolon inside an
// item becomes \;, and every item, the last too, is followed by a semicolon.
// An empty list is empty.
func EscapeList(items []string) string {
	var builder strings.Builder
	for _, item := range items {
		builder.WriteString(strings.ReplaceAll(Escape(item), ";", `\;`))
		builder.WriteByte(';')
	}
	return builder.String()
}

// QuoteExec makes a program path safe to use as the command of an Exec key: a
// path with a space or a reserved character is put in double quotes and the
// characters that stay special inside them (" ` $ \) get a backslash. Escape
// then doubles those backslashes when the file is written, which the spec
// requires.
func QuoteExec(path string) string {
	if !strings.ContainsAny(path, " \t\n\"'\\><~|&;$*?#()`") {
		return path
	}
	var builder strings.Builder
	builder.WriteByte('"')
	for _, r := range path {
		if strings.ContainsRune("\"`$\\", r) {
			builder.WriteByte('\\')
		}
		builder.WriteRune(r)
	}
	builder.WriteByte('"')
	return builder.String()
}
