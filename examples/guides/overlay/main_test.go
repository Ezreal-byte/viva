package main

import (
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/ui"
)

func TestOverlays(t *testing.T) {
	m := &model{}
	var canvas ui.Canvas
	driver, err := app.NewDriver(newConfig(m), &canvas)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()

	clickHit(t, driver, trigger)
	if !m.menuOpen {
		t.Fatal("menu did not open")
	}
	if id, ok := driver.HitTest(500, 300); !ok || id != menuDismiss {
		t.Fatalf("outside press lands on %v, %v", id, ok)
	}
	if err := driver.Click(500, 300); err != nil {
		t.Fatal(err)
	}
	if m.menuOpen {
		t.Fatal("an outside press must close the menu")
	}

	clickHit(t, driver, trigger)
	if err := driver.Key(ui.KeyEscape, 0); err != nil {
		t.Fatal(err)
	}
	if m.menuOpen {
		t.Fatal("Escape must close the menu")
	}

	// Delete… opens the dialog; confirming closes it.
	clickHit(t, driver, trigger)
	clickHit(t, driver, menuDelete)
	if !m.dialogOpen || m.menuOpen {
		t.Fatalf("dialog=%v menu=%v", m.dialogOpen, m.menuOpen)
	}
	clickHit(t, driver, dialogConfirm)
	if m.dialogOpen || m.status != "Deleted" {
		t.Fatalf("dialog=%v status=%q", m.dialogOpen, m.status)
	}
}

func clickHit(t *testing.T, driver *app.Driver, id ui.ID) {
	t.Helper()
	for _, hit := range driver.Snapshot().Hits {
		if hit.ID == id {
			if err := driver.Click(hit.Bounds.X+hit.Bounds.Width/2, hit.Bounds.Y+hit.Bounds.Height/2); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatalf("no hit %v", id)
}
