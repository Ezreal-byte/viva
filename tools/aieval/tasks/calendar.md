---
title: Month calendar with day selection
text: March 2026
text: Selected: 15 March 2026
text: 31
hits: 33
click: 20
text: Selected: 20 March 2026
click: Next
text: April 2026
text: Selected: none
text: 30
not-text: 31
click: Previous
click: Previous
text: February 2026
text: 28
not-text: 29
layout: yes
---
A month calendar, 420x480, no date library needed (the page knows these months): a header row with a 'Previous' button, the month title such as 'March 2026' centered, and a 'Next' button. Under it a weekday row Su Mo Tu We Th Fr Sa and the days as a 7-column grid, each day a clickable cell whose accessible name is just its number ('20') (1 March 2026 is a Sunday; March has 31 days, April 30 with 1 April a Wednesday, February 2026 has 28 days and starts on a Sunday). Days that are not part of the month leave their cell empty. The page starts on March 2026 with day 15 selected (highlighted); below the grid a line 'Selected: 15 March 2026'. Clicking a day selects it. Previous and Next move one month (any number of times) and clear the selection: the line then reads 'Selected: none'.
