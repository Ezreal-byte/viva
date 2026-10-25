#!/bin/sh

# Build the web gallery into build/web with viva build and serve it locally.

set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
out="$root/build/web"

cd "$root"
go run ./cmd/viva build --target web -o "$out"

printf 'serving http://127.0.0.1:8090 (ctrl-c to stop)\n'
cd "$out" && python3 -m http.server 8090
