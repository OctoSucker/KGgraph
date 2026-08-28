#!/usr/bin/env python3
"""Build the thirteenth corpus batch: 增删卜易 (六爻断卦原则).

Output: ingest/zengshan-buyi.json.
"""

import json

SR = "zhouyi-v2/gushu/zengshan-buyi"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="第十章", snippet="", st="corpus_extracted"):
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


node("增删卜易", "concept", ["野鹤老人", "六爻实战"])
node("用神", "concept", ["所占之事所对应的六亲爻"])
node("元神", "concept", ["生用神之爻"])
node("忌神", "concept", ["克用神之爻"])
node("仇神", "concept", ["克元神生忌神之爻"])

edge("元神", "用神", "generates", pol=1, conf=0.95, vol="第十章",
     snippet="元神生用神，須要旺相，方可生得用神。")
edge("忌神", "用神", "overcomes", pol=-1, conf=0.95, vol="第十章",
     snippet="忌神動而克害用神。")
edge("仇神", "元神", "overcomes", pol=-1, conf=0.85, vol="第十章",
     snippet="忌神與仇神同動五也。")

node("元神有力", "rule", ["元神旺相或临日月或动化回头生及化进神者等五者有力"])
edge("元神有力", "元神", "describes", conf=0.85, vol="第十章",
     snippet="元神能生用神者有五：元神旺相或臨日月或日月動爻生扶者一也。元神動化回頭生及化進神者二也。元神長生帝旺於日辰三也。元神與忌神同動四也。元神旺動臨空化空五也。")
node("元神无力", "rule", ["元神休囚不动或动而休囚被伤克等六者无力"])
edge("元神无力", "元神", "describes", conf=0.85, vol="第十章",
     snippet="元神又有不能生用神者有六：元神休囚，不動或動而休囚又被傷克者一也。元神休囚又逢自空、月破二也。元神休囚動化退神三也。元神衰而又絕四也。元神入三墓五也。元神休囚動而化絕、化克、化破、化散六也。")
node("忌神有力", "rule", ["忌神旺相或遇日月动爻生扶等五者有力，诸占大凶"])
edge("忌神有力", "忌神", "describes", conf=0.85, vol="第十章",
     snippet="忌神動而克害用神者有五。以上之忌神者如斧戟之忌神，諸占大凶。")
node("忌神无力", "rule", ["忌神休囚不动或动休囚被日月动爻克等七者无力，化凶为吉"])
edge("忌神无力", "忌神", "describes", conf=0.85, vol="第十章",
     snippet="忌神不能克用神者有七。此忌神者乃無力之忌神也，諸占化凶爲吉。")
node("用神无根", "rule", ["用神无根，元神有力亦难生"])
edge("用神无根", "用神", "describes", conf=0.85, vol="第十章",
     snippet="倘若用神無根謂之元神有力亦難生，忌神無力何足喜。")

node("克处逢生", "rule", ["受此处之克，得他处之生，即为克处逢生"])
edge("克处逢生", "增删卜易", "is_a", conf=0.85, vol="第十三章",
     snippet="受此處之克，得他處之生，卽爲克處逢生。大凡用神、元神克少生多爲吉。")

node("月将", "rule", ["月将即月建又为月令，掌一月之权，万卜以为之纲领"])
edge("月将", "增删卜易", "is_a", conf=0.9, vol="第十六章",
     snippet="月將卽是月建又爲月令。掌一月之權，司三旬之令。月將乃當權之帥，萬卜以之爲綱領。")
node("月破", "rule", ["月建冲爻即为月破，乃无用之爻"])
edge("月破", "月将", "derives_from", conf=0.9, vol="第十六章",
     snippet="月建沖爻卽爲月破，乃無用之爻也。")
node("月合", "rule", ["月建合爻为月合，乃有用之爻"])
edge("月合", "月将", "derives_from", conf=0.9, vol="第十六章",
     snippet="月建合爻爲月合，乃有用之爻也。")
node("日月建权能", "rule", ["月建能生合比拱扶衰弱之爻，能冲克刑破旺强之爻"])
edge("日月建权能", "月将", "describes", conf=0.85, vol="第十六章",
     snippet="爻之衰弱者能生之合之、比之、拱之、扶之，衰而亦旺。爻之彊旺者能沖之克之、刑之、破之，旺而亦衰。")

node("六冲六法", "rule", ["日月冲爻、卦逢六冲、六合变六冲、冲变六冲、动爻变冲、爻与爻冲"])
edge("六冲六法", "增删卜易", "is_a", conf=0.9, vol="第二十章",
     snippet="相沖之法有六。日月沖爻者一也，卦逢六沖二也，六合卦變六沖三也，沖變六沖四也。動爻變沖五也、爻與爻沖六也。沖者散也。")
node("六冲", "rule", ["子午丑未寅申卯酉辰戌巳亥相冲"])
edge("六冲", "六冲六法", "describes", conf=0.95, vol="第二十章",
     snippet="子午相沖、丑未相沖、寅申相沖、卯酉相沖、辰戌相沖、巳亥相沖。")

node("动变生克", "rule", ["变出之爻能生克冲合本位动爻，不能生克他爻"])
edge("动变生克", "增删卜易", "is_a", conf=0.9, vol="第十五章",
     snippet="卦有動爻，動而必變，夫變出之爻能生克沖合本位之動爻，不能生克他爻，而他爻與本位之動爻，亦不能生克變爻。")
node("反伏吟", "rule", ["卦变爻变而反伏者，如乾变坤"])
edge("反伏吟", "增删卜易", "is_a", conf=0.85, vol="第二十五章",
     snippet="卦變者內外動而反伏者同一卦也。如乾卦變坤卦。")

node("六神", "concept", ["青龙朱雀勾陈腾蛇白虎玄武"])
for s in ["青龙", "朱雀", "勾陈", "腾蛇", "白虎", "玄武"]:
    node(s, "concept")
    edge(s, "六神", "belongs_to", conf=0.9, vol="第十八章",
         snippet=f"六神章第十八：{s}")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/zengshan-buyi.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
