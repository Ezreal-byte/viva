# Releases

Viva ships as a Go module tag: there are no prebuilt binaries and no release
assets. A release is an annotated tag on `main` plus the GitHub Release that
carries the notes, and `.github/workflows/release.yml` publishes that Release
once the tag is pushed.

## Who owns what

The releasing agent owns the tag and the notes. The workflow owns going public:
it runs the full gate from `ci.yml`, checks the draft the agent left behind, and
flips it from draft to published. Nothing is published by a local command.

1. The agent writes the bilingual notes for `v<version>` to a file outside the
   repository, then shows them, the exact `main` HEAD, the gate results and the
   notes digest at a confirmation gate.
2. After confirmation the agent creates an annotated tag at that HEAD whose body
   is a short English summary, and pushes it.
3. The agent creates the draft release immediately after the push, so GitHub does
   not create a lightweight tag of its own:

   ```sh
   gh release create "v<version>" --repo vivaui/viva --verify-tag --draft \
     --title "Viva v<version>" --notes-file /path/to/notes.md
   ```

4. The tag push starts `release.yml`. Its `gate` job calls `ci.yml`, and only if
   that passes does the `publish` job run. It waits up to ten minutes for the
   draft, requires the title `Viva v<version>`, a non-empty body and no assets,
   then sets `draft=false` (and `prerelease=true` for a tag that contains `-`).
5. The agent verifies the live release: body identical to the approved notes,
   published state, zero assets, annotated tag pointing at the release commit.
   Then it commits the three "current release" references (`README.md` and the
   two `getting-started` guides) and records the release in its ledger.

A release is a straight sequence: the notes are written before the tag, and the
tag never waits on a background notes task.

## What the workflow cannot check

- Platform conformance. `cmd/viva-conformance` needs a real desktop on macOS, a
  Windows VM and a Linux X session; the release runner has none of them. Run
  `tools/conformance/run.sh` from [conformance.md](conformance.md) before
  tagging, and record what was not run in the confirmation gate.
- The release notes' wording and their consistency with the committed tree. The
  workflow can only require a non-empty body; the agent compares the published
  body with the approved notes byte for byte afterwards.
- The three "current release" references. They are a separate commit after the
  release exists, so the published notes can be the only place that names a
  version for a few minutes.

## Rerunning

`workflow_dispatch` takes a tag, so a tag whose workflow failed can be retried
without moving it:

```sh
gh workflow run release.yml -f tag=v<version>
```

A release that is already published is left alone (the job records "already
published" and skips). If the draft was never created the publish job fails
after ten minutes with the `gh release create --draft` command to run.

## Rolling back

Before anything consumes the tag, delete the release and then the tag:

```sh
gh release delete "v<version>" --yes
git push origin ":refs/tags/v<version>"
git tag -d "v<version>"
```

Once a downstream `go.mod` has pinned the version, do not delete the tag. Ship a
patch release, explain the problem in its notes, and only mark the old release
as a prerelease if it must stop resolving as latest. After either path, check
that `releases/latest` and the three version references point at the release you
meant to keep.
