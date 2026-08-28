#!/usr/bin/env python3
"""Build the twenty-first corpus batch: 易学启蒙 (朱熹象数纲领).

Output: ingest/yixue-qimeng.json.
"""

import json

SR = "zhouyi-v2/gushu/yixue-qimeng"

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


node("易学启蒙", "concept", ["朱熹", "象数", "本图书原卦画明蓍策考变占"])
node("河图洛书", "rule", ["河出图，洛出书，圣人则之；河图载五十有五之数，洛书具九畴之数"])
edge("河图洛书", "易学启蒙", "is_a", conf=0.9, vol="卷上",
     snippet="河出圖洛出書聖人則之。河圖與易之天一地十者合而載天地五十有五之數，則固易之所自生也。洛書與洪範之初一至次九者合而具九疇之數，則固範之所自出也。")
node("河图", "concept", ["五十有五之数"])
node("洛书", "concept", ["九畴之数"])
edge("河图", "八卦", "generates", pol=1, conf=0.9, vol="卷上",
     snippet="伏羲氏繼天而王受河圖而畫之八卦是也。")
node("八卦", "concept")
edge("洛书", "九畴", "generates", pol=1, conf=0.9, vol="卷上",
     snippet="禹治洪水賜洛書法而陳之九疇是也。")
node("九畴", "concept", ["洪范九畴"])
edge("河图洛书", "八卦九章", "relates_to", conf=0.85, vol="卷上",
     snippet="河圖洛書相為經緯，八卦九章相為表裏。")
node("八卦九章", "rule")

node("大衍五十", "rule", ["河图洛书中数皆五，衍之各极其数以合五十"])
edge("大衍五十", "易学启蒙", "is_a", conf=0.9, vol="卷下",
     snippet="河圖洛書之中數皆五，衍之而各極其數以至於十，則合為五十矣。")
node("五为数宗", "rule", ["天地之数不过五，五者数之祖，于五常为信"])
edge("五为数宗", "大衍五十", "describes", conf=0.85, vol="卷下",
     snippet="天地之數不過五而已，五者數之祖也。河圖洛書皆五居中而為數祖宗。此五也於五行為土，於五常為信，水火木金不得土不能各成一器。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/yixue-qimeng.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
