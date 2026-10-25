package main

import (
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"
)

func hasLabel(snapshot app.Snapshot, want string) bool {
	for _, label := range snapshot.Labels {
		if strings.Contains(label.Value, want) {
			return true
		}
	}
	return false
}

// clickID presses the middle of the control with the given identity.
func clickID(t *testing.T, driver *app.Driver, id ui.ID) {
	t.Helper()
	for _, hit := range driver.Snapshot().Hits {
		if hit.ID == id {
			b := hit.Bounds
			if err := driver.Click(b.X+b.Width/2, b.Y+b.Height/2); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatalf("no control %v on screen", id)
}

func TestClickCounts(t *testing.T) {
	var canvas ui.Canvas
	driver, err := app.NewDriver(newConfig(&model{}), &canvas)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()

	if !hasLabel(driver.Snapshot(), "Clicked 0 times") {
		t.Fatal("the button should start at zero")
	}
	clickID(t, driver, click)
	if !hasLabel(driver.Snapshot(), "Clicked 1 times") {
		t.Fatal("the click did not reach the button")
	}
}
