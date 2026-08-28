#!/usr/bin/env python3
"""Build the fourteenth corpus batch: 六壬指南 (心印赋/指掌赋机制).

Output: ingest/liuren-zhinan.json.
"""

import json

SR = "zhouyi-v2/gushu/liuren-zhinan"

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


node("六壬", "concept", ["大六壬", "三式之一"])
node("六壬指南", "concept", ["陈公献", "心印赋", "指掌赋"])
edge("六壬指南", "六壬", "belongs_to", conf=0.95, vol="序")

node("日辰根本", "rule", ["日尊为天干，辰卑为地支，日辰为六壬根本"])
edge("日辰根本", "六壬", "is_a", conf=0.95, vol="卷一",
     snippet="六壬如人，先明日辰。日尊故曰天干，辰卑故曰地支。")

node("地盘天盘", "rule", ["月将加占时之上，顺布十二宫辰为天盘，地支方位为地盘"])
edge("地盘天盘", "六壬", "is_a", conf=0.9, vol="卷一",
     snippet="以月將加占時之上，順布十二宮辰，即天盤也。")

# 十二天将（月将）
yuejiang = [
    ("登明", "正月", "正月雨水後日躔娵訾之次入亥宮，乃登明將也。"),
    ("河魁", "二月", "二月春分後日躔降婁之次入戌宮，乃河魁將也。"),
    ("从魁", "三月", "三月穀雨後日躔大梁之次入酉宮，乃從魁將也。"),
    ("传送", "四月", "四月小滿後日躔實沈之次入申宮，乃傳送將也。"),
    ("小吉", "五月", "五月夏至後日躔鶉首之次入未宮，乃小吉將也。"),
    ("胜光", "六月", "六月大暑後日躔鶉火之次入午宮，乃勝光將也。"),
    ("太乙", "七月", "七月處暑後日躔鶉尾之次入巳宮，乃太乙將也。"),
    ("天罡", "八月", "八月秋分後日躔壽星之次入辰宮，乃天罡將也。"),
    ("太冲", "九月", "九月霜降後日躔大火之次入卯宮，乃太沖將也。"),
    ("功曹", "十月", "十月小雪後日躔析木之次入寅宮，乃功曹將也。"),
    ("大吉", "十一月", "十一月冬至後日躔星紀之次入丑宮，乃大吉將也。"),
    ("神后", "十二月", "十二月大寒後日躔玄枵之次入子宮，乃神後將也。"),
]
node("十二天将", "concept", ["登明河魁从魁传送小吉胜光太乙天罡太冲功曹大吉神后"])
for name, month, snip in yuejiang:
    node(name, "concept")
    edge(name, month, "maps_to", conf=0.9, vol="卷一", snippet=snip)
    node(month, "concept")
    edge(name, "十二天将", "belongs_to", conf=0.9, vol="卷一")

# 四课
node("四课", "rule", ["干上阳神第一课，干上阴神第二课，支上阳神第三课，支上阴神第四课"])
edge("四课", "六壬", "is_a", conf=0.9, vol="卷一",
     snippet="干上陽神為第一課，乃陽中之陽也；干上陰神為第二課，乃陽中之陰也；支上陽神為第三課，乃陰中之陽也；支上陰神為第四課，乃陰中之陰也。")

# 三传课体
kes = [
    ("元首课", "四课中并无下克，唯一上神克下，取而为用", "若四課中並無下克，唯一上神克下，取而用之名曰元首課。上克為元首，理順成而百事咸宜。"),
    ("重审课", "一下克其上神者取用", "若有一下克其上神者，雖有二三之上克下不論矣，名曰重審課。下賊為重審，人事逆而謀為不利。"),
    ("知一课", "二三克贼，看克处与本干有益无益", "二三克賊，知一總名。"),
    ("见机课", "用孟，当因时以致宜", "用孟名曰見機，當因時以致宜。"),
    ("察微课", "仲季，事未萌而预计", "仲季號為察微，事未萌而預計。"),
    ("涉害课", "克贼重重，用辰主外灾害己，用日主我祸延人", "克賊重重比涉害，用辰主外灾害己，用日主我禍延人。"),
    ("蒿矢课", "神遥克日", "蒿矢神遙克日，二克主兩事而合為一事。"),
    ("弹射课", "日遥克神", "彈射日遙克神，一克主一端而分作兩端。"),
    ("昴星课", "如虎对立，视俯仰以定远近", "昴星如虎對立，視俯仰以定遠近之憂危。"),
]
node("三传课体", "rule", ["九宗门发用课体"])
for name, alias, snip in kes:
    nid = "课·" + name
    node(nid, "rule", [alias])
    edge(nid, "三传课体", "is_a", conf=0.85, vol="卷一", snippet=snip)

node("六壬指掌总纲", "rule", ["六壬通万变之机，大为国而小为家，日为人而辰为事"])
edge("六壬指掌总纲", "六壬指南", "is_a", conf=0.85, vol="卷二",
     snippet="六壬通萬變之機，大為國而小為家。日辰定動靜之位，日為人而辰為事。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/liuren-zhinan.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
