# Configuration scenarios

Each file is a complete, strictly validated configuration rather than a partial
snippet.

This repository directory contains templates, not the active runtime
configuration. Copy one to `$XDG_CONFIG_HOME/shelllane/config.yaml` (normally
`$HOME/.config/shelllane/config.yaml`) or `/etc/shelllane/config.yaml`.

| File | Connector | Runtime | Use |
| --- | --- | --- | --- |
| [`local-host.yaml`](local-host.yaml) | local | host | Trusted local execution |
| [`local-docker.yaml`](local-docker.yaml) | local | container | Local user sandbox |
| [`ssh-host.yaml`](ssh-host.yaml) | SSH | host | Personal development machine |
| [`ssh-docker.yaml`](ssh-docker.yaml) | SSH | container | Remote container worker |
| [`mixed.yaml`](mixed.yaml) | mixed | mixed | Admin, developer, and default-user policies |

Run `make configs-check` after any edit. Replace example hosts, key paths,
container UID/GID, image, and state/audit paths before deployment.

The SSH examples can be installed on both initiating and remote machines. The
remote `run-remote` helper consumes the requested runtime and local state root;
routes are used by the initiating frontend.

See the [field reference](../docs/configuration.md) and
[Chinese scenario guide](README.zh-CN.md).
