#!/bin/sh
# Runs the benchmarks that guard Viva's paint and input paths and either records
# them in the history of this machine or checks them against it.
#
#   benchmarks/history.sh record m2-pro physical
#   benchmarks/history.sh check  m2-pro physical
#
# The history of a machine is docs/benchmarks/history/<label>.jsonl. Results of
# different machine labels, CPUs, operating systems or kinds (physical, vm,
# software-gpu) are never compared. Run it on a quiet machine: each benchmark runs
# five times and the median is kept.
set -eu

mode=${1:?usage: history.sh record|check LABEL physical|vm|software-gpu}
label=${2:?machine label}
kind=${3:?physical, vm or software-gpu}
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

output=$(mktemp)
trap 'rm -f "$output"' EXIT

run() {
    go test -C "$root" "$1" -run '^$' -bench "$2" -benchmem -count=5 -benchtime=2000x >>"$output"
}
run ./ui 'Benchmark(DrawIcon|RowIdentity|MountID|IDByWord)'
run ./app 'Benchmark(ClickThrough|PointerMove|PainterButtonPointer)'
run ./internal/input 'Benchmark(HitTest10KMiss|FocusNext10K)$'

if [ -n "${MOTTY:-}" ]; then
    go test -C "$MOTTY" ./internal/workbench -run '^$' -bench 'BenchmarkWholeWindowPaint' -benchmem -count=5 -benchtime=300x >>"$output"
fi

go run -C "$root" ./tools/perfhistory "$mode" -machine "$label" -kind "$kind" -dir "$root/docs/benchmarks/history" <"$output"
