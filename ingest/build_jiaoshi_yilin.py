#!/usr/bin/env python3
"""Build the twenty-fourth corpus batch: 焦氏易林 (体例/卦辞模式).

Output: ingest/jiaoshi-yilin.json.  Snippets copied from simplified corpus.
"""

import json

SR = "zhouyi-v2/gushu/jiaoshi-yilin"

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


node("焦氏易林", "concept", ["焦赣", "易林", "四言卦辞"])
node("易林体例", "rule", ["汉焦赣撰，四言韵语为主，每卦六十四变占，以本卦起句"])
edge("易林体例", "焦氏易林", "is_a", conf=0.9, vol="卷一",
     snippet="焦氏易林卷一　　　　　　汉　焦赣　撰干之第一")

examples = [
    ("乾之乾", "道陟多阪，胡言连蹇，译瘠且聋，莫使道通，请谒不行，求事无功。"),
    ("乾之坤", "招殃来螫，害我邦国，病伤手足，不得安息。"),
    ("乾之屯", "阳孤亢极，多所恨惑，车倾葢亡，身常忧惶，乃得其愿，雌雄相从。"),
    ("乾之蒙", "鹄鵴鸤鸠，专一无尤，君子是则，长受嘉福。"),
    ("乾之需", "目𥆧足动，喜如其愿，举家蒙宠。"),
    ("乾之比", "中夜犬吠，盗在墙外，神明祐助，消散皆去。"),
]
node("卦辞示例", "rule", ["以本卦之变卦为题，四言断占"])
for name, verse in examples:
    nid = "易林·" + name
    node(nid, "rule", [verse[:12]])
    edge(nid, "卦辞示例", "is_a", conf=0.85, vol="卷一", snippet=verse)

node("易林取象", "rule", ["以卦象、干支、历史典故与自然物象入辞断占"])
edge("易林取象", "焦氏易林", "is_a", conf=0.8, vol="原序",
     snippet="较定易林原序")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/jiaoshi-yilin.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
