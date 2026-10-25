---
title: Three-step wizard with validation
text: Step 1 of 3
text: Account
hits: 2
click: Next
text: Name is required
text: Step 1 of 3
click: Name
type: Ada
click: Next
text: Step 2 of 3
click: Pro
click: Next
text: Step 3 of 3
text: Ada
text: Pro
click: Back
text: Step 2 of 3
click: Next
click: Finish
text: All done
not-text: Step 3 of 3
layout: yes
---
A sign-up wizard, 560x480: a title 'Set up your workspace', under it a line 'Step N of 3' and three step names in a row (Account, Plan, Review) with the current one bold. Step 1 'Account' has one text field 'Name'. Step 2 'Plan' lets the user pick one of three plans, Free, Pro or Team, as clickable cards (the picked one highlighted; nothing is picked at start, Free is not preselected). Step 3 'Review' lists the entered name and the chosen plan as 'Name: ...' and 'Plan: ...'. At the bottom right a Back (outline) button, hidden on step 1, and a Next (primary) button, which on the last step reads Finish. Next on step 1 with an empty name does not advance and shows a red 'Name is required' under the field; Next on step 2 without a plan shows 'Pick a plan' and stays. Finish replaces the whole wizard with the line 'All done'.
