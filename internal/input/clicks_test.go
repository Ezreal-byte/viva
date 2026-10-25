package input

import (
	"testing"
	"time"
)

func TestClickCounterCountsSequencesAndResetsOnTimeDistanceAndButton(t *testing.T) {
	counter := ClickCounter{Interval: 500 * time.Millisecond, Distance: 4}
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }

	for step, test := range []struct {
		at      time.Duration
		button  uint8
		x, y    float32
		want    int
		comment string
	}{
		{ms(0), 1, 100, 100, 1, "first press"},
		{ms(200), 1, 102, 99, 2, "second press, close and quick: double click"},
		{ms(400), 1, 103, 101, 3, "third press continues the sequence: triple click"},
		{ms(1200), 1, 103, 101, 1, "too slow starts over"},
		{ms(1300), 1, 150, 101, 1, "too far starts over"},
		{ms(1400), 2, 150, 101, 1, "another button starts over"},
		{ms(1500), 1, 150, 101, 1, "and so does coming back to the first"},
		{ms(1999), 1, 150, 101, 2, "exactly inside the interval continues"},
		{ms(2499), 1, 154, 105, 3, "exactly at the distance limit continues"},
		{ms(2998), 1, 154, 105, 4, "counts keep growing"},
	} {
		if got := counter.Press(test.at, test.button, test.x, test.y); got != test.want {
			t.Errorf("step %d (%s): count = %d, want %d", step, test.comment, got, test.want)
		}
		if counter.Current() != test.want {
			t.Errorf("step %d: Current = %d, want %d (a release reports its press's count)", step, counter.Current(), test.want)
		}
	}
}

func TestClickCounterZeroValueNeverMakesDoubleClicks(t *testing.T) {
	var counter ClickCounter
	if counter.Current() != 1 {
		t.Fatalf("Current before any press = %d, want 1", counter.Current())
	}
	for index := 0; index < 3; index++ {
		if got := counter.Press(time.Duration(index), 1, 10, 10); got != 1 {
			t.Fatalf("press %d on an unconfigured counter = %d, want 1", index, got)
		}
	}
}

func TestClickCounterIgnoresBackwardsTime(t *testing.T) {
	counter := ClickCounter{Interval: time.Second, Distance: 4}
	counter.Press(5*time.Second, 1, 0, 0)
	if got := counter.Press(4*time.Second, 1, 0, 0); got != 1 {
		t.Fatalf("a press older than the previous one continued a sequence: %d", got)
	}
}
