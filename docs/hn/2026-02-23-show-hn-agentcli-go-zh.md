# Show HN: agentcli-go

## 标题建议

1. Show HN: agentcli-go — 为 AI 代理打造可控的 Go CLI 框架
2. Show HN: agentcli-go — 用可验证约束快速生成 Go CLI
3. Show HN: agentcli-go — 从提示词到可上线 CLI 的确定性工程化路线

## 精简版（推荐）

我做了 `agentcli-go`，这是一个让 AI 代理安全、稳定地生成和维护 Go CLI 的工程化框架。

核心目标：让“生成 CLI”的流程从试错式改造，变成可复现的工程过程。

关键点：
- 严格标准化的脚手架项目结构
- 机器可读的健康检查（例如 `agentcli doctor --json`）
- 输出 schema 校验
- CI 约束与回归机制，保障生成产物稳定性

我想解决的问题：AI 生成的 CLI 不应该引入长期漂移，尤其在内部工具和自动化场景。

我想请教社区：
1. 从零到第一条可运行命令，入门是否清晰？
2. 现在的严格程度是否过高，还是刚好够用？
3. 你最关心哪些扩展点来决定是否采纳？

快速上手：
```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.1
agentcli new --module example.com/mycli mycli
agentcli add command --dir ./mycli --preset file-sync sync-data
agentcli doctor --dir ./mycli --json
cd mycli && task verify
```

当前内部试用基线：
- 首次脚手架成功：约 1 分钟
- 首次 `task verify` 通关：约 1 分钟
- `doctor` 从失败到绿色的中位迭代次数：1 次

一句话总结：
让 AI 代理也能稳定、可控地“持续地”演化 Go CLI。

## 详细版

`agentcli-go` 面向一个真实问题：
**如何让 AI 代理在自动化中创建和修改 CLI 时，不让项目逐步失控？**

我把它做成了“显式约束 + 验收入口”的组合：
- 固定的项目骨架，避免每次生成风格不同
- 支持机器读取的输出模式（`--json`）
- 健康检查可解析（`doctor --json`）
- smoke 输出使用 schema 校验
- CI 中加入负向用例，确保异常输出会被拦截

这样生成的 CLI 更容易被团队、流水线和自动化系统长期信任。

欢迎反馈：
1. 首次接入摩擦点
2. 严格性与灵活性的平衡是否合理
3. 在开源场景里，扩展模型还有哪些缺口

有需要的话，我也可以补充一版“代理驱动项目迁移前后对比”的案例。

## 参考链接写法

如果引用外部观点（例如访谈、公开讨论），建议：
- 用自己的话转述核心结论
- 直接给出处链接
- 避免长篇逐字引述
