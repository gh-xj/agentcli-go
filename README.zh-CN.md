# AgentCLI - GO

[![CI](https://github.com/gh-xj/agentcli-go/actions/workflows/ci.yml/badge.svg)](https://github.com/gh-xj/agentcli-go/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gh-xj/agentcli-go)](https://github.com/gh-xj/agentcli-go/releases)
[![License](https://img.shields.io/github/license/gh-xj/agentcli-go)](./LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gh-xj/agentcli-go)](./go.mod)

**语言**: [English](./README.md) | [中文](./README.zh-CN.md)

用于快速构建面向 AI Agent 的 Go CLI 项目的框架与运行时库。

`agentcli-go` 是一个 **agent-first 的 CLI 脚手架与运行时框架**，提供日志、参数解析、命令执行、文件系统辅助、生命周期流程与验证基础能力，帮助 AI Agent 高一致性地构建可维护的 CLI。

---

## 2 分钟价值

- 1 分钟内生成符合规范的 Go CLI 骨架。
- 使用 `task ci` 和文档检查实现可复现、可追溯的质量流程。
- 用一套标准化流程快速带入另一个 AI Agent。

## 30 秒示例

```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.3
agentcli new --module github.com/me/my-tool my-tool
cd my-tool
agentcli doctor
agentcli loop doctor --repo-root .            # 可选：验证痕迹
```

预期结果：可运行的脚手架（`main.go`、`cmd/`、`Taskfile.yml`、`test/`）以及首次质量检查，通常 < 2 分钟完成。

## 为什么选择

- 省去 AI 辅助 CLI 开发中的重复样板代码。
- 全项目统一 CLI 模式，便于多人/多人协作。
- 快速生成可合规项目。
- 从第一天起即支持机器可读输出（`--json`）。

### Harness Engineering 价值

- 使用统一默认值实现可复现的项目构建。
- 内置质量闸道，适配 AI 工作流与 CI 安全验证。
- 合同优先的输出（Schema + 工具）让生成 CLI 更易持续维护。
- 标准化生命周期与错误语义，降低首次运行与接手成本。
- 以小而可审计的运行时 API，支撑大规模 Agent 工具编排。

## 谁应该使用 agentcli-go

- 你希望 AI Agent 可以稳定生成、演进并维护 Go CLI。
- 你在维护内部工具，希望统一模板、可复现输出和 CI 友好检查。
- 你想减少“第一次配置/缺文档导致的反复重试”。
- 你偏好小而清晰、可 review 的运行时 API。

本项目定位为 **AI 生成 CLI 的底层基础设施**，同时也适合追求同等严谨性的人工开发。

## 常见场景

- 内部运维脚本（cron、迁移、维护助手）。
- 数据与 IO 工具（同步、抓取、转换、上报）。
- 需要确定性命令与任务编排的多步 Agent 工作流。
- 团队复用同一脚手架标准和质量闸道构建内部工具集。

## 本项目在生态中的位置

`agentcli-go` 在当前仓库扮演三层角色：
- 作为 **Library** 提供通用 CLI 基元。
- 作为 **脚手架 CLI** 生成合规项目骨架。
- 作为 **Harness 能力** 帮助 AI Agent 产出、验证与迭代更安全。

```text
用户请求
   │
   ▼
AI Agent（Codex/Claude/ClawHub）
   │
   ├─ 读取 onboarding 与 skill 文档
   │
   ▼
agentcli-go（library + scaffold CLI）
   │
   ├─ 生成标准目录与命令结构
   ├─ 统一的日志、错误、配置流程
   └─ 嵌入质量检查入口与文档规范
           │
           ▼
生成项目（task/verify, schema, docs:check, loop/quality）
           │
   ▼
更低认知负担、迭代更安全
```

## 为什么有助于被发现

- 可重复的机器可读输出（`--json`）和任务闸道提高可置信度。
- 明确文档结构（`README` / `agents.md` / 技能文档）减少上手歧义。
- 内置 Agent 入口可用（ClawHub）：
  - https://clawhub.ai/gh-xj/agentcli-go

## 对 Agent 的一键 onboarding 提示词

在会话开始时可直接粘贴到 Agent：

```text
我将使用这个仓库作为 agent skill。
项目地址: https://github.com/gh-xj/agentcli-go
请按 agents.md 的顺序执行，包括标准化 onboarding 命令、文档归档规则、以及记忆/更新动作。
```

## 5 分钟快速上手路径

1. 安装 CLI：

```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.3
```

2. 生成标准项目：

```bash
agentcli new --module github.com/me/my-tool my-tool
cd my-tool
```

3. 运行首轮健康检查：

```bash
go test ./...
agentcli doctor
```

4. 添加命令并快速验证：

```bash
agentcli add command --name sync --preset file-sync
go run . --help
```

若顺利通过，说明已获得一个带 harness 能力的可继续迭代 CLI 骨架。

如果这个工具减少了你的 Agent 设置成本，欢迎给 GitHub 点星，帮助更多团队发现它：
https://github.com/gh-xj/agentcli-go

## FAQ（1 分钟）

- `agentcli loop` 命令执行失败？
  - 先检查 `agents.md` 与 `skills/verification-loop/SKILL.md` 中的当前命令签名。
- 生成项目缺少预期文件？
  - 使用干净目录重跑 `agentcli new`，并确认模板版本兼容性。
- 提交 PR 前应该做什么？
  - 运行 `task verify`，并在 PR 描述中附上关键检查结果。

## 贡献

- 欢迎提交贡献。详见 `CONTRIBUTING.md` 中的评审期望、检查项与 PR 流程。

## 安装

### 库方式（导入到你的项目）

```bash
go get github.com/gh-xj/agentcli-go@v0.2.3
```

### 脚手架 CLI（可选）

```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.3
```

或使用 Homebrew：

```bash
brew tap gh-xj/tap
brew install agentcli
```

或下载预编译二进制（macOS/Linux amd64+arm64）：

- https://github.com/gh-xj/agentcli-go/releases/tag/v0.2.3

## Claude Code Skill

使用 Codex/Claude 工作流的操作说明：[`skills/agentcli-go/SKILL.md`](./skills/agentcli-go/SKILL.md)  
Agent 工作流起点与 harness 入口请先读 [`agents.md`](./agents.md)。

## 在 ClawHub 发布

该仓库已发布为 Agent Skill： https://clawhub.ai/gh-xj/agentcli-go

## HN 草稿

- 英文: `docs/hn/2026-02-22-show-hn-agentcli-go.md`
- 中文: `docs/hn/2026-02-23-show-hn-agentcli-go-zh.md`

## 快速开始：生成新项目

```bash
agentcli new --module github.com/me/my-tool my-tool
cd my-tool
agentcli add command --preset file-sync sync-data
agentcli doctor --json        # 验证合规
task verify                   # 运行完整本地门禁
```

Monorepo 默认推荐：使用 `--in-existing-module`，避免嵌套 `go.mod`：

```bash
agentcli new --dir ./tools --in-existing-module replay-cli
cd tools/replay-cli
task verify
```

最小模式（只生成最小可运行骨架）：

```bash
agentcli new --minimal --module github.com/me/my-mini my-mini
```

项目结构示例：

```
my-tool/
├── main.go
├── cmd/root.go
├── internal/app/{bootstrap,lifecycle,errors}.go
├── internal/config/{schema,load}.go
├── internal/io/output.go
├── internal/tools/smokecheck/main.go
├── pkg/version/version.go
├── test/e2e/cli_test.go
├── test/smoke/version.schema.json
└── Taskfile.yml
```

可用预设：`file-sync`、`http-client`、`deploy-helper`、`task-replay-emit-wrapper`、`task-replay-orchestrator`

跨仓库编排推荐使用 `task-replay-orchestrator`：

```bash
agentcli add command --preset task-replay-orchestrator replay-orchestrate
go run . replay-orchestrate \
  --repo ../external-repo \
  --task replay:emit \
  --env IOC_ID=123 \
  --env MODE=baseline \
  --timeout 90s \
  --timeout-hook 'echo timeout for $AGENTCLI_TIMEOUT_TASK'
```

可选能力降级编译（当 loop/harness 相关包临时漂移时）：

- 仅保留 core scaffold/doctor 能力编译：
  - `go build -tags agentcli_core ./cmd/agentcli`
- `agentcli_core` 构建下，`loop` 与 `loop-server` 命令会被禁用。

## 示例

- [`examples/file-sync-cli`](./examples/file-sync-cli)
- [`examples/http-client-cli`](./examples/http-client-cli)
- [`examples/deploy-helper-cli`](./examples/deploy-helper-cli)

示例索引：[`examples/README.md`](./examples/README.md)

## 项目健康与治理

- License: [Apache-2.0](./LICENSE)
- 安全规范: [SECURITY.md](./SECURITY.md)
- 贡献指南: `CONTRIBUTING.md`
- 行为准则: [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)
- 变更日志: [CHANGELOG.md](./CHANGELOG.md)

## 文档规范

- 文档归属与更新位置定义见：[docs/documentation-conventions.md](./docs/documentation-conventions.md)。

## 面向 Agent 的工程工作流

如果作为 agent skill 使用，请先按 [`agents.md`](./agents.md) 执行初始化与检查。

## API 说明（完整版）

完整 API Reference 保持英文版本一致：[`README.md`](./README.md#api-reference)。

---

## 许可证

Apache-2.0
