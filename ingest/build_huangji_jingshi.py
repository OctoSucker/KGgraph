#!/usr/bin/env python3
"""Build the twenty-fifth corpus batch: 皇极经世 (元会运世/加一倍法).

Output: ingest/huangji-jingshi.json.  Snippets copied from simplified corpus.
"""

import json

SR = "zhouyi-v2/gushu/huangji-jingshi"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="卷一上", snippet="", st="corpus_extracted"):
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


node("皇极经世", "concept", ["邵雍", "元会运世", "先天学"])
node("元会运世", "rule", ["一元十二会，一会三十运，一运十二世，一世三十年；一元十二万九千六百年"])
edge("元会运世", "皇极经世", "is_a", conf=0.85, vol="卷一上",
     snippet="日甲一月卯四星甲九十一辰子一千八十一", cond="观物篇以元经会推步之表")
node("加一倍法", "rule", ["一分为二，二分为四，四分为八，八分为十六，十六分为三十二，三十二分为六十四"])
edge("加一倍法", "皇极经世", "is_a", conf=0.9, vol="卷十三",
     snippet="太极既分，两仪立矣。阳下交于阴，阴上交于阳，四象生矣。是故一分为二，二分为四，四分为八，八分为十六，十六分为三十二，三十二分为六十四。")
node("乾坤策数", "rule", ["干一爻三十六策，六爻二百一十六；坤一爻二十四策，六爻百四十四；二篇之策万有一千五百二十"])
edge("乾坤策数", "皇极经世", "is_a", conf=0.85, vol="卷十三",
     snippet="六其六得三十六，为干一爻之数也。积六爻之策，共得二百一十有六，为干之策。六其四得二十四，为坤一爻之策。积六爻之数，共得一百四十有四，为坤之策。积二篇之策，乃万有一千五百二十也。")
node("乾坤交泰", "rule", ["乾坤定位，震巽一交，兑离坎艮再交；否泰为乾坤之交"])
edge("乾坤交泰", "皇极经世", "is_a", conf=0.85, vol="卷十三",
     snippet="乾坤定位也，震、巽一交也，兑、离、坎、艮再交也。诸卦不交于乾坤者，则生于否泰。否泰，乾坤之交也。乾坤起自奇偶，奇偶生自太极。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/huangji-jingshi.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
