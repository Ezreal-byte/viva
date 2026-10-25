<!-- One kind of change per pull request, with a message that says what changed
and why. See CONTRIBUTING.md. -->

## What this changes

## Why

<!-- The problem it solves. If it fixes an issue, link it. -->

## How it was verified

<!-- Which of these ran, and what they printed. -->

- [ ] `test -z "$(gofmt -l .)"`
- [ ] `go vet ./...`
- [ ] `go test ./... -count=1`
- [ ] platform vet, if the backends are touched
- [ ] benchmarks before and after, if a paint, layout, or input path changed
- [ ] stripped binary size before and after, if what links changed
- [ ] a real window on: <!-- macOS / Windows / Linux — list what you actually ran -->

## Where the boundary sits

<!-- Does this add framework capability, or product behaviour? If it belongs to
an application, say why it is in the framework anyway. Reusable capability goes
in Viva; a protocol, a storage schema, or one application's workflow does not. -->

## Cost

<!-- What a program that does not use this feature pays. Nothing is the expected
answer. -->
