#!/bin/sh

# Lightweight development reload: watch Go sources, rebuild, and restart the
# target command. This is a stopgap until the process-isolated reload host
# exists; state is not preserved across restarts.

set -eu

if [ "$#" -eq 0 ]; then
    echo "usage: dev_watch.sh ./cmd/showcase [args...]" >&2
    exit 1
fi

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
package="$1"
shift || true
binary="$root/build/dev-watch-app"

if ! command -v fswatch >/dev/null 2>&1; then
    echo "dev_watch: fswatch is required (brew install fswatch)" >&2
    exit 1
fi

pid=0

restart() {
    if [ "$pid" -ne 0 ] && kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    fi
    if ! (cd "$root" && go build -o "$binary" "$package"); then
        echo "dev_watch: build failed; keeping previous binary down" >&2
        pid=0
        return
    fi
    "$binary" "$@" &
    pid=$!
    echo "dev_watch: running $package (pid $pid)"
}

trap 'if [ "$pid" -ne 0 ]; then kill "$pid" 2>/dev/null || true; fi' EXIT INT TERM

restart

fswatch -0 "$root" --event Created --event Updated \
    -e 'build/' -e '.git/' -e '*_test.go' | while read -r _; do
    restart
done
