#!/usr/bin/env python3
"""Build the fifteenth corpus batch: 易纬乾坤凿度 (象数易纲领).

Output: ingest/yijing-qiankun-zaodu.json.
"""

import json

SR = "zhouyi-v2/gushu/yijing-qiankun-zaodu"

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


node("乾坤凿度", "concept", ["易纬", "乾凿度", "象数易"])
node("太易太极乾坤", "rule", ["太易始著，太极成，太极成乾坤行"])
edge("太易太极乾坤", "乾坤凿度", "is_a", conf=0.85, vol="卷上",
     snippet="太易始著太極成太極成乾坤行。")
node("太初太始太素", "rule", ["太初而后有太始，太始而后有太素"])
edge("太初太始太素", "乾坤凿度", "is_a", conf=0.85, vol="卷上",
     snippet="太易變教民不倦，太初而後有太始，太始而後有太素，有形始於弗形，有法始於弗法。")
node("乾训健", "rule", ["乾者天也川也先也，乾训健"])
edge("乾训健", "乾坤凿度", "is_a", conf=0.85, vol="卷上",
     snippet="乾鑿度聖人頤乾道浩大以天門為名也。乾者天也川也先也，乾訓健。")
node("易纬十四文", "rule", ["先元皇介垂皇策万形经乾文纬乾凿度考灵经制灵图河图八文等"])
edge("易纬十四文", "乾坤凿度", "is_a", conf=0.8, vol="卷上",
     snippet="先元皇介而後有垂皇筞而後有萬形經而後有乾文緯而後有乾鑿度而後有考靈經而後有制靈圖而後有河圖八文而後有希夷名而後有含文嘉而後有稽命圖而後有墳文而後有八文大籀而後有元命包一十四文。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/yijing-qiankun-zaodu.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
