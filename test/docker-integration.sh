#!/usr/bin/env bash
set -euo pipefail

if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
  echo "docker integration: skipped (Docker daemon unavailable)"
  exit 0
fi

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

go build -trimpath -o "$test_dir/shelllane" "$project_dir/cmd/shelllane"
ln -s "$test_dir/shelllane" "$test_dir/sh"

sed \
  -e "s|STATE_ROOT|$test_dir/state|" \
  -e "s|AUDIT_PATH|$test_dir/audit.jsonl|" \
  -e "s|CONTAINER_USER|$(id -u):$(id -g)|" \
  "$project_dir/test/testdata/docker.yaml" > "$test_dir/config.yaml"

export SHELLLANE_CONFIG="$test_dir/config.yaml"
export SHELLLANE_IDENTITY="docker-integration-user"
export SHELLLANE_CONTEXT_REQUEST="allowed-context"
export SHELLLANE_TEST_SECRET="must-not-pass"

identity_hash="$(printf docker-integration-user | sha256sum | cut -d' ' -f1)"
workspace="$test_dir/state/$identity_hash/shared/workspace"

output="$("$test_dir/sh" -c '
  test "$HOME" = /home/agent
  test "$XDG_CONFIG_HOME" = /home/agent/.config
  test "$SHELLLANE_CONTEXT_REQUEST" = allowed-context
  test "$SHELLLANE_IDENTITY" = docker-integration-user
  test -z "${SHELLLANE_TEST_SECRET:-}"
  test "$(id -u)" = "'"$(id -u)"'"
  test "$(ls /sys/class/net)" = lo
  ! touch /root-filesystem-must-be-read-only
  printf persisted > /workspace/docker-marker
  printf docker-ok
')"

test "$output" = "docker-ok"
test "$(cat "$workspace/docker-marker")" = "persisted"

set +e
"$test_dir/sh" -c 'exit 37'
status=$?
set -e
test "$status" -eq 37

grep '"event":"dispatch"' "$test_dir/audit.jsonl" >/dev/null
test -z "$(docker ps -aq --filter name=shelllane-)"

echo "docker integration: runtime, mounts, env allowlist, network, read-only root, cleanup, and exit status passed"
