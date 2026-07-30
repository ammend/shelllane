# 按场景选择配置

这里的每个 YAML 都是完整配置，可以直接执行 `shelllane config check`，不是省略关键
字段的片段。

代码仓库的 `configs/` 只存模板。真实运行配置遵循 Linux XDG 规范：

```bash
mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/shelllane"
cp configs/local-host.yaml \
  "${XDG_CONFIG_HOME:-$HOME/.config}/shelllane/config.yaml"
```

常驻系统服务可由管理员安装到 `/etc/shelllane/config.yaml`。不要把仓库内模板路径
误当成用户运行时路径。

五个模板都展示了多上游 `identity.envMap` 列表。`make install` 会更新这些具名模板，
但默认不覆盖已经投入使用的 `config.yaml`；明确需要重置时使用
`sudo make install OVERWRITE_CONFIG=1`。

所有模板显式使用默认的 `state.identityNaming: plain`。若身份可能包含路径分隔符，
改用可逆的 `base64url`；若文件系统中不应暴露身份，改用不可逆的 `hash`。修改模式会
改变用户目录名，已有数据需要迁移。

## local + host

使用 [`local-host.yaml`](local-host.yaml)。

请求留在当前机器，ShellLane 直接 `exec` 真实 Shell，但仍按身份替换 HOME、XDG 和
workspace。它最容易部署，也最接近传统命令行；缺点是没有容器安全边界。

适合可信管理员、单机开发、首次验证。不要把它描述成沙箱。

## local + Docker

使用 [`local-docker.yaml`](local-docker.yaml)。

ShellLane 在当前机器准备用户目录，再调用 Docker。容器 rootfs 可只读，网络、CPU、
内存、PID、环境变量和挂载都由配置控制。容器可随请求删除，用户 workspace/config
仍通过 bind mount 持久化。

把 `user: "1000:1000"` 改成运行 ShellLane 的服务账号 UID:GID，并换成包含所需
Agent CLI 的固定 digest 镜像。

## SSH + host

使用 [`ssh-host.yaml`](ssh-host.yaml)。

本机 ShellLane 完成身份和路由，OpenSSH 负责认证、加密、PTY 与 stdio；远端
`shelllane run-remote` 根据同名 runtime 运行真实 Shell。适合把指定团队成员绑定到
自己的开发机。

部署前必须准备：

1. 远端安装兼容版本的 `shelllane`；
2. 创建低权限远端账号；
3. 配置 key 和 `known_hosts`；
4. 让远端账号可写 `state.root`；
5. 在远端放置包含同名 `host` runtime 的配置。

模板默认不设置 `sshArgs`，会复用运行 ShellLane 账号的 `~/.ssh/config`、
`~/.ssh/known_hosts` 和默认私钥，避免引用尚未创建的 `/etc/shelllane/...` 导致开箱
即失败。常驻服务需要专用 key 或 known_hosts 时，再取消 YAML 中的注释或显式增加
`sshArgs`。ShellLane 仍固定启用 `BatchMode=yes` 和 `StrictHostKeyChecking=yes`。

## SSH + Docker

使用 [`ssh-docker.yaml`](ssh-docker.yaml)。

SSH 只是通信层；Docker 真正运行在远端机器。远端 helper 收到版本化请求后准备用户
状态并启动容器。适合集中式工作节点或带 GPU/大磁盘的机器。

除了 SSH + host 的准备项，远端账号还要有容器引擎权限，镜像和数值 UID:GID 也必须
在远端有效。

## mixed

使用 [`mixed.yaml`](mixed.yaml)。

它体现最重要的路由原则：只枚举少量例外，不枚举普通用户。

1. `admin@example`、`operator@example` → 本机 existing-home；
2. `developer@example` → 个人 SSH 开发机；
3. 其他任意身份 → 两台远端 worker 的容器；
4. Rendezvous Hash 让同一普通用户稳定落到同一 worker。

规则按顺序 first-match-wins，默认规则必须最后。修改 `hashSeed` 会重新分布大量用户，
上线后应保持稳定。

## 检查

```bash
make configs-check
shelllane route test --config configs/mixed.yaml --identity admin@example
shelllane route test --config configs/mixed.yaml --identity developer@example
shelllane route test --config configs/mixed.yaml --identity arbitrary-user
```
