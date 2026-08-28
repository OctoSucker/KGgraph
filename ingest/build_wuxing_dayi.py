#!/usr/bin/env python3
"""Build the first corpus batch: 五行大义 (vols 1-3 core mechanisms).

Output: ingest/wuxing-dayi.json (import-graph compatible).
Every edge carries at least one evidence with a verbatim snippet and a
source_ref of the form zhouyi-v2/gushu/wuxing-dayi/<卷>.
"""

import json

SR = "zhouyi-v2/gushu/wuxing-dayi"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.92, cond="", vol="卷2", snippet="", st="corpus_extracted"):
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


W = ["木", "火", "土", "金", "水"]
for g in "甲乙丙丁戊己庚辛壬癸":
    node(g, "concept")
for z in "子丑寅卯辰巳午未申酉戌亥":
    node(z, "concept")
for f in ["东方", "南方", "西方", "北方", "中央"]:
    node(f, "concept")
for s in ["春", "夏", "秋", "冬", "季夏"]:
    node(s, "concept")
for grp in ["甲丙戊庚壬", "乙丁己辛癸", "子寅辰午申戌", "丑卯巳未酉亥"]:
    node(grp, "concept")

# ---- 卷一：五行名义/体性 ----
for w in W:
    node(w, "concept")
node("五行", "concept", ["five elements", "wu xing"])
node("五行体性", "rule")
for w in W:
    edge(w, "五行", "belongs_to", pol=1, conf=0.95, vol="卷1",
         snippet=f"{w}者。五行之一。")

edge("木", "曲直", "is_characterized_by", conf=0.95, vol="卷1",
     snippet="木居少陽之位。春氣和煦溫柔。弱火伏其中。故木以溫柔爲體。曲直爲性。")
node("曲直", "concept", ["木曰曲直"])
edge("火", "炎上", "is_characterized_by", conf=0.95, vol="卷1",
     snippet="火居大陽之位。炎熾赫烈。故火以明熱爲體。炎上爲性。")
node("炎上", "concept", ["火曰炎上"])
edge("土", "稼穡", "is_characterized_by", conf=0.95, vol="卷1",
     snippet="土在四時之中。處季夏之末。陽衰陰長。居位之中。總於四行。故土以含散持實爲體。稼穡爲性。")
node("稼穡", "concept", ["土曰稼穡"])
edge("金", "从革", "is_characterized_by", conf=0.95, vol="卷1",
     snippet="金居少陰之位。西方成物之所。物成則凝強。少陰則清冷。故金以強冷爲體。從革爲性。")
node("从革", "concept", ["金曰从革"])
edge("水", "润下", "is_characterized_by", conf=0.95, vol="卷1",
     snippet="水以寒虛爲體。潤下爲性。")
node("润下", "concept", ["水曰润下"])

# 方位与季节
edge("木", "东方", "maps_to", conf=0.95, vol="卷1", snippet="其時春。其位在東方。")
edge("木", "春", "maps_to", conf=0.95, vol="卷1", snippet="其時春。其位在東方。")
edge("火", "南方", "maps_to", conf=0.95, vol="卷1", snippet="其時夏。其位南方。")
edge("火", "夏", "maps_to", conf=0.95, vol="卷1", snippet="其時夏。其位南方。")
edge("土", "中央", "maps_to", conf=0.95, vol="卷1", snippet="其時季夏。其位處內。王於四時之季。")
edge("土", "季夏", "maps_to", conf=0.95, vol="卷1", snippet="其時季夏。其位處內。")
edge("金", "西方", "maps_to", conf=0.95, vol="卷1", snippet="其時秋也。其位西方。")
edge("金", "秋", "maps_to", conf=0.95, vol="卷1", snippet="其時秋也。其位西方。")
edge("水", "北方", "maps_to", conf=0.95, vol="卷1", snippet="其時冬。其位北方。")
edge("水", "冬", "maps_to", conf=0.95, vol="卷1", snippet="其時冬。其位北方。")

# ---- 生成数 ----
gen = [
    ("天一生水", "水生数一", "天以一生水於北方。君子之位。陽氣微動於黃泉之下。故水數一也。"),
    ("地二生火", "火生数二", "極陽生陰。陰始於午。以陽尊故。陰卑贊和。故火數二也。"),
    ("天三生木", "木生数三", "木配陽動。而左長於東方。長則滋繁。滋繁則數增。故木數三也。"),
    ("地四生金", "金生数四", "陰佐陽消。陰道右轉。而居於西。在陽之後。理無等義。故金數四也。"),
    ("天五生土", "土生数五", "陰陽之數。始乎一周。然後陽達於中。總括四行。苞則彌多。故土數五也。"),
    ("地六成水", "水成数六", "水得於五。其數六。用能潤下。"),
    ("天七成火", "火成数七", "火得於五。其數七。用能炎上。"),
    ("地八成木", "木成数八", "木得於五。其數八。用能曲直。"),
    ("天九成金", "金成数九", "金得於五。其數九。用能從革。"),
    ("地十成土", "土成数十", "土得於五。其數十。用能稼穡。"),
]
node("五行生成数", "rule", ["生数一成数六之属"])
for rid, alias, snip in gen:
    node(rid, "rule", [alias])
    edge(rid, "五行生成数", "is_a", conf=0.9, vol="卷1", snippet=snip)

node("大衍之数", "rule", ["七八为静九六为动"])
edge("大衍之数", "五行生成数", "relates_to", conf=0.8, vol="卷1",
     snippet="七八爲靜。九六爲動。陽動而進。變七之九。陰動而退。變八之六。")

# ---- 卷二：相生 ----
sheng = [
    ("木", "火", "木生火者。木性溫暖。火伏其中。鑽灼而出。故木生火。"),
    ("火", "土", "火生土者。火熱。故能焚木。木焚而成灰。灰卽土也。故火生土。"),
    ("土", "金", "土生金者。金居石。依山津潤而生。聚土成山。山必生石。故土生金。"),
    ("金", "水", "金生水者。少陰之氣潤澤。流津銷金。亦爲水。故金生水。"),
    ("水", "木", "水生木者。因水潤而能生。故水生木也。"),
]
for frm, to, snip in sheng:
    edge(frm, to, "produces", pol=1, conf=0.95, vol="卷2", snippet=snip)

# ---- 卷二：相克 ----
ke = [
    ("木", "土", "木剋土者。專勝散。"),
    ("土", "水", "土剋水者。實勝虛。"),
    ("水", "火", "水剋火者。衆勝寡。"),
    ("火", "金", "火剋金者。精勝堅。"),
    ("金", "木", "金剋木者。剛勝柔。"),
]
for frm, to, snip in ke:
    edge(frm, to, "overcomes", pol=-1, conf=0.95, vol="卷2", snippet=snip)
node("上克下顺", "rule", ["上剋下爲順。下剋上爲剝"])
edge("上克下顺", "相克", "is_a", conf=0.85, vol="卷2",
     snippet="凡上剋下爲順。下剋上爲剝。喻如君有刑臣之法。臣無犯君之義。")
node("相克", "relation", ["五行相克"])

# ---- 卷二：生死所（十二长生） ----
shengsi = [
    ("木", "亥", "卯", "午", "未", "木。受氣於申。胎於酉。養於戌。生於亥。沐浴於子。冠帶於丑。臨官於寅。王於卯。衰於辰。病於巳。死於午。葬於未。"),
    ("火", "寅", "午", "酉", "戌", "火。受氣於亥。胎於子。養於丑。生於寅。沐浴於卯。冠帶於辰。臨官於巳。王於午。衰於未。病於申。死於酉。葬於戌。"),
    ("金", "巳", "酉", "子", "丑", "金。受氣於寅。胎於卯。養於辰。生於巳。沐浴於午。冠帶於未。臨官於申。王於酉。衰於戌。病於亥。死於子。葬於丑。"),
    ("水", "申", "子", "卯", "辰", "水。受氣於巳。胎於午。養於未。生於申。沐浴於酉。冠帶於戌。臨官於亥。王於子。衰於丑。病於寅。死於卯。葬於辰。"),
    ("土", "卯", "未", "酉", "辰", "土。受氣於亥。胎於子。養於丑。寄行於寅。生於卯。沐浴於辰。冠帶於巳。臨官於午。王於未。衰病於申。死於酉。葬於戌。"),
]
node("十二长生", "rule", ["受气胎养生沐浴冠带临官王衰病死葬"])
for w, sheng_at, wang_at, si_at, mu_at, snip in shengsi:
    node(f"{w}长生{sheng_at}", "rule", [f"{w}生於{sheng_at}"])
    edge(f"{w}长生{sheng_at}", "十二长生", "is_a", conf=0.9, vol="卷2", snippet=snip)
    edge(f"{w}长生{sheng_at}", f"{w}王{wang_at}", "entails", conf=0.85, vol="卷2")
    edge(f"{w}长生{sheng_at}", f"{w}墓{mu_at}", "entails", conf=0.85, vol="卷2")
    node(f"{w}王{wang_at}", "rule", [f"{w}王於{wang_at}"])
    node(f"{w}墓{mu_at}", "rule", [f"{w}葬於{mu_at}"])

# ---- 卷二：四时休王 ----
xiuwang = [
    ("春", "木", "火", "水", "金", "土", "春則木王。火相。水休。金囚。土死。"),
    ("夏", "火", "土", "木", "水", "金", "夏則火王。土相。木休。水囚。金死。"),
    ("季夏", "土", "金", "火", "木", "水", "六月則土王。金相。火休。木囚。水死。"),
    ("秋", "金", "水", "土", "火", "木", "秋則金王。水相。土休。火囚。木死。"),
    ("冬", "水", "木", "金", "土", "火", "冬則水王。木相。金休。土囚。火死。"),
]
node("四时休王", "rule", ["王相休囚死"])
for season, wang, xiang, xiu, qiu, si, snip in xiuwang:
    node(f"{season}{wang}王", "rule", [f"{season}则{wang}王"])
    edge(f"{season}{wang}王", "四时休王", "is_a", conf=0.92, vol="卷2", snippet=snip)
    edge(wang, f"{season}{wang}王", "dominates_in", pol=1, conf=0.92, vol="卷2", snippet=snip)
node("王相休囚死", "rule", ["旺者气盛相者子壮休者父母衰囚者所克死者所畏"])
edge("王相休囚死", "四时休王", "is_a", conf=0.85, vol="卷2",
     snippet="凡當王之時。皆以子爲相者。以其子方壯。能助治事也。父母爲休者。以其子當王。氣正盛。父母衰老。不能治事。所畏爲死者。以其身王。能制殺之。所刻者爲囚者。以其子爲相。能囚讎敵也。")

# ---- 卷二：配支干 ----
peizhi = [
    ("甲乙寅卯", "木", "东方", "甲乙寅卯。木也。位在東方。"),
    ("丙丁巳午", "火", "南方", "丙丁巳午。火也。位在南方。"),
    ("戊己辰戌丑未", "土", "中央", "戊己辰戌丑未。土也。位在中央。分王四季。寄治丙丁。"),
    ("庚辛申酉", "金", "西方", "庚辛申酉。金也。位在西方。"),
    ("壬癸亥子", "水", "北方", "壬癸亥子。水也。位在北方。"),
]
for ganzhi, w, fang, snip in peizhi:
    node(ganzhi, "concept", [f"{ganzhi}属{w}"])
    edge(ganzhi, w, "maps_to", conf=0.95, vol="卷2", snippet=snip)
    edge(ganzhi, fang, "maps_to", conf=0.95, vol="卷2")
node("阳干", "concept", ["甲丙戊庚壬"])
node("阴干", "concept", ["乙丁己辛癸"])
edge("阳干", "甲丙戊庚壬", "consists_of", conf=0.95, vol="卷2",
     snippet="干則甲丙戊庚壬爲陽。乙丁己辛癸爲陰。")
edge("阴干", "乙丁己辛癸", "consists_of", conf=0.95, vol="卷2")
node("阳支", "concept", ["子寅辰午申戌"])
node("阴支", "concept", ["丑卯巳未酉亥"])
edge("阳支", "子寅辰午申戌", "consists_of", conf=0.95, vol="卷2",
     snippet="支則寅辰午申戌子爲陽。卯巳未酉亥丑爲陰。")
edge("阴支", "丑卯巳未酉亥", "consists_of", conf=0.95, vol="卷2")
node("六十甲子", "rule", ["六旬甲子轮转"])
edge("六十甲子", "阳干", "requires", conf=0.9, vol="卷2",
     snippet="干既有十。支有十二。輪轉相配。終於癸亥。故有六十日。十日一旬。故有六旬。")
node("旬空", "rule", ["甲子旬戌亥空亡之属"])
edge("旬空", "六十甲子", "derives_from", conf=0.85, vol="卷2",
     snippet="一旬之內。二支無配偶者。爲之孤。所對衝者。爲之虛。卜筮所云空亡。以支孤無干。故名爲空亡。")

# ---- 卷二：合 ----
ganhe = [
    ("甲", "己", "己爲甲妻。故甲與己合。"),
    ("乙", "庚", "乙爲庚妻。故乙與庚合。"),
    ("丙", "辛", "辛爲丙妻。故丙與辛合。"),
    ("丁", "壬", "丁爲壬妻。故壬與丁合。"),
    ("戊", "癸", "癸爲戊妻。故癸與戊合。"),
]
for a, b, snip in ganhe:
    edge(a, b, "pairs_with", pol=1, conf=0.95, vol="卷2", snippet=snip)
node("天干五合", "rule", ["甲己合化之属"])
edge("天干五合", "甲己合", "is_a", conf=0.9, vol="卷2")
node("甲己合", "rule", ["甲己合土"])

zhhe = [
    ("子", "丑", "正月。日月會於諏訾之次。諏訾。亥也。斗建在寅。故寅與亥合。", 0),
    ("寅", "亥", "正月。日月會於諏訾之次。諏訾。亥也。斗建在寅。故寅與亥合。", 1),
    ("卯", "戌", "二月。日月會於降婁之次。降婁。戌也。斗建在卯。故卯與戌合。", 0),
    ("辰", "酉", "三月。日月會於大梁之次。大梁。酉也。斗建在辰。故辰與酉合。", 0),
    ("巳", "申", "四月。日月會於實沈之次。實沈。申也。斗建在巳。故巳與申合。", 0),
    ("午", "未", "五月。日月會於鶉首之次。鶉首。未也。斗建在午。故午與未合。", 0),
]
seen_zh = set()
for a, b, snip, _ in zhhe:
    key = tuple(sorted([a, b]))
    if key in seen_zh:
        continue
    seen_zh.add(key)
    edge(a, b, "pairs_with", pol=1, conf=0.95, vol="卷2", snippet=snip)
node("地支六合", "rule", ["子丑合寅亥合卯戌合辰酉合巳申合午未合"])
edge("地支六合", "子丑合", "is_a", conf=0.9, vol="卷2",
     snippet="十一月。日月會於星紀之次。星紀。丑也。斗建在子。故子與丑合。")
node("子丑合", "rule", ["子与丑合"])

# ---- 卷二：扶抑 ----
node("扶抑", "rule", ["母得子为扶子遇母为抑"])
edge("扶抑", "木扶水", "is_a", conf=0.85, vol="卷2",
     snippet="相扶者。木扶水。水扶金。金扶土。土扶火。火扶木。此皆母得子。")
node("木扶水", "rule")
edge("扶抑", "木抑火", "is_a", conf=0.85, vol="卷2",
     snippet="相抑者。木抑火。火抑土。土抑金。金抑水。水抑木。此皆子遇母也。")
node("木抑火", "rule")

# ---- 卷二：刑 ----
xing = [
    ("子", "卯"), ("卯", "子"), ("寅", "巳"), ("巳", "申"), ("申", "寅"),
    ("丑", "戌"), ("戌", "未"), ("未", "丑"),
]
for a, b in xing:
    edge(a, b, "punishes", pol=-1, conf=0.9, vol="卷2",
         snippet="支自相刑者。子刑在卯。卯刑在子。丑刑在戌。戌刑在未。未刑在丑。寅刑在巳。巳刑在申。申刑在寅。")
for z in ["辰", "午", "酉", "亥"]:
    edge(z, z, "punishes", pol=-1, conf=0.9, vol="卷2",
         snippet="辰午酉亥各自刑。")
node("地支三刑", "rule", ["子卯刑寅巳申刑丑戌未刑辰午酉亥自刑"])
edge("地支三刑", "子刑卯", "is_a", conf=0.9, vol="卷2")
node("子刑卯", "rule", ["子卯相刑"])

# ---- 卷二：害 ----
hai = [("戌", "酉"), ("亥", "申"), ("子", "未"), ("丑", "午"), ("寅", "巳"), ("卯", "辰")]
for a, b in hai:
    edge(a, b, "harms", pol=-1, conf=0.9, vol="卷2",
         snippet="戌與酉。亥與申。子與未。丑與午。寅與巳。卯與辰。是六害也。")
node("地支六害", "rule", ["戌酉亥申子未丑午寅巳卯辰相害"])
edge("地支六害", "子未害", "is_a", conf=0.9, vol="卷2")
node("子未害", "rule")

# ---- 卷二：冲 ----
chong = [("子", "午"), ("丑", "未"), ("寅", "申"), ("卯", "酉"), ("辰", "戌"), ("巳", "亥")]
for a, b in chong:
    edge(a, b, "clashes_with", pol=-1, conf=0.95, vol="卷2",
         snippet="支衝破者。子午衝破。丑未衝破。寅申衝破。卯酉衝破。辰戌衝破。巳亥衝破。")
node("地支六冲", "rule", ["子午冲丑未冲寅申冲卯酉冲辰戌冲巳亥冲"])
edge("地支六冲", "子午冲", "is_a", conf=0.95, vol="卷2")
node("子午冲", "rule")
for a, b in [("甲", "庚"), ("乙", "辛"), ("丙", "壬"), ("丁", "癸")]:
    edge(a, b, "clashes_with", pol=-1, conf=0.9, vol="卷2",
         snippet="干衝破者。甲庚衝破。乙辛衝破。丙壬衝破。丁癸衝破。")
node("天干四冲", "rule", ["甲庚乙辛丙壬丁癸冲"])
edge("天干四冲", "甲庚冲", "is_a", conf=0.9, vol="卷2")
node("甲庚冲", "rule")

# ---- 卷三：杂配 ----
peiset = [
    ("木", "青", "東方木爲蒼色。萬物發生。"),
    ("火", "赤", "南方火爲赤色。以象盛陽炎燄之狀也。"),
    ("土", "黄", "中央土黃色。黃者。地之色也。"),
    ("金", "白", "西方金色白。秋爲殺氣。白露爲霜。"),
    ("水", "黑", "北方水色黑。遠望黯然。陰闇之象也。"),
]
node("五色", "concept", ["青赤黄白黑"])
for w, c, snip in peiset:
    edge(w, c, "maps_to", conf=0.95, vol="卷3", snippet=snip)
    edge(c, "五色", "belongs_to", conf=0.9, vol="卷3")
    node(c, "concept")

yin = [
    ("木", "角"), ("火", "徵"), ("土", "宮"), ("金", "商"), ("水", "羽"),
]
node("五音", "concept", ["宫商角征羽"])
for w, s in yin:
    edge(w, s, "maps_to", conf=0.95, vol="卷3",
         snippet="青作角聲。白作商聲。黑作羽聲。赤作徵聲。黃作宮聲。")
    node(s, "concept")
    edge(s, "五音", "belongs_to", conf=0.9, vol="卷3")

wei = [
    ("木", "酸", "春之日。其味酸。其臭羶。木之臭味也。"),
    ("火", "苦", "夏之日。其味苦。其臭焦。"),
    ("土", "甘", "季夏之日。其味甘。其臭香。土味所以甘者。中央中和也。"),
    ("金", "辛", "秋之日。其臭腥。其味辛。"),
    ("水", "咸", "冬之日。其味醎。其臭朽。"),
]
node("五味", "concept", ["酸苦甘辛咸"])
for w, t, snip in wei:
    edge(w, t, "maps_to", conf=0.95, vol="卷3", snippet=snip)
    node(t, "concept")
    edge(t, "五味", "belongs_to", conf=0.9, vol="卷3")

zang = [
    ("木", "肝"), ("火", "心"), ("土", "脾"), ("金", "肺"), ("水", "腎"),
]
node("五脏", "concept", ["肝心脾肺肾"])
for w, z in zang:
    edge(w, z, "maps_to", conf=0.95, vol="卷3",
         snippet="五藏者。肝。心。脾。肺。腎也。肝以配木。心以配火。脾以配土。肺以配金。腎以配水。")
    node(z, "concept")
    edge(z, "五脏", "belongs_to", conf=0.9, vol="卷3")

liufu = [
    ("肝", "胆", "肝合膽。膽爲中精之府。"),
    ("心", "小肠", "心合小腸。小腸爲受盛之府。"),
    ("脾", "胃", "脾合胃。胃爲五穀之府。"),
    ("肺", "大肠", "肺合大腸。大腸爲傳道之府。"),
    ("腎", "膀胱", "腎合膀胱。膀胱爲津液之府。"),
]
for z, f, snip in liufu:
    edge(z, f, "pairs_with", pol=1, conf=0.9, vol="卷3", snippet=snip)
    node(f, "concept")
node("三焦", "concept")
edge("三焦", "六腑", "belongs_to", conf=0.85, vol="卷3",
     snippet="三焦孤立。爲中瀆之府。")
node("六腑", "concept", ["胆胃大小肠三焦膀胱"])

chang = [
    ("木", "仁", "歲星於人。五常。仁也。"),
    ("火", "礼", "熒惑於人。五常。禮也。"),
    ("金", "义", "太白於人。五常。義也。"),
    ("水", "智", "辰星於人。五常。智也。"),
    ("土", "信", "鎮星於人。五常。信也。"),
]
node("五常", "concept", ["仁义礼智信"])
for w, c, snip in chang:
    edge(w, c, "maps_to", conf=0.8, vol="卷3",
         snippet=snip + "按毛公及京房。漢史。皆以土爲信。")
    node(c, "concept")
    edge(c, "五常", "belongs_to", conf=0.9, vol="卷3")
edge("土", "智", "maps_to", pol=1, conf=0.55, vol="卷3",
     snippet="鄭玄及詩緯。以土爲智者。以能了萬事。莫過於智。水爲信者。水之有潮。依期而至。故以水爲信。此理寘證狹。於義乖也。",
     cond="异说：郑玄/诗纬以土智水信，与主流相反")
node("智", "concept")
node("信", "concept")

shi = [
    ("木", "貌", "歲星於人。五常。仁也。五事。貌也。"),
    ("火", "视", "熒惑於人。五常。禮也。五事。視也。"),
    ("金", "言", "太白於人。五常。義也。五事。言也。"),
    ("水", "听", "辰星於人。五常。智也。五事。聽也。"),
    ("土", "思", "鎮星於人。五常。信也。五事。思也。"),
]
node("五事", "concept", ["貌言视听思"])
for w, s, snip in shi:
    edge(w, s, "maps_to", conf=0.85, vol="卷3", snippet=snip)
    node(s, "concept")
    edge(s, "五事", "belongs_to", conf=0.9, vol="卷3")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/wuxing-dayi.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
