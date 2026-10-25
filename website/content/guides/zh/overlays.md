---
title: 对话框、菜单与弹出层
summary: 图层、焦点范围与关闭方式：点击外部就关闭的下拉菜单，以及模态对话框。
order: 9
group: guides
---

浮层最后绘制，位于更高的图层，并且会改变输入能去往哪里。在 `flex` 里，浮层声明在页面中它打开的那个位置，并且不占那里的空间。Viva 给每个浮层提供三样东西。

- **图层。** `f.Popup()` 和 `f.Dialog` 画在页面之上，声明在它们上面的命中区域优先于下面的。
- **焦点范围。** `DialogSpec.Scope` 在对话框打开期间把 Tab 困在里面，它后面的任何东西都拿不到键盘焦点。
- **关闭标识。** 用户按下 Escape 或点击浮层外部时，报告点击的那个标识。你在声明浮层的地方读取它，和读取其他点击一样。

和 Viva 里一贯的做法一样，浮层是否打开是你的状态。你声明它，它才存在。

<!-- example: overlay/main.go -->
```go
package main

import (
	"log"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/flex"
	"github.com/vivaui/viva/ui"
)

// Identities are names in a tree, not numbers: the application declares the
// tree once and every control it paints is a name inside a feature of it.
var (
	tree   = ui.Tree{}
	page   = tree.Feature("overlays")
	dialog = tree.Sub(page, tree.Name("dialog"))

	trigger     = tree.ID(page, "trigger")
	menuDismiss = tree.ID(page, "menu-dismiss")
	menuNew     = tree.ID(page, "menu-new")
	menuDelete  = tree.ID(page, "menu-delete")

	dialogScope   = tree.ID(dialog, "scope")
	dialogClose   = tree.ID(dialog, "close")
	dialogCancel  = tree.ID(dialog, "cancel")
	dialogConfirm = tree.ID(dialog, "confirm")
)

type model struct {
	menuOpen   bool
	dialogOpen bool
	status     string
	view       flex.View
}

func newConfig(m *model) app.Config {
	theme := ui.LightTheme()
	config := app.Config{Tree: &tree, Title: "Overlays", Width: 640, Height: 360}

	config.Paint = func(canvas *ui.Canvas, size ui.Size) {
		bounds := ui.Rect{Width: size.Width, Height: size.Height}
		canvas.Fill(bounds, theme.Background)

		m.view.Draw(canvas, theme, bounds, func(f *flex.Frame) {
			f.Column().Fill().Pad(24).Gap(12).Items(flex.Start).Children(func() {
				actions := f.Button(trigger, "Actions").Outline()
				if actions.Clicked() {
					m.menuOpen = !m.menuOpen
				}
				f.Text(m.status).Muted()

				if m.menuOpen {
					// A popup takes no room and is painted over the page. The first one
					// covers the window so a press outside the menu lands on menuDismiss,
					// and Escape is routed to the same identity.
					canvas.SetEscapeTarget(menuDismiss)
					f.Popup().At(0, 0).Fill().Hit(menuDismiss, "Close menu").Children(func() {})
					if canvas.Clicked(menuDismiss) {
						m.menuOpen = false
					}
					f.Popup().BelowBox(actions, 4, flex.Start).Width(180).Pad(4).Background(theme.Popover).
						Border(1, theme.Border).Radius(theme.Metrics.RadiusControl).Shadow(ui.ShadowMD).Children(func() {
						if f.Button(menuNew, "New file").Ghost().Clicked() {
							m.menuOpen, m.status = false, "Created"
						}
						if f.Button(menuDelete, "Delete…").Ghost().Destructive().Clicked() {
							m.menuOpen, m.dialogOpen = false, true
						}
					})
				}

				if m.dialogOpen {
					d := f.Dialog(flex.DialogSpec{
						Title: "Delete file?", Description: "This cannot be undone.", Width: 400,
						Scope: dialogScope, DismissID: dialogCancel, CloseID: dialogClose,
					})
					d.Footer(func() {
						f.Button(dialogCancel, "Cancel").Outline()
						if f.Button(dialogConfirm, "Delete").Destructive().Clicked() {
							m.dialogOpen, m.status = false, "Deleted"
						}
					})
					if d.Dismissed() {
						m.dialogOpen = false
					}
				}
			})
		})
	}
	return config
}

func main() {
	var canvas ui.Canvas
	if err := app.RunCanvas(newConfig(&model{}), &canvas); err != nil {
		log.Fatal(err)
	}
}
```

## 菜单与弹出层

`f.Popup()` 不占所在容器的空间，大小由内容决定，始终保持在窗口内，并画在所有东西之上。针对页面上的某个元素，用 `BelowBox`、`AboveBox` 或 `BesideBox` 定位；针对某个点或矩形，用 `At(x, y)`、`Below(rect, ...)` 和 `Above(rect, ...)`。它优先用你指定的一侧，放不下时翻到另一侧。要锚定到触发控件，就保留它的 `Box`（`actions := f.Button(...)`）：`Box.Rect()` 在这一帧结束前为零。菜单的行就是弹出层里普通的按钮或 `Hit` 盒子；可能很长的菜单需要 `MaxHeight` 和 `ScrollArea`。

右键菜单是同样的弹出层，用 `At` 放在指针处。在 `OnPointer` 里收到 `PointerButtonSecondary` 按下时打开它。

### 点击外部时关闭

弹出层不会自己关闭，因为是否打开是你的状态。示例在菜单下面再声明一个与窗口一样大的弹出层并给它一个 `Hit`：在菜单外的任何位置按下（包括打开它的那个按钮）都会落到这个标识上，`canvas.SetEscapeTarget` 把 Escape 也送到同一个标识。用 `canvas.Clicked` 读取它，然后关闭菜单。这次按下不做别的事，所以一次点击不会既关闭菜单又触发它下面的页面内容。

示例的测试把这些全检查了：窗口远角的按下落到关闭标识上，Escape 关闭菜单，选择 Delete… 会关闭菜单并打开对话框。

## 对话框

`f.Dialog(flex.DialogSpec{...})` 把窗口变暗，画出居中的表面，带标题、描述和关闭按钮，并困住焦点。只在对话框打开时声明它。`Body` 声明正文，`Footer` 声明按钮，按钮按给定顺序靠右排列；表面的高度正好等于正文所需，所以正文很短时页脚上方不会留下空隙。点击背景、关闭按钮或按 Escape 时，`Dismissed()` 在递送这次点击的那一遍里为真，Escape 由对话框自己路由。

`DismissID` 是背景和 Escape 报告的标识，`CloseID` 是右上角关闭按钮的。示例把 `DismissID` 设成 Cancel 的标识，所以点击背景和按 Escape 的表现都和 Cancel 一致。除非设置 `InitialFocusID`（应当获得焦点的控件），否则打开对话框不会移动焦点。对话框里有输入框时用它。

## 选择框及其他

`f.Select` 是下拉列表：模型里放一个 `flex.SelectState` 和它自己的 scope，列表在触发器下方展开（放不下时在上方），画在所有东西之上；在对话框里它在对话框上方展开。工具提示和其他非模态面板是不声明关闭标识的弹出层，所以它们不困住焦点，也不吞掉外部点击。Combobox、命令面板、菜单栏和导航菜单遵循同样的图层、范围和关闭约定；[组件目录](/zh/ui/dialog/)为每一个都提供了可运行的示例，[API 参考](/zh/api/components/)列出了全部选项。

## 进场与退场

对话框、Sheet、抽屉，以及组件配方里的菜单、弹出层、提示和 Toast，会在你开始绘制它们时淡入并移动到位，在你停止绘制时淡出，和 shadcn 的 `animate-in`、`animate-out` 一样。你不需要为此保存任何状态：不再声明对话框，画布会在接下来的 100 毫秒里把它最后一帧画完淡出，这期间它没有命中区域、焦点陷阱和 Escape 目标，不会从下面的界面抢走点击或键盘。退场中再次绘制，会从当前位置掉头。它需要稳定的标识（对话框的 `DismissID`，菜单的 `DismissID` 或 `Scope`），菜单、Select 和 Combobox 需要调用 `End()`；主题的 `Transition` 没有时长、减弱动画或运行时没有帧时钟时，直接显示最终状态。
