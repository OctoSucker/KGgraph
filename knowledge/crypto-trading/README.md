# crypto-trading

加密货币交易干货知识图谱种子数据。定位：**可执行的交易机制与条件规则**，而非百科式定义。

## 规模

- 223 个节点、158 条边（去重后）
- 每条信号类边带 `condition_text` 限定条件与失效条件（`invalidates`）
- 来源标记：`source_type=domain_curated`，`source_ref=crypto-trading-v1/<域>`

## 域划分

| 文件 | 内容 |
|---|---|
| 01-liquidity-engine | 美元流动性、稳定币、ETF 与 BTC 的传导机制 |
| 02-price-structure | 通道/关键点/突破/假突破/多周期共振等价格结构规则 |
| 03-positioning | 持仓量/资金费率/爆仓/基差/多空比等合约博弈规则 |
| 04-onchain-flows | 交易所余额、巨鲸、链上活跃度与抛压/囤积信号 |
| 05-livermore | 《股票大作手回忆录》式交易原则：试探建仓、让利润奔跑、截断亏损 |
| 06-execution-risk | 结构位止损、波动率调整仓位、事件降杠杆、逆势抄底条件 |
| 07-mutex-validation | 信号冲突时结构优先、数据分层、时效性分级、互斥降权 |
| 08-extra-rules | 追高风险、清算规模、资金费率急变、流动性脆弱等补充规则 |

## 使用

```bash
# 完整规则检索（含触发条件与失效条件）
kggraph lookup-context --term "上升通道"

# 从任意概念多跳推理
kggraph expand-reasoning --start-id "资金费率" --max-depth 3

# 写入新主张前检查互斥
kggraph conflict-scan --from-id "阶段高低点同步下移" --to-id "上升通道" --relation-type signals
```

`lookup-context` 输出包含 `direction=in` 的入边（什么现象触发该状态、什么条件使其失效）与 `direction=out` 的出边（该状态导向什么动作）。

## 重新导入

```bash
for f in knowledge/crypto-trading/*.json; do
  kggraph import-graph --file "$f"
done
```

默认写入用户数据目录（macOS：`~/Library/Application Support/kggraph/knowledgegraph.sqlite`）。
