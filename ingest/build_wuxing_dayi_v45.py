#!/usr/bin/env python3
"""Build the nineteenth corpus batch: 五行大义 卷4-5 补充
(律吕/七政/五方之神/五帝).

Output: ingest/wuxing-dayi-v45.json.
"""

import json

SR = "zhouyi-v2/gushu/wuxing-dayi"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="卷4", snippet="", st="corpus_extracted"):
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


node("十二律吕", "rule", ["阳六律黄钟大蔟姑洗蕤宾夷则无射，阴六吕林钟南吕应钟大吕夹钟仲吕"])
edge("十二律吕", "五行大义", "is_a", conf=0.9, vol="卷4",
     snippet="陰陽各六。合有十二。陽六爲律。陰六爲呂。律六者。黃鍾。大蔟。姑洗。蕤賓。夷則。無射也。呂六者。林鐘。南呂。應鐘。大呂。夾鐘。仲呂也。")
node("五行大义", "concept", ["萧吉"])

node("七政三解", "rule", ["日月五星为七政，或北斗七星，或二十八宿"])
edge("七政三解", "五行大义", "is_a", conf=0.85, vol="卷4",
     snippet="七者。數有七也。凡有三解。一云。日月五星。合爲七政。二云。北斗七星爲七政。三云。二十八宿。布在四方。方別七宿。共爲七政。")

node("五方之神", "rule", ["勾芒木正，祝融火正，后土土正，蓐收金正，玄冥水正"])
edge("五方之神", "五行大义", "is_a", conf=0.9, vol="卷5",
     snippet="高辛氏立。五行名官。以勾芒爲木正。祝融爲火正。蓐收爲金正。玄冥爲水正。后土爲土正。分掌其職。")
for shen, w, snip in [
    ("勾芒", "木", "東海神名勾芒。春之月。其神勾芒。"),
    ("祝融", "火", "南海神名祝融。夏之月。其神祝融。"),
    ("后土", "土", "中央土。其神后土。"),
    ("蓐收", "金", "西海神名蓐收。秋之月。其神蓐收。"),
    ("玄冥", "水", "北海神名玄冥。冬之月。其神玄冥。"),
]:
    node(shen, "concept")
    node(w, "concept")
    edge(shen, w, "maps_to", conf=0.9, vol="卷5", snippet=snip)
    edge(shen, "五方之神", "belongs_to", conf=0.9, vol="卷5")

node("五帝", "rule", ["东方青帝灵威仰木，南方赤帝赤熛怒火，中央黄帝含枢纽土，西方白帝白招拒金，北方黑帝叶光纪水"])
edge("五帝", "五行大义", "is_a", conf=0.9, vol="卷5",
     snippet="東方青帝。靈威仰。木帝也。南方赤帝。赤熛怒。火帝也。中央黃帝。含樞紐。土帝也。西方白帝。白招拒。金帝也。北方黑帝。叶光紀。水帝也。")
for di, w, name, snip in [
    ("青帝", "木", "灵威仰", "東方青帝。靈威仰。木帝也。"),
    ("赤帝", "火", "赤熛怒", "南方赤帝。赤熛怒。火帝也。"),
    ("黄帝", "土", "含枢纽", "中央黃帝。含樞紐。土帝也。"),
    ("白帝", "金", "白招拒", "西方白帝。白招拒。金帝也。"),
    ("黑帝", "水", "叶光纪", "北方黑帝。叶光紀。水帝也。"),
]:
    node(di, "concept", [name])
    edge(di, w, "maps_to", conf=0.9, vol="卷5", snippet=snip)
    edge(di, "五帝", "belongs_to", conf=0.9, vol="卷5")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/wuxing-dayi-v45.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
