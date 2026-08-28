#!/usr/bin/env python3
"""Build the third corpus batch: 渊海子平 (十神/六亲/格局/歌诀要点).

Output: ingest/yuanhai-zipping.json.
"""

import json

SR = "zhouyi-v2/gushu/yuanhai-zipping"

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


node("渊海子平", "concept", ["子平法", "四柱命理"])
node("十神", "concept", ["比肩劫财食神伤官正偏财正偏官正偏印"])

shishen = [
    ("比肩", "同我", "以甲为例见甲：为比肩、兄弟。"),
    ("劫财", "同我", "以甲为例见乙：为劫财、败财，剋父及妻。"),
    ("食神", "我生", "以甲为例见丙：为食神、天厨、寿星，为男。"),
    ("伤官", "我生", "以甲为例见丁：为伤官、退财、耗气，子甥。"),
    ("偏财", "我克", "以甲为例见戊：为偏财、偏妻、偏妾，剋子。"),
    ("正财", "我克", "以甲为例见己：为正财、正妻，剋母，为合神。"),
    ("七杀", "克我", "以甲为例见庚：为偏官、七杀、官鬼、将星。"),
    ("正官", "克我", "以甲为例见辛：为正官、禄马、荣神，父母。"),
    ("偏印", "生我", "以甲为例见壬：为倒食、偏印、梟神，剋女。"),
    ("正印", "生我", "以甲为例见癸：为印綬、正人、君子，产业。"),
]
for name, kind, snip in shishen:
    node(name, "concept", [f"日干{kind}者为{name}"])
    edge(name, "十神", "belongs_to", conf=0.95, vol="十神基础", snippet=snip)
    edge(name, kind, "defined_by", conf=0.9, vol="十神基础")
for kind in ["同我", "我生", "我克", "克我", "生我"]:
    node(kind, "concept")

node("四柱根苗花果", "rule", ["年为根月为苗日为花时为果"])
edge("四柱根苗花果", "渊海子平", "is_a", conf=0.9, vol="择日三要",
     snippet="以干为天，以支为地，支中所藏者为人元。乃分四柱，以年为根，月为苗，日为花，时为果。")
node("大运岁君", "rule", ["大运看支，岁君看干"])
edge("大运岁君", "渊海子平", "is_a", conf=0.85, vol="大运",
     snippet="子平之法，大运看支，岁君看干，交运同接木。")
node("月令提纲", "rule", ["欲知贵贱先观月令提纲，用神不可损伤，日主最宜健旺"])
edge("月令提纲", "渊海子平", "is_a", conf=0.9, vol="继善篇",
     snippet="欲知贵贱，先观月令乃提纲，次断吉凶。专用日干为主本；三元要成格局，四柱喜见财官。用神不可损伤。日主最宜健旺。")

liuqin = [
    ("正印", "母", "正印正母；偏印偏母及祖父也。"),
    ("偏印", "偏母", "正印正母；偏印偏母及祖父也。"),
    ("偏财", "父", "偏财是父，乃母之夫星也，亦为偏妻。"),
    ("正财", "妻", "正财为妻；偏财为妾，为父是也。"),
    ("比肩", "兄弟姐妹", "比肩为兄弟姐妹也。"),
    ("七杀", "子", "七杀是男。"),
    ("正官", "女", "正官是女。"),
    ("食神", "孙", "食神是男孙。"),
    ("伤官", "孙女", "伤官是女孙及祖母也。"),
]
node("六亲配置", "rule", ["以日干配十神定六亲"])
for s, q, snip in liuqin:
    edge(s, q, "maps_to", conf=0.9, vol="六亲", snippet=snip)
    node(q, "concept")

geju = [
    ("官星须印身旺", "官星须得印绶、身旺则发；若无伤官破印，身不弱者便为贵命。", 0.8),
    ("偏官如抱虎", "人有偏官，如抱虎而眠，虽借其威，足以摄群畜；稍失关防，必为其噬脐。如遇三刑俱全、阳刃在日及时、又有六害；复遇魁罡相冲，凶不可述。", 0.7),
    ("日德五日", "日德有五：甲寅、戊辰、丙辰、庚辰、壬戌日是也。其福要多，而忌刑冲破害，恶官星，憎财旺；加临会合，但空亡而忌魁罡。", 0.7),
    ("魁罡四日", "夫魁罡者有四：壬辰、庚戌、戊戌、庚辰日是也；如日位加临者众，必是福。主人性格聪明、文章振发、临事有断，惟是好杀。", 0.7),
    ("天元一气", "天元一气，羸弱贫薄难治。是乐于身旺，不要行剋制之乡；剋制者，官鬼也。", 0.7),
    ("官旺身弱", "柱中官星太旺，天元羸弱之名。", 0.7),
    ("身旺无依", "天元虽旺，若无依倚是常人。", 0.7),
    ("女命官杀一位", "女命用官为夫、或杀；只喜一位，多者剋夫。", 0.7),
]
node("渊海子平·格局", "rule", ["日德魁罡官印身旺之属"])
for name, snip, conf in geju:
    node("格·" + name, "rule", [snip[:20]])
    edge("格·" + name, "渊海子平·格局", "is_a", conf=conf, vol="格局", snippet=snip,
         cond="经验断语，需结合全局生克验证")

shensha = [
    ("太岁", "太岁乃众杀之主，入命未必为灾；若遇战斗之乡，必主刑于本命。", 0.75),
    ("阳刃桃花等凶煞", "阳刃桃花、伏吟返吟、休、囚、死、绝、衰、败者凶。遇帝旺、临官、禄马、贵人、生、养、冠带、库者吉。", 0.7),
]
node("神煞", "concept", ["太岁阳刃桃花伏吟返吟等"])
for name, snip, conf in shensha:
    node("煞·" + name, "rule", [name])
    edge("煞·" + name, "神煞", "belongs_to", conf=conf, vol="神煞", snippet=snip,
         cond="经验断语")

gejue = [
    ("继善篇·人命禀五行", "人稟天地，命属阴阳，生居覆载之内，尽在五行之中。", 0.9),
    ("继善篇·四柱喜财官", "专用日干为主本；三元要成格局，四柱喜见财官。", 0.85),
    ("心镜歌·官禄贵马", "官禄贵马见合刑，一举便成名。", 0.65),
    ("相心赋·官星贵气", "官星愷悌，贵气轩昂，性优游而仁慈宽大。", 0.6),
    ("相心赋·印绶智慧", "印綬主多智慧、丰身自在心慈。", 0.6),
    ("碧渊赋·五行生克", "人命荣枯得失，尽在五行生剋之中；富贵贫穷，不出乎八字中和之外。", 0.85),
    ("碧渊赋·先观气节", "先观气节之浅深，后看财官之向背。", 0.8),
    ("渊源集说·四土人元", "辰戌丑未，四土之神。人元三用，透旺爲真。寅申巳亥，四生之局。用物身強，遇之發福。", 0.8),
    ("渊源集说·天元一气", "天元一氣，地物相同。人命得此，位列三公。", 0.65),
]
node("渊海子平·歌诀", "rule", ["继善篇心镜歌相心赋碧渊赋等"])
for name, snip, conf in gejue:
    node("诀·" + name, "rule", [snip[:20]])
    edge("诀·" + name, "渊海子平·歌诀", "is_a", conf=conf, vol="歌诀", snippet=snip)

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/yuanhai-zipping.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
