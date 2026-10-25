---
title: Focus timer with pause and reset
text: Focus timer
text: 25:00
text: Ready
hits: 2
click: Start
wait: 3500
text: 24:5
text: Running
click: Pause
text: Paused
wait: 3000
not-text: 24:4
click: Resume
text: Running
click: Reset
text: 25:00
text: Ready
layout: yes
---
A focus timer, 420x360: a title 'Focus timer', the time left in large type as 'MM:SS' starting at '25:00', under it a status line 'Ready', a progress bar showing how much of the 25 minutes has passed, and a row of buttons: 'Start', 'Reset' (outline). While it runs the time counts down by the animation clock (one second per second), the status says 'Running' and the Start button reads 'Pause'; pausing keeps the time and says 'Paused', and the same button then reads 'Resume'. Reset stops it, puts the time back to 25:00 and the status back to 'Ready'. When it reaches 00:00 it stops and says 'Time is up'.
