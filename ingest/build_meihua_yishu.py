#!/usr/bin/env python3
"""Build the seventh corpus batch: 梅花易数 (起卦/体用/断卦机制).

Output: ingest/meihua-yishu.json.
"""

import json

SR = "zhouyi-v2/gushu/meihua-yishu"

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


node("梅花易数", "concept", ["邵雍", "心易", "观梅数"])
node("体用", "rule", ["体卦为主，用卦为事，互卦为事之中间，变卦为事之终"])
node("先天卦数", "rule", ["乾一兑二离三震四巽五坎六艮七坤八"])

# 先天数
nums = [("乾", "一"), ("兑", "二"), ("離", "三"), ("震", "四"),
        ("巽", "五"), ("坎", "六"), ("艮", "七"), ("坤", "八")]
for gua, num in nums:
    node(gua, "concept")
    node(f"数{num}", "concept")
    edge(gua, f"数{num}", "maps_to", conf=0.95, vol="卷一",
         snippet=f"{gua}，{num}；")
    edge(gua, "先天卦数", "part_of", conf=0.9, vol="卷一",
         snippet="乾，一；兌，二；離，三；震，四；巽，五；坎，六；艮，七；坤，八。")

# 八宫五行
bagong = [("乾", "金"), ("兑", "金"), ("坤", "土"), ("艮", "土"),
          ("震", "木"), ("巽", "木"), ("坎", "水"), ("離", "火")]
node("八宫五行", "rule")
for gua, w in bagong:
    edge(gua, w, "maps_to", conf=0.95, vol="卷一",
         snippet="乾、兌，金；坤、艮，土；震、巽，木；坎，水；離，火。")
    edge(gua, "八宫五行", "part_of", conf=0.9, vol="卷一")

# 卦气旺衰
edge("震", "春", "climaxes_in", conf=0.9, vol="卷一",
     snippet="震、巽木旺於春，離火旺于夏，乾、兌金旺於秋，坎水旺於冬，坤、艮土旺於辰戌丑未月。")
edge("離", "夏", "climaxes_in", conf=0.9, vol="卷一")
edge("兑", "秋", "climaxes_in", conf=0.9, vol="卷一")
edge("坎", "冬", "climaxes_in", conf=0.9, vol="卷一")

# 体用吉凶
edge("体用", "用生体", "entails", conf=0.9, vol="卷二",
     snippet="應用生及比和，則吉；體生用及克體，則不吉。")
node("用生体", "rule", ["用生体有进益之喜"])
edge("体用", "体生用", "entails", conf=0.9, vol="卷二",
     snippet="體生用有耗失之患。")
node("体生用", "rule", ["体生用有耗失之患"])
edge("体用", "用克体", "entails", conf=0.9, vol="卷二",
     snippet="用克體不宜，體克用則吉。")
node("用克体", "rule", ["用克体不宜"])
edge("体用", "体克用", "entails", conf=0.9, vol="卷二",
     snippet="體克用則吉。")
node("体克用", "rule")
edge("体用", "比和", "entails", conf=0.9, vol="卷二",
     snippet="體用比和，謀為吉利。")
node("比和", "rule", ["体用比和谋为吉利"])

# 断卦次序
node("断卦次序", "rule", ["先看周易爻辞，次看体用五行生克，又次看克应，复验己身动静"])
edge("断卦次序", "梅花易数", "is_a", conf=0.9, vol="卷二",
     snippet="大抵占卜之法，成卦之後先看《周易》爻辭，以斷吉凶。次看卦之體用，以論五行生克。又次看克應。複驗己身之動靜。")
node("克应", "rule", ["闻吉说见吉兆则吉，见圆物事易成"])
edge("克应", "断卦次序", "part_of", conf=0.8, vol="卷二",
     snippet="如聞吉說見吉兆，則吉；聞凶說見凶兆，則凶。見圓物，事易成；見缺物，事終毀之類。")

# 起卦法
edge("卦以八除", "梅花易数", "part_of", conf=0.95, vol="卷一",
     snippet="凡起卦不問數多少，即以此數作卦數，過八數即以八數遞除。")
node("卦以八除", "rule", ["起卦过八数以八除余数为卦"])
edge("爻以六除", "梅花易数", "part_of", conf=0.95, vol="卷一",
     snippet="爻以六除。")
node("爻以六除", "rule", ["动爻数以六除余数为动爻"])
edge("年月日时起例", "梅花易数", "part_of", conf=0.9, vol="卷一",
     snippet="年月日時起例。")
node("年月日时起例", "rule", ["年月日时数起卦"])
edge("互卦起例", "梅花易数", "part_of", conf=0.9, vol="卷一")
node("互卦起例", "rule", ["互卦取中爻"])

# 先天后天
edge("先天后天", "先天卦断吉凶", "entails", conf=0.85, vol="卷二",
     snippet="先天卦斷吉凶，止以卦論，不甚用《易》之爻辭。後天則用爻辭，兼用卦辭。")
node("先天后天", "rule", ["先天以卦断，后天用爻辞兼卦辞"])
edge("先天后天", "后天定爻加时", "entails", conf=0.8, vol="卷二",
     snippet="後天起卦定爻必加時而後可。")
node("后天定爻加时", "rule")
node("先天卦断吉凶", "rule", ["先天止以卦论"])
for s in ["晴", "雨", "雷", "风"]:
    node(s, "concept")

# 天时占
edge("離", "晴", "signals", conf=0.85, vol="卷二",
     snippet="離多主晴，坎多主雨，坤乃陰晦，乾主晴明，震多則春夏雷轟，巽多則四時風烈，艮多則久雨必晴，兌多則不雨亦陰。")
edge("坎", "雨", "signals", conf=0.85, vol="卷二")
edge("震", "雷", "signals", conf=0.85, vol="卷二")
edge("巽", "风", "signals", conf=0.85, vol="卷二")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/meihua-yishu.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
