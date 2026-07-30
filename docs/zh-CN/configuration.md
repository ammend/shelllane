# 配置教程

完整可运行模板都在 [`configs/`](../../configs/README.zh-CN.md)。本页解释配置如何组合。
逐字段类型与默认值见 [英文完整参考](../configuration.md)。

## 配置加载顺序

1. 诊断命令的 `--config`；
2. `SHELLLANE_CONFIG`；
3. `$XDG_CONFIG_HOME/shelllane/config.yaml`；未设置 XDG 时为
   `$HOME/.config/shelllane/config.yaml`；
4. `/etc/shelllane/config.yaml`。

仓库的 `configs/` 是模板目录，不是运行时自动查找目录。选择一个模板安装到 Linux
用户配置目录：

```bash
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/shelllane"
cp configs/local-host.yaml \
  "${XDG_CONFIG_HOME:-$HOME/.config}/shelllane/config.yaml"
```

因此 `/home/ammend/configs/local-host.yaml` 只在仓库恰好位于
`/home/ammend` 时才是模板路径；正常运行路径应是
`/home/ammend/.config/shelllane/config.yaml`。

未知字段、错误引用和不支持的值会直接报错。部署前执行：

```bash
shelllane config check --config configs/mixed.yaml
shelllane doctor --config configs/mixed.yaml
```

## 最小心智模型

```yaml
routes:
  - match: ...       # 谁
    target: name     # 去哪个组合

targets:
  name:
    node: machine    # 去哪里
    runtime: sandbox # 怎么运行
```

当机器有多个时，Target 使用 `nodePool` 代替 `node`。

## 身份

```yaml
identity:
  env: SHELLLANE_IDENTITY
  envMap:
    SHELLLANE_IDENTITY: "$BOTMUX_USER_ID"
  argvEnvFallback: true
  required: true
```

`env` 是 ShellLane 内部的统一身份名；`envMap` 的 key 必须与它相同，value 是
上层已经提供的环境变量引用。这里表示读取 `BOTMUX_USER_ID`，而不是要求上层创建
`SHELLLANE_IDENTITY`。`${BOTMUX_USER_ID}` 也可使用。映射只接受单一变量引用，不执行
shell 展开、命令替换或字符串拼接。

身份来源必须由已经完成认证的上层应用设置，不能直接信任外部用户提交的任意环境变量。
若 `SHELLLANE_IDENTITY` 与映射来源同时存在但值不同，会以 `identity_conflict` 失败。
原始身份用于精确匹配，目录和审计只用 SHA-256。

如果上层没有可映射的变量，或只能把元数据追加在 Bash 后面，仍可使用兼容模式：

```bash
# 首选：真正的进程环境
SHELLLANE_IDENTITY=user@example /path/to/bash -lc 'codex ...'

# 兼容：launcher 把 KEY=value 放在 bash 后的 argv 中
/path/to/bash SHELLLANE_IDENTITY=user@example -lc 'codex ...'

# envMap 中声明的源变量也可以放在 bash 后
/path/to/bash BOTMUX_USER_ID=user@example -lc 'codex ...'
```

后两种方式只有在 `argvEnvFallback: true` 时启用。ShellLane 扫描 `argv[1:]` 中与
`identity.env` 或 `envMap` 来源同名的 `KEY=value`，完成解析后删除所有这些元数据参数，所以真实 Bash
的 argv 最终只收到 `bash -lc 'codex ...'`；同时 ShellLane 将这些同名同值项保留或恢复
到真实 Shell 的环境中，不删除或覆盖用户已有环境变量。身份不是 secret；argv 在 ShellLane 消费前可能被
同机进程列表短暂看到。

Session ID 不是 ShellLane 必需字段。如果上层确实要传非敏感上下文，使用
`SHELLLANE_CONTEXT_*`，并在 runtime `passEnv` 中显式允许。

## 路由

```yaml
routes:
  - match:
      identityIn: [admin@example, operator@example]
    target: trusted-local
  - match:
      identity: developer@example
    target: developer-host
  - target: ordinary-pool
```

顺序不可交换：只枚举少量管理员和有开发机的成员，最后一条无 match 的规则覆盖数量
无限的普通用户。默认规则必须最后。

用命令查看结果，不要靠猜：

```bash
shelllane route test --config configs/mixed.yaml --identity arbitrary-user
```

## Node 与 SSH

```yaml
nodes:
  worker:
    connector: ssh
    address: worker.example.internal
    user: shelllane
    helper: /usr/local/bin/shelllane
    sshArgs:
      - -o
      - UserKnownHostsFile=/etc/shelllane/known_hosts
      - -i
      - /etc/shelllane/keys/worker
```

远端需要兼容版本的 ShellLane、低权限账号、key/known_hosts、可写 state root 和同名
runtime。SSH 只透传 runtime `passEnv` 允许的环境。

## Host runtime

```yaml
runtimes:
  host:
    type: host
    profile: host
    stateMode: isolated
    workdir: "{workspace}"
```

`isolated` 替换 HOME/XDG/workspace，但不是安全沙箱。`existing-home` 会复用服务账号
真实 HOME，只应由精确管理员规则覆盖。

## Container runtime

```yaml
runtimes:
  sandbox:
    type: container
    engine: docker
    image: registry.example/agent-runtime@sha256:...
    pullPolicy: missing
    user: "1000:1000"
    workdir: /workspace
    network: restricted
    readOnly: true
    resources: {cpus: "2", memory: 4g, pids: 512}
    passEnv: [TERM, LANG, "LC_*", "SHELLLANE_CONTEXT_*"]
```

生产环境必须：

- 使用包含所需 Agent CLI 的固定 digest 镜像；
- `user` 与运行远端 ShellLane 的服务账号数值 UID:GID 一致；
- 不挂载 Docker socket、宿主根目录或宽泛凭据目录；
- 只允许业务真正需要的网络、环境和额外挂载。

## NodePool

```yaml
nodePools:
  workers:
    nodes: [worker-a, worker-b]
    strategy: rendezvous-hash
    storage: local
    hashSeed: stable-deployment-v1
```

`hashSeed` 不是密码，但上线后必须稳定。修改它会让大量用户重新分布。当前版本节点
失败不会自动迁移；本地存储也不会自动同步。

## 上线检查

```bash
make configs-check
shelllane config check --config /etc/shelllane/config.yaml
shelllane route test --config /etc/shelllane/config.yaml --identity admin@example
shelllane route test --config /etc/shelllane/config.yaml --identity normal-user
shelllane doctor --config /etc/shelllane/config.yaml
```

然后分别用一个管理员身份和一个普通身份执行只读命令，检查 HOME、pwd、容器 UID、
网络和审计，再逐步开放真实 Agent CLI。
