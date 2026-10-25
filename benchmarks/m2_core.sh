#!/bin/sh

set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

printf '%s\n' '== input and focus =='
go test -C "$root" ./internal/input \
    -run '^$' \
    -bench 'Benchmark(HitTest10KMiss|FocusNext10K)$' \
    -benchmem \
    -count=1

printf '%s\n' '== text and IME state =='
go test -C "$root" ./text \
    -run '^$' \
    -bench 'Benchmark(CursorMoveMixedText|ReplaceComposition|MeasureCacheHit)$' \
    -benchmem \
    -count=1

printf '%s\n' '== semantic tree =='
go test -C "$root" ./internal/semantic \
    -run '^$' \
    -bench 'BenchmarkSemantic(NoChanges|LocalizedUpdate)$' \
    -benchmem \
    -count=1

if [ "$(go env GOOS)" = darwin ]; then
    printf '%s\n' '== CoreText shaping =='
    go test -C "$root" ./internal/platform/darwin \
        -run '^$' \
        -bench 'BenchmarkMeasureTextMixed$' \
        -benchmem \
        -count=1
fi
