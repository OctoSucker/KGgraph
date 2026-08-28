#!/usr/bin/env python3
"""Build the seventeenth corpus batch: 麻衣相法 (形神气血/面部分区断法).

Output: ingest/mayi-xiangfa.json.
"""

import json

SR = "zhouyi-v2/gushu/mayi-xiangfa"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="论形", snippet="", st="corpus_extracted"):
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


node("麻衣相法", "concept", ["麻衣", "相学", "人相"])
node("相法总纲", "rule", ["头象天，足象地，眼象日月，声音象雷霆，血脉象江河"])
edge("相法总纲", "麻衣相法", "is_a", conf=0.9, vol="论形",
     snippet="人秉陰陽之氣，肖天地之形，受五行之資，為萬物之靈者也。故頭象天，足象地，眼象日月，聲音象雷霆，血脈象江河，骨節象金石，鼻額象山岳，毫發象草木。")

node("形神气血", "rule", ["形养血，血养气，气养神；形全则血全，血全则气全，气全则神全"])
edge("形神气血", "麻衣相法", "is_a", conf=0.9, vol="论神",
     snippet="夫形以養血，血以養氣，氣以養神，故形全則血全，血全則氣全，氣全則神全。")
edge("眼明神清", "贵", "signals", pol=1, conf=0.75, vol="论神",
     snippet="眼明則神清，眼昏則神濁。清則貴，濁則賤。")
node("眼明神清", "rule", ["眼明神清则贵，眼昏神浊则贱"])
node("贵", "concept")

node("声相", "rule", ["器大声宏，器小声短；贵人声出丹田，清而圆竖而亮"])
edge("声相", "麻衣相法", "is_a", conf=0.85, vol="论声",
     snippet="器大則聲宏，器小則聲短。神清則氣和、氣和則聲潤，深而圓暢也。貴人之聲，多出於丹田之中。")
node("气相", "rule", ["神完则气宽，神安则气静，重厚有福"])
edge("气相", "麻衣相法", "is_a", conf=0.85, vol="论气",
     snippet="神完則氣寬，神安則氣靜。得失不足以暴其氣，喜怒不足以驚其神。則於德為有容，於量為有度，乃重厚有福之人也。")

node("面相总断", "rule", ["五岳四渎相朝，三停丰满为富贵之基"])
edge("面相总断", "麻衣相法", "is_a", conf=0.85, vol="论面",
     snippet="故五岳四瀆欲得相朝，三停諸部欲得豐滿也，貌端神靜氣和者乃富貴之基也。")
edge("面赤如火", "命短", "signals", pol=-1, conf=0.7, vol="论面",
     snippet="若面色赤爆如火者，命短卒亡。")
node("面赤如火", "rule", ["面色赤爆如火主命短卒亡"])
node("命短", "concept")

node("眉相", "rule", ["眉是人伦紫气星，宜细平阔秀长，主聪明"])
edge("眉相", "麻衣相法", "is_a", conf=0.85, vol="论眉",
     snippet="眉是人倫紫氣星，稜高疏淡秀兼清。眉毛適宜長得又細又平又闊，眉毛長得秀氣而長的，主其人聰明。")
node("三停", "rule", ["上停主少年，中停主中年，下停主晚年"])
edge("三停", "麻衣相法", "is_a", conf=0.85, vol="论上停吉气",
     snippet="三停諸部欲得豐滿。")

edge("额", "官禄之宫", "maps_to", conf=0.85, vol="论上停吉气",
     snippet="離位為官祿之宮，橫連坤巽，宜高廣而有角。額為南方離位。")
node("额", "concept")
node("官禄之宫", "concept", ["额头主官禄"])
edge("气色青", "忧惊", "signals", pol=-1, conf=0.7, vol="论上停吉气",
     snippet="赤主口舌爭訟，白主喪服折傷，青主憂驚，黑主牢獄死亡。")
edge("气色赤", "口舌争讼", "signals", pol=-1, conf=0.7, vol="论上停吉气")
edge("气色黑", "牢狱死亡", "signals", pol=-1, conf=0.7, vol="论上停吉气")
for s in ["气色青", "气色赤", "气色黑", "忧惊", "口舌争讼", "牢狱死亡"]:
    node(s, "concept")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/mayi-xiangfa.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
