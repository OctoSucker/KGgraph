#!/usr/bin/env python3
"""Build the twenty-ninth corpus batch: 星学大成 (星命推步).

Output: ingest/xingxue-dacheng.json.  Snippets copied from simplified corpus.
"""

import json

SR = "zhouyi-v2/gushu/xingxue-dacheng"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="卷01", snippet="", st="corpus_extracted"):
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


node("星学大成", "concept", ["万民英", "星命", "五星"])
node("周天度数", "rule", ["周天三百六十五度二十分五十秒分十二宫，子午两宫各三十度四十三分八十秒，余十宫各三十度四十三分七十九秒"])
edge("周天度数", "星学大成", "is_a", conf=0.9, vol="卷01",
     snippet="右周天都计三百六十五度二十分五十秒分布十二宫，惟子午两宫每宫计三十度四十三分八十秒，其余十宫每宫计三十度四十三分七十九秒。")
node("天盘立命", "rule", ["天盘顺数天左旋，以天盘之卯加之，卯上分明是命宫"])
edge("天盘立命", "星学大成", "is_a", conf=0.85, vol="卷01",
     snippet="天盘之法顺数天左旋也。人之立命在十二宫有不同，而以天盘之卯加之则同。经云：天盘转出地盘上，卯上分明是命宫是也。")
node("十二宫分野", "rule", ["析木在寅，大火在卯，寿星在辰，鹑尾在巳，鹑火在午，鹑首在未，实沈在申，大梁在酉，降娄在戌，娵訾在亥，玄枵在子，星纪在丑"])
edge("十二宫分野", "星学大成", "is_a", conf=0.9, vol="卷01",
     snippet="析木原来本在寅，大火在卯寿星辰，鹑尾在已鹑火午，鹑首在未实沈申，大梁居酉降娄戌，娵訾在亥定其真，玄枵在子星纪丑，十二宫中仔细寻。")
node("变曜管库", "rule", ["天元禄管官禄，天暗属相貌，天福属财帛福德迁移，天耗属兄弟，天荫属妻妾田宅，天贵属男女，天刑属奴仆，天囚属疾厄，天权属命宫"])
edge("变曜管库", "星学大成", "is_a", conf=0.85, vol="卷01",
     snippet="凡当年之变为天元禄者，即其星之管官禄也。变为暗者属相貌，变为福者属财帛福德迁移，变为耗者属兄弟，变为荫者属妻妾，变为贵者属男女，变为刑者属奴仆，变为囚者属疾厄，变为权者属命宫。此又变曜之所属也，故名为管库星云。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/xingxue-dacheng.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
