//go:build darwin || windows || linux

package headless

import (
	"errors"
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
)

func TestOpenFilesControlQueuesPathsAndCancel(t *testing.T) {
	session, err := Open(app.Config{}, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	reply, _ := execute(session, `openfiles "/tmp/id one" /tmp/id_two`)
	if reply != "ok\n" {
		t.Fatalf("paths reply = %q", reply)
	}
	paths, err := app.OpenFiles(app.OpenFileOptions{Title: "Keys", AllowMultiple: true})
	if err != nil || len(paths) != 2 || paths[0] != "/tmp/id one" || paths[1] != "/tmp/id_two" {
		t.Fatalf("OpenFiles = %v, %v", paths, err)
	}

	reply, _ = execute(session, "openfiles cancel")
	if reply != "ok\n" {
		t.Fatalf("cancel reply = %q", reply)
	}
	if _, err := app.OpenFiles(app.OpenFileOptions{}); !errors.Is(err, app.ErrCanceled) {
		t.Fatalf("canceled OpenFiles error = %v", err)
	}

	reply, _ = execute(session, "openfiles")
	if !strings.HasPrefix(reply, "error openfiles needs") {
		t.Fatalf("missing argument reply = %q", reply)
	}
}
