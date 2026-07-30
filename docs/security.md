# Security model

ShellLane isolates filesystem state and controls placement; it is not itself a
complete sandbox. The strength of isolation is determined by the chosen
runtime, operating-system account, container engine, mounts, image, and network.

## Trust boundaries

- The upstream launcher is trusted to supply the configured identity.
- Configuration files and the ShellLane binary are administrator-controlled.
- SSH authentication maps ShellLane to an explicitly provisioned remote account.
- Container images and registries are part of the trusted computing base.
- AI-generated commands and ordinary-user prompts are untrusted.

Never accept the identity environment variable directly from an unauthenticated
remote client. ShellLane detects env/argv conflicts, but cannot prove who
created the process.

## Defaults and controls

- Identity is required. State paths use the configured `plain`, `base64url`, or
  `hash` naming mode; `plain` is path-validated and `hash` offers the strongest
  identity privacy.
- Filesystem state is created with mode `0700`; symlinked state components are
  rejected.
- Routes require a final default so unlisted users do not accidentally inherit
  an administrator route.
- SSH uses batch mode and strict host-key checking.
- Remote environment propagation is allowlist-based.
- Containers drop all capabilities, enable `no-new-privileges`, use a private
  tmpfs, and can be read-only, resource-limited, and network-disabled.
- Audit records contain identity hashes, not raw identities or command content.

## Deployment requirements

1. Run the service and SSH helper as dedicated, unprivileged OS users.
2. Make configuration, binaries, SSH keys, and `known_hosts` non-writable by
   request identities.
3. Pin container image digests and scan/sign images in the deployment pipeline.
4. Configure a non-root container user. Do not mount Docker sockets, host roots,
   or broad credential directories.
5. Keep per-user state on a filesystem with appropriate quotas and backups.
6. Rotate SSH credentials and audit access to remote worker accounts.
7. Treat `existing-home` as a privileged escape hatch for explicitly listed
   administrators only.

## Known v0.1 boundaries

- Container isolation shares the host kernel.
- Local host runtime is process/state separation, not a sandbox.
- Mount ownership must match the configured container UID/GID.
- SSH request payloads are visible to processes able to inspect that remote
  command line; do not add secrets to `passEnv`.
- There is no centralized revocation, remote-agent attestation, health-aware
  scheduling, or cross-node state synchronization.
- Audit append storage is local; ship it externally if tamper resistance or
  centralized retention is required.

Report vulnerabilities privately as described in [SECURITY.md](../SECURITY.md).
