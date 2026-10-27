package components

import "github.com/vivaui/viva/ui"

// FormFieldSnapshot is the caller-owned state for one form item. Error takes
// precedence over Message, matching shadcn's FormMessage behavior without a
// form registry, context, or validation runtime.
type FormFieldSnapshot struct {
	Label         string
	Description   string
	Message       string
	Error         string
	ControlHeight float32
	Required      bool
	Disabled      bool
	Invalid       bool
	// ControlID, when set, is the identity of the control painted into the
	// layout's Control slot; the label then labels it (see ui.Canvas.HitLabel),
	// as `htmlFor` does, so pressing the label focuses and clicks the control.
	ControlID ui.ID
}

// FormControlSemantics carries the typed metadata that the caller passes to
// the Input, Select, Checkbox, or other control painted into FormFieldLayout.Control.
type FormControlSemantics struct {
	Label    string
	Hint     string
	Error    string
	Required bool
	Disabled bool
	Invalid  bool
}

// AssistiveText selects the most important single hint for controls whose
// current platform semantic contract accepts one hint string. Error remains a
// separate field so richer accessibility adapters can expose both values.
func (semantics FormControlSemantics) AssistiveText() string {
	if semantics.Invalid && semantics.Error != "" {
		return semantics.Error
	}
	return semantics.Hint
}

// FormFieldLayout reports every deterministic slot painted for a form item.
// Zero Description or Message rectangles mean that the corresponding row was
// omitted. The caller paints the actual control into Control.
type FormFieldLayout struct {
	Bounds      ui.Rect
	Label       ui.Rect
	Control     ui.Rect
	Description ui.Rect
	Message     ui.Rect
	Semantics   FormControlSemantics
}

// PaintFormField paints shadcn's label, description, and message recipe, in the
// same rhythm as Field, and returns the control slot. It never paints or registers the control itself,
// so editor state, values, validation, and interaction remain caller-owned.
func PaintFormField(canvas *ui.Canvas, theme ui.Theme, bounds ui.Rect, snapshot FormFieldSnapshot) FormFieldLayout {
	metrics := theme.Metrics
	gap := fieldGap(metrics)
	labelHeight := fieldTitleHeight(metrics)
	textHeight := fieldTextHeight(metrics)
	controlHeight := snapshot.ControlHeight
	if controlHeight <= 0 {
		controlHeight = metrics.ControlHeight
	}

	invalid := snapshot.Invalid || snapshot.Error != ""
	message := snapshot.Message
	if snapshot.Error != "" {
		message = snapshot.Error
	}
	semanticError := ""
	if invalid {
		semanticError = message
	}

	layout := FormFieldLayout{
		Label: ui.Rect{
			X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: labelHeight,
		},
		Semantics: FormControlSemantics{
			Label: snapshot.Label, Hint: snapshot.Description, Error: semanticError,
			Required: snapshot.Required, Disabled: snapshot.Disabled, Invalid: invalid,
		},
	}
	layout.Control = ui.Rect{
		X: bounds.X, Y: layout.Label.Y + layout.Label.Height + gap,
		Width: bounds.Width, Height: controlHeight,
	}
	nextY := layout.Control.Y + layout.Control.Height
	if snapshot.Description != "" {
		nextY += gap
		layout.Description = ui.Rect{X: bounds.X, Y: nextY, Width: bounds.Width, Height: textHeight}
		nextY += textHeight
	}
	if message != "" {
		nextY += gap
		layout.Message = ui.Rect{X: bounds.X, Y: nextY, Width: bounds.Width, Height: textHeight}
		nextY += textHeight
	}
	layout.Bounds = ui.Rect{X: bounds.X, Y: bounds.Y, Width: bounds.Width, Height: nextY - bounds.Y}

	if canvas == nil || bounds.Width <= 0 {
		return layout
	}
	labelColor := theme.Foreground
	if invalid {
		labelColor = theme.Danger
	}
	if snapshot.Disabled {
		labelColor.Alpha *= 0.5
	}
	labelWidth := canvas.TextWidthWeight(snapshot.Label, metrics.FontSizeM, ui.WeightMedium)
	if labelWidth > layout.Label.Width {
		labelWidth = layout.Label.Width
	}
	canvas.Label(snapshot.Label, ui.Rect{
		X: layout.Label.X, Y: layout.Label.Y, Width: labelWidth, Height: layout.Label.Height,
	}, ui.TextStyle{Size: metrics.FontSizeM, Weight: ui.WeightMedium, Color: labelColor})
	canvas.HitLabel(snapshot.ControlID, ui.Rect{X: layout.Label.X, Y: layout.Label.Y, Width: labelWidth, Height: layout.Label.Height}, snapshot.Label, snapshot.Disabled)
	if snapshot.Required && labelWidth < layout.Label.Width {
		requiredColor := theme.Danger
		if snapshot.Disabled {
			requiredColor.Alpha *= 0.5
		}
		canvas.Label("*", ui.Rect{
			X: layout.Label.X + labelWidth + 3, Y: layout.Label.Y,
			Width: layout.Label.Width - labelWidth - 3, Height: layout.Label.Height,
		}, ui.TextStyle{Size: metrics.FontSizeM, Weight: ui.WeightMedium, Color: requiredColor})
	}
	if snapshot.Description != "" {
		descriptionColor := theme.MutedForeground
		if snapshot.Disabled {
			descriptionColor.Alpha *= 0.5
		}
		canvas.Label(snapshot.Description, layout.Description, ui.TextStyle{
			Size: metrics.FontSizeM, Weight: ui.WeightRegular, Color: descriptionColor,
		})
	}
	if message != "" {
		messageColor := theme.Danger
		if snapshot.Disabled {
			messageColor.Alpha *= 0.5
		}
		canvas.Label(message, layout.Message, ui.TextStyle{
			Size: metrics.FontSizeM, Weight: ui.WeightRegular, Color: messageColor,
		})
	}
	return layout
}
