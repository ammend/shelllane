# Shell compatibility

ShellLane is selected through the executable basename. Symlinks named `bash`,
`zsh`, and `sh` all point to the same binary.

For a host runtime ShellLane calls `exec(2)` with:

- the configured real shell as the executable path;
- the original argv, including its original `argv[0]`;
- the original environment with HOME/XDG/TMP state variables replaced;
- the selected work directory; and
- the original stdin, stdout, stderr, signals, and controlling terminal.

For containers the real shell is the container command and every argument after
the original `argv[0]` is preserved. For SSH, PTY allocation follows the local
TTY and the remote helper repeats the same host/container execution.

The optional identity argv item is metadata for launchers that cannot set
environment variables. It is consumed and is not forwarded to the real shell.

## Compatibility policy

Configuration schema and remote request payloads are explicitly versioned.
Patch releases preserve schema v1. New optional fields may be added in minor
releases. Breaking schema/protocol changes require a new version and a migration
note. A sender must not assume an upgraded remote helper; deploy helpers before
using new runtime fields.

## Current limits

- ShellLane supports Linux process semantics.
- A requested host cwd is mapped by the runtime's `workdir`; the recommended
  isolated value is `{workspace}`.
- Shell startup files naturally come from the selected isolated HOME.
- Docker and Podman CLI compatibility is limited to the flags emitted by
  ShellLane and tested in CI at argument-contract level.
