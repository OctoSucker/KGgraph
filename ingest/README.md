# 图谱入库批次（v2 重建）

2026-08-28 起图谱 v2 重建：清空旧数据，从最基础的**八卦周易**开始，
先建好、评审通过，再做拓展导入。旧的手写批次（agent 逐书挑重点、无校验）
已全部移除，不再入库。

## 设计原则

1. **程序化、确定性**：基础层由 `kggraph seed-foundation` 生成，零 LLM；
   数据表在代码里，循环、校验都由程序完成，不依赖 agent 自驱动。
2. **逐字可回溯**：`corpus_extracted` 片段在生成时即与语料校验
   （去空白后必须逐字命中）；概括性常识标 `corpus_summary` 并附现代汉语译文。
3. **v2 结构化**：
   - 节点带 `domain`（子学科轴：周易 / 易学基础…）
   - 边带 `edge_kind`（structural / matrix / attribution…）与
     `condition_json`（结构化条件，如 先天/后天、上卦/下卦）
   - 证据带 `translation`（文言→现代汉语），`snippet` 保留原文
4. **跨体系重名加前缀**：如 `八卦·乾`（三爻经卦）与 `卦·乾为天`（六爻别卦）
   是不同节点；后续 `子平·七杀` / `紫微·七杀` 同理。

## 命令

```bash
# 生成基础批次（确定性，含原文校验）
kggraph seed-foundation --corpus gushu/zhouyi --out ingest/zhouyi-foundation.json

# 导入（不需要 embedding 时关掉，避免配额干扰）
kggraph import-graph --db data/kggraph.sqlite --file ingest/zhouyi-foundation.json \
  -embedding-model ""
```

## 进度

| 批次 | 内容 | 节点/边/证据 | 状态 |
| --- | --- | --- | --- |
| 01 | 八卦周易基础（阴阳/五行生克/八卦全属性/六十四卦构成与卦序） | 204 / 434 / 154 | 已入库 |

当前合计：204 节点 / 434 边 / 154 证据（56 逐字原文 + 98 概括标注）。
