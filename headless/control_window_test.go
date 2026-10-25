//go:build darwin || windows || linux

package headless

import (
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"
)

// TestControlWindowMoveMinimizeAndTitle drives the window state the way a
// window manager would, without opening a window.
func TestControlWindowMoveMinimizeAndTitle(t *testing.T) {
	var frames []ui.Rect
	var minimizes []bool
	origin := ui.Point{X: 10, Y: 20}
	config := app.Config{
		Title: "main", Width: 300, Height: 200, Origin: &origin,
		OnFrameChange: func(frame ui.Rect) { frames = append(frames, frame) },
		OnMinimize:    func(minimized bool) { minimizes = append(minimizes, minimized) },
	}
	session, err := Open(config, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	reply := func(line string) string { r, _ := execute(session, line); return r }

	if got := reply("windowmove 60 70"); !strings.HasPrefix(got, "ok") {
		t.Fatalf("windowmove: %s", got)
	}
	if len(frames) != 1 || frames[0] != (ui.Rect{X: 60, Y: 70, Width: 300, Height: 200}) {
		t.Fatalf("frames after move = %+v", frames)
	}
	if got := reply("minimize on"); !strings.HasPrefix(got, "ok") {
		t.Fatalf("minimize on: %s", got)
	}
	if got := reply("minimize off"); !strings.HasPrefix(got, "ok") {
		t.Fatalf("minimize off: %s", got)
	}
	if len(minimizes) != 2 || !minimizes[0] || minimizes[1] {
		t.Fatalf("minimize changes = %v", minimizes)
	}
	if got := reply("minimize sideways"); !strings.HasPrefix(got, "error ") {
		t.Fatalf("minimize sideways = %s", got)
	}
}
