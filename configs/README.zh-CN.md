# 按场景选择配置

这里的每个 YAML 都是完整配置，可以直接执行 `shelllane config check`，不是省略关键
字段的片段。

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
