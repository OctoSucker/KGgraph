#!/usr/bin/env python3
"""Build the twenty-sixth corpus batch: 三命通会 (八字总纲).

Output: ingest/sanming-tonghui.json.  Snippets copied from simplified corpus.
"""

import json

SR = "zhouyi-v2/gushu/sanming-tonghui"

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


node("三命通会", "concept", ["万民英", "八字命理", "集大成"])
node("五行生成", "rule", ["水一火二木三金四土五，以阴阳奇偶配数；水生于一，太一者水之尊号"])
edge("五行生成", "三命通会", "is_a", conf=0.9, vol="卷一",
     snippet="一曰水，二曰火，三曰木，四曰金，五曰土者，咸有所自也。水，北方子之位也，子者，阳之初一，阳数也，故水曰一。水生于一。天地未分，万物未成之初，莫不先见于水，故《灵枢经》曰：太一者，水之尊号。")
node("五行生克总纲", "rule", ["十干十二支、五运六气、岁月日时皆自此立，形气相感而化生万物"])
edge("五行生克总纲", "三命通会", "is_a", conf=0.9, vol="卷一",
     snippet="五行相生相克，其理昭然。十干十二支、五运六气、岁月日时皆自此立，更相为用，在天则为气：寒、暑、燥、湿、风；在地则成形：金、木、水、火、土。形气相感而化生万物，此造化生成之大纪也。")
node("木主东春", "rule", ["木主于东应春，木之为言触也，阳气触动冒地而生"])
edge("木主东春", "三命通会", "is_a", conf=0.9, vol="卷一",
     snippet="木主于东；应春。木之为言触也，阳气触动，冒地而生也。水流趋东以生木也。")
node("火主南夏", "rule", ["火主于南应夏，火之为言化也毁也，钻木取火木所生"])
edge("火主南夏", "三命通会", "is_a", conf=0.9, vol="卷一",
     snippet="火主于南，应夏。火之为言化也，毁也，阳在上，阴在下；毁然盛而变化万物也。钻木取火，木所生也。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/sanming-tonghui.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
