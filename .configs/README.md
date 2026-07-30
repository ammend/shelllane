# Configuration scenarios

Each file is a complete, strictly validated configuration rather than a partial
snippet.

The directory is hidden by convention. Select a file explicitly from the
project root:

```bash
export SHELLLANE_CONFIG="$PWD/.configs/local-host.yaml"
```

ShellLane intentionally does not auto-load configuration from the current
working directory: a checked-out project must not be able to redirect execution
to an arbitrary SSH node or image without administrator intent.

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
