#!/usr/bin/env python3
"""Build the eighth corpus batch: 周易 (64 卦 + 十翼核心命题).

Output: ingest/zhouyi.json.  The 64 hexagrams are scanned from the
zhouyi corpus files so node ids match the source chapter names.
"""

import json
import os
import re

SR = "zhouyi-v2/gushu/zhouyi"

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


node("周易", "concept", ["易经", "六十四卦"])
node("六十四卦", "concept")
node("十翼", "concept", ["易传", "彖象系辞文言说卦序卦杂卦"])

# 64 卦：扫描 zhouyi 语料文件
skip = {"full.txt", "周易.txt", "大象.txt", "小象.txt", "彖.txt", "文言.txt",
        "繫辭上.txt", "繫辭下.txt", "說卦.txt", "序卦.txt", "雜卦.txt"}
gua_dir = "gushu/zhouyi"
gua_files = sorted(f for f in os.listdir(gua_dir)
                   if f.endswith(".txt") and f not in skip)
for fname in gua_files:
    gua = fname[:-4]
    node(gua, "concept", [f"{gua}卦"])
    edge(gua, "六十四卦", "belongs_to", conf=0.95, vol=gua)
    # 卦辞首句作为 evidence
    raw = open(os.path.join(gua_dir, fname), encoding="utf-8").read()
    lines = [ln.strip() for ln in raw.splitlines() if ln.strip()]
    snippet = ""
    for ln in lines:
        if re.match(rf"^{gua}：", ln) or re.match(rf"^{gua}:", ln):
            snippet = ln
            break
    if snippet:
        edge(gua, "卦辞", "described_by", conf=0.9, vol=gua, snippet=snippet[:120])
node("卦辞", "concept", ["卦名下的断占之辞"])

# 八卦取象（说卦）
quxiang = [
    ("乾", "天"), ("坤", "地"), ("震", "雷"), ("巽", "风"),
    ("坎", "水"), ("離", "火"), ("艮", "山"), ("兑", "泽"),
]
node("八卦取象", "rule", ["乾天坤地震雷巽风坎水离火艮山兑泽"])
for gua, xiang in quxiang:
    node(xiang, "concept")
    edge(gua, xiang, "represents", conf=0.95, vol="說卦",
         snippet=f"乾爲天，坤爲地，震爲雷，巽爲風，坎爲水，離爲火，艮爲山，兌爲澤。")
    edge(gua, "八卦取象", "part_of", conf=0.9, vol="說卦")

# 先天八卦方位
edge("天地定位", "先天八卦", "describes", conf=0.9, vol="說卦",
     snippet="天地定位，山澤通氣，雷風相薄，水火不相射，八卦相錯。")
node("天地定位", "rule", ["先天八卦方位"])
node("先天八卦", "rule")
# 后天八卦方位
edge("帝出乎震", "后天八卦", "describes", conf=0.9, vol="說卦",
     snippet="帝出乎震，齊乎巽，相見乎離，致役乎坤，說言乎兌，戰乎乾，勞乎坎，成言乎艮。")
node("帝出乎震", "rule", ["后天八卦方位"])
node("后天八卦", "rule")

# 系辞核心命题
xici = [
    ("天尊地卑", "天尊地卑，乾坤定矣。卑高以陳，貴賤位矣。動靜有常，剛柔斷矣。"),
    ("易有太极", "易有太極，是生兩儀，兩儀生四象，四象生八卦。"),
    ("一阴一阳之谓道", "一陰一陽之謂道。"),
    ("生生之谓易", "生生之謂易。"),
    ("易与天地准", "易與天地準，故能彌綸天地之道。"),
    ("形上形下", "形而上者謂之道，形而下者謂之器。"),
    ("神无方易无体", "神無方而易無體。"),
    ("乾坤易之蕴", "乾坤其易之蘊邪。"),
    ("六爻之动", "六爻之動，三極之道也。"),
    ("刚柔相推", "剛柔相推，而生變化。"),
]
for name, snip in xici:
    nid = "系辞·" + name
    node(nid, "rule", [snip[:18]])
    edge(nid, "十翼", "belongs_to", conf=0.9, vol="繫辭上", snippet=snip)

# 序卦
edge("有天地然后万物生", "十翼", "belongs_to", conf=0.9, vol="序卦",
     snippet="有天地，然後萬物生焉。盈天地之間者唯萬物，故受之以屯。")
node("有天地然后万物生", "rule", ["序卦卦序逻辑"])

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/zhouyi.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("gua files:", len(gua_files))
print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
