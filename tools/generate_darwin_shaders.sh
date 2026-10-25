#!/bin/sh

set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
shader_dir="$root/internal/platform/darwin/shaders"
air="$root/build/scene.air"

mkdir -p "$root/build"

xcrun --sdk macosx metal \
    -c "$shader_dir/scene.metal" \
    -o "$air"
xcrun --sdk macosx metallib \
    "$air" \
    -o "$shader_dir/scene.metallib"

unlink "$air"
