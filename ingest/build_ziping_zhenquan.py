#!/usr/bin/env python3
"""Build the sixth corpus batch: 子平真诠 (格局法核心).

Output: ingest/ziping-zhenquan.json.
"""

import json

SR = "zhouyi-v2/gushu/ziping-zhenquan"

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


node("子平真诠", "concept", ["沈孝瞻", "格局法"])
node("格局法", "rule", ["八字用神专求月令"])

rules = [
    ("用神专求月令", "八字用神，專求月令，以日乾配月令地支，而生克不同，格局分焉。", 0.95),
    ("顺用逆用", "財官印食，此用神之善而順用之者也；煞傷劫刃，用神之不善而逆用之者也。當順而順，當逆而逆，配合得宜，皆為貴格。", 0.9),
    ("扶抑取用", "日元強者抑之，日元弱者扶之，此以扶抑為用神也。", 0.9),
    ("病药取用", "以扶為喜，則以傷其扶者為病；以抑為喜，則以去其抑者為病。除其病神，即謂之藥。", 0.85),
    ("调候取用", "金水生於冬令，木火生於夏令，氣候太寒太燥，以調和氣候為急。此以調候為用神也。", 0.85),
    ("建禄月劫另取", "木生寅卯，日與月同，本身不可為用，必看四柱有無財官煞食透幹會支，另取用神；然終以月令為主。", 0.85),
    ("四柱有根", "十干不論月令休囚，只要四柱有根，便能受財官食神而當傷官七煞。", 0.9),
    ("根之轻重", "長生祿旺，根之重者也；墓庫餘氣，根之輕者也。天乾得一比肩，不如得支中一墓庫。", 0.85),
    ("日主不必旺", "人之日主，不必生逢祿旺，即月令休囚，而年日時中得長祿旺，便不為弱，就使逢庫，亦為有根。", 0.85),
    ("刑冲解刑", "刑衝俱不為美，而刑衝用神，尤為破格，不如以另位之刑衝，解月令之刑衝。", 0.8),
    ("相神紧要", "用神之情，不向日主，或日主之情，不向用神，皆非美朕也。", 0.8),
    ("四吉神破格", "四吉神能破格（财官印食顺用，逢忌神则破）。", 0.75),
    ("四凶神成格", "四凶神能成格（煞伤劫刃逆用，制化得宜则成）。", 0.75),
]
for name, snip, conf in rules:
    nid = "子平·" + name
    node(nid, "rule", [snip[:20]])
    edge(nid, "格局法", "is_a", conf=conf, vol="论用神", snippet=snip)

cheng = [
    ("官格成", "官逢財印，又無刑衝破害，官格成也。"),
    ("财格成", "財生官旺，或財逢食生而身強帶比，或財格透印而位置妥貼，兩不相克，財格成也。"),
    ("印格成", "印輕逢煞，或官印雙全，或身印兩旺而用食傷洩氣，或印多逢財而財透根輕，印格成也。"),
    ("食神格成", "食神生財，或食帶煞而無財，棄食就煞而透印，食格成也。"),
    ("煞格成", "身強七煞逢制，煞格成也。"),
    ("伤官格成", "傷官生財，或傷官佩印而傷官旺、印有根，或傷官帶煞而無財，傷官格成也。"),
    ("阳刃格成", "陽刃透官煞而露財印，不見傷官，陽刃格成也。"),
    ("建禄格成", "建祿月劫，透官而逢財印，透財而逢食傷，透煞而遇制伏，建祿月劫之格成也。"),
]
node("八格成败", "rule", ["用神成败救应"])
for name, snip in cheng:
    nid = "格成·" + name
    node(nid, "rule", [snip[:18]])
    edge(nid, "八格成败", "is_a", conf=0.85, vol="论用神成败救应", snippet=snip)

for g in ["正官格", "财格", "印绶格", "食神格", "偏官格", "伤官格", "阳刃格", "建禄月劫格"]:
    node(g, "concept")
    edge(g, "八格成败", "part_of", conf=0.8, vol="论正官")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/ziping-zhenquan.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
