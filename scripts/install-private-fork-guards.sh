#!/usr/bin/env bash
set -euo pipefail

git config remote.upstream.pushurl DISABLED
git config core.hooksPath githooks

if command -v gh >/dev/null 2>&1; then
  gh repo set-default Fankouzu/new-api-offical >/dev/null
fi

echo "Installed private fork guards:"
echo "- remote.upstream.pushurl=DISABLED"
echo "- core.hooksPath=githooks"
echo "- gh repo default=Fankouzu/new-api-offical"
echo
echo "Use scripts/create-private-pr.sh instead of bare 'gh pr create'."
