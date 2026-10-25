#!/bin/bash
# Cross-compile a package's tests, run them in a test environment, print the
# verdicts. Neither environment has Go; the test binary does not need it.
#
#   go-test.sh linux|windows ./internal/pkg [test flags]     (run from a Go module)
#
# Tests that open windows need the desktop session, which the environments have,
# and on Linux a Vulkan driver: the private copy under /tmp/viva-vulkan when there is
# one, otherwise the system's (mesa-vulkan-drivers). With neither, graphics tests skip.
set -eu
here=$(dirname "$0")
platform=$1 package=$2
shift 2
name=$(echo "$package" | tr '/.' '__')
out=$(mktemp -d)
case "$platform" in
linux)
	GOOS=linux GOARCH=amd64 go test -c -o "$out/t" "$package"
	"$here/upload.sh" linux "$out/t" "/home/user/gotest-$name"
	"$here/exec.sh" linux "chmod +x /home/user/gotest-$name; mkdir -p /tmp/gotest-home /tmp/gotest-run-$name; cd /tmp/gotest-run-$name; export HOME=/tmp/gotest-home SHELL=/bin/bash DISPLAY=:1 XAUTHORITY=/home/user/.Xauthority; if [ -f /tmp/viva-vulkan/lvp-local.json ]; then export VK_ICD_FILENAMES=/tmp/viva-vulkan/lvp-local.json LD_LIBRARY_PATH=/tmp/viva-vulkan/root/usr/lib/x86_64-linux-gnu; fi; timeout 100 /home/user/gotest-$name -test.count=1 $* 2>&1 | tail -40"
	;;
windows)
	GOOS=windows GOARCH=amd64 go test -c -o "$out/t.exe" "$package"
	"$here/upload.sh" windows "$out/t.exe" "C:\\Users\\Docker\\Desktop\\gotest-$name.exe"
	# cmd.exe runs it, because PowerShell 5.1 splits -test.run=x at the dot. The log
	# is then reduced to its verdict lines in printable text, since escape sequences
	# in a log break the reply.
	"$here/exec.sh" cmd "cd /d C:\\Users\\Docker\\Desktop && gotest-$name.exe -test.count=1 $* > gotest-$name.log 2>&1" || true
	"$here/exec.sh" windows "\$bytes = [IO.File]::ReadAllBytes('C:\\Users\\Docker\\Desktop\\gotest-$name.log'); \$text = -join (\$bytes | ForEach-Object { if (\$_ -ge 32 -and \$_ -lt 127 -or \$_ -eq 10) { [char]\$_ } else { '?' } }); (\$text -split [char]10 | Where-Object { \$_ -match '^(--- FAIL|FAIL|PASS|ok|panic)' }) -join [char]10"
	;;
*)
	echo "first argument is linux or windows" >&2
	exit 2
	;;
esac
