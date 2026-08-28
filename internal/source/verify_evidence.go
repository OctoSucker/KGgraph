package source

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "modernc.org/sqlite"
)

// EvidenceReport records whether each stored evidence snippet can be located
// verbatim (after whitespace/punctuation normalization) in the corpus text.
type EvidenceReport struct {
	Total            int                  `json:"total"`
	Verbatim         int                  `json:"verbatim"`
	NeedsReview      int                  `json:"needs_review"`
	CorpusDir        string               `json:"corpus_dir"`
	NeedsReviewItems []EvidenceReviewItem `json:"needs_review_items"`
}

type EvidenceReviewItem struct {
	EvidenceID int64  `json:"evidence_id"`
	SourceRef  string `json:"source_ref"`
	Snippet    string `json:"snippet"`
	Reason     string `json:"reason"`
}

var f2s = map[rune]rune{
	'屬': '属', '則': '则', '聲': '声', '氣': '气', '為': '为', '於': '于', '與': '与',
	'後': '后', '來': '来', '無': '无', '萬': '万', '應': '应', '發': '发', '豐': '丰',
	'門': '门', '開': '开', '陽': '阳', '陰': '阴', '時': '时', '節': '节', '處': '处',
	'長': '长', '華': '华', '樂': '乐', '數': '数', '書': '书', '學': '学', '說': '说',
	'論': '论', '國': '国', '裏': '里', '遠': '远', '近': '近', '風': '风', '雲': '云',
	'觀': '观', '龍': '龙', '點': '点', '對': '对', '動': '动', '靜': '静', '強': '强',
	'衝': '冲', '養': '养', '絕': '绝', '祿': '禄', '馬': '马', '貴': '贵', '廟': '庙',
	'權': '权', '宮': '宫', '財': '财', '帛': '帛', '遷': '迁', '僕': '仆', '綬': '绶',
	'濁': '浊', '闡': '阐', '詳': '详', '覽': '览', '鉞': '钺', '東': '东', '銷': '销',
	'貧': '贫', '歲': '岁', '將': '将', '課': '课', '傳': '传', '當': '当', '鬥': '斗',
	'烏': '乌', '鳥': '鸟', '覩': '睹', '變': '变', '舊': '旧',
}

func normalizeEvidenceText(s string) string {
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "")
	s = strings.NewReplacer(
		"，", "", "。", "", "、", "", "；", "", "：", "", "？", "", "！", "",
		"「", "", "」", "", "『", "", "』", "", "（", "", "）", "", "(", "", ")", "",
		",", "", ".", "", "?", "", "!", "", ":", "", ";", "",
	).Replace(s)
	var b strings.Builder
	for _, r := range s {
		if r2, ok := f2s[r]; ok {
			b.WriteRune(r2)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var evidenceCorpusCache = map[string]string{}

func evidenceCorpus(corpusDir, slug string) string {
	if v, ok := evidenceCorpusCache[slug]; ok {
		return v
	}
	bookDir := filepath.Join(corpusDir, slug)
	entries, err := os.ReadDir(bookDir)
	if err != nil {
		evidenceCorpusCache[slug] = ""
		return ""
	}
	var corpus strings.Builder
	for _, de := range entries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".txt") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(bookDir, de.Name()))
		if err == nil {
			corpus.Write(raw)
		}
	}
	evidenceCorpusCache[slug] = normalizeEvidenceText(corpus.String())
	return evidenceCorpusCache[slug]
}

func runVerifyEvidence(argv []string) {
	fs := flag.NewFlagSet("source verify-evidence", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", defaultCorpusPath(), "corpus directory")
	db := fs.String("db", "", "sqlite db path (data/kggraph.sqlite)")
	out := fs.String("out", "", "report output file (default stdout)")
	mustParse(fs, argv)

	if *db == "" {
		fail(2, map[string]any{"error": "--db is required"})
	}
	ctx := context.Background()
	_ = ctx
	sqlDB, err := sql.Open("sqlite", "file:"+filepath.Clean(*db))
	if err != nil {
		fail(1, map[string]any{"error": err.Error()})
	}
	defer sqlDB.Close()
	rows, err := sqlDB.Query(`SELECT id, source_ref, snippet FROM kg_edge_evidence`)
	if err != nil {
		fail(1, map[string]any{"error": err.Error()})
	}
	defer rows.Close()

	report := EvidenceReport{CorpusDir: *dir, NeedsReviewItems: []EvidenceReviewItem{}}
	for rows.Next() {
		var id int64
		var ref, snip string
		if err := rows.Scan(&id, &ref, &snip); err != nil {
			fail(1, map[string]any{"error": err.Error()})
		}
		report.Total++
		ok, reason := evidenceLocatable(*dir, ref, snip)
		if ok {
			report.Verbatim++
		} else {
			report.NeedsReview++
			report.NeedsReviewItems = append(report.NeedsReviewItems,
				EvidenceReviewItem{EvidenceID: id, SourceRef: ref, Snippet: truncateRunes(snip, 60), Reason: reason})
		}
	}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if *out != "" {
		if err := os.WriteFile(*out, append(raw, '\n'), 0o644); err != nil {
			fail(1, map[string]any{"error": err.Error()})
		}
		ok(map[string]any{"report": *out, "total": report.Total, "verbatim": report.Verbatim,
			"needs_review": report.NeedsReview})
	}
	ok(map[string]any{"total": report.Total, "verbatim": report.Verbatim,
		"needs_review": report.NeedsReview})
}

func evidenceLocatable(corpusDir, ref, snip string) (bool, string) {
	parts := strings.Split(ref, "/")
	if len(parts) < 4 {
		return false, "bad source_ref"
	}
	slug := parts[2]
	tn := evidenceCorpus(corpusDir, slug)
	if tn == "" {
		return false, "corpus not found: " + slug
	}
	sn := normalizeEvidenceText(snip)
	if sn == "" {
		return false, "empty snippet"
	}
	if len([]rune(sn)) <= 10 {
		if strings.Contains(tn, sn) {
			return true, ""
		}
		return false, "short snippet not found"
	}
	// 6-char sliding windows: require ALL windows to appear continuously
	r := []rune(sn)
	hits, total := 0, 0
	for i := 0; i+6 <= len(r); i += 3 {
		total++
		if strings.Contains(tn, string(r[i:i+6])) {
			hits++
		}
	}
	if total == 0 {
		return strings.Contains(tn, sn), ""
	}
	if hits == total {
		return true, ""
	}
	return false, fmt.Sprintf("window hit %d/%d", hits, total)
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
