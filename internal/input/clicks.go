package input

import "time"

// ClickCounter numbers consecutive presses of one button: 1 for a lone press,
// 2 for the second of a double click, 3 for a triple click, and so on. Adapters
// whose operating system does not count clicks feed it their presses; macOS and
// browsers already report a count and do not need it.
//
// A press continues a sequence when it uses the same button, lands within
// Distance of the previous press on both axes, and arrives within Interval of
// it. The zero value never continues a sequence, so adapters set both fields
// from the system's double-click settings.
type ClickCounter struct {
	// Interval is the longest gap between presses of one sequence.
	Interval time.Duration
	// Distance is how far (in logical pixels, on each axis) a press may land
	// from the previous one and still count as the same sequence.
	Distance float32

	count  int
	last   time.Duration
	x, y   float32
	button uint8
}

// Press registers a press at time now (any monotonic clock) and returns its
// click count. Move events never reset a sequence on their own; a press that
// lands too far away does.
func (counter *ClickCounter) Press(now time.Duration, button uint8, x, y float32) int {
	continues := counter.count > 0 && button == counter.button &&
		now >= counter.last && now-counter.last <= counter.Interval &&
		abs(x-counter.x) <= counter.Distance && abs(y-counter.y) <= counter.Distance
	if continues {
		counter.count++
	} else {
		counter.count = 1
	}
	counter.last, counter.x, counter.y, counter.button = now, x, y, button
	return counter.count
}

// Current is the count of the most recent press, which its release reports
// too.
func (counter *ClickCounter) Current() int {
	return max(counter.count, 1)
}

func abs(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}
