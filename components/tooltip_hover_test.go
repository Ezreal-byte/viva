package components

import (
	"testing"
	"time"

	"github.com/vivaui/viva/ui"
)

// The delay behavior is tested independently of the shipped default, which the
// pinned shadcn provider sets to zero.
const testTooltipDelay = 500 * time.Millisecond

func hoverFrame(now time.Duration, hovered, pressed uint32, hint string, viewport ui.Rect, hover *HoverTooltip) (*ui.Canvas, bool, time.Duration, bool) {
	return hoverFrameAt(10, now, hovered, pressed, hint, viewport, hover)
}

// hoverFrameAt is hoverFrame with the hovered control y pixels from the top.
func hoverFrameAt(y float32, now time.Duration, hovered, pressed uint32, hint string, viewport ui.Rect, hover *HoverTooltip) (*ui.Canvas, bool, time.Duration, bool) {
	var canvas ui.Canvas
	canvas.SetClock(func() time.Duration { return now })
	canvas.BeginFrame()
	canvas.HitWithOptions(ui.Item(7), ui.Rect{X: 40, Y: y, Width: 28, Height: 28}, "Refresh", ui.HitOptions{Hint: hint})
	canvas.SetInteraction(ui.Item(int(hovered)), ui.Item(int(pressed)), ui.ID{}, false)
	painted := hover.Paint(&canvas, ui.DarkTheme(), viewport, TooltipOptions{}, testTooltipDelay)
	canvas.EndFrame()
	delay, wake := canvas.NextFrame()
	return &canvas, painted, delay, wake
}

func tooltipText(canvas *ui.Canvas, value string) (ui.Text, bool) {
	for _, text := range canvas.Texts() {
		if text.Value == value {
			return text, true
		}
	}
	return ui.Text{}, false
}

func TestHoverTooltipWaitsForTheDelayThenShowsTheHint(t *testing.T) {
	var hover HoverTooltip
	viewport := ui.Rect{Width: 300, Height: 200}

	// Just arrived: nothing yet, and exactly one wake-up is requested for the
	// moment the tooltip is due.
	canvas, painted, delay, wake := hoverFrame(0, 7, 0, "Refresh the list", viewport, &hover)
	if painted || len(canvas.Texts()) != 0 || !wake || delay != testTooltipDelay {
		t.Fatalf("on arrival: painted %v texts %d wake %v after %v", painted, len(canvas.Texts()), wake, delay)
	}
	canvas, painted, delay, wake = hoverFrame(200*time.Millisecond, 7, 0, "Refresh the list", viewport, &hover)
	if painted || !wake || delay != testTooltipDelay-200*time.Millisecond {
		t.Fatalf("partway: painted %v wake %v after %v", painted, wake, delay)
	}

	// Due: the hint appears below the control, and its entrance asks for frames
	// at display cadence until it is in.
	canvas, painted, delay, wake = hoverFrame(testTooltipDelay, 7, 0, "Refresh the list", viewport, &hover)
	text, ok := tooltipText(canvas, "Refresh the list")
	if !painted || !ok || !wake || delay != 0 {
		t.Fatalf("when due: painted %v shown %v wake %v after %v", painted, ok, wake, delay)
	}
	if text.Bounds.Y <= 38 {
		t.Fatalf("tooltip at y=%v, want it below the control (bottom 38)", text.Bounds.Y)
	}
}

func TestHoverTooltipStartsOverWhenThePointerMovesOnOrPresses(t *testing.T) {
	var hover HoverTooltip
	viewport := ui.Rect{Width: 300, Height: 200}
	hoverFrame(0, 7, 0, "Hint", viewport, &hover)
	if _, painted, _, _ := hoverFrame(testTooltipDelay, 7, 0, "Hint", viewport, &hover); !painted {
		t.Fatal("the tooltip did not appear before the press")
	}
	if _, painted, _, _ := hoverFrame(time.Second, 7, 7, "Hint", viewport, &hover); painted {
		t.Fatal("a tooltip stayed over a pressed button")
	}
	// A click hides the tooltip until the pointer leaves, even though the
	// pointer is still over the control after the release.
	if _, painted, _, wake := hoverFrame(time.Second, 7, 0, "Hint", viewport, &hover); painted || wake {
		t.Fatalf("after the press: painted %v wake %v, want it stay hidden", painted, wake)
	}
	if _, painted, _, wake := hoverFrame(time.Second, 0, 0, "Hint", viewport, &hover); painted || wake {
		t.Fatalf("with nothing hovered: painted %v wake %v", painted, wake)
	}
	if _, painted, _, wake := hoverFrame(2*time.Second, 7, 0, "Hint", viewport, &hover); painted || !wake {
		t.Fatalf("after leaving and returning the delay did not start: painted %v wake %v", painted, wake)
	}
}

func TestHoverTooltipNeedsAHintAndFlipsAboveWhenThereIsNoRoomBelow(t *testing.T) {
	var hover HoverTooltip
	hoverFrame(0, 7, 0, "", ui.Rect{Width: 300, Height: 200}, &hover)
	if canvas, painted, _, wake := hoverFrame(time.Second, 7, 0, "", ui.Rect{Width: 300, Height: 200}, &hover); painted || wake || len(canvas.Texts()) != 0 {
		t.Fatal("a hit without a hint showed a tooltip")
	}

	// A control near the bottom of a short window: no room below it, room above.
	short := ui.Rect{Width: 300, Height: 130}
	hoverFrameAt(90, 0, 7, 0, "Refresh", short, &hover)
	canvas, painted, _, _ := hoverFrameAt(90, time.Second, 7, 0, "Refresh", short, &hover)
	text, ok := tooltipText(canvas, "Refresh")
	if !painted || !ok || text.Bounds.Y+text.Bounds.Height > 90 {
		t.Fatalf("with no room below, the tooltip is at y=%v (shown %v); want it above the control at y=90", text.Bounds.Y, ok)
	}
}

// With ShowOnFocus, Tab focus shows the hint at once, without waiting for a
// pointer rest.
func TestHoverTooltipShowsOnKeyboardFocusWhenAsked(t *testing.T) {
	var hover HoverTooltip
	hover.ShowOnFocus()
	var canvas ui.Canvas
	canvas.SetInteraction(ui.ID{}, ui.ID{}, ui.Item(7), true)
	canvas.HitWithOptions(ui.Item(7), ui.Rect{X: 10, Y: 10, Width: 40, Height: 28}, "Refresh", ui.HitOptions{Role: ui.RoleButton, Hint: "Refresh the list"})
	if painted := hover.Paint(&canvas, ui.DarkTheme(), ui.Rect{Width: 300, Height: 200}, TooltipOptions{}, testTooltipDelay); !painted {
		t.Fatal("a focused control with a hint did not show its tooltip")
	}
}

// Escape (Dismiss) and a press anywhere hide the shown tooltip until the pointer
// leaves the control, as Base UI's tooltip dismiss does.
func TestHoverTooltipDismissesOnEscapeAndPress(t *testing.T) {
	viewport := ui.Rect{Width: 300, Height: 200}
	var hover HoverTooltip
	hoverFrame(0, 7, 0, "Hint", viewport, &hover)
	if _, painted, _, _ := hoverFrame(testTooltipDelay, 7, 0, "Hint", viewport, &hover); !painted {
		t.Fatal("the tooltip did not show")
	}
	if !hover.Dismiss() || !hover.Showing().IsZero() {
		t.Fatal("Dismiss did not hide the shown tooltip")
	}
	if _, painted, _, _ := hoverFrame(testTooltipDelay+time.Second, 7, 0, "Hint", viewport, &hover); painted {
		t.Fatal("Escape let the tooltip return while the pointer stayed")
	}
	hoverFrame(testTooltipDelay+time.Second, 0, 0, "Hint", viewport, &hover)
	if _, _, _, wake := hoverFrame(3*time.Second, 7, 0, "Hint", viewport, &hover); !wake {
		t.Fatal("returning to the control did not start a new tooltip delay")
	}

	var second HoverTooltip
	hoverFrame(0, 7, 0, "Hint", viewport, &second)
	if _, painted, _, _ := hoverFrame(testTooltipDelay, 7, 0, "Hint", viewport, &second); !painted {
		t.Fatal("the second tooltip did not show")
	}
	if _, painted, _, _ := hoverFrame(testTooltipDelay+time.Second, 0, 7, "Hint", viewport, &second); painted {
		t.Fatal("a press outside the control left the tooltip up")
	}
}
