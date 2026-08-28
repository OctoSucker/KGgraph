#!/usr/bin/env python3
"""Build the ninth corpus batch: 京氏易传 (八宫卦序/纳甲/飞伏/五行配位).

Output: ingest/jingshi-yizhuan.json.
"""

import json

SR = "zhouyi-v2/gushu/jingshi-yizhuan"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="卷上", snippet="", st="corpus_extracted"):
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


node("京氏易传", "concept", ["京房", "纳甲筮法源头"])
node("京房八宫", "rule", ["八纯卦各领七变卦，分八宫"])

palaces = {
    "乾": ["乾", "姤", "遁", "否", "觀", "剝", "晉", "大有"],
    "震": ["震", "豫", "解", "恆", "升", "井", "大過", "隨"],
    "坎": ["坎", "節", "屯", "既濟", "革", "豐", "明夷", "師"],
    "艮": ["艮", "賁", "大畜", "損", "睽", "履", "中孚", "漸"],
    "坤": ["坤", "復", "臨", "泰", "大壯", "夬", "需", "比"],
    "巽": ["巽", "小畜", "家人", "益", "無妄", "噬嗑", "頤", "蠱"],
    "離": ["離", "旅", "鼎", "未濟", "蒙", "渙", "訟", "同人"],
    "兌": ["兌", "困", "萃", "咸", "蹇", "謙", "小過", "歸妹"],
}
for palace, guas in palaces.items():
    nid = f"宫·{palace}"
    node(nid, "rule", [f"{palace}宫八卦"])
    edge(nid, "京房八宫", "is_a", conf=0.95, vol="卷上",
         snippet=f"{palace}宫：" + " ".join(guas))
    for g in guas:
        node(g, "concept")
        edge(g, nid, "belongs_to", conf=0.95, vol="卷上")

# 飞伏（对宫）
feifu = [
    ("乾", "坤", "與坤為飛伏，居世。"),
    ("震", "巽", "與巽為飛伏。"),
    ("坎", "離", "分北方之卦也，與離為飛伏。"),
    ("艮", "兌", "上艮下艮二象，土木分氣候，與兌為飛伏。"),
]
node("飞伏对宫", "rule", ["乾坤震巽坎离艮兑互为飞伏"])
for a, b, snip in feifu:
    edge(a, b, "pairs_with", pol=1, conf=0.9, vol="卷上", snippet=snip)
    edge(a, "飞伏对宫", "part_of", conf=0.9, vol="卷上")

# 纳甲与卦象五行
edge("乾", "甲壬", "maps_to", conf=0.9, vol="卷上",
     snippet="甲壬配外內二象。象配天，屬金。")
node("甲壬", "concept", ["乾纳甲壬"])
edge("乾", "金", "maps_to", conf=0.9, vol="卷上",
     snippet="象配天，屬金。")
edge("坤", "土", "maps_to", conf=0.9, vol="卷上",
     snippet="純陰用事，象配地，屬土。")
node("纳甲", "rule", ["八纯卦纳甲"])
edge("乾", "纳甲", "part_of", conf=0.9, vol="卷上")

# 乾宫五行配位（六亲）
node("乾宫五行配位", "rule", ["水为福德，木为宝贝，土为父母，火为相敌，金为比和"])
edge("乾宫五行配位", "京房八宫", "describes", conf=0.85, vol="卷上",
     snippet="水配位為褔德，木入金鄉居寶貝，土臨內象為父母，火來四上嫌相敵，金入金鄉木漸微。")

# 世应五等
node("世应五等", "rule", ["元士大夫三公诸侯天子为世，宗庙为应"])
edge("世应五等", "京房八宫", "describes", conf=0.85, vol="卷上",
     snippet="九三，三公為應。元士居世。大夫居世。諸侯居世。宗廟居世。")

# 阴阳消息与积算
node("阳极阴生", "rule", ["阳极则阴生"])
edge("阳极阴生", "京氏易传", "is_a", conf=0.85, vol="卷上",
     snippet="六位純陽，陰象在中。陽極陰生。")
edge("乾", "积算", "described_by", conf=0.8, vol="卷上",
     snippet="積算起己巳火，至戊辰土，周而復始。")
node("积算", "rule", ["京房积算推吉凶"])

# 取象
edge("乾", "马", "represents", conf=0.85, vol="卷上",
     snippet="於類：為馬，為龍。配於人事：為首，為君父。")
edge("乾", "君父", "represents", conf=0.85, vol="卷上")
edge("坤", "母", "represents", conf=0.85, vol="卷上",
     snippet="配於人事：為腹，為母。于類為馬。")
for s in ["马", "君父", "母"]:
    node(s, "concept")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/jingshi-yizhuan.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
