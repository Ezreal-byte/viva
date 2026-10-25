#!/bin/bash
# Save a 1920x1080 PNG of a test environment's desktop.   screenshot.sh linux|windows OUT.png
set -eu
. "$(dirname "$0")/env.sh"
case "$1" in
linux) url=$VIVA_LINUX_ENV ;;
windows) url=$VIVA_WINDOWS_ENV ;;
*) echo "first argument is linux or windows" >&2; exit 2 ;;
esac
if [ -z "$url" ]; then
	echo "set VIVA_LINUX_ENV and VIVA_WINDOWS_ENV, or put them in tools/testenv/local.sh" >&2
	exit 2
fi
curl -sf -m 30 -o "$2" "$url/screenshot"
