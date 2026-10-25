// Package tarball writes the reproducible archives viva package makes for Linux:
// the .tar.gz with an install script (Linux), and the tar and gzip layers the
// Debian package builder (tools/debpkg) shares.
package tarball

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// defaultDate is the timestamp of every entry when SOURCE_DATE_EPOCH is not
// set: 1980-01-01, the earliest time a zip entry can have, which viva package
// also uses for its zip archives.
var defaultDate = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// SourceDate is the modification time given to every archive entry:
// SOURCE_DATE_EPOCH (seconds since 1970) when the environment sets it, the
// reproducible-builds convention, else 1980-01-01.
func SourceDate() time.Time {
	if text := os.Getenv("SOURCE_DATE_EPOCH"); text != "" {
		if seconds, err := strconv.ParseInt(text, 10, 64); err == nil && seconds >= 0 {
			return time.Unix(seconds, 0).UTC()
		}
	}
	return defaultDate
}

// File is one regular file of an archive.
type File struct {
	// Path is slash-separated and relative, such as usr/bin/app.
	Path string
	Data []byte
	// Mode is the permission bits: 0o755 for a program, 0o644 otherwise.
	Mode int64
}

// WriteTar writes files as an uncompressed tar archive. Every file and every
// directory above one is owned by root (uid and gid 0), has the given
// modification time and appears once, in name order, directories before what is
// inside them, so the same input always gives the same bytes. root, when not
// empty, is a directory every name is put under, with an entry of its own:
// "." gives the ./usr/bin/app names of a Debian package, a name such as
// app-1.0-linux-amd64 the single top-level directory of a release archive.
func WriteTar(output io.Writer, files []File, root string, modified time.Time) error {
	seen := map[string]bool{}
	directories := map[string]bool{}
	for _, file := range files {
		if err := checkPath(file.Path); err != nil {
			return err
		}
		if seen[file.Path] {
			return fmt.Errorf("tarball: %s appears twice", file.Path)
		}
		seen[file.Path] = true
		for dir := path.Dir(file.Path); dir != "."; dir = path.Dir(dir) {
			directories[dir] = true
		}
	}
	for dir := range directories {
		if seen[dir] {
			return fmt.Errorf("tarball: %s is both a file and a directory", dir)
		}
	}

	type entry struct {
		name string
		file *File
	}
	var entries []entry
	for dir := range directories {
		entries = append(entries, entry{name: dir})
	}
	for index := range files {
		entries = append(entries, entry{name: files[index].Path, file: &files[index]})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	writer := tar.NewWriter(output)
	header := func(name string, mode int64, size int64, kind byte) *tar.Header {
		return &tar.Header{
			Typeflag: kind, Name: name, Mode: mode, Size: size, Uid: 0, Gid: 0, Uname: "root", Gname: "root",
			ModTime: modified, Format: tar.FormatGNU,
		}
	}
	if root != "" {
		if err := writer.WriteHeader(header(root+"/", 0o755, 0, tar.TypeDir)); err != nil {
			return err
		}
	}
	prefix := ""
	if root != "" {
		prefix = root + "/"
	}
	for _, item := range entries {
		if item.file == nil {
			if err := writer.WriteHeader(header(prefix+item.name+"/", 0o755, 0, tar.TypeDir)); err != nil {
				return err
			}
			continue
		}
		if err := writer.WriteHeader(header(prefix+item.name, item.file.Mode, int64(len(item.file.Data)), tar.TypeReg)); err != nil {
			return err
		}
		if _, err := writer.Write(item.file.Data); err != nil {
			return err
		}
	}
	return writer.Close()
}

func checkPath(name string) error {
	if name == "" || strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") || path.Clean(name) != name || strings.HasPrefix(name, "../") {
		return fmt.Errorf("tarball: %q is not a relative, clean path", name)
	}
	return nil
}

// Gzip compresses data at the best level with an empty header (no name, no
// time), so the output depends on the data alone (and on the Go release's
// deflate implementation).
func Gzip(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buffer, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
