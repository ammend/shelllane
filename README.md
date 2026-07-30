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
2. Pick a copy-ready template from [`configs/`](configs/README.md).
3. Follow the [configuration reference](docs/configuration.md).
4. Review the [security model](docs/security.md).

```bash
make build
sudo install -m 0755 bin/shelllane /usr/local/bin/shelllane
sudo install -d /usr/local/lib/shelllane
sudo ln -s /usr/local/bin/shelllane /usr/local/lib/shelllane/bash
sudo install -d -m 0700 -o "$USER" /var/lib/shelllane/users
sudo install -d -m 0700 -o "$USER" /var/log/shelllane

mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/shelllane"
cp configs/local-host.yaml \
  "${XDG_CONFIG_HOME:-$HOME/.config}/shelllane/config.yaml"
export SHELLLANE_IDENTITY="user@example"
/usr/local/lib/shelllane/bash -lc 'printf "%s\n" "$HOME"'
```

Validate before deployment:

```bash
shelllane config check --config configs/local-host.yaml
shelllane route test --config configs/mixed.yaml --identity developer@example
shelllane doctor --config configs/mixed.yaml
```

## Template directory vs runtime configuration

`<repo>/configs/*.yaml` contains copy-ready templates. ShellLane does not use
that repository directory as its default runtime location.

The user runtime path follows the XDG Base Directory convention:

```text
$XDG_CONFIG_HOME/shelllane/config.yaml
# or, when XDG_CONFIG_HOME is unset:
$HOME/.config/shelllane/config.yaml
```

System services may use `/etc/shelllane/config.yaml`.

## Map an upstream identity

The launcher does not need to synthesize `SHELLLANE_IDENTITY`. Map the trusted
variable it already exports into ShellLane's canonical identity:

```yaml
identity:
  env: SHELLLANE_IDENTITY
  envMap:
    SHELLLANE_IDENTITY: "$BOTMUX_USER_ID"
  required: true
```

If both variables exist and disagree, ShellLane fails closed with
`identity_conflict`.

## Identity appended after Bash (compatibility)

Some launchers append metadata after the shell executable:

```bash
/path/to/bash SHELLLANE_IDENTITY=user@example -lc 'agent-cli ...'
```

Enable:

```yaml
identity:
  env: SHELLLANE_IDENTITY
  argvEnvFallback: true
  required: true
```

ShellLane consumes the matching `KEY=value` item before executing the real
Bash. The remaining argv is exactly `bash -lc 'agent-cli ...'`. If environment
and argv identities both exist but disagree, execution fails closed with
`identity_conflict`.

## Visual guides

![SSH and container request sequence](docs/diagrams/ssh-container-sequence.png)

GitHub renders the PNG previews directly. The same directory also contains
standalone interactive Archify HTML with dark/light themes and one-click
PNG/JPEG/WebP/SVG export. Maintainable JSON sources live in
[`docs/diagrams/source/`](docs/diagrams/source/).

## Integrations

Any launcher that can choose a shell executable and provide a trusted identity
can use ShellLane. For example, botmux can set `launchShell` to a ShellLane
symlink; ShellLane can map botmux's existing `BOTMUX_USER_ID` to its canonical
identity in configuration. The same
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
