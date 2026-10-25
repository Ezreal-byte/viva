---
title: Login form with validation and show password
text: Sign in
hits: 4
click: Sign in
text: Enter your email
text: Password must be at least 8 characters
click: Email
type: ana@example.com
click: Password
type: hunter22x
not-text: Enter your email
not-text: Password must be
not-text: hunter22x
click: Show password
text: hunter22x
click: Show password
not-text: hunter22x
click: Sign in
text: Signed in as ana@example.com
layout: yes
---
A login page, 440x420: a title 'Sign in', a field 'Email', a field 'Password' (its text hidden), a checkbox with the label text 'Show password' that reveals what was typed in the password field while it is checked, and a 'Sign in' button. Nothing is validated until Sign in is pressed. Pressing it with an empty email shows a red 'Enter your email' under the email field, and with a password shorter than 8 characters a red 'Password must be at least 8 characters' under the password field; each message disappears as soon as its field is valid (the check runs on every repaint after the first Sign in). The form is replaced by 'Signed in as <email>' only by a press of Sign in that finds both valid; typing alone never leaves the form.
