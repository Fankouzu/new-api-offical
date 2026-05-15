#!/usr/bin/env bash
set -euo pipefail

readonly PRIVATE_REPO="Fankouzu/new-api-offical"
readonly FORBIDDEN_REPO="QuantumNous/new-api"

usage() {
  cat <<'USAGE'
Usage:
  scripts/create-private-pr.sh [gh pr create options]

Creates a pull request only in Fankouzu/new-api-offical.

Examples:
  scripts/create-private-pr.sh --base main --head my-branch --title "Fix task logs" --body "..."
  scripts/create-private-pr.sh --fill
USAGE
}

previous_arg=""
for arg in "$@"; do
  case "${previous_arg}" in
    --head)
      if [[ "${arg}" == QuantumNous:* ]]; then
        echo "error: refusing to use a head branch from QuantumNous" >&2
        exit 2
      fi
      ;;
  esac

  case "$arg" in
    -R|--repo|--head-repo|-R=*|--repo=*|--head-repo=*)
      echo "error: repository override is not allowed; PRs must target ${PRIVATE_REPO}" >&2
      exit 2
      ;;
    --head=QuantumNous:*)
      echo "error: refusing to use a head branch from QuantumNous" >&2
      exit 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
  esac

  previous_arg="${arg}"
done

exec gh pr create -R "${PRIVATE_REPO}" "$@"
