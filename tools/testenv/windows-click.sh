#!/bin/bash
# Click on the Windows test VM's desktop.   windows-click.sh X Y [left|right|move|double|drag X2 Y2]
# Windows 11 keeps a new tray icon in the overflow flyout: click the arrow near
# 1725,1056 first, then the icon in the flyout (about 1686,995).
set -eu
here=$(dirname "$0")
"$here/upload.sh" windows "$here/windows-click.ps1" 'C:\Users\Docker\Desktop\windows-click.ps1'
"$here/exec.sh" cmd "powershell -NoProfile -ExecutionPolicy Bypass -File C:\\Users\\Docker\\Desktop\\windows-click.ps1 $1 $2 ${3:-left} ${4:-0} ${5:-0}"
