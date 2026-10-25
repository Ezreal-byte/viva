#!/bin/bash
# Run one command in a test environment and print its output.
#
#   exec.sh linux   'bash command'
#   exec.sh windows 'PowerShell command'
#   exec.sh cmd     'cmd.exe command line'    (Windows; use it when a program takes
#                                              -flags.with.dots, which PowerShell 5.1 splits)
#
# The call is synchronous and the environments cut it off after about a minute;
# start long work in the background and read its log. Output with control
# characters breaks the reply, so filter logs to printable text.
set -eu
. "$(dirname "$0")/env.sh"
case "$1" in
linux) url=$VIVA_LINUX_ENV ;;
windows | cmd) url=$VIVA_WINDOWS_ENV ;;
*) echo "first argument is linux, windows or cmd" >&2; exit 2 ;;
esac
if [ -z "$url" ]; then
	echo "set VIVA_LINUX_ENV and VIVA_WINDOWS_ENV, or put them in tools/testenv/local.sh" >&2
	exit 2
fi
python3 - "$1" "$2" "$url" <<'PY'
import json, sys, urllib.request
kind, text, url = sys.argv[1:4]
if kind == "linux":
    command = ["bash", "-c", text]
elif kind == "windows":
    command = ["powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", text]
elif kind == "cmd":
    command = ["cmd.exe", "/d", "/s", "/c", text]
else:
    sys.exit("first argument is linux, windows or cmd")
body = json.dumps({"command": command, "shell": False}).encode()
request = urllib.request.Request(url + "/execute", data=body, headers={"Content-Type": "application/json"})
reply = json.load(urllib.request.urlopen(request, timeout=120))
print(reply.get("output") or "", end="")
sys.exit(0 if reply.get("returncode") == 0 else 1)
PY
