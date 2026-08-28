# 周易命理知识图谱入库批次

本目录存放**从原文抽取的图谱批次**（import-graph JSON 格式），是
`gushu/` 原文语料 → 知识图谱之间的中间产物。

## 约定

- 每本书一个批次文件：`ingest/<slug>.json`，结构为
  `{nodes, edges, evidence}`，与 `kggraph import-graph` 兼容。
- `source_type` 统一为 `corpus_extracted`；`source_ref` 格式：
  `zhouyi-v2/gushu/<slug>/<章节>`。
- 证据（evidence）携带原文摘录 `snippet`，一条边至少一条证据。
- 经典机制性规则（五行生克、排盘、冲合、历法）置信度 0.85-0.95；
  经验断语（神煞吉凶、体用吉凶等）0.55-0.75 并标注条件/争议。

## 入库命令

```bash
kggraph import-graph --db data/kggraph.sqlite --file ingest/<slug>.json
kggraph conflict-scan --db data/kggraph.sqlite
```

## 进度

| 批次 | 书 | 状态 |
| --- | --- | --- |
| 01 | 五行大义 | 待抽取 |
