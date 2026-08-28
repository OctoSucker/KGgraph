#!/usr/bin/env python3
"""Build the twenty-third corpus batch: 周易正义 (孔颖达疏义理).

Output: ingest/zhouyi-zhengyi.json.  Snippets are copied from the
simplified-Chinese corpus text.
"""

import json

SR = "zhouyi-v2/gushu/zhouyi-zhengyi"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="07-01", snippet="", st="corpus_extracted"):
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


node("周易正义", "concept", ["孔颖达", "五经正义", "义疏"])
node("易简", "rule", ["干以易知，坤以简能；天地之道不为而善始，不劳而善成"])
edge("易简", "周易正义", "is_a", conf=0.9, vol="07-01",
     snippet="干以易知者，易谓易略，无所造为，以此为知，故曰干以易知也。坤以简能者，简谓简省凝静，不须繁劳，以此为能，故曰坤以简能也。天地之道，不为而善始，不劳而善成，故曰易简。")
node("刚柔立本", "rule", ["刚柔之象立在其卦之根本，皆由刚柔阴阳往来"])
edge("刚柔立本", "周易正义", "is_a", conf=0.9, vol="08-1",
     snippet="刚柔者，立本者也，言刚柔之象，立在其卦之根本者也。言卦之根本，皆由刚柔阴阳往来。")
node("卦者时也", "rule", ["卦者时也，爻者趣时者也（王弼略例）"])
edge("卦者时也", "周易正义", "is_a", conf=0.85, vol="08-1",
     snippet="故《略例》云：卦者，时也；变通者，趣时者也。卦既总主一时，爻则就一时之中，各趣其所宜之时，故《略例》云：爻者，趣时者也。")
node("八卦成列", "rule", ["八卦成列，象在其中；因而重之，爻在其中"])
edge("八卦成列", "周易正义", "is_a", conf=0.9, vol="08-1",
     snippet="八卦成列，象在其中矣。夫八卦备天下之理，而未极其变，故因而重之以象其动用。")
node("极数知来", "rule", ["穷极蓍策之数豫知来事，谓之占；物穷则变，万事乃生"])
edge("极数知来", "周易正义", "is_a", conf=0.9, vol="07-05",
     snippet="极数知来之谓占者，谓穷极蓍策之数，豫知来事，占问吉凶，故云谓之占也。通变之谓事者，物之穷极，欲使开通，须知其变化，乃得通也。凡天下之事，穷则须变，万事乃生。")
node("系辞章数", "rule", ["系辞下章数诸儒不同，刘瓛为十二章，周氏庄氏并为九章，今从九章"])
edge("系辞章数", "周易正义", "is_a", conf=0.8, vol="08-1",
     snippet="此篇章数，诸儒不同，刘瓛为十二章，以对上《系》十二章也。周氏、庄氏并为九章，今从九章为说也。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/zhouyi-zhengyi.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
