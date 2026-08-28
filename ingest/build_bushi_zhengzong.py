#!/usr/bin/env python3
"""Build the eighteenth corpus batch: 卜筮正宗 (六爻断卦歌诀).

Output: ingest/bushi-zhengzong.json.
"""

import json

SR = "zhouyi-v2/gushu/bushi-zhengzong"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="卜筮格言", snippet="", st="corpus_extracted"):
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


node("卜筮正宗", "concept", ["王洪绪", "六爻"])
node("诚为卜本", "rule", ["至诚之道可以前知，卜者不诚不格，占卦者妄断不灵"])
edge("诚为卜本", "卜筮正宗", "is_a", conf=0.85, vol="卜筮格言",
     snippet="聖經曰：至誠之道，可以前知故問，卜者不誠不格，占卦者妄斷不靈，此二語實定論也。")

node("通玄赋", "rule", ["先论用神，次看原神，三合会用吉，六冲刑克俱主伤"])
edge("通玄赋", "卜筮正宗", "is_a", conf=0.85, vol="通玄赋",
     snippet="始須論用神，次必看原神，三合會用吉，祿馬最為良。爻動始為定，次者論空亡，六沖主沖并，刑克俱主傷。")
node("六神动应", "rule", ["龙动家有喜，虎动主有丧，勾陈朱雀动，田土与文章"])
edge("六神动应", "通玄赋", "is_a", conf=0.8, vol="通玄赋",
     snippet="龍動家有喜，虎動主有喪，勾陳朱雀動，田土與文章，財動憂尊長，父動損兒郎，子動男人滯，兄動女人殃。")

node("诸爻持世诀", "rule", ["世爻旺相最为强；父母持世主身劳；子孙持世事无忧；官鬼持世事难安；财爻持世益财荣；兄弟持世莫求财"])
edge("诸爻持世诀", "卜筮正宗", "is_a", conf=0.85, vol="诸爻持世",
     snippet="世爻旺相最為強，作事亨通大吉昌。父母持世主身勞。子身持世事無憂。鬼爻持世事難安。財爻持世益財榮。兄弟持世莫求財。")
edge("世爻旺相", "诸爻持世诀", "is_a", conf=0.85, vol="诸爻持世",
     snippet="世爻旺相最為強，作事亨通大吉昌，謀望諸般皆遂意。")
node("世爻旺相", "rule", ["世爻旺相作事亨通"])

node("忌神歌", "rule", ["看卦先须看忌神，忌神宜静不宜兴，宜逢伤克不宜生扶"])
edge("忌神歌", "卜筮正宗", "is_a", conf=0.85, vol="忌神歌",
     snippet="看卦先須看忌神，忌神宜靜不宜興，忌神急要逢傷克，若遇生扶用受刑。")
node("原神歌", "rule", ["原神发动志扬扬，须要生扶兼旺相，最嫌化克及逢伤"])
edge("原神歌", "卜筮正宗", "is_a", conf=0.85, vol="原神歌",
     snippet="原神發動志揚揚，用伏藏兮也不妨，須要生扶兼旺相，最嫌化克及逢傷。")
node("用神不上卦", "rule", ["正卦如无变又无，就将首卦六亲攻"])
edge("用神不上卦", "卜筮正宗", "is_a", conf=0.85, vol="用神不上卦诀",
     snippet="正卦如無變又無，就將首卦六親攻，動爻生用終須吉，若遇交重克用凶。")
node("用神空亡", "rule", ["发动逢冲不谓空，静空遇克却为空；忌神喜空，用神原神不可空"])
edge("用神空亡", "卜筮正宗", "is_a", conf=0.85, vol="用神空亡诀",
     snippet="發動逢沖不謂空，靜空遇克卻為空，忌神最喜逢空吉，用與原神不可空。")
node("用神发动", "rule", ["用爻发动纵值休囚亦不凶，得生扶旺相则亨通"])
edge("用神发动", "卜筮正宗", "is_a", conf=0.85, vol="用神发动诀",
     snippet="用爻發動在宮中，縱值休囚亦不凶，更得生扶兼旺相，管教作事永亨通。")
node("日辰诀", "rule", ["问卦先须看日辰，日辰克用不堪亲，相生合则作事遂心"])
edge("日辰诀", "卜筮正宗", "is_a", conf=0.85, vol="日辰诀",
     snippet="問卦先須看日辰，日辰克用不堪親，日辰與用相生合，作事何愁不起心。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/bushi-zhengzong.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
