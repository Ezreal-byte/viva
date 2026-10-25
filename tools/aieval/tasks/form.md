---
title: Sign-up form
text: Name
text: Email
text: Password
text: Create account
hits: 4
click: Create account
text: Welcome
not-text: I agree
layout: yes
---
A sign-up page, 480x560 window: a title, a one-line description under it, then labelled text fields for Name, Email and Password (use f.Input with components.InputSnapshot; the password one Secure), a checkbox 'I agree to the terms' with its label beside it, and at the bottom right a Cancel (outline) and a Create account (default) button. Clicking Create account sets m.created=true and the page then shows a 'Welcome' line instead of the form.
