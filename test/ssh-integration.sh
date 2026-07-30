#!/usr/bin/env bash
set -euo pipefail

for command_name in ssh sshd ssh-keygen ssh-keyscan docker; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "ssh integration: skipped ($command_name unavailable)"
    exit 0
  fi
done
if ! docker info >/dev/null 2>&1; then
  echo "ssh integration: skipped (Docker daemon unavailable)"
  exit 0
fi

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
sshd_pid=""
use_sudo=false
if test "$(id -u)" -ne 0; then
  if sudo -n true >/dev/null 2>&1; then
    use_sudo=true
  else
    echo "ssh integration: skipped (root or passwordless sudo required for ephemeral sshd)"
    exit 0
  fi
fi
cleanup() {
  if test -n "$sshd_pid"; then
    if $use_sudo; then
      sudo kill "$sshd_pid" >/dev/null 2>&1 || true
    else
      kill "$sshd_pid" >/dev/null 2>&1 || true
    fi
    wait "$sshd_pid" >/dev/null 2>&1 || true
  fi
  if $use_sudo; then
    sudo rm -rf "$test_dir"
  else
    chmod -R u+rwX "$test_dir" >/dev/null 2>&1 || true
    rm -rf "$test_dir"
  fi
}
trap cleanup EXIT

port="$(shuf -i 20000-40000 -n 1)"
while ss -ltn | grep -q ":$port "; do
  port="$(shuf -i 20000-40000 -n 1)"
done

go build -trimpath -o "$test_dir/shelllane" "$project_dir/cmd/shelllane"
ln -s "$test_dir/shelllane" "$test_dir/sh"
ssh-keygen -q -t ed25519 -N '' -f "$test_dir/host_key"
ssh-keygen -q -t ed25519 -N '' -f "$test_dir/client_key"
cp "$test_dir/client_key.pub" "$test_dir/authorized_keys"

sed \
  -e "s|AUTHORIZED_KEYS|$test_dir/authorized_keys|" \
  -e "s|HOST_KEY|$test_dir/host_key|" \
  -e "s|PID_FILE|$test_dir/sshd.pid|" \
  -e "s|SSH_PORT|$port|" \
  "$project_dir/test/testdata/sshd_config" > "$test_dir/sshd_config"

if $use_sudo; then
  sudo "$(command -v sshd)" -t -f "$test_dir/sshd_config"
  sudo "$(command -v sshd)" -D -e -f "$test_dir/sshd_config" >"$test_dir/sshd.log" 2>&1 &
else
  "$(command -v sshd)" -t -f "$test_dir/sshd_config"
  "$(command -v sshd)" -D -e -f "$test_dir/sshd_config" >"$test_dir/sshd.log" 2>&1 &
fi
sshd_pid=$!
for _ in $(seq 1 50); do
  if ssh-keyscan -p "$port" 127.0.0.1 >"$test_dir/known_hosts" 2>/dev/null; then
    break
  fi
  sleep 0.1
done
if ! kill -0 "$sshd_pid" 2>/dev/null || test ! -s "$test_dir/known_hosts"; then
  cat "$test_dir/sshd.log" >&2
  exit 1
fi

sed \
  -e "s|REMOTE_CONFIG|$test_dir/remote.yaml|" \
  -e "s|SHELLLANE_BINARY|$test_dir/shelllane|" \
  "$project_dir/test/testdata/remote-helper.sh" > "$test_dir/remote-helper"
chmod +x "$test_dir/remote-helper"

for template in ssh-client.yaml ssh-remote.yaml; do
  output_name="${template#ssh-}"
  sed \
    -e "s|STATE_ROOT|$test_dir/state|" \
    -e "s|AUDIT_PATH|$test_dir/audit.jsonl|" \
    -e "s|SSH_PORT|$port|" \
    -e "s|REMOTE_HELPER|$test_dir/remote-helper|" \
    -e "s|KNOWN_HOSTS|$test_dir/known_hosts|" \
    -e "s|CLIENT_KEY|$test_dir/client_key|" \
    -e "s|CONTAINER_USER|0:0|" \
    "$project_dir/test/testdata/$template" > "$test_dir/$output_name"
done

export SHELLLANE_CONFIG="$test_dir/client.yaml"
export SHELLLANE_CONTEXT_REQUEST="ssh-allowed"
export SHELLLANE_TEST_SECRET="must-not-pass"

export SHELLLANE_IDENTITY="ssh-host-user"
host_hash="$(printf ssh-host-user | sha256sum | cut -d' ' -f1)"
host_output="$("$test_dir/sh" -c '
  test "$SHELLLANE_CONTEXT_REQUEST" = ssh-allowed
  test -z "${SHELLLANE_TEST_SECRET:-}"
  printf host-persisted > "$XDG_STATE_HOME/host-marker"
  printf "%s|ssh-host-ok" "$HOME"
')"
test "$host_output" = "$test_dir/state/$host_hash/profiles/host/home|ssh-host-ok"
test "$(cat "$test_dir/state/$host_hash/shared/state/host-marker")" = host-persisted

set +e
"$test_dir/sh" -c 'exit 37'
status=$?
set -e
test "$status" -eq 37

export SHELLLANE_IDENTITY="ssh-container-user"
container_hash="$(printf ssh-container-user | sha256sum | cut -d' ' -f1)"
container_output="$("$test_dir/sh" -c '
  test "$HOME" = /home/agent
  test "$SHELLLANE_CONTEXT_REQUEST" = ssh-allowed
  test -z "${SHELLLANE_TEST_SECRET:-}"
  test "$(ls /sys/class/net)" = lo
  ! touch /root-filesystem-must-be-read-only
  printf container-persisted > /workspace/ssh-container-marker
  printf ssh-container-ok
')"
test "$container_output" = ssh-container-ok
test "$(cat "$test_dir/state/$container_hash/shared/workspace/ssh-container-marker")" = container-persisted
test -z "$(docker ps -aq --filter name=shelllane-)"

echo "ssh integration: localhost transport, remote helper, host/container runtimes, state, env, cleanup, and exit status passed"
