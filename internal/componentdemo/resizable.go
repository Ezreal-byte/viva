package componentdemo

import (
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/flex"
	"github.com/vivaui/viva/ui"
)

var resizableComponent = Component{
	ID: "resizable", Title: "Resizable", Category: "Layout", Summary: "返回精确面板几何的横向与纵向分隔器。",
	paint: (*State).paintResizable, pointer: (*State).pointerResizable, initialize: (*State).initializeResizable,
	Examples: []Example{
		{ID: "horizontal", Title: "横向分栏", Description: "捕获主指针拖动分隔器，激活仍可循环调用方持有的分栏比例。", Snippet: `bounds := stage.Rect()
handle := tree.ID(page, "split")
resizable := components.BeginResizable(canvas, theme, handle, bounds, fraction).
    Limits(0.2, 0.8).HandleSize(12).WithHandle().ValueText(valueText)
layout := resizable.Paint("Resize editor panels")
canvas.Rounded(layout.First.Inset(ui.All(8)), 6, theme.Muted)
canvas.Rounded(layout.Second.Inset(ui.All(8)), 6, theme.CardElevated)`},
		{ID: "vertical", Title: "纵向分栏", Description: "同一生产组件切换为上下堆叠布局。", Snippet: `bounds := stage.Rect()
handle := tree.ID(page, "split")
layout := components.BeginResizable(canvas, theme, handle, bounds, fraction).
    Vertical().Limits(0.2, 0.8).HandleSize(12).WithHandle().
    Paint("Resize editor rows")
canvas.Rounded(layout.First.Inset(ui.All(8)), 6, theme.Muted)
canvas.Rounded(layout.Second.Inset(ui.All(8)), 6, theme.CardElevated)`},
		{ID: "disabled", Title: "禁用分隔器", Description: "保留布局与抓手，但不允许交互。", Snippet: `bounds := stage.Rect()
handle := tree.ID(page, "split")
layout := components.BeginResizable(canvas, theme, handle, bounds, fraction).
    Limits(0.2, 0.8).HandleSize(12).WithHandle().Disabled().
    Paint("Resize editor panels")
canvas.Rounded(layout.First.Inset(ui.All(8)), 6, theme.Muted)
canvas.Rounded(layout.Second.Inset(ui.All(8)), 6, theme.CardElevated)`},
	},
}

var (
	resizableUI     = tree.Feature("resizable")
	resizableHandle = tree.ID(resizableUI, "handle")
)

func (state *State) initializeResizable() {
	state.values[0] = 50
	state.bag()
}

func (state *State) pointerResizable(event ui.PointerEvent) ui.EventResult {
	if event.Target != resizableHandle || state.exampleID == "disabled" {
		return ui.EventResult{}
	}
	if event.Kind != ui.PointerDown && !(event.Kind == ui.PointerMove && event.Captured) {
		return ui.EventResult{Handled: event.Kind == ui.PointerUp}
	}
	if event.Kind == ui.PointerDown && event.Button != ui.PointerButtonPrimary {
		return ui.EventResult{}
	}
	resizable := state.previewResizable()
	position := event.X
	if state.exampleID == "vertical" {
		position = event.Y
	}
	value := int32(resizable.FractionAt(position)*100 + 0.5)
	changed := state.values[0] != value
	state.values[0] = value
	return ui.EventResult{Focus: resizableHandle, Handled: true, Repaint: changed}
}

// previewResizable is the splitter as the last paint placed it, for the pointer to
// turn a coordinate into a fraction.
func (state *State) previewResizable() components.Resizable {
	resizable := components.BeginResizable(nil, ui.Theme{}, resizableHandle, state.bag().placed, resizableFraction(state.values[0])).Limits(0.2, 0.8)
	if state.exampleID == "vertical" {
		resizable = resizable.Vertical()
	}
	return resizable
}

func (state *State) paintResizable(canvas *ui.Canvas, theme ui.Theme, size ui.Size) {
	var stage flex.Box
	state.page(canvas, theme, size, 600, "Editor layout", func(f *flex.Frame) {
		stage = f.Slot().Height(200)
		caption(f, state.Text("拖动抓手精确调整；激活循环 30% / 50% / 70%。", "Drag the handle to resize precisely; clicking cycles 30% / 50% / 70%."))
	})

	bounds := stage.Rect()
	if bag := state.slider; bag != nil {
		bag.placed = bounds
	}
	fraction := resizableFraction(state.values[0])
	resizable := components.BeginResizable(canvas, theme, resizableHandle, bounds, fraction).
		Limits(0.2, 0.8).
		HandleSize(12).
		WithHandle().
		ValueText(resizableValueText(state.values[0]))
	if state.exampleID == "vertical" {
		resizable = resizable.Vertical()
	}
	if state.exampleID == "disabled" {
		resizable = resizable.Disabled()
	}
	layout := resizable.Paint("Resize editor panels")

	canvas.Rounded(layout.First.Inset(ui.All(8)), 6, theme.Muted)
	canvas.Rounded(layout.Second.Inset(ui.All(8)), 6, theme.CardElevated)
	canvas.LabelCenter("Navigator", layout.First.Inset(ui.All(16)), ui.TextStyle{Size: theme.Metrics.FontSizeM, Weight: ui.WeightMedium, Color: theme.Foreground})
	canvas.LabelCenter("Preview", layout.Second.Inset(ui.All(16)), ui.TextStyle{Size: theme.Metrics.FontSizeM, Weight: ui.WeightMedium, Color: theme.Foreground})

	if canvas.Clicked(resizableHandle) {
		switch state.values[0] {
		case 30:
			state.values[0] = 50
		case 50:
			state.values[0] = 70
		default:
			state.values[0] = 30
		}
	}
}

func resizableFraction(value int32) float32 {
	return float32(value) / 100
}

func resizableValueText(value int32) string {
	switch value {
	case 50:
		return "50 percent"
	case 30:
		return "30 percent"
	case 70:
		return "70 percent"
	case 20:
		return "20 percent"
	case 40:
		return "40 percent"
	case 60:
		return "60 percent"
	case 80:
		return "80 percent"
	default:
		return "custom split"
	}
}
