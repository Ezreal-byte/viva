---
title: 组件
summary: 直接使用组件目录，或者把组件的 Go 源码复制进自己的项目并拥有它。
order: 10
group: guides
---

## 目录里有什么

组件目录是对固定版本 shadcn/ui 组件集的静态类型 Go 翻译：按钮与表单、浮层与菜单、日历、表格、图表和反馈界面。它们都是通过 `ui.Canvas` 绘制的普通 Go 函数，没有反射，没有运行时样式解析器，底下也没有浏览器。[组件目录](/zh/ui/button/)展示了每一个组件的运行效果、示例、源码和键盘行为。

## 使用组件

引入 `github.com/vivaui/viva/components`，在绘制时声明组件。这是常规用法，[第一个窗口](/zh/docs/getting-started/)就是这样做的。

```go
components.BeginButton(canvas, theme, hitSave, bounds).Outline("Save")
```

主题是一个值，`ui.LightTheme()` 或 `ui.DarkTheme()`。切换外观就是改用另一个主题绘制。

## 拥有一份副本

想修改组件时，就复制它。`viva list` 列出安装器能复制的配方，`viva add` 把它们作为普通 Go 文件写进你的项目：

```sh
go install github.com/vivaui/viva/cmd/viva@latest    # 只需一次：安装 viva 命令
viva list
viva add --out ./components/ui button card badge
```

| 参数 | 含义 |
| --- | --- |
| `--out` | 写入的目录，默认 `components/ui`。 |
| `--package` | 生成文件的包名，默认 `ui`。 |
| `--force` | 覆盖已存在的文件。 |

复制出来的文件由你修改。它们仍然引入 Viva 的 `ui` 包来使用画布和主题类型，所以依赖 Viva 模块，但没有任何东西在中央注册或更新它们。每个组件都是安装器能复制的配方；一个配方若调用了另一个（例如 `badge` 用到 `button`），会连同它一起复制，保证复制出的这一组能独立编译。这些副本由 `github.com/vivaui/viva/components` 包的同一批源文件生成，所以与它保持一致。导入 `github.com/vivaui/viva/components` 但不调用任何配方，不会链接其中任何内容，因此只有在选项不够用时才需要复制组件。目录中每个组件页都有源码文件的**下载**链接和安装命令。
