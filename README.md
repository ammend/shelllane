# ShellLane

**[中文文档](README.zh-CN.md)**

ShellLane is a shell-compatible execution router. It sits between an upstream
launcher and a real `bash`, `zsh`, or `sh`, then selects an execution machine,
runtime, and isolated user state from a trusted identity.

## Why

Many applications invoke Agent CLIs—Codex, Claude Code, Trae Agent, and
others—through ordinary system shell commands. That keeps integrations simple,
but couples every request to the launcher's machine, HOME, credentials, and
workspace.

ShellLane preserves that command-line contract while moving execution policy
into a small, reusable middle layer:

- a developer can run on their own machine through SSH;
- ordinary users can run in isolated local or remote containers;
- administrators can use a trusted local host runtime;
- the upstream application and Agent CLI require no vendor-specific adapter.

ShellLane is not a bot gateway or Agent framework. It routes shell execution.

![ShellLane architecture](docs/diagrams/shelllane-architecture.png)

## Backends

| Connector | Runtime | Result |
| --- | --- | --- |
| `local` | `host` | Real shell on the launcher machine |
| `local` | `container` | Local Docker/Podman sandbox |
| `ssh` | `host` | Real shell on a remote development machine |
| `ssh` | `container` | Docker/Podman sandbox on a remote worker |

Targets combine placement (`node` or `nodePool`) with a runtime. Ordered routes
map exceptional identities explicitly and send everyone else to a required
default target.

## Start here

1. Read the [architecture and request flow](docs/architecture.md).
2. Pick a copy-ready configuration from [`.configs/`](.configs/README.md).
3. Follow the [configuration reference](docs/configuration.md).
4. Review the [security model](docs/security.md).

```bash
make build
sudo install -m 0755 bin/shelllane /usr/local/bin/shelllane
sudo install -d /usr/local/lib/shelllane
sudo ln -s /usr/local/bin/shelllane /usr/local/lib/shelllane/bash
sudo install -d -m 0700 -o "$USER" /var/lib/shelllane/users
sudo install -d -m 0700 -o "$USER" /var/log/shelllane

export SHELLLANE_CONFIG="$PWD/.configs/local-host.yaml"
export SHELLLANE_IDENTITY="user@example"
/usr/local/lib/shelllane/bash -lc 'printf "%s\n" "$HOME"'
```

Validate before deployment:

```bash
shelllane config check --config .configs/local-host.yaml
shelllane route test --config .configs/mixed.yaml --identity developer@example
shelllane doctor --config .configs/mixed.yaml
```

## Visual guides

![SSH and container request sequence](docs/diagrams/ssh-container-sequence.png)

GitHub renders the PNG previews directly. The same directory also contains
standalone interactive Archify HTML with dark/light themes and one-click
PNG/JPEG/WebP/SVG export. Maintainable JSON sources live in
[`docs/diagrams/source/`](docs/diagrams/source/).

## Integrations

Any launcher that can choose a shell executable and provide a trusted identity
can use ShellLane. For example, botmux can set `launchShell` to a ShellLane
symlink and map its authenticated sender to `SHELLLANE_IDENTITY`. The same
pattern applies to IDE services, internal automation, web terminals, and job
runners.

## Quality

```bash
make check
make configs-check
make integration
make docker-integration
make ssh-integration
```

CI exercises real local host, Docker, SSH-to-host, and SSH-to-Docker paths.

Apache-2.0. See [LICENSE](LICENSE).
