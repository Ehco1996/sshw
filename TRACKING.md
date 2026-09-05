# sshw 架构重构与演进路线 (Tracking Plan)

本规划用于追踪 `sshw` 从个人轻量工具向生产级、Agent 友好、符合社区标准的 CLI/运维调度底座演进的阶段性工作。为保证代码稳定与审查质量，采用**小步快跑、多 PR 增量交付**的策略。

---

## 路线总览

```
  Phase 1: CLI 现代化改造 (Cobra)       ───►  [当前阶段 - 本 PR]
     │
     ▼
  Phase 2: 并发调度引擎优化 (errgroup)   ───►  [下一步]
     │
     ▼
  Phase 3: MCP 协议服务内置 (mcp-go)     ───►  [Agent 增强]
     │
     ▼
  Phase 4: Shell 自动补全与交互体验升级   ───►  [体验收尾]
```

---

## 阶段细分与追踪

### Phase 1: CLI 架构现代化（基于 Cobra） ✅ [Completed]
> 目标：消除手写的参数解析循环与 Flag 散落，全面接入 Go 社区事实标准的 CLI 框架 `spf13/cobra`。

- [x] 抽离核心逻辑并完成 SSOT 重构（`FlattenLeaves`, `LoadInventory`, `MatchTargets`）。
- [x] 引入依赖 `github.com/spf13/cobra`。
- [x] 构建 `rootCmd`，定义 `PersistentFlags`（`--config`, `-s`, `-i`, `-k`）自动下发到所有子命令。
- [x] 子命令重构：
  - `sshw list [target] [--json]`
  - `sshw run [flags] <target> <command>`（含别名 `exec`，支持 `--dry-run`, `-c`, `-t`, `-f`）
  - 默认无子命令行为：单参数直连指定节点，无参数开启 TUI。
- [x] 完备的单元测试与回归测试验证。
- [x] 提交并创建 Draft PR (#7)。

### Phase 2: 并发调度引擎与代码精简（基于 `errgroup`） ✅ [Completed]
> 目标：使用 Go 官方扩展库现代化并发控制，消除手写信号量与锁，消除废弃 API 与冗余逻辑。

- [x] 引入 `golang.org/x/sync/errgroup`。
- [x] 使用 `errgroup.Group.SetLimit(concurrency)` 替换 `batch_runner.go` 中的通道信号量 `sem` 和 `sync.WaitGroup`。
- [x] 迁移废弃的 `golang.org/x/crypto/ssh/terminal` 到 `golang.org/x/term`。
- [x] 消除所有 `golangci-lint` 警告（S1005, S1009, S1025, errcheck, unused）。
- [x] 抽象 `FilterNodeInfos` 保持节点树层级面包屑路径（`Path`）归一。
- [x] 统一 `LoadConfigFile` 清除配置文件加载重复逻辑。

### Phase 3: 内置 MCP (Model Context Protocol) 协议服务 ⏳ [Planned]
> 目标：让 `sshw` 成为无本地 Bash 权限 Agent（如 Claude Desktop / Cherry Studio）的原生工具服务。

- [ ] 引入 `github.com/mark3labs/mcp-go`。
- [ ] 新增 `sshw mcp` 子命令（标准输入输出 stdio JSON-RPC）。
- [ ] 注册核心 Tools：
  - `list_hosts`: 获取脱敏后的节点清单。
  - `exec`: 在单个/批量主机上执行指令并获得结构化执行结果。
- [ ] 注入高危指令防护（非交互环境强制拦截未经授权的破坏性指令）。

### Phase 4: Shell 自动补全与体验细节 ⏳ [Planned]
> 目标：提供开箱即用的终端开发体验。

- [ ] 利用 Cobra 生成 Bash / Zsh / Fish 补全脚本（`sshw completion <shell>`）。
- [ ] 支持根据当前 Inventory 动态补全节点名和别名。
