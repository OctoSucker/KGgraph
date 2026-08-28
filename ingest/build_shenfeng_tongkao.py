#!/usr/bin/env python3
"""Build the sixteenth corpus batch: 神峰通考 (病药说/八法/七杀论).

Output: ingest/shenfeng-tongkao.json.
"""

import json

SR = "zhouyi-v2/gushu/shenfeng-tongkao"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="病药说类", snippet="", st="corpus_extracted"):
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


node("神峰通考", "concept", ["张楠", "命理正宗", "病药说"])
node("病药说", "rule", ["有病方为贵，无伤不是奇；格中如去病，财禄两相随"])
edge("病药说", "神峰通考", "is_a", conf=0.9, vol="病药说类",
     snippet="何以為之病？原八字中原所害之神也；何以為之藥？如八字原有所害之字，而得一字以去之謂了。故書云：有病方為貴，無傷不是奇；格中如去病，財祿兩相隨。")
node("病", "concept", ["八字中原所害之神"])
node("药", "concept", ["得一字以去其病"])
edge("病", "药", "cured_by", pol=1, conf=0.85, vol="病药说类",
     snippet="如八字原有所害之字，而得一字以去之謂了，如朱子所謂各因其病而藥之也。")

examples = [
    ("土病木药", "四柱纯土水，日干为杀重身轻；如金日干则为土厚埋金；俱喜木为医药以去其病。"),
    ("比肩病官杀药", "用财见比肩为病，喜见官杀为药。"),
    ("印病财药", "用食神伤官，以印为病，喜财为药。"),
]
for name, snip in examples:
    nid = "病药例·" + name
    node(nid, "rule", [snip[:18]])
    edge(nid, "病药说", "is_a", conf=0.85, vol="病药说类", snippet=snip)

node("八法", "rule", ["雕枯弱旺损益生长八法"])
edge("八法", "神峰通考", "is_a", conf=0.85, vol="叙",
     snippet="立五星正說，五星謬說，子平諸格正說，子平諸格謬說，動靜說，蓋頭說，六親說，病藥說，雕枯弱旺損益生長八法說。")

node("有杀先论杀", "rule", ["有杀须论杀，无杀方论用"])
edge("有杀先论杀", "神峰通考", "is_a", conf=0.9, vol="偏官",
     snippet="凡看命先看七殺，若有七殺，就要將此七殺處置了，方能用得別物。原書去云：有殺須論殺，無殺方論用。")
node("偏官七杀", "rule", ["偏官无制曰七杀，宜制伏，畏太过亦畏不及"])
edge("偏官七杀", "有杀先论杀", "describes", conf=0.9, vol="偏官",
     snippet="偏官即七殺也，如甲日干，數至七個字逢庚字，號為七殺，乃克身之刀劍一般，偏官無制曰七殺，故宜制伏，亦畏太過，亦畏不及。")
node("弃命从杀", "rule", ["日主全无一点生气，四柱纯然有官杀，不得已而从杀"])
edge("弃命从杀", "有杀先论杀", "is_a", conf=0.85, vol="偏官",
     snippet="棄命從殺格，緣日主全無一點生氣，四柱純然有官殺，則不得已而只得從殺也。")

node("正官论", "rule", ["月上官星要官旺，官星太弱喜财生，太旺喜食制"])
edge("正官论", "神峰通考", "is_a", conf=0.85, vol="正官",
     snippet="大抵用月上官星，要官旺，官旺方好取用。要官星有病，各因病而藥之，官旺官多，喜食神以制去之，官星氣弱，喜財神以生之。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/shenfeng-tongkao.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
