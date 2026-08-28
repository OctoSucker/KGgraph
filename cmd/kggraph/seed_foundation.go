package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// runSeedFoundation deterministically generates the 八卦/周易 foundation
// batch (ingest/zhouyi-foundation.json). No LLM is involved: every node/edge
// comes from the fixed tables below, and every verbatim snippet is asserted
// against the corpus at generation time.
func runSeedFoundation(ctx context.Context, argv []string) {
	_ = ctx
	fs := flag.NewFlagSet("seed-foundation", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var corpusDir, outPath string
	fs.StringVar(&corpusDir, "corpus", "gushu/zhouyi", "corpus dir containing 說卦.txt")
	fs.StringVar(&outPath, "out", "ingest/zhouyi-foundation.json", "output batch JSON path")
	mustParse(fs, argv)

	corpus := loadCorpusForSeed(corpusDir)
	gb := newFoundationBuilder(corpus)
	gb.build()
	gb.assertSnippets()

	payload := map[string]any{
		"mode":           "import_graph",
		"schema_version": 1,
		"nodes":          gb.nodes,
		"edges":          gb.edges,
		"evidence":       gb.evidence,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("seed-foundation: marshal: %v", err)})
	}
	if err := os.WriteFile(outPath, raw, 0o644); err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("seed-foundation: write: %v", err)})
	}
	writeJSONAndExit(0, map[string]any{
		"output":         outPath,
		"node_count":     len(gb.nodes),
		"edge_count":     len(gb.edges),
		"evidence_count": len(gb.evidence),
		"note":           "deterministic foundation batch; snippets verified verbatim against corpus",
	})
}

// ---------- data tables ----------

type baguaRow struct {
	name      string // 乾
	koujue    string // 乾三连
	de        string // 卦德
	nature    string // 自然取象
	family    string // 家人
	body      string // 身体
	animal    string // 动物
	wuxing    string // 五行
	yang      bool   // 阳卦?
	houtian   string // 后天方位
	xiantian  string // 先天方位
	xiantianN string // 先天数
	houtianN  string // 后天数（洛书）
	xiang     string // 说卦取象原文
}

var baguaRows = []baguaRow{
	{"乾", "乾三连", "健", "天", "父", "首", "马", "金", true, "西北", "南", "一", "六",
		"干为天、为圜、为君、为父、为玉、为金、为寒、为冰、为大赤、为良马、为老马、为瘠马、为驳马、为木果。"},
	{"兑", "兑上缺", "说", "泽", "少女", "口", "羊", "金", false, "西", "东南", "二", "七",
		"兑为泽、为少女、为巫、为口舌、为毁折、为附决。其于地也，为刚卤，为妾、为羊。"},
	{"离", "离中虚", "丽", "火", "中女", "目", "雉", "火", false, "南", "东", "三", "九",
		"离为火、为日、为电、为中女、为甲胄、为戈兵。其于人也，为大腹、为乾卦、为鳖、为蟹、为蠃、为蚌、为龟。其于木也，为科上槁。"},
	{"震", "震仰盂", "动", "雷", "长男", "足", "龙", "木", true, "东", "东北", "四", "三",
		"震为雷、为龙、为玄黄、为敷、为大涂、为长子、为决躁、为苍筤竹、为萑苇。其于马也，为善鸣、为馵足，为的颡。其于稼也，为反生。其究为健，为蕃鲜。"},
	{"巽", "巽下断", "入", "风", "长女", "股", "鸡", "木", false, "东南", "西南", "五", "四",
		"巽为木、为风、为长女、为绳直、为工、为白、为长、为高、为进退、为不果、为臭。其于人也，为寡发、为广颡、为多白眼、为近利市三倍。其究为躁卦。"},
	{"坎", "坎中满", "陷", "水", "中男", "耳", "豕", "水", true, "北", "西", "六", "一",
		"坎为水、为沟渎、为隐伏、为矫𫐓、为弓轮。其于人也，为加忧、为心病、为耳痛、为血卦、为赤。其于马也，为美脊、为亟心、为下首、为薄蹄、为曳。其于舆也，为多眚、为通、为月、为盗。其于木也，为坚多心。"},
	{"艮", "艮覆碗", "止", "山", "少男", "手", "狗", "土", true, "东北", "西北", "七", "八",
		"艮为山、为径路、为小石、为门阙、为果蓏、为阍寺、为指、为狗、为鼠、为黔喙之属。其于木也，为坚多节。"},
	{"坤", "坤六断", "顺", "地", "母", "腹", "牛", "土", false, "西南", "北", "八", "二",
		"坤为地、为母、为布、为釜、为吝啬、为均、为子母牛、为大舆、为文、为众、为柄。其于地也为黑。"},
}

type gua64Row struct {
	seq   int
	name  string
	upper string
	lower string
}

var gua64Rows = []gua64Row{
	{1, "乾为天", "乾", "乾"}, {2, "坤为地", "坤", "坤"},
	{3, "水雷屯", "坎", "震"}, {4, "山水蒙", "艮", "坎"},
	{5, "水天需", "坎", "乾"}, {6, "天水讼", "乾", "坎"},
	{7, "地水师", "坤", "坎"}, {8, "水地比", "坎", "坤"},
	{9, "风天小畜", "巽", "乾"}, {10, "天泽履", "乾", "兑"},
	{11, "地天泰", "坤", "乾"}, {12, "天地否", "乾", "坤"},
	{13, "天火同人", "乾", "离"}, {14, "火天大有", "离", "乾"},
	{15, "地山谦", "坤", "艮"}, {16, "雷地豫", "震", "坤"},
	{17, "泽雷随", "兑", "震"}, {18, "山风蛊", "艮", "巽"},
	{19, "地泽临", "坤", "兑"}, {20, "风地观", "巽", "坤"},
	{21, "火雷噬嗑", "离", "震"}, {22, "山火贲", "艮", "离"},
	{23, "山地剥", "艮", "坤"}, {24, "地雷复", "坤", "震"},
	{25, "天雷无妄", "乾", "震"}, {26, "山天大畜", "艮", "乾"},
	{27, "山雷颐", "艮", "震"}, {28, "泽风大过", "兑", "巽"},
	{29, "坎为水", "坎", "坎"}, {30, "离为火", "离", "离"},
	{31, "泽山咸", "兑", "艮"}, {32, "雷风恒", "震", "巽"},
	{33, "天山遁", "乾", "艮"}, {34, "雷天大壮", "震", "乾"},
	{35, "火地晋", "离", "坤"}, {36, "地火明夷", "坤", "离"},
	{37, "风火家人", "巽", "离"}, {38, "火泽睽", "离", "兑"},
	{39, "水山蹇", "坎", "艮"}, {40, "雷水解", "震", "坎"},
	{41, "山泽损", "艮", "兑"}, {42, "风雷益", "巽", "震"},
	{43, "泽天夬", "兑", "乾"}, {44, "天风姤", "乾", "巽"},
	{45, "泽地萃", "兑", "坤"}, {46, "地风升", "坤", "巽"},
	{47, "泽水困", "兑", "坎"}, {48, "水风井", "坎", "巽"},
	{49, "泽火革", "兑", "离"}, {50, "火风鼎", "离", "巽"},
	{51, "震为雷", "震", "震"}, {52, "艮为山", "艮", "艮"},
	{53, "风山渐", "巽", "艮"}, {54, "雷泽归妹", "震", "兑"},
	{55, "雷火丰", "震", "离"}, {56, "火山旅", "离", "艮"},
	{57, "巽为风", "巽", "巽"}, {58, "兑为泽", "兑", "兑"},
	{59, "风水涣", "巽", "坎"}, {60, "水泽节", "坎", "兑"},
	{61, "风泽中孚", "巽", "兑"}, {62, "雷山小过", "震", "艮"},
	{63, "水火既济", "坎", "离"}, {64, "火水未济", "离", "坎"},
}

// verbatim snippets from gushu/zhouyi/说卦.txt
const (
	sShuoguaDe       = "干，健也。坤，顺也。震，动也。巽，入也。坎，陷也。离，丽也。艮，止也。兑，说也。"
	sShuoguaAnimal   = "干为马，坤为牛，震为龙，巽为鸡，坎为豕，离为雉，艮为狗，兑为羊。"
	sShuoguaBody     = "干为首，坤为腹，震为足，巽为股，坎为耳，离为目，艮为手，兑为口。"
	sShuoguaFamily   = "干，天也，故称乎父；坤，地也，故称乎母；震一索而得男，故谓之长男；巽一索而得女，故谓之长女；坎再索而得男，故谓之中男；离再索而得女，故谓之中女；艮三索而得男，故谓之少男；兑三索而得女，故谓之少女。"
	sShuoguaXiantian = "天地定位，山泽通气，雷风相薄，水火不相射。八卦相错，数往者顺，知来者逆。是故易逆数也。"
	sShuoguaHoutian  = "帝出乎震，齐乎巽，相见乎离，致役乎坤，说言乎兑，战乎干，劳乎坎，成言乎艮。万物出乎震，震，东方也。齐乎巽，巽，东南也，齐也者，言万物之洁齐也。离也者，明也，万物皆相见，南方之卦也。圣人南面而听天下，向明而治，盖取诸此也。坤也者，地也，万物皆致养焉，故曰致役乎坤。兑，正秋也。万物之所说也。故曰说；言乎兑。战乎干，干，西北之卦也。言阴阳相薄也。坎者，水也，正北方之卦也，劳卦也，万物之所归也，故曰劳乎坎。艮，东北之卦也，万物之所成，终而所成始也，故曰成言乎艮。"
)

// ---------- builder ----------

type foundationBuilder struct {
	corpus   string // whitespace-stripped corpus
	nodes    []map[string]any
	edges    []map[string]any
	evidence []map[string]any
	edgeID   int
	verified []string
}

func newFoundationBuilder(corpus string) *foundationBuilder {
	return &foundationBuilder{corpus: stripWS(corpus)}
}

func (b *foundationBuilder) node(id, typ string, aliases, domain []string) {
	if typ == "" {
		typ = "concept"
	}
	b.nodes = append(b.nodes, map[string]any{
		"id": id, "node_type": typ, "aliases": aliases, "domain": domain,
	})
}

func (b *foundationBuilder) edge(from, rel, to string, conf float64, condText, edgeKind, condJSON string) {
	b.edgeID++
	e := map[string]any{
		"id": b.edgeID, "from_id": from, "to_id": to, "relation_type": rel,
		"polarity": 1, "confidence": conf,
	}
	if condText != "" {
		e["condition_text"] = condText
	}
	if edgeKind != "" {
		e["edge_kind"] = edgeKind
	}
	if condJSON != "" {
		e["condition_json"] = condJSON
	}
	b.edges = append(b.edges, e)
}

func (b *foundationBuilder) ev(edgeID int, sourceType, sourceRef, snippet, translation string) {
	ev := map[string]any{
		"edge_id": edgeID, "source_type": sourceType, "source_ref": sourceRef,
		"supports": true, "weight": 1.0,
	}
	if snippet != "" {
		ev["snippet"] = stripWS(snippet)
		b.verified = append(b.verified, stripWS(snippet))
	}
	if translation != "" {
		ev["translation"] = translation
	}
	b.evidence = append(b.evidence, ev)
}

func (b *foundationBuilder) build() {
	const (
		zy   = "周易"
		base = "易学基础"
		ref  = "zhouyi-v2/gushu/zhouyi"
		sg   = ref + "/說卦"
	)

	// 顶层概念
	b.node("周易", "concept", []string{"易经", "易"}, []string{zy})
	b.node("八卦", "concept", []string{"经卦", "三爻卦"}, []string{zy})
	b.node("六十四卦", "concept", []string{"别卦", "重卦"}, []string{zy})
	b.node("六十四卦序", "concept", []string{"周易卦序"}, []string{zy})
	b.node("阴阳", "concept", nil, []string{base})
	b.node("五行", "concept", nil, []string{base})
	b.edge("八卦", "part_of", "周易", 0.95, "", "structural", "")
	b.edge("六十四卦", "part_of", "周易", 0.95, "", "structural", "")
	b.edge("六十四卦序", "part_of", "周易", 0.95, "", "structural", "")

	// 五行 + 生克矩阵
	for _, el := range []string{"木", "火", "土", "金", "水"} {
		b.node(el, "concept", nil, []string{base})
		b.edge(el, "belongs_to", "五行", 0.95, "", "structural", "")
	}
	sheng := [][2]string{{"木", "火"}, {"火", "土"}, {"土", "金"}, {"金", "水"}, {"水", "木"}}
	for _, p := range sheng {
		b.edge(p[0], "produces", p[1], 0.95, "五行相生", "matrix", `{"关系":"相生"}`)
	}
	ke := [][2]string{{"木", "土"}, {"土", "水"}, {"水", "火"}, {"火", "金"}, {"金", "木"}}
	for _, p := range ke {
		b.edge(p[0], "overcomes", p[1], 0.95, "五行相克", "matrix", `{"关系":"相克"}`)
	}
	b.ev(b.edgeID-9, "corpus_summary", ref+"/周易", "", "五行相生：木生火、火生土、土生金、金生水、水生木。")
	b.ev(b.edgeID-4, "corpus_summary", ref+"/周易", "", "五行相克：木克土、土克水、水克火、火克金、金克木。")

	// 阴阳
	b.node("阳", "concept", nil, []string{base})
	b.node("阴", "concept", nil, []string{base})

	// 八卦
	bagua := "八卦"
	for _, r := range baguaRows {
		id := "八卦·" + r.name
		b.node(id, "concept", []string{r.name, r.koujue}, []string{zy, base})
		b.edge(id, "part_of", bagua, 0.95, "", "structural", "")

		// 五行
		b.edge(id, "belongs_to", r.wuxing, 0.9, "五行属性", "attribution", `{"属性":"五行"}`)
		b.ev(b.edgeID, "corpus_summary", ref+"/周易", "", "八卦·"+r.name+"五行属"+r.wuxing+"（后天配卦）。")

		// 自然取象（说卦 第十一章 逐字）
		b.edge(id, "maps_to", r.nature, 0.9, "自然取象", "attribution", `{"取象":"自然"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, r.xiang, "八卦·"+r.name+"取象为"+r.nature+"。")

		// 卦德（说卦 第七章 逐字）
		b.edge(id, "described_by", r.de, 0.9, "卦德", "attribution", `{"取象":"卦德"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, sShuoguaDe, "卦德："+r.de+"。")

		// 家庭（说卦 第十章 逐字）
		b.edge(id, "maps_to", r.family, 0.9, "家庭取象", "attribution", `{"取象":"家庭"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, sShuoguaFamily, "八卦·"+r.name+"取象为"+r.family+"。")

		// 身体（说卦 第九章 逐字）
		b.edge(id, "maps_to", r.body, 0.9, "身体取象", "attribution", `{"取象":"身体"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, sShuoguaBody, "八卦·"+r.name+"取象为"+r.body+"。")

		// 动物（说卦 第八章 逐字）
		b.edge(id, "maps_to", r.animal, 0.9, "动物取象", "attribution", `{"取象":"动物"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, sShuoguaAnimal, "八卦·"+r.name+"取象为"+r.animal+"。")

		// 方位（后天：说卦 第五章 逐字；先天：说卦 第三章 逐字）
		b.edge(id, "maps_to", r.houtian, 0.9, "后天方位", "attribution", `{"取象":"方位","frame":"后天"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, sShuoguaHoutian, "后天八卦中，八卦·"+r.name+"居"+r.houtian+"方。")
		b.edge(id, "maps_to", r.xiantian, 0.9, "先天方位", "attribution", `{"取象":"方位","frame":"先天"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, sShuoguaXiantian, "先天八卦中，八卦·"+r.name+"居"+r.xiantian+"方。")

		// 数（先天数、洛书九宫数，传统常识，概括标注）
		b.edge(id, "maps_to", r.xiantianN, 0.85, "先天八卦数", "attribution", `{"取象":"数","frame":"先天","system":"先天八卦数"}`)
		b.ev(b.edgeID, "corpus_summary", ref+"/周易", "", "先天八卦数：八卦·"+r.name+"为数"+r.xiantianN+"（邵雍）。")
		b.edge(id, "maps_to", r.houtianN, 0.85, "洛书九宫数", "attribution", `{"取象":"数","frame":"后天","system":"洛书九宫数"}`)
		b.ev(b.edgeID, "corpus_summary", ref+"/周易", "", "洛书九宫数：八卦·"+r.name+"为数"+r.houtianN+"。")

		// 阴阳
		yy := "阴"
		if r.yang {
			yy = "阳"
		}
		b.edge(id, "belongs_to", yy, 0.9, "阴阳属性", "attribution", `{"属性":"阴阳"}`)
		b.ev(b.edgeID, "corpus_summary", ref+"/周易", "", "八卦·"+r.name+"为"+yy+"卦。")
	}

	// 取象值节点
	for _, v := range []string{"天", "泽", "雷", "风", "山", "地"} {
		b.node(v, "concept", nil, []string{zy})
	}
	for _, v := range []string{"父", "母", "长男", "长女", "中男", "中女", "少男", "少女"} {
		b.node(v, "concept", nil, []string{zy})
	}
	for _, v := range []string{"首", "腹", "足", "股", "耳", "目", "手", "口"} {
		b.node(v, "concept", nil, []string{zy})
	}
	for _, v := range []string{"马", "牛", "龙", "鸡", "豕", "雉", "狗", "羊"} {
		b.node(v, "concept", nil, []string{zy})
	}
	for _, v := range []string{"健", "顺", "动", "入", "陷", "丽", "止", "说"} {
		b.node(v, "concept", nil, []string{zy})
	}
	for _, v := range []string{"南", "北", "东", "西", "东南", "西北", "东北", "西南"} {
		b.node(v, "concept", nil, []string{zy, base})
	}
	for _, v := range []string{"一", "二", "三", "四", "五", "六", "七", "八", "九"} {
		b.node(v, "concept", nil, []string{base})
	}

	// 六十四卦
	for _, g := range gua64Rows {
		gid := "卦·" + g.name
		b.node(gid, "concept", []string{g.name}, []string{zy})
		b.edge(gid, "part_of", "六十四卦", 0.95, "", "structural", "")
		b.edge(gid, "consists_of", "八卦·"+g.upper, 0.95, "上卦", "structural", `{"role":"上卦"}`)
		b.edge(gid, "consists_of", "八卦·"+g.lower, 0.95, "下卦", "structural", `{"role":"下卦"}`)

		seqID := fmt.Sprintf("卦序·%d", g.seq)
		b.node(seqID, "concept", nil, []string{zy})
		b.edge(seqID, "belongs_to", "六十四卦序", 0.95, "", "structural", "")
		b.edge(gid, "ordered_at", seqID, 0.95, "", "structural", "")
		b.ev(b.edgeID, "corpus_summary", ref+"/周易", "",
			fmt.Sprintf("周易六十四卦第%d卦：%s（上%s下%s）。", g.seq, g.name, g.upper, g.lower))
	}
}

func (b *foundationBuilder) assertSnippets() {
	for _, s := range b.verified {
		if !strings.Contains(b.corpus, s) {
			writeJSONAndExit(2, map[string]any{
				"error": "seed-foundation: snippet not found verbatim in corpus",
				"miss":  s[:40],
			})
		}
	}
}

func loadCorpusForSeed(dir string) string {
	data, err := os.ReadFile(dir + "/說卦.txt")
	if err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("seed-foundation: read corpus: %v", err)})
	}
	return string(data)
}

var wsRe = regexp.MustCompile(`\s+`)

func stripWS(s string) string {
	return wsRe.ReplaceAllString(s, "")
}
