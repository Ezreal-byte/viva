//go:build vivahead && darwin

package gallery

import (
	"errors"
	"testing"

	"github.com/vivaui/viva/headless"
)

// This test needs the offscreen renderer, which only -tags vivahead links in.
func TestAnInputOnlyEditorLeavesNoMarkOnScreen(t *testing.T) {
	state := New()
	state.EnableTextEditing()
	session, err := headless.Open(galleryConfig(state), nil, headless.Options{Scale: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for index := range catalog {
		if catalog[index].ID == "input-otp" {
			state.openPage(index + 1)
		}
	}
	if err := session.Resize(DefaultWidth, DefaultHeight); err != nil {
		t.Fatal(err)
	}
	field, ok := session.Hit("Verification code")
	if !ok {
		t.Fatal("the OTP field is not on screen")
	}
	// Focus the first slot: the platform text service now has an editor there.
	if err := session.Click(field.Bounds.X+8, field.Bounds.Y+field.Bounds.Height/2); err != nil {
		t.Fatal(err)
	}
	image, err := session.Image()
	if errors.Is(err, headless.ErrNoRenderer) {
		t.Skip("no offscreen renderer")
	}
	if err != nil {
		t.Fatal(err)
	}

	// The OTP paints its own slots and caret. The invisible editor sits at the
	// left edge of the active slot, and its own caret would be drawn right there,
	// 2px wide, over the slot's accent border.
	y := int(field.Bounds.Y + field.Bounds.Height/2)
	x := int(field.Bounds.X)
	pixel := func(x int) (r, g, b uint32) {
		r, g, b, _ = image.At(x, y).RGBA()
		return r >> 8, g >> 8, b >> 8
	}
	if r, g, b := pixel(x); !(b > r+100 && b > g+100) {
		t.Fatalf("the active slot's left border is (%d,%d,%d), want the accent blue: the hidden editor drew its caret over it", r, g, b)
	}
	if r, g, b := pixel(x + 1); r < 200 || g < 200 || b < 200 {
		t.Fatalf("the pixel inside the slot's left border is (%d,%d,%d), want the slot's empty fill", r, g, b)
	}
}
