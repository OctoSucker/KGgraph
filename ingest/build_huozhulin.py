#!/usr/bin/env python3
"""Build the fifth corpus batch: 火珠林 (六爻纳甲机制).

Output: ingest/huozhulin.json.
"""

import json

SR = "zhouyi-v2/gushu/huozhulin"

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


node("火珠林", "concept", ["六爻纳甲", "京房纳甲"])
node("六爻", "concept", ["六爻预测", "纳甲筮法"])

node("六亲", "concept", ["父母兄弟妻财子孙官鬼"])
for q in ["父母", "兄弟", "妻财", "子孙", "官鬼"]:
    node(q, "concept")
edge("火珠林", "六爻", "belongs_to", conf=0.95, vol="易中明义",
     snippet="四营成易，八卦为体；三才变化，六爻为义。")
edge("六亲", "六爻", "part_of", conf=0.95, vol="六亲根源",
     snippet="卦定根源，六亲为主；又究旁通，五行而取。")
edge("卦身", "六亲", "belongs_to", conf=0.9, vol="六亲根源",
     snippet="父母、兄弟、妻财、子孙、官鬼，只有五件，而曰六亲何也？答曰：卦身当一亲。")
node("卦身", "rule", ["阳世从子月起，阴世还当午月生"])
edge("卦身", "六爻", "part_of", conf=0.9, vol="六亲根源",
     snippet="阳世则从子月起，阴世还当午月生，此即卦身也。")

node("飞伏神", "rule", ["本宫六亲在飞象之下为伏神"])
edge("飞伏神", "六爻", "part_of", conf=0.9, vol="六亲根源",
     snippet="本宫之六亲在飞象之下，为之亲王，为之伏神。旁宫之飞象加伏神之上，为飞象。知飞伏二爻之来历，然后可与言八卦、六亲矣。")
node("出现伏藏", "rule", ["出现旺相为久为远，伏藏有气只利暂时"])
edge("出现伏藏", "飞伏神", "describes", conf=0.85, vol="出现伏藏",
     snippet="出现旺相，为久为远；伏藏有气，只利暂时。")

node("用神取法", "rule", ["乱动之卦只取旺爻，旺爻即用神；伏爻要旺相，动爻要生世"])
edge("用神取法", "六爻", "part_of", conf=0.9, vol="独发乱动",
     snippet="乱动之卦只取旺爻，旺爻即用神也。生克吉凶皆在此爻。若伏藏安静要旺相，若发动，却要生世之爻为用神。")
node("官用取官", "rule", ["官用取官，私用取财"])
edge("官用取官", "用神取法", "is_a", conf=0.9, vol="公私用事",
     snippet="官用取官，私用取财。占病鬼祟，占失看贼，占求官事，占官词讼，占婚问夫，以上皆看官爻。")
edge("私用取财", "用神取法", "is_a", conf=0.9, vol="公私用事",
     snippet="占买卖财，占家事，占婚婢事，占求财事，占婚姻事，以上皆看财爻。")
node("私用取财", "rule", ["占财看财爻"])

node("世应", "rule", ["占财要财爻持世，占官要官爻持世"])
edge("世应", "六爻", "part_of", conf=0.9, vol="世应相克",
     snippet="占财要财爻持世，占官要官爻持世，若应又是世之墓，动又是世之墓，皆不中矣。")
node("辅爻", "rule", ["财以子孙为辅，官以父母为辅"])
edge("辅爻", "财官辅助", "is_a", conf=0.85, vol="财官辅助")
node("财官辅助", "rule", ["用有辅助，类可忖量"])
edge("财官辅助", "六爻", "part_of", conf=0.85, vol="财官辅助",
     snippet="财官异路，可辨五乡；用有辅助，类可忖量。")
edge("官鬼", "父母", "requires", conf=0.8, vol="公私用事",
     snippet="占官必用父母，占财必用子孙。")
edge("妻财", "子孙", "requires", conf=0.8, vol="公私用事",
     snippet="占官必用父母，占财必用子孙。兄弟是破财之人，不为主、不为辅。")

node("纳音三才", "rule", ["天干管天文，地支管人事，纳音管地理"])
edge("纳音三才", "六爻", "part_of", conf=0.85, vol="易中明义",
     snippet="六十甲子生成，变化而行鬼神，是故天干管天文，地支管人事，纳音管地理。")

wangxiang = [
    ("春", "木", "火", "水", "金", "土", "春，寅卯木旺，巳午火相，亥子水休，申酉金囚，辰戌丑未土死。"),
    ("夏", "火", "土", "木", "水", "金", "夏，巳午火旺，辰戌丑未土相，寅卯木休，亥子水囚，申酉金死。"),
    ("秋", "金", "水", "土", "火", "木", "秋，申酉金旺，亥子水相，辰戌丑未土休，巳午火囚，寅卯木死。"),
    ("冬", "水", "木", "金", "土", "火", "冬，亥子水旺，寅卯木相，申酉金休，辰戌丑未土囚，巳午火死。"),
]
node("六爻旺相", "rule", ["春木旺火相水休金囚土死之属"])
for season, wang, xiang, xiu, qiu, si, snip in wangxiang:
    nid = f"旺相·{season}"
    node(nid, "rule", [f"{season}{wang}旺{xiang}相"])
    edge(nid, "六爻旺相", "is_a", conf=0.9, vol="财官辅助", snippet=snip)

cai = [
    ("财伏鬼乡", "财伏鬼乡，买卖遭伤；日辰福德，方始荣昌。", 0.8),
    ("财伏兄弟", "用财伏兄，口舌相侵；若在世下，旺相可成。", 0.8),
    ("财伏父母", "财伏父母，旺相得半。", 0.8),
    ("财伏子孙", "财伏子孙，有气必满。", 0.8),
    ("鬼伏兄弟", "用鬼伏兄，同类欺凌；若不虚诈，人不一心。", 0.8),
]
node("六爻伏神断", "rule", ["财官伏于六亲之下的吉凶断法"])
for name, verse, conf in cai:
    nid = "伏断·" + name
    node(nid, "rule", [verse[:20]])
    edge(nid, "六爻伏神断", "is_a", conf=conf, vol=name, snippet=verse)

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/huozhulin.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
