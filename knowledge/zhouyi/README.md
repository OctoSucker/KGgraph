# zhouyi

周易、八卦与八字命理知识图谱种子数据。定位：**Agent 记忆与知识基础设施的演示领域**——选择周易是因为它是 LLM 本身做不好的符号规则系统（生克、冲合刑害、十神、大运流年），恰好能体现确定性知识层的价值。

## 规模

- 334 个节点、491 条边（去重后）
- 来源标记：`source_type=domain_curated`，`source_ref=zhouyi-v1/<域>`
- 经典机制（生克、排盘、干支作用）置信度 0.85+；经验断语（神煞吉凶、体用吉凶）0.5-0.7 并标注争议

## 域划分

| 文件 | 内容 |
|---|---|
| 01-yinyang-wuxing | 阴阳、五行生克、旺相休囚死、方位/季节/颜色/脏腑映射 |
| 02-tiangan-dizhi | 十天干十二地支、五合六合三合、六冲三刑六害、藏干、月建、时辰 |
| 03-shishen-wangshuai | 十神规则、日主旺衰（得令得地得势）、用神喜忌与调候 |
| 04-bazi-paipan | 四柱排盘、立春/节气换柱、五虎遁五鼠遁、大运起运、流年应期 |
| 05-bagua-64gua | 八卦五行阴阳、先天后天卦序、后天方位、六十四卦（部分） |
| 06-liuyao-meihua | 六爻起卦、动爻变卦、体用、世应、六亲六神、月建日辰 |
| 07-shensha | 天乙贵人、桃花、驿马、羊刃、空亡等（统一标注争议） |
| 08-methodology | 经典来源、派别分歧、不可验证性、来源分级与互斥检查 |

## 使用示例

```bash
# 完整规则检索（含触发与失效条件）
kggraph lookup-context --term "身强"

# 从任意概念多跳推理（甲日主 -> 比肩/木 -> 十神/火/土 ...）
kggraph expand-reasoning --start-id "甲" --max-depth 3

# 写入新主张前检查互斥（木 相克 火 会命中库中 相生）
kggraph conflict-scan --from-id "木" --to-id "火" --relation-type 相克
```

## 重新导入

```bash
for f in knowledge/zhouyi/*.json; do
  kggraph import-graph --file "$f"
done
```

默认写入用户数据目录（macOS：`~/Library/Application Support/kggraph/knowledgegraph.sqlite`）。

## 诚实边界

- 经典机制（生克冲合、排盘规则）有明确文本依据，按高置信建模；
- 断语类知识（神煞吉凶、体用吉凶、六冲六合应期）派别分歧大且无法验证，统一低置信并标注争议；
- 本图谱是符号规则系统的结构化演示，不提供也不宣称“算命”准确性。
