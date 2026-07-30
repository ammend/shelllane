# Architecture and request flow

Open the interactive diagrams alongside this page:

- [System architecture](diagrams/shelllane-architecture.html)
- [SSH + container request sequence](diagrams/ssh-container-sequence.html)

## The boundary ShellLane owns

An upstream application already knows **who** is making a request and already
knows **which command** it wants to run. ShellLane accepts the trusted identity
and ordinary shell argv. It owns only:

1. identity validation;
2. static policy routing;
3. execution-node selection;
4. local or SSH transport;
5. host or container runtime setup; and
6. per-identity state mapping.

It does not parse prompts or recognize Codex, Claude Code, Trae, or any other
Agent CLI. This separation is why one integration can serve many CLIs.

## Model

```text
Route(identity) → Target(Node | NodePool, Runtime)

Node.connector = local | ssh
Runtime.type   = host  | container
```

`Node` answers “where and how can it be reached?” `Runtime` answers “how does
the shell run there?” Their product gives four backends without duplicating
policy.

`NodePool` applies rendezvous hashing to the identity digest. It is
deterministic placement, not a health/capacity scheduler. If a selected node
fails, ShellLane fails closed rather than silently exposing a different state
directory.

## Local request

1. The launcher executes a ShellLane symlink named `bash`, `zsh`, or `sh`.
2. ShellLane strictly loads YAML and resolves `SHELLLANE_IDENTITY`.
3. The first matching route selects a target.
4. ShellLane prepares the hashed per-user state.
5. A host runtime calls `exec(2)`; a container runtime replaces the process
   with Docker/Podman.
6. stdio, TTY, signals, and the final exit code remain on the original process
   path.

## SSH request

1. The initiating side resolves identity, route, node, and runtime.
2. It serializes a versioned request containing the identity hash, shell argv,
   selected runtime, cwd/TTY metadata, and allowlisted environment.
3. OpenSSH authenticates the remote account, verifies the host key, and carries
   stdio/PTY.
4. The remote `shelllane run-remote` helper loads local configuration and state.
5. The helper executes the requested host or container runtime.
6. Output and exit status flow back through SSH.

The raw identity is not sent to the remote helper; routing has already selected
the node and the digest is sufficient for state isolation.

## State

Shared config/data/state/credentials/workspace survive runtime replacement.
HOME and cache are profile-specific so a host CLI and a container image do not
silently corrupt each other's startup files.

State exists on the selected execution node. With local node-pool storage,
stable hashing is what keeps a user attached to the same data. Replication and
migration remain external concerns.

## Failure rules

- missing/conflicting identity: fail;
- invalid configuration or route: fail;
- selected SSH node unavailable: fail, no local fallback;
- Docker unavailable: fail, no host fallback;
- unknown remote runtime/protocol: fail.

These rules prevent a security or placement failure from becoming an
unexpected execution on a more privileged machine.

Chinese explanation: [架构与请求流](zh-CN/architecture.md).
