# KGgraph

KGgraph 是一个 Go 实现的知识图谱与规则推理引擎，定位是 AI Agent 的「确定性知识 / 记忆层」：图存储、来源溯源、冲突检查、确定性决策闸门，以及 MCP 集成。

## 文档导航

- [基础架构设计](docs/01-architecture.md)：设计目的、思路、改进过程与值得记录的决策
- [产品实践：周易八卦算术](docs/02-product-zhouyi-bagua.md)：古籍语料 → 知识图谱 → 前端与 Agent 的产品规划
- [Vibe Coding 哲学](docs/03-vibe-coding-philosophy.md)：基建先行、小步验证、流程固化的开发记录

## 特性

- CLI（`kggraph ...`）、MCP stdio server（`kggraph serve-mcp`）、本地图浏览器（`kggraph graph-view`）
- 确定性决策工作流：`record-decision` → `evaluate-decision` → `strict-ask` → `pre-trade-check`（决策时零 LLM，同一输入永远同一结论）
- 知识生命周期：来源、证据、验证、退休、冲突扫描，逐条可审计
- 本地优先：纯 Go SQLite，无外部数据库，离线可用

## 快速开始

```bash
make demo
# 或
bash examples/demo.sh
```

安装（Homebrew）：

```bash
brew tap 0xfakeSpike/tap
brew install kggraph
```

## 周易语料

31 部古籍语料（`gushu/`）、27 个入库批次（`ingest/`）。当前库内 927 节点 / 1099 边 / 798 证据（2026-08-28），详见 [产品实践文档](docs/02-product-zhouyi-bagua.md)。

## 设计边界

- 不是图数据库的替代品
- 不是无约束聊天机器人，决策模式刻意严格
- LLM 只做受限抽取，决策时不被调用
- 决策质量取决于用户提供的证据与失效条件
