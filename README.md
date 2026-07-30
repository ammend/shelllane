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
sudo make install
export SHELLLANE_IDENTITY="user@example"
/usr/local/lib/shelllane/bash -lc 'printf "%s\n" "$HOME"'
```

`make install` builds the binary, installs it under `/usr/local/bin`, creates
the `bash`, `zsh`, and `sh` wrapper links, and prepares state/log directories
for the invoking user. It also copies every `configs/*.yaml` template to the
user's `$HOME/.config/shelllane/` and initializes `config.yaml` from
`local-host.yaml` only when `config.yaml` does not already exist. Reinstalling
therefore preserves the active configuration. Under `sudo`, `SUDO_USER` and
that account's HOME are used instead of `root`. If sudo's `secure_path` cannot
find Go, the build step resolves Go from `SUDO_USER`'s login environment and
runs the compiler as that unprivileged user.
Install locations are customizable:

```bash
sudo make install PREFIX=/opt/shelllane RUNTIME_USER=agent
make install DESTDIR=/tmp/package-root RUNTIME_USER="$USER"
sudo make install CONFIG_DIR=/custom/xdg/shelllane
```

`make uninstall` removes only the binary and wrapper links; configuration,
state, and logs are deliberately preserved.

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
    SHELLLANE_IDENTITY:
      - "$BOTMUX_USER_ID"
      - "$IDE_USER_ID"
      - "$JOB_USER_ID"
  argvEnvFallback: true
  required: true
```

The list declares alternative upstream applications in priority order. One
non-empty source is enough; multiple equal values are accepted, while
different values fail closed with `identity_conflict`. The original scalar
form remains valid for one source.

The mapped source name is also accepted in argv compatibility mode:

```bash
/usr/local/lib/shelllane/bash BOTMUX_USER_ID=alice -lc 'pwd'
```

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
Bash, restores the same assignment in its process environment, and leaves all
existing environment entries unchanged. The remaining argv is exactly
`bash -lc 'agent-cli ...'`. If environment and argv identities both exist but
disagree, execution fails closed with `identity_conflict`.

## State directory names

```yaml
state:
  root: /var/lib/shelllane/users
  identityNaming: plain
```

The supported modes are `plain` (default and readable), `base64url`
(reversible, path-safe encoding), and `hash` (irreversible SHA-256). Base64 is
not encryption. Audit and node-pool selection use SHA-256 in every mode.

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
