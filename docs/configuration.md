# Configuration reference

ShellLane reads one YAML file. Resolution order is:

1. the path passed to `--config` for diagnostic commands;
2. `SHELLLANE_CONFIG`;
3. `/etc/shelllane/config.yaml`;
4. `$HOME/.config/shelllane/config.yaml`.

Unknown fields, invalid references, and unsupported values are rejected. Check a
file before deployment:

```bash
shelllane config check --config ./config.yaml
shelllane route test --config ./config.yaml --identity user@example
```

## Minimal local configuration

```yaml
version: 1

shells:
  bash: /bin/bash
  zsh: /bin/zsh
  sh: /bin/sh

identity:
  env: SHELLLANE_IDENTITY
  required: true

state:
  root: /var/lib/shelllane/users

routes:
  - target: local-host

nodes:
  this-machine:
    connector: local

runtimes:
  host:
    type: host

targets:
  local-host:
    node: this-machine
    runtime: host
```

The empty `match` on the only route is the required default. Every identity gets
its own isolated HOME, XDG directories, workspace, cache, and temporary area.

## Top-level fields

| Field | Required | Meaning |
| --- | --- | --- |
| `version` | yes | Configuration schema. The only supported value is `1`. |
| `shells` | for host use | Absolute real-shell paths keyed by `bash`, `zsh`, and `sh`. |
| `identity` | no | Identity source and validation behavior. |
| `state` | no | Per-user state root. |
| `routes` | yes | Ordered, first-match-wins routing rules. Must end in a default rule. |
| `nodes` | yes | Local or SSH-reachable execution machines. |
| `nodePools` | when referenced | Deterministic groups of nodes. |
| `runtimes` | yes | Host or container execution definitions. |
| `targets` | yes | Named combinations of placement and runtime. |
| `audit` | no | Local JSONL audit destination. |

All map keys are administrator-chosen stable names. Renaming a runtime profile
changes its profile-specific HOME/cache path; renaming a node-pool changes its
rendezvous-hash placement.

## `shells`

```yaml
shells:
  bash: /bin/bash
  zsh: /usr/bin/zsh
  sh: /bin/sh
```

Paths must be absolute. ShellLane rejects a path that resolves back to its own
binary. Container runtimes may override these paths under `runtime.shell`;
otherwise conventional `/bin/bash`, `/bin/zsh`, and `/bin/sh` paths are used
inside the image.

## `identity`

```yaml
identity:
  env: SHELLLANE_IDENTITY
  argvEnvFallback: false
  required: true
  hash: sha256
```

| Field | Default | Meaning |
| --- | --- | --- |
| `env` | `SHELLLANE_IDENTITY` | Trusted environment variable containing the upstream identity. |
| `argvEnvFallback` | `false` | Also accept and consume an argv item such as `SHELLLANE_IDENTITY=alice`. |
| `required` | `true` | Fail closed when no identity exists. If false, use `anonymous`. |
| `hash` | `sha256` | Path-safe identity derivation; currently the only supported algorithm. |

The launcher, not the end user, must set the identity. If env and argv values
disagree, ShellLane rejects the invocation. Raw values are used only for exact
route matching; audit and filesystem paths use the digest.

Additional non-secret request context can use environment names such as
`SHELLLANE_CONTEXT_REQUEST` or `SHELLLANE_CONTEXT_TENANT`. It affects execution
only when explicitly allowlisted by a container runtime. Session IDs are not
required by ShellLane.

## `state`

```yaml
state:
  root: /var/lib/shelllane/users
```

`root` must be absolute and defaults to `/var/lib/shelllane/users`. The service
account must be able to create it. Each identity receives a SHA-256-named
directory with shared config/data/state/credentials/workspace, profile-specific
home/cache, and tmp directories.

State is created on the execution node. Node-pool users remain stable through
rendezvous hashing; state replication and migration are external concerns.

## `routes`

```yaml
routes:
  - match:
      identityIn: [admin@example, operator@example]
    target: privileged-local
  - match:
      identity: developer@example
    target: developer-box
  - target: ordinary-sandbox
```

Routes are evaluated in order:

- `match.identity` matches one exact identity.
- `match.identityIn` matches a small exact allowlist.
- no `match` is the default and must be last.
- `identity` and `identityIn` cannot appear together.
- `target` must name an existing target.

Prefer enumerating only exceptional administrators/team members. The final
default handles an unbounded ordinary-user population.

## `nodes`

Local node:

```yaml
nodes:
  service:
    connector: local
```

SSH node:

```yaml
nodes:
  worker-a:
    connector: ssh
    address: worker-a.example.internal
    user: shelllane
    port: 22
    helper: /usr/local/bin/shelllane
    sshArgs:
      - -o
      - UserKnownHostsFile=/etc/shelllane/known_hosts
```

| Field | Applies to | Default | Meaning |
| --- | --- | --- | --- |
| `connector` | all | none | `local` or `ssh`. |
| `address` | SSH | none | Hostname/IP, required for SSH. |
| `user` | SSH | none | Remote OS account, required for SSH. |
| `port` | SSH | `22` | SSH port. |
| `helper` | SSH | `shelllane` | Remote ShellLane executable. A fixed simple path/name only. |
| `sshArgs` | SSH | empty | Administrator-controlled extra OpenSSH arguments. |

ShellLane always adds `BatchMode=yes` and `StrictHostKeyChecking=yes`. Provision
keys and `known_hosts` before routing traffic. The remote helper needs a
compatible schema/runtime configuration and access to its local state root.

## `nodePools`

```yaml
nodePools:
  workers:
    nodes: [worker-a, worker-b]
    strategy: rendezvous-hash
    storage: local
    hashSeed: stable-deployment-seed
```

| Field | Default | Meaning |
| --- | --- | --- |
| `nodes` | none | Non-empty list of existing node names. |
| `strategy` | rendezvous behavior | Only `rendezvous-hash` is supported. |
| `storage` | informational | Document whether storage is `local` or externally shared. |
| `hashSeed` | empty | Stable placement namespace; changing it reshuffles users. It need not be secret. |

The pool is deterministic placement, not a load/health scheduler. A failed
selected node fails closed; v0.1 does not silently move the request and expose a
different state directory.

## `runtimes`

### Host

```yaml
runtimes:
  host:
    type: host
    profile: host
    stateMode: isolated
    workdir: "{workspace}"
    shell:
      bash: /bin/bash
```

| Field | Default | Meaning |
| --- | --- | --- |
| `type` | none | `host`. |
| `profile` | `host` | Namespace for HOME/cache. |
| `stateMode` | `isolated` | `isolated` or privileged `existing-home`. |
| `workdir` | `{workspace}` | Directory template; supports `{workspace}` and `{home}`. |
| `shell` | top-level `shells` | Per-runtime real-shell overrides. |
| `env` | empty | Fixed environment values (primarily useful for containers). |

`existing-home` reuses the remote/local OS account's real home and is intended
only for explicitly routed trusted administrators. Host execution is not a
sandbox.

### Container

```yaml
runtimes:
  sandbox:
    type: container
    profile: sandbox-v1
    engine: docker
    image: ghcr.io/example/runtime@sha256:immutable-digest
    platform: linux/amd64
    user: "1000:1000"
    pullPolicy: if-not-present
    workdir: /workspace
    network: restricted
    readOnly: true
    shell:
      bash: /bin/bash
      sh: /bin/sh
    mounts:
      - source: shared/credentials
        target: /home/agent/.credentials
        readOnly: true
    resources:
      cpus: "2"
      memory: 4g
      pids: 512
    env:
      CI: "true"
    passEnv:
      - TERM
      - LANG
      - LC_*
      - SHELLLANE_CONTEXT_*
```

| Field | Default | Meaning |
| --- | --- | --- |
| `type` | none | `container`. |
| `profile` | runtime name | Namespace for HOME/cache. |
| `engine` | `docker` | `docker` or `podman`. |
| `image` | none | Required image; pin a digest in production. |
| `platform` | engine default | Optional image platform. |
| `user` | image default | Container UID/GID; configure a non-root value. |
| `pullPolicy` | `if-not-present` | Passed to the container engine. |
| `workdir` | `/workspace` | Container working directory. |
| `network` | engine default | Use `restricted` for `--network none`, or an engine network name. |
| `readOnly` | `false` | Make the container root filesystem read-only. |
| `shell` | conventional paths | Per-kind shell path inside the image. |
| `mounts` | standard state mounts | Extra/overriding bind mounts. |
| `resources.cpus` | unlimited | CPU limit accepted by the engine. |
| `resources.memory` | unlimited | Memory limit accepted by the engine. |
| `resources.pids` | unlimited | Positive PID limit. |
| `env` | empty | Fixed environment values. |
| `passEnv` | terminal, locale, context, proxy allowlist | Upstream environment patterns; suffix `*` means prefix match. |

Mount `source` is always relative to that user's state root and cannot escape
it. Mount `target` must be absolute. ShellLane automatically mounts user
home/config/data/cache/workspace, drops all capabilities, enables
`no-new-privileges`, and provides a private `/tmp`.

Do not place secrets in `passEnv`; mount a narrowly scoped credentials path
read-only instead. Ensure host directory ownership matches the container UID.

## `targets`

```yaml
targets:
  ordinary-sandbox:
    nodePool: workers
    runtime: sandbox
  privileged-local:
    node: service
    runtime: host
    stateMode: existing-home
```

A target sets exactly one of `node` or `nodePool`, plus one `runtime`.
`stateMode` optionally overrides the runtime for that route. It is transmitted
to the SSH helper, so local and remote semantics remain the same.

## `audit`

```yaml
audit:
  path: /var/log/shelllane/audit.jsonl
```

When `path` is empty, audit output is disabled. Otherwise ShellLane appends
mode-`0600` JSON Lines records with request ID, identity hash, source type,
route, node, connector, runtime, shell/TTY metadata, and dispatch errors. It
does not record raw identities or commands.

## Two-machine ordinary-user example

The complete [example configuration](../config.example.yaml) demonstrates:

- named administrators running locally;
- a named developer running on a dedicated SSH machine;
- every other identity routed to one of two SSH workers;
- deterministic placement through a node pool; and
- a configured, resource-limited container image on the chosen worker.

Install the same ShellLane binary and a compatible configuration on both
workers. The initiating machine needs node/route/target definitions; each
worker must define the referenced runtime name and its local state root.
