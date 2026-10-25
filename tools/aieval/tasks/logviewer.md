---
title: Log viewer with level chips
source: Mono
text: 60 lines
text: [INFO] message 1
text: Following the tail
hits: 6
click: Error
text: 12 lines
text: [ERROR] message 5
not-text: [INFO]
not-text: [WARN]
click: Warn
text: 16 lines
text: [WARN] message 3
not-text: [ERROR]
click: All
text: 60 lines
click: Auto-scroll
text: Paused
not-text: Following the tail
click: Clear
text: No log lines
text: 0 lines
layout: yes
---
A log viewer, 800x520, for 60 sample lines that the page generates itself: line n reads '[LEVEL] message n' where the level is ERROR when n is divisible by 5, otherwise WARN when n is divisible by 3, otherwise INFO. At the top a row with filter chips All, Info, Warn, Error (rounded pill buttons, one selected at a time, All at start, the selected one filled), a Switch labelled 'Auto-scroll' that is on at start with a muted text beside it, 'Following the tail' when on and 'Paused' when off, and at the far right a 'Clear' outline button. Below, a scrolling area with the lines that pass the filter in a monospace font, one line each, and under it a muted status line such as '60 lines' or '12 lines' (the count of shown lines). With auto-scroll on the view sticks to the last line; the mouse wheel scrolls it. Clear removes all lines and shows 'No log lines' centered.
