//go:build darwin || windows || linux

package headless

import (
	"errors"
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
)

func TestSaveFileControlQueuesPathAndCancel(t *testing.T) {
	session, err := Open(app.Config{}, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	reply, _ := execute(session, `savefile "/tmp/report final.txt"`)
	if reply != "ok\n" {
		t.Fatalf("path reply = %q", reply)
	}
	path, err := app.SaveFile(app.SaveFileOptions{SuggestedName: "report.txt"})
	if err != nil || path != "/tmp/report final.txt" {
		t.Fatalf("SaveFile = %q, %v", path, err)
	}

	reply, _ = execute(session, "savefile cancel")
	if reply != "ok\n" {
		t.Fatalf("cancel reply = %q", reply)
	}
	if _, err := app.SaveFile(app.SaveFileOptions{}); !errors.Is(err, app.ErrCanceled) {
		t.Fatalf("canceled SaveFile error = %v", err)
	}

	reply, _ = execute(session, "savefile")
	if !strings.HasPrefix(reply, "error savefile needs") {
		t.Fatalf("missing argument reply = %q", reply)
	}
}
