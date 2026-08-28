#!/usr/bin/env python3
"""Build the eleventh corpus batch: 紫微斗数全书 (安星法/主星/四化).

Output: ingest/ziwei-doushu.json.
"""

import json

SR = "zhouyi-v2/gushu/ziwei-doushu"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="卷一", snippet="", st="corpus_extracted"):
    global edge_id
    edge_id += 1
    edges.append({
        "id": edge_id, "from_id": frm, "to_id": to, "relation_type": rel,
        "polarity": pol, "confidence": conf, "condition_text": cond,
        "source_type": st, "source_ref": f"{SR}/{vol}",
    })
    if snippet:
        evidence.append({
            "edge_id": edge_id, "source_type": st,
            "source_ref": f"{SR}/{vol}", "snippet": snippet,
            "supports": True, "weight": 1.0,
        })


node("紫微斗数", "concept", ["紫微", "斗数", "陈抟"])
node("安身命", "rule", ["寅上起正月顺数至生月，生月起子时逆安命顺安身"])
edge("安身命", "紫微斗数", "is_a", conf=0.9, vol="卷二",
     snippet="大抵人命俱从寅上起正月，顺数至本生月止，又自人生月起子时逆至本生时安命，顺至本生时安身。")

node("十二宫", "rule", ["命宫兄弟妻妾子女财帛疾厄迁移奴仆官禄田宅福德父母"])
edge("十二宫", "紫微斗数", "is_a", conf=0.95, vol="卷二",
     snippet="安十二宫例 男女俱从逆转切忌莫顺去：一 命宫、二兄弟、三妻妾、四子女、五财帛、六疾厄、七迁移、八奴仆、九官禄、十田宅、十一福德、十二父母。")

wuhu = [
    ("甲己", "丙寅", "甲己之岁起丙寅"),
    ("乙庚", "戊寅", "乙庚之岁起戊寅"),
    ("丙辛", "庚寅", "丙辛之岁起庚寅"),
    ("丁壬", "壬寅", "丁壬之岁起壬寅"),
    ("戊癸", "甲寅", "戊癸之岁起甲寅"),
]
node("五虎遁", "rule", ["起五行寅诀"])
for a, b, snip in wuhu:
    edge(a, b, "maps_to", conf=0.95, vol="卷二", snippet=snip)
    node(a, "concept")
    node(b, "concept")
    edge(a, "五虎遁", "part_of", conf=0.9, vol="卷二")

# 十四主星
zhuxing = [
    ("紫微", "土", "中天之尊星为帝座，主掌造化枢机，人生主宰。诸宫降福能消百恶。"),
    ("天机", "木", "南斗第三益算之善星，化气曰善，解诸星之顺逆。"),
    ("太阳", "火", "日之精也，主人有贵气，能为文为武，乃官禄之枢机。"),
    ("武曲", "金", "北斗第六星，乃财帛宫主。性刚果决，在天司寿，在数司财。"),
    ("天同", "水", "南方第四星，为福德宫之主宰，化福，十二宫中皆曰福。"),
    ("廉贞", "火", "北斗第五星，在斗司品秩，在数司权令，化囚为杀。"),
    ("天府", "土", "南斗主令第一星，为财帛之主宰，司命延寿解厄。"),
    ("太阴", "水", "水之精，为田宅主，化富，与日为配。"),
    ("贪狼", "水", "北斗解厄之神第一星，化气为桃花，主祸福。"),
    ("巨门", "水金", "北斗第二星，为阴精之星，化气为暗，主口舌是非。"),
    ("天相", "水", "南斗第五星，为司爵之宿，化气曰印，官禄文星。"),
    ("天梁", "土", "南斗第二星，司寿化气为荫为福寿，化暴戾为祥和。"),
    ("七杀", "火金", "南斗第六星，斗中上将，成败之孤辰，主风宪。"),
    ("破军", "水", "北斗第七星，司夫妻子息奴仆之神，化气曰耗。"),
    ("文昌", "金", "主科甲，守身命主人幽闲儒雅，一举成名。"),
]
node("十四主星", "concept", ["紫微天机太阳武曲天同廉贞天府太阴贪狼巨门天相天梁七杀破军"])
for name, w, desc in zhuxing:
    node(name, "concept", [f"{name}星"])
    for ww in w:
        node(ww, "concept")
        edge(name, ww, "maps_to", conf=0.9, vol="卷一",
             snippet=f"{name}屬{ww}。{desc}")
    edge(name, "十四主星", "belongs_to", conf=0.9, vol="卷一")

node("四化", "concept", ["化禄化权化科化忌"])
for h in ["化禄", "化权", "化科", "化忌"]:
    node(h, "concept")
    edge(h, "四化", "belongs_to", conf=0.9, vol="卷一",
         snippet=f"问{h}星所主若何？")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/ziwei-doushu.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
