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
unset __OWNER_OPEN_ID BOTMUX_USER_ID IDE_USER_ID JOB_USER_ID UPSTREAM_USER_ID
export SHELLLANE_IDENTITY="integration-user"

output="$("$test_dir/sh" -c 'printf "%s|%s" "$HOME" "$XDG_CONFIG_HOME"')"
expected_home="$test_dir/state/$(printf integration-user | sha256sum | cut -d' ' -f1)/profiles/host/home"
expected_config="$test_dir/state/$(printf integration-user | sha256sum | cut -d' ' -f1)/shared/config"
test "$output" = "$expected_home|$expected_config"

# The launcher may expose its existing authenticated identity variable.
unset SHELLLANE_IDENTITY
export UPSTREAM_USER_ID="mapped-user"
mapped_output="$("$test_dir/sh" -c 'printf "%s" "$HOME"')"
mapped_home="$test_dir/state/$(printf mapped-user | sha256sum | cut -d' ' -f1)/profiles/host/home"
test "$mapped_output" = "$mapped_home"
unset UPSTREAM_USER_ID

# The mapped upstream variable may also be appended after the shell executable.
mapped_argv_output="$("$test_dir/sh" UPSTREAM_USER_ID=argv-mapped-user -c '
  test "$#" -eq 0
  printf "%s|%s" "$HOME" "$UPSTREAM_USER_ID"
')"
mapped_argv_home="$test_dir/state/$(printf argv-mapped-user | sha256sum | cut -d' ' -f1)/profiles/host/home"
test "$mapped_argv_output" = "$mapped_argv_home|argv-mapped-user"

# Botmux appends __OWNER_OPEN_ID after the shell command and working-directory
# argument. ShellLane must resolve it, remove only that metadata argv item, and
# restore it in the real shell environment without disturbing other arguments.
botmux_output="$("$test_dir/sh" -c '
  test "$1" = /workspace
  test "$2" = tail-argument
  printf "%s|%s" "$HOME" "$__OWNER_OPEN_ID"
' _ /workspace __OWNER_OPEN_ID=ou_botmux tail-argument)"
botmux_home="$test_dir/state/$(printf ou_botmux | sha256sum | cut -d' ' -f1)/profiles/host/home"
test "$botmux_output" = "$botmux_home|ou_botmux"

# Some launchers append metadata after the shell executable instead of setting
# process environment: sh SHELLLANE_IDENTITY=value -c ...
unset SHELLLANE_IDENTITY
argv_output="$("$test_dir/sh" SHELLLANE_IDENTITY=argv-user -c '
  test "$#" -eq 0
  printf "%s|argv-ok" "$HOME"
')"
argv_home="$test_dir/state/$(printf argv-user | sha256sum | cut -d' ' -f1)/profiles/host/home"
test "$argv_output" = "$argv_home|argv-ok"

export SHELLLANE_IDENTITY="integration-user"
set +e
"$test_dir/sh" SHELLLANE_IDENTITY=conflicting-user -c 'exit 0' 2>"$test_dir/conflict.log"
conflict_status=$?
set -e
test "$conflict_status" -ne 0
grep 'identity_conflict' "$test_dir/conflict.log" >/dev/null

set +e
"$test_dir/sh" -c 'exit 37'
status=$?
set -e
test "$status" -eq 37

"$test_dir/shelllane" config check --config "$test_dir/config.yaml" >/dev/null
"$test_dir/shelllane" route test --config "$test_dir/config.yaml" --identity anyone | grep '"target": "default"' >/dev/null
grep '"event":"dispatch"' "$test_dir/audit.jsonl" >/dev/null

echo "integration: shell argv, environment, routing, audit, and exit status passed"
