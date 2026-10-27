package componentdemo

import (
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/flex"
	"github.com/vivaui/viva/ui"
)

var colorSwatchComponent = Component{
	ID: "color-swatch", Title: "Color Swatch", Category: "Forms", Summary: "带选中环、键盘焦点和禁用状态的颜色选择。",
	paint: (*State).paintColorSwatch,
	Examples: []Example{{ID: "palette", Title: "颜色选择", Description: "选择一种颜色，或恢复默认颜色。最后一个选项展示禁用状态。", Snippet: `swatch := components.BeginColorSwatch(canvas, theme, id, bounds, color).
    Label("Blue").Selected(selected == id)
if swatch.Paint().Clicked() {
    selected = id
}`}},
}

var colorSwatchItems = tree.Sub(tree.Feature("color-swatch"), tree.Name("color"))

func (state *State) paintColorSwatch(canvas *ui.Canvas, theme ui.Theme, size ui.Size) {
	var slots [4]flex.Box
	state.page(canvas, theme, size, 360, state.Text("连接颜色", "Connection color"), func(f *flex.Frame) {
		f.Row().Gap(12).Children(func() {
			for i := range slots {
				slots[i] = f.Slot().Size(36, 36)
			}
		})
		caption(f, state.Text("点击或使用 Tab / Space 选择颜色。", "Click or use Tab / Space to select a color."))
	})
	colors := [...]ui.Color{{}, {Green: 0.77, Red: 0.13, Blue: 0.37, Alpha: 1}, {Red: 0.23, Green: 0.51, Blue: 0.96, Alpha: 1}, {Red: 0.94, Green: 0.27, Blue: 0.27, Alpha: 1}}
	labels := [...]string{"No color", "Green", "Blue", "Red (disabled)"}
	for i, color := range colors {
		swatch := components.BeginColorSwatch(canvas, theme, colorSwatchItems.At(i), slots[i].Rect(), color).Label(labels[i]).Selected(state.values[0] == int32(i))
		if i == 3 {
			swatch = swatch.Disabled()
		}
		if swatch.Paint().Clicked() {
			state.values[0] = int32(i)
		}
	}
}
