# ShellLane

ShellLane is an identity-aware shell wrapper. It looks like `bash`, `zsh`, or
`sh` to an existing application, but selects where that shell runs and which
user state it sees.

It is designed for launchers that already know the requesting user but should
not bind every request to the service host. A small list of
team members can use their own development machines, while all unlisted users
fall through to an isolated default runtime.

> Status: early development (`v0.1`). Review the [security model](docs/security.md)
> before exposing ShellLane to untrusted users.

## Principles

- Preserve the shell contract; do not add Codex-, Claude-, or agent-specific logic.
- Treat identity, placement, transport, and runtime as separate concerns.
- Enumerate exceptional users, not the potentially unbounded default population.
- Keep routing static and deterministic. ShellLane exposes an abstraction seam,
  but is not a general-purpose scheduler.
- Start with local/SSH transport and host/container runtimes.

## Model

```text
shell invocation
  └─ ShellLane (bash | zsh | sh)
       └─ identity → ordered route → target
                                    ├─ node or nodePool
                                    └─ host or container runtime
```

A node chooses the transport (`local` or `ssh`). A runtime chooses execution
(`host` or `container`). Their product provides four useful backends without
four separate implementations:

| Connector | Runtime | Result |
| --- | --- | --- |
| local | host | Shell on the service machine |
| local | container | Isolated local container |
| ssh | host | Shell on a developer/worker machine |
| ssh | container | Container on a developer/worker machine |

Node pools use rendezvous hashing, so an ordinary user is pinned
deterministically to one of several machines. This is placement, not a dynamic
capacity scheduler.

## Quick start

Requirements: Go 1.24 or newer. Docker/Podman and SSH are only needed for their
respective backends.

```bash
make build
cp config.example.yaml ~/.config/shelllane/config.yaml
mkdir -p ~/.local/lib/shelllane/users ~/.local/bin/shelllane-shells
ln -s "$PWD/bin/shelllane" ~/.local/bin/shelllane-shells/bash
ln -s "$PWD/bin/shelllane" ~/.local/bin/shelllane-shells/zsh
ln -s "$PWD/bin/shelllane" ~/.local/bin/shelllane-shells/sh

SHELLLANE_CONFIG="$HOME/.config/shelllane/config.yaml" \
SHELLLANE_IDENTITY=user@example \
~/.local/bin/shelllane-shells/bash -lc 'echo "$HOME"'
```

Validate configuration and preview a route before changing `launchShell`:

```bash
shelllane config check --config /etc/shelllane/config.yaml
shelllane route test --identity ou_example --config /etc/shelllane/config.yaml
shelllane doctor --config /etc/shelllane/config.yaml
```

## Identity and state

The default trusted identity is `SHELLLANE_IDENTITY`. An argv-style
`SHELLLANE_IDENTITY=value` fallback can be enabled for launchers that cannot
set process environment; ShellLane consumes that argument before invoking the
real shell. If both sources exist and disagree, execution fails closed.

Raw identities are never used in filesystem paths. A SHA-256 digest selects:

```text
<state-root>/<identity-hash>/
├── shared/
│   ├── config/
│   ├── credentials/
│   ├── data/
│   ├── state/
│   └── workspace/
├── profiles/<runtime-profile>/
│   ├── home/
│   └── cache/
└── tmp/
```

This state lives on the execution node. With a node pool, rendezvous hashing
keeps a user on the same node. Moving state between nodes is deliberately
outside v0.1; use shared storage or an external synchronization mechanism if
mobility is required.

## Configuration

See [configuration.md](docs/configuration.md) for a field-by-field reference,
[config.example.yaml](config.example.yaml) for a complete non-secret example,
and [architecture.md](docs/architecture.md) for execution details.
Ordered routes are first-match-wins and must end in a default route.

Container images are explicit configuration. Production deployments should
pin an immutable digest, use a non-root image user, and grant only the mounts
and network access that workload needs.

## Compatibility

ShellLane passes the original shell argv through unchanged and uses
`exec(2)` for the final process. TTY/stdin/stdout/stderr therefore remain
attached end-to-end. SSH forces or disables a remote PTY to match the incoming
invocation. See [compatibility.md](docs/compatibility.md) for the exact contract
and current boundaries.

## Development

```bash
make check
make integration
```

The project uses the standard Go toolchain and has no runtime Go dependencies
except YAML parsing. See [CONTRIBUTING.md](CONTRIBUTING.md).

## Non-goals for v0.1

- Bot request distribution, chat protocol handling, or device registration
- A reverse-connected agent or public control-plane port
- Kubernetes integration or a dynamic scheduler
- Credential synchronization across machines
- Web UI, database, queues, or agent-specific adapters

## License

Apache License 2.0. See [LICENSE](LICENSE).

## Integrations

ShellLane is launcher-agnostic. For botmux, configure a bot's custom
`launchShell` as the matching ShellLane symlink, such as
`/opt/shelllane/bin/bash`, and map the authenticated sender identity into
`SHELLLANE_IDENTITY`. This is one integration example, not part of ShellLane's
architecture.
