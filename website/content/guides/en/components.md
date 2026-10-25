---
title: Components
summary: Use the catalog directly, or copy a component's Go source into your project and own it.
order: 10
group: guides
---

## What is in the catalog

The catalog is a statically typed Go translation of the pinned shadcn/ui component set: buttons and forms, overlays and menus, calendars, tables, charts and feedback surfaces. They are plain Go functions that paint through `ui.Canvas`. There is no reflection, no runtime style parser and no browser underneath. The [component catalog](/en/ui/button/) shows each one running, with its examples, source and keyboard behavior.

## Using a component

Import `github.com/vivaui/viva/components` and declare the component while painting. This is the normal path, and it is what the [first window](/en/docs/getting-started/) does.

```go
components.BeginButton(canvas, theme, hitSave, bounds).Outline("Save")
```

A theme is a value, `ui.LightTheme()` or `ui.DarkTheme()`. Switching appearance means painting with the other one.

## Owning a copy

If you want to change a component, copy it. `viva list` names the recipes the installer can copy, and `viva add` writes them into your project as ordinary Go files:

```sh
go install github.com/vivaui/viva/cmd/viva@latest    # once: installs the viva command
viva list
viva add --out ./components/ui button card badge
```

| Flag | Meaning |
| --- | --- |
| `--out` | Directory to write into. Default `components/ui`. |
| `--package` | Package name of the generated files. Default `ui`. |
| `--force` | Replace files that already exist. |

The copied files are yours to edit. They still import Viva's `ui` package for the canvas and theme types, so they depend on the Viva module, but nothing registers or updates them centrally. Every component is a recipe the installer can copy, and a recipe that calls into another one, such as `badge` into `button`, is copied together with it so the set compiles on its own. The copies are generated from the same source files the `github.com/vivaui/viva/components` package builds, so they match it. Importing `github.com/vivaui/viva/components` without calling a recipe links nothing from it, so copy a component only when you need to change more than its options allow. Every component page in the catalog has a **Download** link for its source file and its install command.
