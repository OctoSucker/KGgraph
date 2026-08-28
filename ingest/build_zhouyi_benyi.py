#!/usr/bin/env python3
"""Build the twentieth corpus batch: 周易本义 (朱熹易学命题).

Output: ingest/zhouyi-benyi.json.
"""

import json

SR = "zhouyi-v2/gushu/zhouyi-benyi"

nodes = {}
edges = []
evidence = []
edge_id = 0


def node(nid, ntype="concept", aliases=None):
    if nid not in nodes:
        nodes[nid] = {"id": nid, "node_type": ntype, "aliases": aliases or []}
    return nid


def edge(frm, to, rel, pol=1, conf=0.8, cond="", vol="第一卷", snippet="", st="corpus_extracted"):
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


node("周易本义", "concept", ["朱熹", "易学"])
node("易名之义", "rule", ["易有交易、变易之义"])
edge("易名之义", "周易本义", "is_a", conf=0.9, vol="第一卷",
     snippet="易，書名也。其卦本伏羲所畫，有交易、變易之義，故謂之易。")
node("伏羲画卦", "rule", ["一奇象阳，一耦象阴，再倍而三成八卦，三倍成六十四卦"])
edge("伏羲画卦", "周易本义", "is_a", conf=0.9, vol="第一卷",
     snippet="見陰陽有奇耦之數，故畫一奇以象陽，畫一耦以象陰。見一陰一陽有各生一陰一陽之象，故自下而上，再倍而三，以成八卦。三畫已具，八卦已成，則又三倍其畫，以成六畫，而於八卦之上，各加八卦，以成六十四卦也。")
node("乾为健", "rule", ["乾者健也，阳之性也"])
edge("乾为健", "周易本义", "is_a", conf=0.9, vol="第一卷",
     snippet="乾者，健也，陽之性也。")

node("大衍筮法", "rule", ["大衍之数五十其用四十有九，分二挂一揲四归奇，三变成爻十八变成卦"])
edge("大衍筮法", "周易本义", "is_a", conf=0.9, vol="第三卷",
     snippet="大衍之數五十，其用四十有九，分而爲二，以象兩，掛一以象三，揲之以四以象四時，歸奇於扐以象閏。四營而成易，十有八變而成卦。")
node("七八九六", "rule", ["余三奇则九为太阳，二奇一偶则八为少阴，二偶一奇则七为少阳，三偶则六为老阴"])
edge("七八九六", "大衍筮法", "describes", conf=0.9, vol="第三卷",
     snippet="餘三奇則九，而其揲亦九，策亦四九三十六，是爲居一之太陽。餘二奇一偶則八，是爲居二之少陰。二偶一奇則七，是爲居三之少陽。三偶則六，是爲居四之老陰。")
node("河图数", "rule", ["一六居下，二七居上，三八居左，四九居右，五十居中"])
edge("河图数", "周易本义", "is_a", conf=0.9, vol="第三卷",
     snippet="其位一、六居下，二、七居上，三、八居左，四、九居右，五、十居中。就此章而言之，則中五爲衍母，次十爲衍子。")
node("太极两仪四象八卦", "rule", ["易有太极是生两仪，两仪生四象，四象生八卦；太极为理，两仪阴阳"])
edge("太极两仪四象八卦", "周易本义", "is_a", conf=0.9, vol="第三卷",
     snippet="易有大極，是生兩儀。大極者，其理也。兩儀者，始爲一畫以分陰陽。四象者，次爲二畫以分大少。八卦者，次爲三畫而三才之象始備。")
node("易道四用", "rule", ["以言者尚其辞，以动者尚其变，以制器者尚其象，以卜筮者尚其占"])
edge("易道四用", "周易本义", "is_a", conf=0.9, vol="第三卷",
     snippet="易有聖人之道四焉：以言者尚其辭，以動者尚其變，以制器者尚其象，以卜筮者尚其占。")
node("生生之谓易", "rule", ["阴生阳，阳生阴，其变无穷"])
edge("生生之谓易", "周易本义", "is_a", conf=0.9, vol="第三卷",
     snippet="生生之謂易，陰生陽，陽生陰，其變無窮，理與書皆然也。")

payload = {
    "mode": "import_graph",
    "schema_version": 1,
    "nodes": sorted(nodes.values(), key=lambda n: n["id"]),
    "edges": edges,
    "evidence": evidence,
}
with open("ingest/zhouyi-benyi.json", "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=False, indent=1)

print("nodes:", len(payload["nodes"]), "edges:", len(payload["edges"]),
      "evidence:", len(payload["evidence"]))
