#!/bin/bash
# Build viva-conformance for a platform, run it there, and print its report.
#
#   tools/conformance/run.sh mac        run on this Mac (opens real windows)
#   tools/conformance/run.sh linux      run in the Linux test container
#   tools/conformance/run.sh windows    run in the Windows test VM
#   tools/conformance/run.sh all        all three, one after another
#
# The container and VM are machines you provide: set VIVA_LINUX_ENV and
# VIVA_WINDOWS_ENV to their addresses (or put them in tools/testenv/local.sh).
# Exit status is 1 if any platform reported a FAIL or could not be reached.
set -u
root=$(cd "$(dirname "$0")/../.." && pwd)
linux_env=${VIVA_LINUX_ENV:-}
windows_env=${VIVA_WINDOWS_ENV:-}
export no_proxy="${no_proxy:-},localhost,127.0.0.1"
out=$(mktemp -d)
status=0

# execute URL JSON: run a command through a test environment's /execute endpoint.
execute() {
	python3 - "$1" "$2" <<'PY'
import json, sys, urllib.request
request = urllib.request.Request(sys.argv[1] + "/execute", data=sys.argv[2].encode(), headers={"Content-Type": "application/json"})
reply = json.load(urllib.request.urlopen(request, timeout=300))
print(reply.get("output") or "", end="")
sys.exit(0 if reply.get("returncode") == 0 else 1)
PY
}

upload() { curl -sf -X POST "$1/setup/upload" -F "file_path=$2" -F "file_data=@$3" >/dev/null; }

run_mac() {
	(cd "$root" && go build -o "$out/mac" ./cmd/viva-conformance) && (cd "$out" && ./mac)
}

run_linux() {
	if [ -z "$linux_env" ]; then
		echo "set VIVA_LINUX_ENV to a Linux test environment" >&2
		return 1
	fi
	(cd "$root" && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o "$out/linux" ./cmd/viva-conformance) || return 1
	upload "$linux_env" /home/user/viva-conformance "$out/linux" || return 1
	# The container's X server is :1; its GPU is software Vulkan (lavapipe). The session bus
	# address changes with each login, so it is read from the desktop shell's environment.
	execute "$linux_env" '{"command":["bash","-c","chmod +x /home/user/viva-conformance; export DISPLAY=:1 XAUTHORITY=/home/user/.Xauthority; pid=$(pgrep -u user plasmashell | head -1); if [ -r /proc/$pid/environ ]; then export $(xargs -0 -n1 -a /proc/$pid/environ | grep ^DBUS_SESSION_BUS_ADDRESS=); fi; if [ -f /tmp/viva-vulkan/lvp-local.json ]; then export VK_ICD_FILENAMES=/tmp/viva-vulkan/lvp-local.json LD_LIBRARY_PATH=/tmp/viva-vulkan/root/usr/lib/x86_64-linux-gnu; fi; cd /tmp && timeout 50 /home/user/viva-conformance"],"shell":false}'
}

run_windows() {
	if [ -z "$windows_env" ]; then
		echo "set VIVA_WINDOWS_ENV to a Windows test environment" >&2
		return 1
	fi
	(cd "$root" && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o "$out/windows.exe" ./cmd/viva-conformance) || return 1
	upload "$windows_env" 'C:\Users\Docker\Desktop\viva-conformance.exe' "$out/windows.exe" || return 1
	# The report goes to a file: the program's own console is not the desktop's.
	execute "$windows_env" '{"command":["cmd.exe","/d","/s","/c","cd /d C:\\Users\\Docker\\Desktop && (viva-conformance.exe > conformance.txt 2>&1 & echo exit=%errorlevel% > conformance.exit) & type conformance.txt"]}'
}

run() {
	echo "== $1"
	local report
	report=$(run_"$1" 2>&1)
	echo "$report"
	if [ -z "$report" ] || echo "$report" | grep -q '^FAIL'; then
		status=1
	fi
}

case "${1:-}" in
mac | linux | windows) run "$1" ;;
all) run mac; run linux; run windows ;;
*) sed -n '2,12p' "$0"; exit 2 ;;
esac
exit $status
