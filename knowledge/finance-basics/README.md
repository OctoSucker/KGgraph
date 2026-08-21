# finance-basics

基础金融与交易知识图谱种子数据（人工整理、领域直录），供 kggraph 个人知识库使用。

- 规模：418 个节点、328 条边（去重后）
- 关系词汇：`is_a` / `part_of` / `example_of` / `measured_by` / `requires` / `enables` / `signals` / `precedes` / `hedges` / `increases` / `decreases` / `correlates_with` / `affects` / `contrasts_with` / `derived_from`
- 来源标记：`source_type=domain_curated`，`source_ref=finance-basics-v1/<域>`
- 置信度约定：结构性定义 0.9+；条件性/方向性关系 0.6-0.85，并附 `condition_text` 限定条件

## 域划分

| 文件 | 内容 |
|---|---|
| 01-market-structure | 市场、资产类别、交易所、订单类型、交易时段、结算、指数 |
| 02-microstructure | 订单簿、价差、滑点、保证金/杠杆/强平、仓位与盈亏 |
| 03-technical-analysis | 趋势、支撑阻力、均线/指标、K线与形态、量价、背离 |
| 04-risk-money | 风险度量、止损止盈、仓位管理、回测陷阱、绩效指标 |
| 05-fundamental | 财报三表、估值指标、盈利质量、回购分红解禁 |
| 06-macro | 通胀、利率、央行、收益率曲线、避险与风险资产、跨资产关系 |
| 07-crypto | 链上、钱包、稳定币、DEX/AMM、永续合约、减半、共识机制 |
| 08-psychology | 交易纪律、认知偏差、情绪化交易、复盘与绩效归因 |
| 09-supplement | 跨域共用节点补充 |

## 重新导入

```bash
for f in knowledge/finance-basics/*.json; do
  kggraph import-graph --file "$f"
done
```

默认写入用户数据目录（macOS：`~/Library/Application Support/kggraph/knowledgegraph.sqlite`）。想用独立工作区时加 `--workspace <dir>`。
