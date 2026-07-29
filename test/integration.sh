#!/usr/bin/env bash
set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

go build -trimpath -o "$test_dir/shelllane" "$project_dir/cmd/shelllane"
ln -s "$test_dir/shelllane" "$test_dir/sh"

sed \
  -e "s|STATE_ROOT|$test_dir/state|" \
  -e "s|AUDIT_PATH|$test_dir/audit.jsonl|" \
  "$project_dir/test/testdata/local.yaml" > "$test_dir/config.yaml"

export SHELLLANE_CONFIG="$test_dir/config.yaml"
export SHELLLANE_IDENTITY="integration-user"

output="$("$test_dir/sh" -c 'printf "%s|%s" "$HOME" "$XDG_CONFIG_HOME"')"
expected_home="$test_dir/state/$(printf integration-user | sha256sum | cut -d' ' -f1)/profiles/host/home"
expected_config="$test_dir/state/$(printf integration-user | sha256sum | cut -d' ' -f1)/shared/config"
test "$output" = "$expected_home|$expected_config"

set +e
"$test_dir/sh" -c 'exit 37'
status=$?
set -e
test "$status" -eq 37

"$test_dir/shelllane" config check --config "$test_dir/config.yaml" >/dev/null
"$test_dir/shelllane" route test --config "$test_dir/config.yaml" --identity anyone | grep '"target": "default"' >/dev/null
grep '"event":"dispatch"' "$test_dir/audit.jsonl" >/dev/null

echo "integration: shell argv, environment, routing, audit, and exit status passed"
