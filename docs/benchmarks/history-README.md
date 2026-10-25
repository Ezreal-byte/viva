# Benchmark history

`benchmarks/history.sh` runs the benchmarks that guard Gop's paint and input paths
five times each and either records the medians (`record`) or compares them with the
history of the same machine (`check`). The data lives in
`docs/benchmarks/history/<machine>.jsonl`, one line per run, with the commit, whether
the tree was dirty, the Go version, and the machine: label, kind, OS, architecture
and the CPU line.

```sh
benchmarks/history.sh record m2-pro-darwin physical
MOTTY=../motty benchmarks/history.sh check m2-pro-darwin physical   # adds the workbench
go run ./tools/perfhistory steps -machine m2-pro-darwin -kind physical  # lasting changes in a history
```

Two runs on the same machine can also be compared without a history file, which
is how `.github/workflows/ci.yml` guards a pull request against its base
revision:

```sh
VIVA_ROOT=../base ./benchmarks/ci.sh /tmp/base.txt   # base worktree
./benchmarks/ci.sh /tmp/head.txt                     # candidate
go run ./tools/perfhistory compare -threshold 0.20 -min-ns 100 /tmp/base.txt < /tmp/head.txt
```

`compare` applies the same rule as `check` (a slowdown above the threshold and
above the absolute floor, or any new allocation; the CI runner uses 20% and
100 ns because it is shared and noisier than a quiet machine) and refuses to compare results from different CPUs, operating
systems or architectures. `benchmarks/ci.sh` runs the pure-Go scene, render,
input, text, wrap and painter benchmarks that are reproducible on a shared
runner; platform, GPU and window benchmarks stay out of this gate.

Rules the tool enforces:

- A result is compared only with results of the same machine label, kind (`physical`,
  `vm` or `software-gpu`), OS, architecture and CPU. A WARP or lavapipe number is
  never a point on a physical-GPU line.
- A slowdown is a finding when it is above the threshold (15%) **and** at least 20 ns,
  or when allocations per operation rise at all. A 3 ns benchmark moving by 1 ns is
  noise.
- `steps` looks for lasting changes anywhere in the series (the median of the next
  three runs against the previous three), not only the last run.

Record the first baseline of a machine when nothing else is running. A baseline taken
under load is worse than none: on the machine this was written on, with a load average
of about ten from other applications, the same code measured 35 us and 134 us for one
benchmark within minutes, and a `check` flagged regressions that were not there. No
history is committed yet for that reason.
