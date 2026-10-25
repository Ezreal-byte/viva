//go:build darwin || windows || linux

package headless

import (
	"errors"
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
)

func TestOpenDirectoryControlQueuesPathAndCancel(t *testing.T) {
	session, err := Open(app.Config{}, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	reply, _ := execute(session, `opendirectory "/tmp/project folder"`)
	if reply != "ok\n" {
		t.Fatalf("path reply = %q", reply)
	}
	path, err := app.OpenDirectory(app.OpenDirectoryOptions{Title: "Project"})
	if err != nil || path != "/tmp/project folder" {
		t.Fatalf("OpenDirectory = %q, %v", path, err)
	}

	reply, _ = execute(session, "opendirectory cancel")
	if reply != "ok\n" {
		t.Fatalf("cancel reply = %q", reply)
	}
	if _, err := app.OpenDirectory(app.OpenDirectoryOptions{}); !errors.Is(err, app.ErrCanceled) {
		t.Fatalf("canceled OpenDirectory error = %v", err)
	}

	reply, _ = execute(session, "opendirectory")
	if !strings.HasPrefix(reply, "error opendirectory needs") {
		t.Fatalf("missing argument reply = %q", reply)
	}
}
