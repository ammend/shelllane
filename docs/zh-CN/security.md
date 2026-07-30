# 安全模型

ShellLane 负责身份路由、状态隔离和运行边界，但不是完整的安全沙箱。最终安全强度由
OS 账号、SSH、容器引擎、镜像、mount、网络和宿主机共同决定。

## 信任边界

- 上层 launcher 必须提供可信的身份变量；ShellLane 可通过 `identity.envMap` 将其映射为统一身份；
- 配置、二进制、SSH key 与 known_hosts 只能由管理员修改；
- SSH 远端账号和容器镜像属于可信计算基础；
- 普通用户输入、prompt 和 Agent 生成的命令均视为不可信。

ShellLane 能发现 env/argv 身份冲突，但无法证明是谁创建了进程。不要把未经认证的
HTTP 参数或用户输入直接变成身份环境变量。

## 推荐策略

普通用户优先使用 container runtime：

- 专用非 root 服务账号；
- 容器 `user` 使用同一数值 UID:GID；
- 固定 image digest；
- `readOnly: true`；
- `network: restricted`，按需要再放开；
- CPU、内存、PID 限制；
- 环境变量 allowlist；
- 不挂 Docker socket、宿主根目录或宽泛凭据目录。

SSH 节点应使用专用远端账号、BatchMode、严格 host key、独立 key 和最小 sudo/Docker
权限。ShellLane 默认 SSH 或 Docker 失败即失败，不回落到更高权限的本机 host。

## Host 的真实含义

`stateMode: isolated` 只替换 HOME、XDG、workspace 和 tmp；进程仍共享宿主内核、账号
权限和可访问文件，因此不是沙箱。

`existing-home` 会复用 ShellLane 服务账号的真实 HOME，应只放在显式管理员规则，
不能成为默认目标。

## 凭据

优先把凭据放在每个身份的 `shared/credentials`，按最小目录只读挂载。不要通过
`passEnv` 发送 token；SSH 版本化请求会出现在远端命令上下文中，环境 allowlist 应
只包含终端、locale 和非敏感业务上下文。

完整边界与已知限制见 [英文安全模型](../security.md)。
