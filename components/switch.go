package components

import "github.com/vivaui/viva/ui"

// SwitchSize selects the official default or small switch geometry.
type SwitchSize uint8

const (
	SwitchSizeDefault SwitchSize = iota
	SwitchSizeSmall
)

// switchSlotChecked is the animation slot holding thumb travel, 0 (off) to 1 (on).
const switchSlotChecked uint8 = 0

// Switch paints one two-state toggle with caller-owned state. When the
// runtime provides a frame clock, a change of the checked value slides the
// thumb and fades the track over the theme's Transition; the caller still
// passes only the destination state.
type Switch struct {
	canvas   *ui.Canvas
	theme    ui.Theme
	id       ui.ID
	bounds   ui.Rect
	label    string
	checked  bool
	focused  bool
	invalid  bool
	size     SwitchSize
	disabled bool
}

// BeginSwitch starts a switch declaration. Zero-sized bounds select official
// preset geometry. ID zero is a static preview and declares no hit.
func BeginSwitch(canvas *ui.Canvas, theme ui.Theme, id ui.ID, bounds ui.Rect, checked bool) Switch {
	focused := false
	if canvas != nil && !id.IsZero() {
		focused = canvas.FocusVisible() == id
	}
	return Switch{
		canvas: canvas, theme: theme, id: id, bounds: bounds,
		label: "switch", checked: checked, focused: focused,
	}
}

// Small selects the official compact switch size.
func (switchControl Switch) Small() Switch {
	switchControl.size = SwitchSizeSmall
	return switchControl
}

// Label sets the accessible name without painting text.
func (switchControl Switch) Label(label string) Switch {
	switchControl.label = label
	return switchControl
}

// Invalid applies the official destructive validation border and ring.
func (switchControl Switch) Invalid() Switch {
	switchControl.invalid = true
	return switchControl
}

// Disabled paints the official 50% opacity state and retains disabled
// accessibility semantics for nonzero IDs.
func (switchControl Switch) Disabled() Switch {
	switchControl.disabled = true
	return switchControl
}

// SwitchResult is what painting reports back; Clicked is true in the event pass of a
// click on the control (see ui.Canvas.Clicked), so the code that declares it
// handles it.
type SwitchResult struct {
	canvas *ui.Canvas
	id     ui.ID
}

// Clicked reports whether this paint is delivering a click on the control.
func (result SwitchResult) Clicked() bool {
	return result.canvas != nil && result.canvas.Clicked(result.id)
}

// SwitchExtent is the width and height of a switch at the theme's density.
func SwitchExtent(theme ui.Theme, small bool) (width, height float32) {
	scale := theme.Metrics.ControlHeight / 32
	if small {
		return 24 * scale, 14 * scale
	}
	return 32 * scale, 18.4 * scale
}

// SwitchHitExtent is the width and height of the interaction region Paint
// declares around a switch. The padding is the fixed 12 by 8 logical pixels
// Paint adds, not scaled by density.
func SwitchHitExtent(theme ui.Theme, small bool) (width, height float32) {
	width, height = SwitchExtent(theme, small)
	return width + 24, height + 16
}

// Paint draws the switch and declares its enlarged desktop hit region.
func (switchControl Switch) Paint() SwitchResult {
	result := SwitchResult{canvas: switchControl.canvas, id: switchControl.id}
	if switchControl.canvas == nil {
		return result
	}
	scale := switchControl.theme.Metrics.ControlHeight / 32
	defaultWidth, defaultHeight := SwitchExtent(switchControl.theme, switchControl.size == SwitchSizeSmall)
	thumbSize := float32(16) * scale
	if switchControl.size == SwitchSizeSmall {
		thumbSize = 12 * scale
	}
	if switchControl.bounds.Width <= 0 {
		switchControl.bounds.Width = defaultWidth
	}
	if switchControl.bounds.Height <= 0 {
		switchControl.bounds.Height = defaultHeight
	}
	if switchControl.bounds.Height != defaultHeight {
		thumbSize = switchControl.bounds.Height - 2*scale
	}
	if thumbSize < 1 {
		thumbSize = 1
	}

	// checked is the destination; progress is how far the thumb has travelled
	// toward it. Without a frame clock progress is exactly 0 or 1.
	target := float32(0)
	if switchControl.checked {
		target = 1
	}
	progress := switchControl.canvas.Animate(switchControl.id, switchSlotChecked, target, switchControl.theme.Transition)

	trackOff, trackOn := switchControl.theme.Input, switchControl.theme.Primary
	thumbOff, thumbOn := switchControl.theme.Background, switchControl.theme.Background
	if isDark(switchControl.theme) {
		trackOff.Alpha *= 0.80
		thumbOff, thumbOn = switchControl.theme.Foreground, switchControl.theme.PrimaryForeground
	}
	track := trackOff.Mix(trackOn, progress)
	thumb := thumbOff.Mix(thumbOn, progress)
	switchControl.canvas.Elevate(switchControl.bounds, switchControl.bounds.Height/2, ui.ShadowXS) // shadow-xs
	var border ui.Color
	bordered := switchControl.focused || switchControl.invalid
	if bordered {
		ring := scaledAlpha(switchControl.theme.Ring, 0.50)
		border = switchControl.theme.Ring
		if switchControl.invalid {
			ring, border = invalidRing(switchControl.theme), invalidBorder(switchControl.theme)
		}
		switchControl.canvas.Rounded(expand(switchControl.bounds, 3), switchControl.bounds.Height/2+3, ring)
	}
	if switchControl.disabled {
		track.Alpha *= 0.50
		thumb.Alpha *= 0.50
		border.Alpha *= 0.50
	}
	if bordered {
		paintBordered(switchControl.canvas, switchControl.theme, switchControl.bounds, switchControl.bounds.Height/2, border, track)
	} else {
		switchControl.canvas.Rounded(switchControl.bounds, switchControl.bounds.Height/2, track)
	}

	// The thumb sits inside the track's 1px border: one pixel from the left edge
	// when off and one from the right when on (translate-x-[calc(100%-2px)]).
	inset := 1 * scale
	thumbX := ui.Lerp(switchControl.bounds.X+inset, switchControl.bounds.X+switchControl.bounds.Width-thumbSize-inset, progress)
	thumbY := switchControl.bounds.Y + (switchControl.bounds.Height-thumbSize)/2
	switchControl.canvas.Rounded(ui.Rect{
		X: thumbX, Y: thumbY, Width: thumbSize, Height: thumbSize,
	}, thumbSize/2, thumb)

	if !switchControl.id.IsZero() {
		hitBounds := ui.Rect{
			X: switchControl.bounds.X - 12, Y: switchControl.bounds.Y - 8,
			Width: switchControl.bounds.Width + 24, Height: switchControl.bounds.Height + 16,
		}
		switchControl.canvas.HitWithOptions(switchControl.id, hitBounds, switchControl.label, ui.HitOptions{
			Role: ui.RoleSwitch, Disabled: switchControl.disabled,
			Checked: switchControl.checked,
		})
	}
	return result
}
