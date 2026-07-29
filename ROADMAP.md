# Roadmap

## v0.1 — useful minimum

- Shell basename compatibility for bash/zsh/sh
- Trusted identity resolution and deterministic ordered routing
- Per-user state layout
- Local host and local container runtimes
- SSH host and SSH container execution
- Rendezvous-hash node pools
- Configuration validation, route preview, audit events, tests, and docs

## v0.2 — operational hardening

- Backend capability/version handshake
- Graceful signal and cancellation conformance suite
- Optional health input and explicit fail-closed behavior
- Image policy hooks and stronger audit sinks
- Packaging and signed release artifacts

## Later, only with demonstrated demand

- A connector plugin boundary
- External scheduler interface
- Reverse-connected workers for machines that cannot accept SSH
- Shared-state adapters

Bot protocol handling, chat request brokering, Kubernetes orchestration, and a
general control plane remain separate projects unless the core use case proves
they belong here.
