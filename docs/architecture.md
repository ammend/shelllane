# Architecture

## Request lifecycle

1. An upstream process invokes a ShellLane symlink named `bash`, `zsh`, or `sh`.
2. ShellLane loads and strictly validates one YAML configuration.
3. It resolves a trusted identity from the configured environment variable,
   with an optional argv fallback.
4. Ordered routes select a target. A target combines one node (or node pool)
   with one runtime.
5. A node pool uses rendezvous hashing over the identity digest.
6. State paths are prepared on the selected execution node.
7. The connector transports the invocation locally or over SSH.
8. The runtime replaces ShellLane with a host shell or container engine.

The outer process contract is stable; the selected backend is not visible to
the program running inside the shell.

## Core abstractions

`Node` owns reachability: connector, address, SSH user, port, and remote helper.

`Runtime` owns execution: host/container, image, shell paths, profile, mounts,
environment allowlist, workdir, network, and resource constraints.

`Target` is only a reference to `Node | NodePool` plus `Runtime`. Keeping it
small prevents policy, transport, and container mechanics from becoming one
large scheduler abstraction.

`Route` is an ordered identity match. Exact matches cover the small exceptional
population (administrators and team members); a required final default covers
all ordinary users.

## SSH protocol

SSH is a connector, not a runtime. The local side serializes a versioned request
as base64url JSON and invokes:

```text
SHELLLANE_INTERNAL_REQUEST=<payload> shelllane run-remote
```

The remote helper loads the remote machine's own configuration and executes the
named runtime. Only allowlisted environment variables enter the payload.
OpenSSH handles authentication, encryption, host verification, stdin/stdout,
and PTY transport. The remote helper version and runtime names must be
compatible with the initiating host.

## Extension seam

Future connectors or runtime drivers should implement the same internal request
contract. A future scheduler may return a node name, but the v0.1 core will not
contain health scoring, queues, autoscaling, leases, or Kubernetes objects.
This keeps the first implementation useful without turning it into a bot
request broker.
