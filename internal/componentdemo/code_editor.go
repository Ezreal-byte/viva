package componentdemo

import (
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/ui"
	"strings"
)

var codeEditorField = tree.ID(tree.Feature("code-editor"), "field")
var codeEditorBar = tree.ID(tree.Feature("code-editor"), "scroll")
var codeEditorDemoValue = strings.Repeat("name: Viva\nenabled: true\n# application-owned grammar\n", 12)

var codeEditorComponent = Component{
	ID: "code-editor", Title: "Code Editor", Category: "Forms", Summary: "Caller-owned text with syntax colors and native text placement.",
	initialize: func(state *State) { state.codeEditor = new(components.CodeEditor) },
	paint: func(state *State, canvas *ui.Canvas, theme ui.Theme, size ui.Size) {
		state.codeEditor.Paint(canvas, theme, ui.Rect{X: 24, Y: 24, Width: max(0, size.Width-48), Height: max(64, size.Height-48)}, components.CodeEditorSnapshot{TextareaSnapshot: components.TextareaSnapshot{ID: codeEditorField, Label: "Configuration source", Value: codeEditorDemoValue, ReadOnly: true, SelectionStart: 0, SelectionEnd: 4, Focused: true}, Language: "yaml", ScrollbarID: codeEditorBar, Highlight: demoCodeHighlight})
	},
	pointer: func(state *State, event ui.PointerEvent) ui.EventResult { return state.codeEditor.Pointer(event) },
	scroll: func(state *State, event ui.ScrollEvent) ui.EventResult {
		state.codeEditor.Layout.ScrollBy(event.DeltaY)
		return ui.EventResult{Handled: true, Repaint: true}
	},
	Examples: []Example{{ID: "default", Title: "Highlighted configuration", Description: "The highlighter colors byte ranges without changing text, selection or composition. Drag the scrollbar to inspect more lines.", Height: 300, Snippet: `editor.Paint(canvas, theme, bounds, components.CodeEditorSnapshot{
    TextareaSnapshot: components.TextareaSnapshot{ID: field, Value: source, ReadOnly: true},
    Language: "yaml", Highlight: highlight, ScrollbarID: scrollbar,
})`}},
}

func demoCodeHighlight(_, line string, dst []components.CodeEditorRun) []components.CodeEditorRun {
	if strings.HasPrefix(line, "#") {
		return append(dst, components.CodeEditorRun{End: len(line), Color: ui.RGB(100, 130, 110)})
	}
	if end := strings.IndexByte(line, ':'); end >= 0 {
		return append(dst, components.CodeEditorRun{End: end, Color: ui.RGB(70, 130, 220)})
	}
	return dst
}
