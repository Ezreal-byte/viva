---
title: File tree sidebar with detail pane
text: docs
text: src
text: go.mod
text: Select a file
not-text: main.go
hits: 3
click: src
text: main.go
text: util.go
not-text: button.go
click: components
text: button.go
click: button.go
text: src/components/button.go
text: Size: 3 KB
click: util.go
text: src/util.go
text: Size: 7 KB
click: src
not-text: main.go
not-text: components
layout: yes
---
A file browser, 900x560. On the left a 260 px wide sidebar showing a tree: at the top level 'docs', 'src' and 'go.mod'; docs holds 'readme.md' (4 KB); src holds 'components' (a folder holding 'button.go', 3 KB), 'main.go' (2 KB) and 'util.go' (7 KB); go.mod is 1 KB. Folders show a small chevron and start collapsed; clicking a folder row expands or collapses it, and its children appear below it indented one level more than the folder, so the tree can nest. Files and folders are clickable rows with a hover background, the selected file highlighted. A vertical divider, then the detail pane taking the rest: at first the muted text 'Select a file'; once a file is clicked it shows the file name as a heading, its full path such as 'src/util.go' and a line 'Size: 7 KB'. Collapsing a folder hides everything below it, including nested open folders.
