---
title: Test without a window
summary: Drive an application from a test, script it over a line protocol, and check that the platforms still agree.
order: 13
group: distribute
---

Verifying a desktop application usually means opening a window, which interrupts you, depends on which display it lands on and cannot run unattended. Viva gives you three layers that avoid that, each usable without the next.

## app.Driver

`app.NewDriver` runs your application's real painter, input router and focus rules from explicit calls: `Move`, `Click`, `Key`, `Type`, `Shortcut`, `Scroll`, `Resize` and `Advance`. It has no platform code, and its animation clock moves only when you tell it to. `Snapshot` returns what was painted: primitives, labels, hit regions and focus. This test drives the program from [Get started](/en/docs/getting-started/):

<!-- example: hello/main_test.go -->
```go
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
```

Background goroutines reach a driver only through `Config.Invalidator`, exactly as they reach a window, and `Pump` applies the wake. `examples/guides/background/main_test.go` shows a test that waits for a worker goroutine's result this way. The driver models the text system's editing commands and the Command-A, C, X and V shortcuts. It does not model input-method composition.

## The line protocol

Import `github.com/vivaui/viva/headless/auto` from a file guarded by a build tag, so release builds never contain it:

```go
//go:build vivahead

package main

import _ "github.com/vivaui/viva/headless/auto"
```

Build with `-tags vivahead` and start the program with `VIVA_HEADLESS=1`. `RunCanvas` then reads commands from standard input instead of opening a window. Controls are addressed by label rather than by coordinates:

```text
click "New Host"
type example.com
key tab
waitfor "Connected" 5000
texts
shot /tmp/frame.png
quit
```

Every request gets zero or more data lines and one final `ok` or `error` line, so a script, another program or a coding agent can drive and inspect an application without touching the screen. The full command list is in the doc comment of `headless/control.go`.

`layout` reads the frame for geometry that is almost surely a mistake and prints one line each: text cut off by a clip, text that runs out of its button or field, one line drawn over another on the same layer, text that leaves the window, and focusable controls that cross without one containing the other. It never fails, so run it after every change to a screen; a few lines of text are quicker to read than a picture. It judges text by the widths the session lays out with, the platform's text engine as in a window, so a line is reported when it is cut or crosses another on that platform.

**Screenshots exist on macOS only.** There the offscreen renderer uses the window's own Metal pipelines, so a capture is the window's pixels by construction. On Windows and Linux the protocol operates and answers queries, but a screenshot returns `ErrNoRenderer`.

## Platform conformance

`cmd/viva-conformance` runs the same lifecycle scenarios through the public `app` package on each platform and prints PASS, FAIL or SKIP for each: closing from another goroutine, an idle window doing no work, `Invalidate` waking exactly once, hidden windows, panels, status items, and `Quit` ending the loop whatever the windows say. Each check exists because a real platform once got it wrong. It opens real windows, so it needs a desktop session. What it cannot check, and how that is done by hand, is listed in the repository's `docs/conformance.md`.
