package flex

import (
	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/text"
	"github.com/vivaui/viva/ui"
)

// Field is one single-line text field's state, kept in the application's model; the
// zero value is an empty field. It owns the editor the platform's text service
// edits, so typing, the caret, selection, clipboard and input methods work without
// any further wiring beyond giving Fields to Config.TextInput.
//
//	type model struct{ name, password flex.Field }
//
//	m.password.Secure = true
//	config.TextInput = flex.Fields(&m.name, &m.password)
//	f.Field(&m.name, nameID, "Name", "Your name")
//	f.Field(&m.password, passwordID, "Password", "")
//	m.name.Text()                       // what was typed
type Field struct {
	// Secure shows bullets instead of the text and keeps it out of accessibility.
	Secure bool

	editor *text.Editor
	id     ui.ID
	// input is the stable TextInput the platform provider hands out. Returning
	// a fresh one on every repaint allocated once per focused frame.
	input app.TextInput
}

func (field *Field) ensure() *text.Editor {
	if field.editor == nil {
		field.editor = text.NewEditor("")
	}
	return field.editor
}

// Text is what is in the field.
func (field *Field) Text() string { return field.ensure().String() }

// SetText replaces the contents and puts the caret at the end.
func (field *Field) SetText(value string) {
	editor := field.ensure()
	editor.SetSelection(text.Range{Start: 0, End: editor.RuneCount()})
	editor.Insert(value)
}

// Field adds a text field for field, named id (a stable identity from your Tree),
// with an accessible label and placeholder. It is Input with the snapshot taken from
// the field: focus, platform editing and the caret are filled in for you.
func (f *Frame) Field(field *Field, id ui.ID, label, placeholder string) Box {
	editor := field.ensure()
	field.id = id
	value := editor.String()
	focused := f.canvas != nil && f.canvas.Focused() == id
	return f.Input(components.InputSnapshot{
		ID: id, Label: label, Value: value, Placeholder: placeholder, Secure: field.Secure,
		Focused:         focused,
		PlatformEditing: focused && value != "",
		ShowCaret:       focused && value == "",
	})
}

// Submitted reports whether this paint is delivering Enter pressed in the text field the
// box declares (f.Field or f.Input), and false in every other paint: a click, a typed
// character or a held key does not count. It is how a chat box sends and a search box
// runs, where the field is declared, like Clicked:
//
//	if f.Field(&m.draft, draftID, "Message", "Type a message").Submitted() { m.send(); m.draft.SetText("") }
//
// A field is one line, so Enter is free to mean "done"; a Textarea keeps Enter for new
// lines and has no Submitted.
func (b Box) Submitted() bool {
	n := b.n()
	if n.kind != kindInput || b.f.canvas == nil {
		return false
	}
	return b.f.canvas.ClickedByKey(b.f.inputs[n.aux].ID)
}

// Fields answers Config.TextInput for the fields: the platform asks which editor
// belongs to the focused control, and gets the one of the field declared with that
// identity.
func Fields(fields ...*Field) func(app.TextInputContext) *app.TextInput {
	return func(context app.TextInputContext) *app.TextInput {
		for _, field := range fields {
			if field.id == context.ID && !context.ID.IsZero() {
				field.input.Editor = field.ensure()
				return &field.input
			}
		}
		return nil
	}
}
