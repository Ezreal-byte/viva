package winpkg

import (
	"archive/zip"
	"io"
	"os"
	"time"
)

// zipTime is the timestamp of the entry, fixed so the same program always
// makes the same archive.
var zipTime = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// ZipProgram writes an archive holding only the program, under name. The
// timestamp, mode and compression are fixed, so the bytes depend only on the
// program and the name.
func ZipProgram(program, name, archive string) error {
	source, err := os.Open(program)
	if err != nil {
		return err
	}
	defer source.Close()
	out, err := os.Create(archive)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(out)
	header := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: zipTime}
	header.SetMode(0o755)
	entry, err := writer.CreateHeader(header)
	if err == nil {
		_, err = io.Copy(entry, source)
	}
	if err == nil {
		err = writer.Close()
	}
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(archive)
	}
	return err
}
