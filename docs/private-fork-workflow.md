# Private fork workflow

This repository is used as a private fork workspace. Local changes must not be
published to `QuantumNous/new-api`.

## Rules

- Push branches only to `origin` (`Fankouzu/new-api-offical`).
- Create pull requests only in `Fankouzu/new-api-offical`.
- Do not run bare `gh pr create`, because GitHub CLI can infer the upstream
  repository from fork metadata.

## Setup

Run once in each checkout:

```bash
scripts/install-private-fork-guards.sh
```

This installs local Git guards by setting `core.hooksPath=githooks` and disables
pushes to the `upstream` remote by setting `remote.upstream.pushurl=DISABLED`.

## Creating PRs

Use:

```bash
scripts/create-private-pr.sh --base main --head <branch> --title "<title>" --body "<body>"
```

The script always passes `-R Fankouzu/new-api-offical` to GitHub CLI and rejects
repository override arguments.
