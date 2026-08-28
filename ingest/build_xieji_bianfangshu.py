#!/usr/bin/env python3
"""Build the twelfth corpus batch: 钦定协纪辨方书 (建除/月建/本原数理).

Output: ingest/xieji-bianfangshu.json.
"""

import json

SR = "zhouyi-v2/gushu/xieji-bianfangshu"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="卷04", snippet="", st="corpus_extracted"):
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


node("协纪辨方书", "concept", ["钦定协纪辨方书", "历法择吉"])
node("建除十二神", "concept", ["建除满平定执破危成收开闭"])

ji = ["除", "定", "执", "成", "开", "危"]
xiong = ["建", "破", "平", "收", "满", "闭"]
for p in ji:
    node(p, "concept")
    edge(p, "建除十二神", "belongs_to", pol=1, conf=0.9, vol="卷04")
for p in xiong:
    node(p, "concept")
    edge(p, "建除十二神", "belongs_to", pol=-1, conf=0.9, vol="卷04")

node("建除吉凶", "rule", ["除危定执成开为吉，建破平收满闭为凶；建满平收黑，除危定执黄"])
edge("建除吉凶", "建除十二神", "describes", conf=0.9, vol="卷04",
     snippet="除危定執成開為吉，建破平收滿閉為凶。歴書所謂建滿平收黑，除危定執黄，成開皆可用，閉破不相當者也。")

node("建除起例", "rule", ["正月建寅，寅日起建，顺行十二辰"])
edge("建除起例", "建除十二神", "describes", conf=0.9, vol="卷04",
     snippet="其法從月建上起建與斗杓所指相應如正月建寅則寅日起建順行十二辰是也。")

node("月建", "rule", ["建为岁君元神，吉凶众神之主帅，月中天子"])
edge("月建", "建除十二神", "describes", conf=0.9, vol="卷04",
     snippet="建為歲君為元神為吉凶衆神之主帥可坐不可向。俗謂之月中天子。")

detail = [
    ("建", "岁君元神，吉凶众神之主帅，可坐不可向"),
    ("除", "四利太阳，小吉"),
    ("满", "土瘟，四利丧门，又为天富，小吉"),
    ("平", "三台，土曲，大吉"),
    ("定", "岁三合，显星吉，又地官符畜官，次凶"),
    ("执", "四利之死符，又小耗，凶"),
    ("破", "岁破，大耗，大凶"),
    ("危", "极富星，谷将星，四利龙德，吉"),
    ("成", "三合吉，又飞廉，四利白虎，小凶"),
    ("收", "四利福德，小吉，又八座，小凶"),
    ("开", "青龙太阴，生气华盖，上吉，又四利吊客，小凶"),
    ("闭", "病符，凶"),
]
for p, desc in detail:
    node(f"建除·{p}", "rule", [f"{p}日{desc[:12]}"])
    edge(f"建除·{p}", p, "describes", conf=0.85, vol="卷04",
         snippet=f"{p}為{desc}。")

# 本原数理
edge("河图", "洛书", "relates_to", conf=0.9, vol="卷01",
     snippet="河圖一六為水居北二七為火居南三八為木居東四九為金居西五十為土居中。")
node("河图", "concept", ["一六水二七火三八木四九金五十土"])
node("洛书", "concept", ["戴九履一左三右七二四为肩六八为足五居中央"])
edge("河图", "五行相生", "describes", conf=0.9, vol="卷01",
     snippet="北方水生東方木，東方木生南方火，南方火生中央土，中央土生西方金，西方金生北方水。此五行相生之序也。")
node("五行相生", "rule")
edge("洛书", "五行相克", "describes", conf=0.9, vol="卷01",
     snippet="一六水尅二七火，二七火尅四九金，四九金尅三八木，三八木尅五中土，五中土尅一六水。此五行相尅之序也。")
node("五行相克", "rule")

node("先天八卦方位", "rule", ["乾南坤北离东坎西兑东南震东北巽西南艮西北"])
edge("先天八卦方位", "河图洛书数", "relates_to", conf=0.9, vol="卷01",
     snippet="邵子曰乾南坤北離東坎西兌居東南震居東北巽居西南艮居西北所謂先天之學也。")
node("河图洛书数", "rule")
node("后天八卦方位", "rule", ["帝出乎震齐乎巽相见乎离致役乎坤说言乎兑战乎乾劳乎坎成言乎艮"])
edge("后天八卦方位", "河图洛书数", "relates_to", conf=0.9, vol="卷01",
     snippet="說卦傳曰帝出乎震齊乎巽相見乎離致役乎坤說言乎兌戰乎乾勞乎坎成言乎艮。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/xieji-bianfangshu.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
