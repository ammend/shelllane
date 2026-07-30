# 架构与请求流

建议同时打开：

- [交互式总体架构图](../diagrams/shelllane-architecture.html)
- [交互式 SSH + Container 顺序图](../diagrams/ssh-container-sequence.html)

## ShellLane 负责什么

上层应用已经知道“谁发起请求”和“要运行什么命令”。ShellLane 接收可信身份和原始
Shell argv，只负责：

1. 校验身份；
2. 按静态规则选择 Target；
3. 选择 Node 或 NodePool；
4. 通过 local 或 SSH 连接；
5. 通过 host 或 container 运行；
6. 映射该身份独有的 HOME、配置、凭据和 workspace。

它不解析 prompt，也不区分 Codex、CC、Trae 或其他 Agent CLI。正因为不理解具体
Agent，一个 ShellLane 接入才能复用到多个上层应用和 CLI。

## 三个核心抽象

```text
Route(identity)
  → Target
      ├── Node 或 NodePool
      │     └── Connector: local | ssh
      └── Runtime: host | container
```

- `Route` 是策略：谁去哪个 Target，按顺序 first-match-wins。
- `Node` 是位置和通信：在哪台机器、用 local 还是 SSH。
- `Runtime` 是运行方法：直接 host exec，还是 Docker/Podman。

SSH 不是一种运行环境，它只是通信方式；物理机也不是一种 Connector，它对应 host
runtime。把这两个维度分开，四种组合自然出现，而不需要四套代码。

## 本地请求

1. 上层执行名为 `bash`、`zsh` 或 `sh` 的 ShellLane 软链；
2. ShellLane 严格读取 YAML 和 `SHELLLANE_IDENTITY`；
3. 第一条匹配规则选中 Target；
4. 根据身份哈希准备独立用户状态；
5. host 使用 `exec(2)` 替换进程，container 使用 Docker/Podman；
6. stdio、TTY、signal 和最终退出码沿原进程链返回。

## SSH + Container 请求

1. 发起端完成身份、路由、节点和 runtime 选择；
2. 生成版本化请求，只包含身份哈希、argv、runtime、cwd/TTY 和 allowlist 环境；
3. OpenSSH 完成远端账号认证、host key 校验、加密、stdio 和 PTY；
4. 远端 `shelllane run-remote` helper 读取远端配置；
5. 远端准备该身份状态并启动 Docker；
6. 真实 Shell 和 Agent CLI 在容器内运行；
7. 输出和退出码经 SSH 回到上层应用。

原始身份不会发送到远端；发起端已经完成路由，远端只需要哈希来选择用户目录。

## 多台机器

NodePool 使用 Rendezvous Hash：

```text
hash(identity_hash, pool, node) → score
选择 score 最高的 node
```

同一个普通用户会稳定落到同一台机器。增加或删除节点时，大多数用户不会移动。它不
检测 CPU、队列或机器健康度，因此是确定性放置，不是动态调度器。

## 为什么失败时不回落

- SSH 失败后回落本机，可能让普通用户进入服务机；
- Docker 失败后回落 host，可能绕过容器安全策略；
- 节点失败后随机换机，可能让用户看到一份空的或错误的状态。

因此 ShellLane 默认 fail closed。需要故障迁移时，应同时设计共享状态和明确的外部
调度策略，而不是静默回落。
