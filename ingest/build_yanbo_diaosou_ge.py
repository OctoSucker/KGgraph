#!/usr/bin/env python3
"""Build the tenth corpus batch: 烟波钓叟歌 (奇门遁甲核心歌诀).

Output: ingest/yanbo-diaosou-ge.json.
"""

import json

SR = "zhouyi-v2/gushu/yanbo-diaosou-ge"

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


node("烟波钓叟歌", "concept", ["奇门遁甲", "遁甲歌诀"])
node("奇门遁甲", "concept", ["三式之一", "奇门"])

node("奇门源流", "rule", ["一千八十局，太公七十二，张子房一十八局"])
edge("奇门源流", "奇门遁甲", "describes", conf=0.85, vol="全文",
     snippet="一千八十當時制，太公成七十二。逮於漢代張子房，一十八局為精藝。")
node("阴阳二遁", "rule", ["阴阳二遁分顺逆，五日一元，接气超神"])
edge("阴阳二遁", "奇门遁甲", "describes", conf=0.85, vol="全文",
     snippet="陰陽二遁分順逆，一氣三元人莫測。五日都來接一元，接氣超神為準則。")
node("九宫九星", "rule", ["九宫逢甲为直符，八门值使，十时一易"])
edge("九宫九星", "奇门遁甲", "describes", conf=0.85, vol="全文",
     snippet="認取九宮為九星，八門又逐九宮行。九宮逢甲為直符，八門值使自分明。符上之門為直使，十時一易堪憑據。")
node("三奇六仪", "rule", ["六甲为六仪，三奇乙丙丁；阳遁顺仪奇逆布，阴遁逆仪奇顺行"])
edge("三奇六仪", "奇门遁甲", "describes", conf=0.9, vol="全文",
     snippet="六甲元號六儀名，三奇即是乙丙丁。陽遁順儀奇逆布，陰遁逆儀奇順行。")

# 九星配宫配五行
xing = [
    ("天蓬", "坎", "水", "坎蓬水星離英火"),
    ("天英", "離", "火", "坎蓬水星離英火"),
    ("天冲", "震", "木", "乾兌為金震巽木"),
    ("天辅", "巽", "木", "乾兌為金震巽木"),
    ("天心", "兑", "金", "乾兌為金震巽木"),
    ("天柱", "乾", "金", "乾兌為金震巽木"),
    ("天芮", "坤", "土", "中宮坤艮土為營"),
    ("天任", "艮", "土", "中宮坤艮土為營"),
]
node("九星", "concept", ["蓬任冲辅禽英芮柱心"])
for name, gua, w, snip in xing:
    node(name, "concept")
    edge(name, w, "maps_to", conf=0.9, vol="全文", snippet=snip)
    edge(name, "九星", "belongs_to", conf=0.9, vol="全文")

node("九星吉凶", "rule", ["辅禽心上吉，冲任小吉，蓬芮大凶，英柱小凶"])
edge("九星吉凶", "九星", "describes", conf=0.85, vol="全文",
     snippet="蓬任衝輔禽陽星，英芮柱心陰宿名。輔禽心星為上吉，衝任小吉未全亨。大凶蓬芮不堪使，小凶英柱不精明。")

# 八门
node("八门", "concept", ["开休生伤杜景死惊"])
for m in ["开", "休", "生", "伤", "杜", "景", "死", "惊"]:
    node(m + "门", "concept")
    edge(m + "门", "八门", "belongs_to", conf=0.9, vol="全文")
edge("开休生", "三吉门", "is_a", conf=0.9, vol="全文",
     snippet="八門若遇開休生，諸事逢之總趁情。")
node("开休生", "concept", ["三吉门"])
node("三吉门", "rule")
edge("死门", "凶门", "is_a", conf=0.85, vol="全文",
     snippet="若死門何所主，只宜弔死與行刑。")
node("凶门", "concept")

# 三遁
node("三遁", "rule", ["天遁地遁人遁"])
edge("天遁", "三遁", "is_a", conf=0.85, vol="全文",
     snippet="生門六丙合六丁，此為天遁自分明。")
edge("地遁", "三遁", "is_a", conf=0.85, vol="全文",
     snippet="開門六乙合六己，地遁如斯而已矣。")
edge("人遁", "三遁", "is_a", conf=0.85, vol="全文",
     snippet="休門六丁共太陰，欲求人遁無過此。")
for d in ["天遁", "地遁", "人遁"]:
    node(d, "rule")

# 吉格
node("鸟跌穴", "rule", ["丙加甲为鸟跌穴，吉"])
edge("鸟跌穴", "奇门遁甲", "describes", conf=0.85, vol="全文",
     snippet="丙加甲兮鳥跌穴，甲加丙兮龍返首。只此二者是吉神，為事如意十八九。")
node("龙返首", "rule", ["甲加丙为龙返首，吉"])
edge("龙返首", "奇门遁甲", "describes", conf=0.85, vol="全文")

# 凶格
xiongge = [
    ("白入荧", "六庚加丙白入熒，賊即來", "六庚加丙白入熒，六丙加庚熒入白。白入熒兮賊即來，熒入白兮賊即去。"),
    ("荧入白", "六丙加庚熒入白，賊即去", "六庚加丙白入熒，六丙加庚熒入白。"),
    ("蛇夭矫", "六癸加丁蛇夭矫", "六癸加丁蛇夭矯，六丁加癸雀投江。"),
    ("雀投江", "六丁加癸雀投江", "六丁加癸雀投江。"),
    ("龙逃走", "六乙加辛龙逃走", "六乙加辛龍逃走，六辛加乙虎猖狂。"),
    ("虎猖狂", "六辛加乙虎猖狂", "六辛加乙虎猖狂。"),
    ("六仪击刑", "甲子值符愁向东，戌刑未上申刑虎", "六儀擊刑何太凶，甲子值符愁向東。戌刑未上申刑虎，寅巳辰辰午刑午。"),
    ("三奇入墓", "丙奇属火火墓戌，诸事不宜", "三奇入墓宜細推。丙奇屬火火墓戌，此時諸事不宜為。"),
    ("五不遇时", "时干克日干，甲日忌庚时", "五不遇時龍不精，號為日月損光明。時干來剋日干上，甲日須知時忌庚。"),
]
node("奇门凶格", "rule", ["白入荧荧入白蛇夭矫雀投江龙逃走虎猖狂等"])
for name, alias, snip in xiongge:
    nid = "凶格·" + name
    node(nid, "rule", [alias])
    edge(nid, "奇门凶格", "is_a", pol=-1, conf=0.85, vol="全文", snippet=snip)

# 旺相休囚
node("九星旺相休囚", "rule", ["与我同行即为，我生之月诚为旺，废于父母休于财，囚于鬼"])
edge("九星旺相休囚", "九星", "describes", conf=0.8, vol="全文",
     snippet="與我同行即為，我生之月誠為旺。廢於父母休於財，囚於鬼兮真不妄。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/yanbo-diaosou-ge.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
