# sshw — Agent 维护与开发指南

`sshw` 是一个用于服务器资产管理与自动化登录的 SSH 客户端包装工具，同时提供现代化的终端 UI（TUI）和非交互式的批量执行（Headless Batch Execution）能力。

本项目为通用开源项目，不包含特定业务系统的定制代码或专有依赖。

---

## 核心工程原则 (Core Discipline)

- **单一事实来源 (SSOT)**：
  - 目标解析逻辑由 `target.go` 独占，任何 CLI 或 TUI 逻辑严禁私自解析节点别名或标签。
  - 节点信息脱敏与对外导出以 `node_info.go` 为唯一视图。
  - 主机树状层级与展平规则以 `flatten.go` 为单一事实。
- **拥抱社区成熟库，拒绝造轮子**：
  - 底层 SSH 协议全面依托 `golang.org/x/crypto/ssh`；
  - 终端交互与渲染使用 Charm 家族（`bubbletea`、`bubbles`、`lipgloss`）；
  - CLI 参数与命令编排使用 `spf13/cobra`；并发调度基于 `golang.org/x/sync/errgroup`。
  - 严禁自行手写 ANSI 转义码解析器、自制并发线程池或手写 OpenSSH 替代语法解析。


---

## 项目定位与核心特性

1. **多源资产清单管理 (Inventory Management)**：
   - 本地 YAML 配置（支持分组、层级嵌套、跳板机链式代理）。
   - OpenSSH 配置（直接读取并解析 `~/.ssh/config`）。
   - 动态远端清单（支持通过 HTTP API 结合 API Key 动态拉取机器列表）。
2. **多模式运行 (Multi-Mode Operations)**：
   - **交互式 TUI**：基于 Charm Bubble Tea 构建，支持实时搜索过滤、分组浏览、全局搜索调色板、单机/多机连接与标记。
   - **单机快捷直连**：`sshw <name-or-alias>` 快速登录。
   - **Headless 自动化调度 (Agent/CI 友好)**：
     - `sshw list [--json]`：列表查询与机器资产信息脱敏导出。
     - `sshw run/exec [flags] <target> "<command>"`：基于通配符或分组进行多机并发远程命令执行。
3. **企业级运维保障 (Safety & Auditing)**：
   - **高危命令拦截 (Danger Guard)**：自动启发式识别 `rm -rf`、`reboot`、`mkfs` 等危险指令；非交互模式默认拦截，TUI 模式需输入验证词。
   - **全量审计日志 (Audit Logging)**：每次批量执行自动持久化运行元数据、单机标准输出/错误输出至本地运行日志目录（默认 `~/.local/state/sshw/runs/`）。

---

## 仓库目录与架构

```
sshw/
├── cmd/sshw/           # CLI 二进制入口与子命令编排 (main, list, run)
├── internal/
│   └── tui/            # 交互式终端 UI 逻辑 (Bubble Tea 模型、样式、视图、按键绑定)
├── audit.go            # 批量执行审计日志记录与索引 (runs.jsonl, <run_id>/<host>.log)
├── batch_runner.go     # Headless 批量并发调度器 (并发池控制、超时控制、聚合统计)
├── client.go           # 底层 SSH 客户端实现 (连接拨号、Jump Host 代理、认证、PTY 终端)
├── client_exec.go      # 底层非交互命令执行 (RunCommand, ExecNode)
├── config.go           # 清单加载逻辑 (YAML / OpenSSH / HTTP Dynamic) 与 Node 结构定义
├── danger.go           # 高危命令正则匹配引擎
├── flatten.go          # 节点树线性展平与层级面包屑计算 (IndexedHost, FlattenLeaves)
├── node_info.go        # 面向外部/JSON 输出的节点安全脱敏视图 (NodeInfo)
└── target.go           # 目标选择器 (支持别名、组展开、通配符 glob、多目标逗号分隔)
```

---

## 核心设计原则 (SSOT & Maintainability)

在维护与扩展本项目时，所有 Agent 必须遵循以下架构原则：

### 1. 严格遵守单一真实源 (Single Source of Truth)
- **树结构计算归一**：所有遍历叶子节点或计算层级路径的逻辑，**统一使用 `FlattenLeaves`**。严禁在具体功能模块中重复编写私有递归遍历。
- **节点匹配归一**：所有通过名称、别名或模式筛选机器的逻辑，**统一使用 `MatchTargets` / `FindConnectableByNameOrAlias`**。
- **配置加载归一**：所有读取本地或动态清单的逻辑，**统一走 `LoadInventory(opts)`**。
- **凭证安全原则**：任何面向 CLI 文本或 JSON 序列化输出的数据结构（如 `NodeInfo`），**严禁包含 `Password`、`Passphrase` 等敏感凭据**。

### 2. 核心下沉，外层消费
- 将网络连接、并发控制、安全匹配、日志落盘等能力收敛在根包 `sshw`。
- `internal/tui/`（界面交互）与 `cmd/sshw/`（命令行）仅作为消费层，不硬编码底层运维与并发策略。

---

## 日常开发与验证工作流

### 1. Worktree 规范
在修改代码前，请在仓库内部的 `.worktrees/` 目录下创建独立的 worktree 进行开发：
```bash
git worktree add .worktrees/feat-<name> -b feat/<name>
```

### 2. 构建与验证命令
- **执行单元测试**：
  ```bash
  go test -v ./...
  ```
- **代码构建**：
  ```bash
  make build
  ```
- **静态代码检查**：
  ```bash
  golangci-lint run ./...
  ```

---

## Agent 指导建议 (How Agents Should Maintain)

1. **扩展命令行功能**：
   - 保持非交互命令对标准输入/输出的纯粹性，便于下游通过管道、`jq` 或 Agent 自动化解析。
   - 对批量操作必须提供 `--dry-run` 预览机制。
2. **高危操作防线**：
   - 若引入新的系统管理操作，需在 `danger.go` 评估是否加入高危拦截模式。
3. **避免过度造轮子**：
   - 优先复用 Go 官方扩展库（如 `golang.org/x/sync`）与社区事实标准的成熟库。
