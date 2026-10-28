#!/usr/bin/env bash
# Prints a hash of the whole working tree (tracked changes and untracked files,
# respecting .gitignore) without touching the real index. /tarea saves it after
# verifying; /cerrar-tarea compares it to skip `make check` when nothing changed.
set -euo pipefail

git_dir=$(git rev-parse --git-dir)
tmp_index=$(mktemp)
trap 'rm -f "$tmp_index"' EXIT

cp "$git_dir/index" "$tmp_index"
GIT_INDEX_FILE=$tmp_index git add -A
GIT_INDEX_FILE=$tmp_index git write-tree
