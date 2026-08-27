package source

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	reComment   = regexp.MustCompile(`(?s)<!--.*?-->`)
	reRefTag    = regexp.MustCompile(`(?s)<ref[^>]*/>|<ref[^>]*>.*?</ref>`)
	reRefs      = regexp.MustCompile(`(?i)<references\s*/>`)
	reBlockTag  = regexp.MustCompile(`(?i)</?(?:poem|blockquote|center|br|p|div|span|small|big|sup|sub|li|ul|ol)[^>]*>`)
	reAnyTag    = regexp.MustCompile(`<[^>]+>`)
	reTemplate  = regexp.MustCompile(`\{\{[^{}]*\}\}`)
	reLangConv  = regexp.MustCompile(`-\{([^{}]*)\}-`)
	reWikiLink  = regexp.MustCompile(`\[\[(?:[^\]|]*\|)?([^\]]*)\]\]`)
	reMagicWord = regexp.MustCompile(`__[A-Z]+__`)
	reListMark  = regexp.MustCompile(`(?m)^[*#;:]+\s*`)
	reAltLine   = regexp.MustCompile(`(?m)^alt=.*$`)
)

// CleanWikitext converts zh.wikisource wikitext into plain text.
func CleanWikitext(text string) string {
	text = reComment.ReplaceAllString(text, "")
	text = reRefTag.ReplaceAllString(text, "")
	text = reRefs.ReplaceAllString(text, "")
	text = reBlockTag.ReplaceAllString(text, "\n")
	text = reAnyTag.ReplaceAllString(text, "")
	// entities must resolve before quote-marker removal
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", `"`)
	text = strings.ReplaceAll(text, "&#39;", "'")
	// language-conversion markers first: they may contain braces that would
	// otherwise hide surrounding templates from the template stripper
	text = reLangConv.ReplaceAllStringFunc(text, func(s string) string {
		inner := s[2 : len(s)-2]
		if i := strings.LastIndex(inner, "|"); i >= 0 {
			return inner[i+1:]
		}
		return inner
	})
	for i := 0; i < 30; i++ {
		next := reTemplate.ReplaceAllString(text, "")
		if next == text {
			break
		}
		text = next
	}
	text = reWikiLink.ReplaceAllString(text, "$1")
	text = reMagicWord.ReplaceAllString(text, "")
	text = reListMark.ReplaceAllString(text, "")
	text = reAltLine.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "''", "")
	text = strings.ReplaceAll(text, "'", "")
	return squeezeBlankLines(text)
}

func squeezeBlankLines(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	blank := 0
	for _, ln := range lines {
		ln = strings.TrimRight(ln, " \t")
		if strings.TrimSpace(ln) == "" {
			blank++
			if blank <= 1 {
				out = append(out, "")
			}
			continue
		}
		blank = 0
		out = append(out, ln)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// Issue is one suspected bad segment found by VerifyText.
type Issue struct {
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Reason  string `json:"reason"`
	Snippet string `json:"snippet"`
}

// VerifyText scans a plain-text chapter for corruption: replacement chars,
// private-use glyphs (common in ctext OCR breakage) and paragraphs whose
// CJK ratio is implausibly low.
func VerifyText(text string) []Issue {
	var issues []Issue
	if strings.ContainsRune(text, utf8.RuneError) {
		issues = append(issues, Issue{
			Start: strings.IndexRune(text, utf8.RuneError), Reason: "replacement_char",
			Snippet: snippetAround(text, strings.IndexRune(text, utf8.RuneError)),
		})
	}
	for _, r := range text {
		if r >= 0xE000 && r <= 0xF8FF {
			pos := strings.IndexRune(text, r)
			issues = append(issues, Issue{
				Start: pos, End: pos + len(string(r)),
				Reason: "private_use_glyph", Snippet: snippetAround(text, pos),
			})
			break
		}
	}
	// paragraph-level CJK ratio check
	for _, para := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(para)
		if len([]rune(trimmed)) < 4 {
			continue
		}
		cjk, total := 0, 0
		for _, r := range trimmed {
			if unicode.Is(unicode.Han, r) || (r >= 0x3400 && r <= 0x9FFF) {
				cjk++
			}
			if !unicode.IsSpace(r) {
				total++
			}
		}
		if total > 0 && cjk*100/total < 40 && countBad(trimmed) > 0 {
			pos := strings.Index(text, trimmed)
			issues = append(issues, Issue{
				Start: pos, End: pos + len(trimmed),
				Reason: "low_cjk_ratio", Snippet: truncated(trimmed, 80),
			})
		}
	}
	return issues
}

func countBad(s string) int {
	n := 0
	for _, r := range s {
		if r == utf8.RuneError || (r >= 0xE000 && r <= 0xF8FF) {
			n++
		}
	}
	return n
}

func snippetAround(text string, pos int) string {
	start := pos - 30
	if start < 0 {
		start = 0
	}
	end := pos + 40
	if end > len(text) {
		end = len(text)
	}
	return truncated(text[start:end], 90)
}

func truncated(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// scanBookDir lists saved .txt chapter files (excluding full.txt) for a book.
func scanBookDir(dir, slug string) []ChapterEntry {
	bookDir := filepath.Join(dir, slug)
	names, err := os.ReadDir(bookDir)
	if err != nil {
		return nil
	}
	var out []ChapterEntry
	for _, de := range names {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".txt") || de.Name() == "full.txt" {
			continue
		}
		p := filepath.Join(bookDir, de.Name())
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		out = append(out, ChapterEntry{File: de.Name(), Chars: int(info.Size())})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out
}
