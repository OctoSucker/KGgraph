package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// runSeedFoundation deterministically generates the 八卦/周易 foundation
// batch from the corpus. Code contains only parsing/validation MECHANISM;
// domain facts come from two places:
//  1. corpus: 說卦.txt (八卦取象/卦德/家庭/身体/动物/后天方位) and the
//     64 卦 files (卦序 in header, 上下卦 in 大象传)
//  2. ingest/foundation-axioms.json: 约定性知识 not stated verbatim in
//     this corpus (五行配卦/先天数/洛书数/先天方位/正象/卦象口诀), each
//     with explicit provenance.
func runSeedFoundation(ctx context.Context, argv []string) {
	_ = ctx
	fs := flag.NewFlagSet("seed-foundation", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var corpusDir, axiomsPath, outPath string
	fs.StringVar(&corpusDir, "corpus", "gushu/zhouyi", "corpus dir containing 說卦.txt and the 64 卦 files")
	fs.StringVar(&axiomsPath, "axioms", "ingest/foundation-axioms.json", "axioms data file")
	fs.StringVar(&outPath, "out", "ingest/zhouyi-foundation.json", "output batch JSON path")
	mustParse(fs, argv)

	axioms := loadAxioms(axiomsPath)
	shuogua := string(readFileMust(corpusDir + "/說卦.txt"))
	bg, err := parseShuogua(shuogua)
	if err != nil {
		writeJSONAndExit(2, map[string]any{"error": "seed-foundation: " + err.Error()})
	}
	gua, err := parseGua64(corpusDir)
	if err != nil {
		writeJSONAndExit(2, map[string]any{"error": "seed-foundation: " + err.Error()})
	}
	for i := range gua {
		if err := gua[i].parseXiang(string(readFileMust(filepath.Join(corpusDir, gua[i].file))), axioms); err != nil {
			writeJSONAndExit(2, map[string]any{"error": "seed-foundation: " + err.Error()})
		}
	}
	if err := validateGua64(gua, axioms); err != nil {
		writeJSONAndExit(2, map[string]any{"error": "seed-foundation: " + err.Error()})
	}

	corpusMap := map[string]string{}
	ref := "zhouyi-v2/gushu/zhouyi"
	corpusMap[ref+"/說卦"] = stripWS(shuogua)
	for _, g := range gua {
		corpusMap[ref+"/"+strings.TrimSuffix(g.file, ".txt")] = stripWS(string(readFileMust(filepath.Join(corpusDir, g.file))))
	}
	gb := newFoundationBuilder(corpusMap)
	gb.build(axioms, bg, gua)
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
		"note":           "deterministic foundation batch; facts from corpus + axioms file; snippets verified verbatim",
	})
}

// ---------- axioms ----------

type foundationAxioms struct {
	NaturalXiang   map[string]string `json:"natural_xiang"`
	Koujue         map[string]string `json:"koujue"`
	Wuxing         map[string]string `json:"wuxing"`
	XiantianNum    map[string]string `json:"xiantian_num"`
	HoutianNum     map[string]string `json:"houtian_num"`
	XiantianDir    map[string]string `json:"xiantian_dir"`
	HoutianDirFill map[string]string `json:"houtian_dir_fill"`
	Provenance     map[string]string `json:"provenance"`
}

func loadAxioms(path string) foundationAxioms {
	raw := readFileMust(path)
	var a foundationAxioms
	if err := json.Unmarshal(raw, &a); err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("seed-foundation: axioms: %v", err)})
	}
	return a
}

// ---------- corpus parsing ----------

var baGuaNames = []string{"乾", "兑", "离", "震", "巽", "坎", "艮", "坤"}

// 说卦 uses 干 for 乾 (wikisource rendering); normalize only in 卦名 contexts.
var guaNameNormalize = map[string]string{"干": "乾"}

type baguaAttr struct {
	de           string
	family       string
	body         string
	animal       string
	houtianDir   string // explicit in 说卦; "" means axiom fill
	xiangSnippet string
}

func parseShuogua(raw string) (map[string]baguaAttr, error) {
	s := stripWS(raw)
	out := map[string]baguaAttr{}
	for _, n := range baGuaNames {
		out[n] = baguaAttr{}
	}

	// 卦德：干，健也。坤，顺也。...
	deTok := mustSlice(s, "干，健也", "兑，说也。", "卦德")
	for _, tok := range strings.Split(deTok, "。") {
		m := regexp.MustCompile(`^([干乾坤震巽坎离艮兑])，([^，]+)也$`).FindStringSubmatch(tok)
		if len(m) == 3 {
			name := normGuaName(m[1])
			a := out[name]
			a.de = m[2]
			out[name] = a
		}
	}
	// 动物：干为马，坤为牛...
	anTok := mustSlice(s, "干为马", "兑为羊。", "动物")
	for _, tok := range strings.Split(anTok, "，") {
		tok = strings.TrimSuffix(tok, "。")
		m := regexp.MustCompile(`^([干乾坤震巽坎离艮兑])为(.+)$`).FindStringSubmatch(tok)
		if len(m) == 3 {
			name := normGuaName(m[1])
			a := out[name]
			a.animal = m[2]
			out[name] = a
		}
	}
	// 身体：干为首，坤为腹...
	bdTok := mustSlice(s, "干为首", "兑为口。", "身体")
	for _, tok := range strings.Split(bdTok, "，") {
		tok = strings.TrimSuffix(tok, "。")
		m := regexp.MustCompile(`^([干乾坤震巽坎离艮兑])为(.+)$`).FindStringSubmatch(tok)
		if len(m) == 3 {
			name := normGuaName(m[1])
			a := out[name]
			a.body = m[2]
			out[name] = a
		}
	}
	// 家庭：干，天也，故称乎父；...震一索而得男，故谓之长男；...
	faTok := mustSlice(s, "干，天也，故称乎父", "兑三索而得女，故谓之少女。", "家庭")
	for _, tok := range strings.Split(faTok, "；") {
		tok = strings.TrimSuffix(tok, "。")
		if m := regexp.MustCompile(`^([干坤])，[^，]+，故称乎(父|母)$`).FindStringSubmatch(tok); len(m) == 3 {
			name := normGuaName(m[1])
			a := out[name]
			a.family = m[2]
			out[name] = a
			continue
		}
		if m := regexp.MustCompile(`^([震巽坎离艮兑])(一索|再索|三索)而得(男|女)，故谓之(长|中|少)(男|女)$`).FindStringSubmatch(tok); len(m) == 6 {
			name := normGuaName(m[1])
			a := out[name]
			a.family = m[4] + m[3]
			out[name] = a
		}
	}
	// 后天方位（说卦直言者）：震东 巽东南 离南 乾西北 坎北 艮东北
	htTok := mustSlice(s, "万物出乎震", "故曰成言乎艮。", "后天方位")
	dirRules := []struct {
		re   string
		name string
	}{
		{`(震)，东方也`, "震"},
		{`(巽)，东南也`, "巽"},
		{`(离)也者[^。]*南方之卦也`, "离"},
		{`(干)，西北之卦也`, "乾"},
		{`(坎)者，水也，正北方之卦也`, "坎"},
		{`(艮)，东北之卦也`, "艮"},
	}
	for _, r := range dirRules {
		m := regexp.MustCompile(r.re).FindStringSubmatch(htTok)
		if len(m) < 2 {
			return nil, fmt.Errorf("parse 说卦 后天方位: pattern %q not found", r.re)
		}
		name := normGuaName(m[1])
		a := out[name]
		switch name {
		case "震":
			a.houtianDir = "东"
		case "巽":
			a.houtianDir = "东南"
		case "离":
			a.houtianDir = "南"
		case "乾":
			a.houtianDir = "西北"
		case "坎":
			a.houtianDir = "北"
		case "艮":
			a.houtianDir = "东北"
		}
		out[name] = a
	}
	// 自然取象（说卦 第十一章 每卦取象句）
	xiTok := mustSlice(s, "干为天、为圜", "为妾、为羊。", "取象")
	starts := []struct {
		anchor string
		name   string
	}{
		{"干为天、为圜", "乾"}, {"坤为地、为母", "坤"}, {"震为雷、为龙", "震"},
		{"巽为木、为风", "巽"}, {"坎为水、为沟渎", "坎"}, {"离为火、为日", "离"},
		{"艮为山、为径路", "艮"}, {"兑为泽、为少女", "兑"},
	}
	type posT struct {
		idx  int
		name string
	}
	var pos []posT
	for _, st := range starts {
		i := strings.Index(xiTok, st.anchor)
		if i < 0 {
			return nil, fmt.Errorf("parse 说卦 取象: anchor %q not found", st.anchor)
		}
		pos = append(pos, posT{i, st.name})
	}
	sort.Slice(pos, func(i, j int) bool { return pos[i].idx < pos[j].idx })
	for i, p := range pos {
		end := len(xiTok)
		if i+1 < len(pos) {
			end = pos[i+1].idx
		}
		a := out[p.name]
		a.xiangSnippet = strings.Trim(xiTok[p.idx:end], "。，")
		out[p.name] = a
	}

	// 完整性校验
	for _, n := range baGuaNames {
		a := out[n]
		if a.de == "" || a.family == "" || a.body == "" || a.animal == "" || a.xiangSnippet == "" {
			return nil, fmt.Errorf("parse 说卦: incomplete attributes for 八卦·%s", n)
		}
	}
	if out["震"].houtianDir == "" || out["巽"].houtianDir == "" || out["离"].houtianDir == "" ||
		out["乾"].houtianDir == "" || out["坎"].houtianDir == "" || out["艮"].houtianDir == "" {
		return nil, fmt.Errorf("parse 说卦: incomplete 后天方位 (expected 震巽离乾坎艮)")
	}
	return out, nil
}

func normGuaName(s string) string {
	if v, ok := guaNameNormalize[s]; ok {
		return v
	}
	return s
}

func mustSlice(s, start, end, label string) string {
	i := strings.Index(s, start)
	if i < 0 {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("seed-foundation: 说卦 %s: start anchor %q missing", label, start)})
	}
	j := strings.Index(s[i:], end)
	if j < 0 {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("seed-foundation: 说卦 %s: end anchor %q missing", label, end)})
	}
	return s[i : i+j+len(end)]
}

// ---------- 六十四卦 parsing ----------

type guaRow struct {
	file     string
	seq      int
	nameTrad string
	name     string // simplified 卦名
	full     string // 水雷屯 / 乾为天
	upper    string // 八卦名
	lower    string
	header   string // 周易　第X卦
	xiang    string // 大象 sentence verbatim
	phrase   string // 象辞（去掉卦名与君子以）
}

var guaNameSimplify = map[string]string{
	"干": "乾", "訟": "讼", "師": "师", "謙": "谦", "隨": "随", "蠱": "蛊",
	"觀": "观", "賁": "贲", "剝": "剥", "復": "复", "頤": "颐", "大過": "大过",
	"離": "离", "遯": "遁", "大壯": "大壮", "晉": "晋", "損": "损", "漸": "渐",
	"歸妹": "归妹", "豐": "丰", "渙": "涣", "節": "节", "既濟": "既济", "未濟": "未济",
	"兌": "兑",
}

var guaHeaderRe = regexp.MustCompile(`周易[　 ]*第([一二三四五六七八九十]+)卦`)
var guaDetectRe = regexp.MustCompile(`第[一二三四五六七八九十]+卦`)

var chineseDigits = map[rune]int{
	'一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
	'六': 6, '七': 7, '八': 8, '九': 9, '十': 10,
}

func chineseNumeral(s string) int {
	total, cur := 0, 0
	for _, r := range s {
		v := chineseDigits[r]
		if v == 10 {
			if cur == 0 {
				cur = 10
			} else {
				cur *= 10
			}
			total += cur
			cur = 0
		} else {
			cur = v
		}
	}
	return total + cur
}

func parseGua64(dir string) ([]guaRow, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []guaRow
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		if e.Name() == "full.txt" || e.Name() == "周易.txt" {
			continue
		}
		content := string(readFileMust(filepath.Join(dir, e.Name())))
		if !guaDetectRe.MatchString(content) {
			continue
		}
		g, err := parseGuaFile(e.Name(), content)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func parseGuaFile(file, content string) (guaRow, error) {
	lines := strings.Split(content, "\n")
	for i, ln := range lines {
		if m := guaHeaderRe.FindStringSubmatch(ln); len(m) == 2 {
			seq := chineseNumeral(m[1])
			nameTrad := ""
			for _, ln2 := range lines[i+1:] {
				if strings.TrimSpace(ln2) != "" {
					nameTrad = strings.TrimSpace(ln2)
					break
				}
			}
			if nameTrad == "" {
				return guaRow{}, fmt.Errorf("%s: 卦名 missing", file)
			}
			name := nameTrad
			if v, ok := guaNameSimplify[nameTrad]; ok {
				name = v
			}
			return guaRow{file: file, seq: seq, nameTrad: nameTrad, name: name, header: strings.TrimSpace(ln)}, nil
		}
	}
	return guaRow{}, fmt.Errorf("%s: 卦序 header missing", file)
}

var xiangCutRe = regexp.MustCompile(`^(.*?)(；|，君子以|，先王以|，后以|，大人以|，上以)`)

func (g *guaRow) parseXiang(content string, axioms foundationAxioms) error {
	m := regexp.MustCompile(`象曰：\n(.+?。)`).FindStringSubmatch(content)
	if len(m) != 2 {
		return fmt.Errorf("%s: 大象传 missing", g.file)
	}
	g.xiang = m[1]
	phrase := m[1]
	if c := xiangCutRe.FindStringSubmatch(phrase); len(c) == 3 {
		phrase = c[1]
	}
	// 先去句号，再去尾部“，卦名”或“，习卦名”
	phrase = strings.TrimSuffix(phrase, "。")
	for _, nm := range []string{g.nameTrad, g.name} {
		phrase = regexp.MustCompile(`，物与`+regexp.QuoteMeta(nm)+`$`).ReplaceAllString(phrase, "")
		phrase = regexp.MustCompile(`，?(习)?`+regexp.QuoteMeta(nm)+`$`).ReplaceAllString(phrase, "")
	}
	g.phrase = phrase

	// 八纯卦：上下同卦
	if contains(baGuaNames, g.name) {
		g.upper, g.lower = g.name, g.name
		return nil
	}
	special := map[string][2]string{
		"泰": {"坤", "乾"}, "否": {"乾", "坤"},
		"噬嗑": {"离", "震"}, "丰": {"震", "离"},
	}
	if v, ok := special[g.name]; ok {
		g.upper, g.lower = v[0], v[1]
		return nil
	}
	up, low, err := parseXiangPhrase(phrase, g.name)
	if err != nil {
		return fmt.Errorf("%s (%s): %w", g.file, phrase, err)
	}
	g.upper, g.lower = up, low
	return nil
}

var imageToGua = map[string]string{
	"天": "乾", "地": "坤", "雷": "震", "风": "巽",
	"水": "坎", "云": "坎", "雨": "坎", "泉": "坎",
	"火": "离", "明": "离", "电": "离", "泽": "兑",
	"山": "艮", "木": "巽",
}

func parseXiangPhrase(phrase, gua string) (string, string, error) {
	rules := []struct {
		re *regexp.Regexp
		up int
		lo int
	}{
		{regexp.MustCompile(`^上(.+?)下(.+)$`), 1, 2},
		{regexp.MustCompile(`^(.+)上有(.+)$`), 2, 1},
		{regexp.MustCompile(`^(.+)上于(.+)$`), 1, 2},
		{regexp.MustCompile(`^(.+)在(.+)上$`), 1, 2},
		{regexp.MustCompile(`^(.+)行(.+)上$`), 1, 2},
		{regexp.MustCompile(`^(.+)中有(.+)$`), 1, 2},
		{regexp.MustCompile(`^(.+)在(.+)中$`), 2, 1},
		{regexp.MustCompile(`^(.+)入(.+)中$`), 2, 1},
		{regexp.MustCompile(`^(.+)中生(.+)$`), 1, 2},
		{regexp.MustCompile(`^(.+)下(.+)行$`), 1, 2},
		{regexp.MustCompile(`^(.+)下有(.+)$`), 1, 2},
		{regexp.MustCompile(`^(.+)下出(.+)$`), 1, 2},
		{regexp.MustCompile(`^(.+)下(.+)$`), 1, 2},
		{regexp.MustCompile(`^(.+)出(.+)上$`), 1, 2},
		{regexp.MustCompile(`^(.+)出(.+)奋$`), 1, 2},
		{regexp.MustCompile(`^(.+)自(.+)出$`), 1, 2},
		{regexp.MustCompile(`^(.+)附(.+)上$`), 1, 2},
		{regexp.MustCompile(`^(.+)与(.+)违行$`), 1, 2},
		{regexp.MustCompile(`^(.+)与(.+)$`), 1, 2},
		{regexp.MustCompile(`^(.+)灭(.+)$`), 1, 2},
		{regexp.MustCompile(`^(.+)无(.+)$`), 1, 2},
	}
	for _, r := range rules {
		if m := r.re.FindStringSubmatch(phrase); len(m) == 3 {
			up, ok1 := imageToGua[m[r.up]]
			low, ok2 := imageToGua[m[r.lo]]
			if ok1 && ok2 {
				return up, low, nil
			}
			return "", "", fmt.Errorf("unknown images %q -> %q", m[1], m[2])
		}
	}
	// 两个象直接连写：云雷、雷风、风雷；或 X 作：雷雨作
	img := strings.TrimSuffix(phrase, "作")
	runes := []rune(img)
	if len(runes) == 2 {
		up, ok1 := imageToGua[string(runes[0])]
		low, ok2 := imageToGua[string(runes[1])]
		if ok1 && ok2 {
			return up, low, nil
		}
	}
	return "", "", fmt.Errorf("cannot parse 象辞 %q for 卦·%s", phrase, gua)
}

func validateGua64(gua []guaRow, axioms foundationAxioms) error {
	if len(gua) != 64 {
		return fmt.Errorf("expected 64 卦 files, got %d", len(gua))
	}
	seqs := map[int]bool{}
	names := map[string]bool{}
	pairs := map[string]bool{}
	for i := range gua {
		g := &gua[i]
		if g.seq < 1 || g.seq > 64 || seqs[g.seq] {
			return fmt.Errorf("duplicate/invalid 卦序 %d for %s", g.seq, g.file)
		}
		seqs[g.seq] = true
		if names[g.name] {
			return fmt.Errorf("duplicate 卦名 %s", g.name)
		}
		names[g.name] = true
		upX, okU := axioms.NaturalXiang[g.upper]
		loX, okL := axioms.NaturalXiang[g.lower]
		if !okU || !okL {
			return fmt.Errorf("%s: upper/lower not resolved: %s/%s", g.file, g.upper, g.lower)
		}
		if g.upper == g.lower {
			g.full = g.name + "为" + upX
		} else {
			g.full = upX + loX + g.name
		}
		key := g.upper + "/" + g.lower
		if pairs[key] {
			return fmt.Errorf("duplicate 上下卦 pair %s", key)
		}
		pairs[key] = true
	}
	if len(pairs) != 64 {
		return fmt.Errorf("上下卦 pairs must cover 8x8 exactly once, got %d", len(pairs))
	}
	return nil
}

// ---------- builder ----------

type foundationBuilder struct {
	corpus   map[string]string
	nodes    []map[string]any
	edges    []map[string]any
	evidence []map[string]any
	edgeID   int
	verified [][2]string // [sourceRef, snippet]
}

func newFoundationBuilder(corpus map[string]string) *foundationBuilder {
	return &foundationBuilder{corpus: corpus}
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
		b.verified = append(b.verified, [2]string{sourceRef, stripWS(snippet)})
	}
	if translation != "" {
		ev["translation"] = translation
	}
	b.evidence = append(b.evidence, ev)
}

func (b *foundationBuilder) build(ax foundationAxioms, bg map[string]baguaAttr, gua []guaRow) {
	const (
		zy   = "周易"
		base = "易学基础"
		ref  = "zhouyi-v2/gushu/zhouyi"
		sg   = ref + "/說卦"
	)

	b.node("周易", "concept", []string{"易经", "易"}, []string{zy})
	b.node("八卦", "concept", []string{"经卦", "三爻卦"}, []string{zy})
	b.node("六十四卦", "concept", []string{"别卦", "重卦"}, []string{zy})
	b.node("六十四卦序", "concept", []string{"周易卦序"}, []string{zy})
	b.node("阴阳", "concept", nil, []string{base})
	b.node("五行", "concept", nil, []string{base})
	b.edge("八卦", "part_of", "周易", 0.95, "", "structural", "")
	b.edge("六十四卦", "part_of", "周易", 0.95, "", "structural", "")
	b.edge("六十四卦序", "part_of", "周易", 0.95, "", "structural", "")

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
	b.node("阳", "concept", nil, []string{base})
	b.node("阴", "concept", nil, []string{base})

	// 爻位（bottom→top，1=阳）为卦形定义，用于推阴阳：阳爻数为奇 → 阳卦
	yangByGua := map[string]bool{}
	for n, bits := range map[string][3]int{
		"乾": {1, 1, 1}, "兑": {1, 1, 0}, "离": {1, 0, 1}, "震": {1, 0, 0},
		"巽": {0, 1, 1}, "坎": {0, 1, 0}, "艮": {0, 0, 1}, "坤": {0, 0, 0},
	} {
		yangByGua[n] = (bits[0]+bits[1]+bits[2])%2 == 1
	}

	for _, n := range baGuaNames {
		id := "八卦·" + n
		a := bg[n]
		b.node(id, "concept", []string{n, ax.Koujue[n]}, []string{zy, base})
		b.edge(id, "part_of", "八卦", 0.95, "", "structural", "")

		b.edge(id, "belongs_to", ax.Wuxing[n], 0.9, "五行属性", "attribution", `{"属性":"五行"}`)
		b.ev(b.edgeID, "corpus_summary", ref+"/周易", "", "八卦·"+n+"五行属"+ax.Wuxing[n]+"（"+ax.Provenance["wuxing"]+"）。")

		b.edge(id, "maps_to", ax.NaturalXiang[n], 0.9, "自然取象", "attribution", `{"取象":"自然"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, a.xiangSnippet, "八卦·"+n+"取象为"+ax.NaturalXiang[n]+"。")

		b.edge(id, "described_by", a.de, 0.9, "卦德", "attribution", `{"取象":"卦德"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, "干，健也。坤，顺也。震，动也。巽，入也。坎，陷也。离，丽也。艮，止也。兑，说也。", "卦德："+a.de+"。")

		b.edge(id, "maps_to", a.family, 0.9, "家庭取象", "attribution", `{"取象":"家庭"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, "干，天也，故称乎父；坤，地也，故称乎母；震一索而得男，故谓之长男；巽一索而得女，故谓之长女；坎再索而得男，故谓之中男；离再索而得女，故谓之中女；艮三索而得男，故谓之少男；兑三索而得女，故谓之少女。", "八卦·"+n+"取象为"+a.family+"。")

		b.edge(id, "maps_to", a.body, 0.9, "身体取象", "attribution", `{"取象":"身体"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, "干为首，坤为腹，震为足，巽为股，坎为耳，离为目，艮为手，兑为口。", "八卦·"+n+"取象为"+a.body+"。")

		b.edge(id, "maps_to", a.animal, 0.9, "动物取象", "attribution", `{"取象":"动物"}`)
		b.ev(b.edgeID, "corpus_extracted", sg, "干为马，坤为牛，震为龙，巽为鸡，坎为豕，离为雉，艮为狗，兑为羊。", "八卦·"+n+"取象为"+a.animal+"。")

		if a.houtianDir != "" {
			b.edge(id, "maps_to", a.houtianDir, 0.9, "后天方位", "attribution", `{"取象":"方位","frame":"后天"}`)
			b.ev(b.edgeID, "corpus_extracted", sg, "帝出乎震，齐乎巽，相见乎离，致役乎坤，说言乎兑，战乎干，劳乎坎，成言乎艮。万物出乎震，震，东方也。齐乎巽，巽，东南也，齐也者，言万物之洁齐也。离也者，明也，万物皆相见，南方之卦也。圣人南面而听天下，向明而治，盖取诸此也。坤也者，地也，万物皆致养焉，故曰致役乎坤。兑，正秋也。万物之所说也。故曰说；言乎兑。战乎干，干，西北之卦也。言阴阳相薄也。坎者，水也，正北方之卦也，劳卦也，万物之所归也，故曰劳乎坎。艮，东北之卦也，万物之所成，终而所成始也，故曰成言乎艮。", "后天八卦中，八卦·"+n+"居"+a.houtianDir+"方。")
		} else {
			b.edge(id, "maps_to", ax.HoutianDirFill[n], 0.85, "后天方位", "attribution", `{"取象":"方位","frame":"后天"}`)
			b.ev(b.edgeID, "corpus_summary", sg, "", "后天八卦中，八卦·"+n+"居"+ax.HoutianDirFill[n]+"方（"+ax.Provenance["houtian_dir_fill"]+"）。")
		}

		b.edge(id, "maps_to", ax.XiantianDir[n], 0.85, "先天方位", "attribution", `{"取象":"方位","frame":"先天"}`)
		b.ev(b.edgeID, "corpus_summary", sg, "天地定位，山泽通气，雷风相薄，水火不相射。八卦相错，数往者顺，知来者逆。是故易逆数也。", "先天八卦中，八卦·"+n+"居"+ax.XiantianDir[n]+"方（"+ax.Provenance["xiantian_dir"]+"）。")

		b.edge(id, "maps_to", ax.XiantianNum[n], 0.85, "先天八卦数", "attribution", `{"取象":"数","frame":"先天","system":"先天八卦数"}`)
		b.ev(b.edgeID, "corpus_summary", ref+"/周易", "", "先天八卦数：八卦·"+n+"为数"+ax.XiantianNum[n]+"（"+ax.Provenance["xiantian_num"]+"）。")
		b.edge(id, "maps_to", ax.HoutianNum[n], 0.85, "洛书九宫数", "attribution", `{"取象":"数","frame":"后天","system":"洛书九宫数"}`)
		b.ev(b.edgeID, "corpus_summary", ref+"/周易", "", "洛书九宫数：八卦·"+n+"为数"+ax.HoutianNum[n]+"（"+ax.Provenance["houtian_num"]+"）。")

		yy := "阴"
		if yangByGua[n] {
			yy = "阳"
		}
		b.edge(id, "belongs_to", yy, 0.9, "阴阳属性", "attribution", `{"属性":"阴阳"}`)
		b.ev(b.edgeID, "corpus_summary", ref+"/周易", "", "八卦·"+n+"为"+yy+"卦（阳爻数为奇则阳）。")
	}

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

	for _, g := range gua {
		gid := "卦·" + g.full
		b.node(gid, "concept", []string{g.full, g.name}, []string{zy})
		b.edge(gid, "part_of", "六十四卦", 0.95, "", "structural", "")
		refFile := ref + "/" + strings.TrimSuffix(g.file, ".txt")
		b.edge(gid, "consists_of", "八卦·"+g.upper, 0.95, "上卦", "structural", `{"role":"上卦"}`)
		b.ev(b.edgeID, "corpus_extracted", refFile, g.xiang, g.full+"：上卦"+g.upper+"。")
		b.edge(gid, "consists_of", "八卦·"+g.lower, 0.95, "下卦", "structural", `{"role":"下卦"}`)
		b.ev(b.edgeID, "corpus_extracted", refFile, g.xiang, g.full+"：下卦"+g.lower+"。")

		seqID := fmt.Sprintf("卦序·%d", g.seq)
		b.node(seqID, "concept", nil, []string{zy})
		b.edge(seqID, "belongs_to", "六十四卦序", 0.95, "", "structural", "")
		b.edge(gid, "ordered_at", seqID, 0.95, "", "structural", "")
		b.ev(b.edgeID, "corpus_extracted", refFile, g.header, "周易六十四卦第"+g.name+"卦为第"+fmt.Sprintf("%d", g.seq)+"卦。")
	}
}

func (b *foundationBuilder) assertSnippets() {
	for _, v := range b.verified {
		corpus, ok := b.corpus[v[0]]
		if !ok {
			writeJSONAndExit(2, map[string]any{"error": "seed-foundation: no corpus registered for " + v[0]})
		}
		if !strings.Contains(corpus, v[1]) {
			writeJSONAndExit(2, map[string]any{
				"error": "seed-foundation: snippet not found verbatim in corpus",
				"ref":   v[0],
				"miss":  v[1][:40],
			})
		}
	}
}

func readFileMust(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("seed-foundation: read %s: %v", path, err)})
	}
	return data
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

var wsRe = regexp.MustCompile(`\s+`)

func stripWS(s string) string {
	return wsRe.ReplaceAllString(s, "")
}
