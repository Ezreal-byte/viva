#!/bin/sh
# Materialize the pinned shadcn/ui reference, outside /tmp, so a catalog
# audit always reads the same commit instead of whatever main is today.
#
#   tools/shadcnpin/fetch.sh            # into ~/.cache/viva/shadcn-ui-<commit>
#   tools/shadcnpin/fetch.sh /some/dir  # into a directory of your choice
#
# Prints the directory. Read geometry and variants from
# <dir>/apps/v4/registry/styles/style-vega.css and behavior from
# <dir>/apps/v4/registry/bases/base/ui/*.tsx. Network access goes through the
# usual https_proxy/all_proxy variables.
set -eu

# Keep in step with the pinned shadcn reference.
COMMIT=30a58d4498f270af1c1e981a7d78a15cf5740841

dest=${1:-${XDG_CACHE_HOME:-$HOME/.cache}/viva/shadcn-ui-$(printf %.12s "$COMMIT")}
if [ -f "$dest/.pinned-commit" ] && [ "$(cat "$dest/.pinned-commit")" = "$COMMIT" ]; then
	echo "$dest"
	exit 0
fi
if [ -e "$dest" ] && [ -n "$(ls -A "$dest" 2>/dev/null)" ]; then
	echo "fetch.sh: $dest exists and is not this pin; choose another directory" >&2
	exit 1
fi

mkdir -p "$dest"
cd "$dest"
git init -q
git remote add origin https://github.com/shadcn-ui/ui.git
git sparse-checkout init --cone
git sparse-checkout set apps/v4/registry apps/v4/content/docs/components
git fetch -q --depth 1 --filter=blob:none origin "$COMMIT"
git checkout -q FETCH_HEAD
test "$(git rev-parse HEAD)" = "$COMMIT"
printf '%s\n' "$COMMIT" >.pinned-commit
echo "$dest"
