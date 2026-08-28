#!/usr/bin/env python3
"""Build the twenty-eighth corpus batch: 六壬大全 (起例九宗门).

Output: ingest/liuren-daquan.json.  Snippets copied from simplified corpus.
"""

import json

SR = "zhouyi-v2/gushu/liuren-daquan"

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


node("六壬大全", "concept", ["起例", "九宗门"])
node("十干寄宫", "rule", ["甲课寅乙课辰丙戊课巳丁己课未庚申辛戌壬亥癸丑，不用四正神"])
edge("十干寄宫", "六壬大全", "is_a", conf=0.9, vol="卷一",
     snippet="甲课寅兮乙课辰，丙戊课巳不须论。丁己课未庚申上，辛戌壬亥是其真。癸课原来丑宫坐，分明不用四正神。")
node("贼克法", "rule", ["一下克上曰重审，一上克下曰元首"])
edge("贼克法", "六壬大全", "is_a", conf=0.9, vol="卷一",
     snippet="取课先从下贼呼，如无下贼上克初。初传之上名中次，中上加临是末居。三传既定天盘将，此是入式法第一。")
node("比用知一", "rule", ["常将天日比神用，阳日用阳阴用阴"])
edge("比用知一", "六壬大全", "is_a", conf=0.9, vol="卷一",
     snippet="下贼或三二四侵，若逢上克亦同云。常将天日比神用，阳日用阳阴用阴。若或俱比俱不比，立法别有渉害陈。")
node("涉害法", "rule", ["涉害行来本家止，路逢多克为用取，孟深仲浅季当休"])
edge("涉害法", "六壬大全", "is_a", conf=0.9, vol="卷一",
     snippet="渉害行来本家止，路逢多克为用取。孟深仲浅季当休，复等柔辰刚日宜。")
node("遥克法", "rule", ["神遥克日曰蒿矢，日遥克神曰弹射"])
edge("遥克法", "六壬大全", "is_a", conf=0.9, vol="卷一",
     snippet="神遥克日曰蒿矢，日遥克神曰弹射。四课无克号为遥，日与神兮逓互招。先取神遥克其日，如无方取日来遥。")
node("昴星法", "rule", ["无遥无克昴星穷，阳仰阴俯酉位中"])
edge("昴星法", "六壬大全", "is_a", conf=0.9, vol="卷一",
     snippet="无遥无克昴星穷，阳仰阴俯酉位中论初传也。刚日先辰而后日，柔日先日而后辰论中末也。")
node("别责法", "rule", ["四课不全另取别责（戊辰戊午丙辰三刚日各一课等）"])
edge("别责法", "六壬大全", "is_a", conf=0.85, vol="卷一",
     snippet="别责法：戊辰、戊午、丙辰三刚日各一课，辛未二课，辛丑二课，丁酉。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/liuren-daquan.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
