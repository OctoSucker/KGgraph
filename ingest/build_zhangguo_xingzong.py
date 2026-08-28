#!/usr/bin/env python3
"""Build the twenty-seventh corpus batch: 张果星宗 (七政四余星命).

Output: ingest/zhangguo-xingzong.json.  Snippets copied from simplified corpus.
"""

import json

SR = "zhouyi-v2/gushu/zhangguo-xingzong"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="全文", snippet="", st="corpus_extracted"):
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


node("张果星宗", "concept", ["果老星宗", "七政四余", "星命"])
node("七政四余", "rule", ["以日月五星（七政）加四余排星盘断命"])
edge("七政四余", "张果星宗", "is_a", conf=0.9, vol="全文",
     snippet="排下七政四余原，掌身命、官福、田财、妻嗣及文魁、经纬、三元、四元等星。")
node("周天度数", "rule", ["周天三百六十五度四分度之一，分配十二宫"])
edge("周天度数", "张果星宗", "is_a", conf=0.9, vol="全文",
     snippet="度数过宫，周天三百六十五度四分，度之一分配十二宫，其过宫，分秒具于图中。")
node("强宫", "rule", ["强宫谓命宫、官禄、田宅、妻妾、男女、福德、财帛，居强宫者旺"])
edge("强宫", "张果星宗", "is_a", conf=0.9, vol="全文",
     snippet="强宫者，谓命宫、官禄、田宅、妻妾、男女、福德、财帛，又云：财帛次弱，与其命宫相违故耳。居强宫者旺。")
node("弱宫", "rule", ["弱宫谓兄弟、奴仆、疾厄、相貌、迁移，临弱宫者衰"])
edge("弱宫", "张果星宗", "is_a", conf=0.9, vol="全文",
     snippet="弱宫者，谓兄弟、奴仆、疾厄、相貌、迁移，又云：迁移近强，与其命宫相向故也。临弱宫者衰。")
node("变曜配宫", "rule", ["天禄管官禄，天暗属相貌，天福属财帛福德迁移，天耗属兄弟，天荫属妻妾"])
edge("变曜配宫", "张果星宗", "is_a", conf=0.85, vol="全文",
     snippet="凡当年星变为天禄者，即其星管官禄也。变为暗者属相貌，变为福者属财帛、福德迁移，变为耗者属兄弟，变为荫者属妻妾。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/zhangguo-xingzong.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
