#!/usr/bin/env bash
set -euo pipefail

GPT_REPO='/home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace'
MINIMAX_REPO='/home/manuel/workspaces/2026-04-08/sqleton-minitrace-minimax/go-minitrace'
GPT_PHASE1_COMMIT='7cc5370cb7f60fca8069642ef3d95d1c085686bc'
MINIMAX_PHASE1_COMMIT='5bf8958dd7a9cb4b850aa3a0f7f24ef1681d0b50'

TMP_GPT="$(mktemp -d /tmp/gpt-phase1-tree-XXXXXX)"
trap 'git -C "$GPT_REPO" worktree remove --force "$TMP_GPT" >/dev/null 2>&1 || true' EXIT

git -C "$GPT_REPO" worktree add --detach "$TMP_GPT" "$GPT_PHASE1_COMMIT" >/dev/null

printf 'GPT phase1 files\n'
find "$TMP_GPT/pkg/minitracecmd" -maxdepth 2 -type f | sort
printf '\nMiniMax phase1 files\n'
find "$MINIMAX_REPO/pkg/minitracecmd" -maxdepth 2 -type f | sort
printf '\nDiff summary\n'
diff -ru "$TMP_GPT/pkg/minitracecmd" "$MINIMAX_REPO/pkg/minitracecmd" || true
