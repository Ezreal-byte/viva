package windows

import (
	"testing"

	"github.com/vivaui/viva/ui"
)

// A single click arrives as several messages; exactly one of them may count, or
// a panel that a click opens is closed again by the same click.
func TestStatusMessagesOfOneClickCountOnce(t *testing.T) {
	const (
		leftUp   = 0x0202
		rightUp  = 0x0205
		moveOver = 0x0200
	)
	leftClick := []uint32{leftUp, ninSelect}
	rightClick := []uint32{rightUp, wmContextMenu}
	count := func(messages []uint32) (primary, secondary int) {
		for _, message := range messages {
			if button, ok := statusButton(message); ok {
				if button == ui.PointerButtonPrimary {
					primary++
				} else {
					secondary++
				}
			}
		}
		return primary, secondary
	}
	if primary, secondary := count(leftClick); primary != 1 || secondary != 0 {
		t.Fatalf("a left click counted %d primary and %d secondary", primary, secondary)
	}
	if primary, secondary := count(rightClick); primary != 0 || secondary != 1 {
		t.Fatalf("a right click counted %d primary and %d secondary", primary, secondary)
	}
	// The keyboard sends only the select and context-menu messages.
	if primary, _ := count([]uint32{ninKeySelect}); primary != 1 {
		t.Fatal("selecting the icon from the keyboard did nothing")
	}
	if primary, secondary := count([]uint32{moveOver, 0x0201, 0x0203}); primary+secondary != 0 {
		t.Fatal("a message that is not a click counted as one")
	}
}
