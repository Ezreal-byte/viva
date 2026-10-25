//go:build darwin || windows || linux

package headless

import (
	"testing"
	"time"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"
)

// A dialog's entrance plays on the animation clock; Rest plays it out, so a
// screenshot after an input shows the interface at rest, and Wait steps it.
func TestRestPlaysTransitionsToTheirEnd(t *testing.T) {
	var canvas ui.Canvas
	session, err := Open(app.Config{
		Title: "rest", Width: 200, Height: 100,
		Paint: func(c *ui.Canvas, size ui.Size) {
			c.BeginPresence(ui.Item(1), 0, ui.Presence{Motion: ui.Motion{Duration: 100 * time.Millisecond}, Fade: true}, ui.Point{})
			c.Fill(ui.Rect{Width: 50, Height: 50}, ui.Color{Alpha: 1})
			c.EndPresence()
		},
	}, &canvas, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	alpha := func() float32 { return canvas.Rectangles()[0].Color.Alpha }
	if alpha() != 0 || !session.Driver().Transitioning() {
		t.Fatalf("opening frame: alpha %v transitioning %v", alpha(), session.Driver().Transitioning())
	}
	if err := session.Wait(50 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if a := alpha(); a <= 0 || a >= 1 {
		t.Fatalf("halfway alpha = %v", a)
	}
	if err := session.Rest(); err != nil {
		t.Fatal(err)
	}
	if alpha() != 1 || session.Driver().Transitioning() {
		t.Fatalf("at rest: alpha %v transitioning %v", alpha(), session.Driver().Transitioning())
	}
}
