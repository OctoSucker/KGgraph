#!/usr/bin/env python3
"""Build the twenty-second corpus batch: 周易集解 (象数集解体例).

Output: ingest/zhouyi-jijie.json.
"""

import json

SR = "zhouyi-v2/gushu/zhouyi-jijie"

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


node("周易集解", "concept", ["李鼎祚", "象数易", "集解"])
node("集解体例", "rule", ["采子夏马融郑玄王弼虞翻荀爽干宝崔憬等三十余家注"])
edge("集解体例", "周易集解", "is_a", conf=0.9, vol="卷01",
     snippet="唐李鼎祚撰。")
node("乾健", "rule", ["天之体以健为用，运行不息，应化无穷"])
edge("乾健", "周易集解", "is_a", conf=0.9, vol="卷01",
     snippet="乾健也，言天之體以健為用，運行不息，應化无窮，故聖人則之，欲使人法天之用，不法天之體，故名乾不名天也。")
node("四德", "rule", ["元始亨通利和贞正，君子法乾而行四德"])
edge("四德", "周易集解", "is_a", conf=0.9, vol="卷01",
     snippet="子夏傳曰：元始也，亨通也，利和也，貞正也。言乾稟純陽之性，故能首出庶物，各得元始開通和諧貞固不失其宜，是以君子法乾而行四德。")
node("老阳动占", "rule", ["九者老阳之数，动之所占"])
edge("老阳动占", "周易集解", "is_a", conf=0.9, vol="卷01",
     snippet="崔憬曰：九者老陽之數，動之所占，故陽稱焉。")
node("爻辰消息", "rule", ["初九建子之月阳气始动黄泉，十二消息卦配十二爻辰"])
edge("爻辰消息", "周易集解", "is_a", conf=0.85, vol="卷01",
     snippet="干寳曰：初九建子之月陽氣始動於黃泉，既未萌牙，猶是潛伏，故曰潛龍也。陽在九二十二月之時自臨來也。陽在九三正月之時自泰來也。")
node("纯阳生众卦", "rule", ["乾者纯阳众卦所生，万一千五百二十策皆受始于乾"])
edge("纯阳生众卦", "周易集解", "is_a", conf=0.85, vol="卷01",
     snippet="九家易曰：陽稱大，六爻純陽，故曰大，乾者純陽衆卦所生，天之象也。荀爽曰：謂分為六十四卦，萬一千五百二十策皆受始於乾也。")
node("卦变虞翻", "rule", ["阳息至三已变成离，离为日坤为夕（卦变取象）"])
edge("卦变虞翻", "周易集解", "is_a", conf=0.85, vol="卷01",
     snippet="虞翻曰：謂陽息至三，已變成離，離為日，坤為夕。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/zhouyi-jijie.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
