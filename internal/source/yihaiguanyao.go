package source

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reYhgyBook    = regexp.MustCompile(`/book/(\d+)-([^<"&]+)`)
	reYhgyContent = regexp.MustCompile(`(?s)<div class="reader-content-v2">(.*?)<div class="reader-content-footer"`)
	reYhgyPages   = regexp.MustCompile(`(\d+)\s*页`)
	reYhgyLine    = regexp.MustCompile(`^第\d+-\d+行\s*·\s*共\d+行\s*$`)
	reYhgyChapter = regexp.MustCompile(`^(\d{1,3})章\s*(.+)$`)
	reYhgyDup     = regexp.MustCompile(`^《.*》\d{1,3}章`)
)

// SearchYihaiguanyao scans the site sitemap for matching book titles.
func SearchYihaiguanyao(ctx context.Context, query string) ([]SearchHit, error) {
	body, err := httpGet(ctx, "https://yihaiguanyao.com/sitemap-main.xml")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var hits []SearchHit
	for _, m := range reYhgyBook.FindAllStringSubmatch(string(body), -1) {
		title := strings.TrimSpace(m[2])
		if title == "" || !strings.Contains(title, query) || seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		hits = append(hits, SearchHit{
			Provider: ProviderYihaiguanyao,
			Title:    title,
			URL:      "https://yihaiguanyao.com/book/" + m[1] + "-" + title,
			Snippet:  "site book id=" + m[1],
		})
	}
	return hits, nil
}

// FetchYihaiguanyao downloads a book page by page, splits chapters and
// (by default) drops the 小雅译 translation paragraphs.
func FetchYihaiguanyao(ctx context.Context, book Book, dir string, force bool) (*BookManifest, error) {
	if book.SiteBookID == "" {
		return nil, fmt.Errorf("yihaiguanyao book %s missing site book id", book.ID)
	}
	p1, err := fetchYhgyPage(ctx, book.SiteBookID, 1)
	if err != nil {
		return nil, err
	}
	totalPages := 0
	if m := reYhgyPages.FindStringSubmatch(p1); m != nil {
		totalPages, _ = strconv.Atoi(m[1])
	}
	if totalPages <= 0 {
		totalPages = 100
	}
	if book.MaxPages > 0 && book.MaxPages < totalPages {
		totalPages = book.MaxPages
	}

	var sb strings.Builder
	sb.WriteString(cleanYhgyPage(p1))
	for p := 2; p <= totalPages; p++ {
		page, err := fetchYhgyPage(ctx, book.SiteBookID, p)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", p, err)
		}
		sb.WriteString("\n")
		sb.WriteString(cleanYhgyPage(page))
		time.Sleep(250 * time.Millisecond)
	}
	full := squeezeBlankLines(sb.String())

	bookDir := filepath.Join(dir, book.ID)
	if err := os.MkdirAll(bookDir, 0o755); err != nil {
		return nil, err
	}
	chapters, preface := splitYhgyChapters(full, book.SkipTranslation)

	var entries []ChapterEntry
	if preface != "" {
		fname := "00-序.txt"
		if err := writeIfChanged(filepath.Join(bookDir, fname), preface, force); err != nil {
			return nil, err
		}
		entries = append(entries, ChapterEntry{File: fname, Title: "序", Chars: len([]rune(preface))})
	}
	for _, ch := range chapters {
		fname := ch.Num + "-" + slugify(ch.Name) + ".txt"
		if err := writeIfChanged(filepath.Join(bookDir, fname), ch.Text, force); err != nil {
			return nil, err
		}
		entries = append(entries, ChapterEntry{File: fname, Title: ch.Num + "章 " + ch.Name, Chars: len([]rune(ch.Text))})
	}
	if err := writeFull(bookDir, entries, force); err != nil {
		return nil, err
	}
	total := 0
	for _, e := range entries {
		total += e.Chars
	}
	return &BookManifest{
		Slug:       book.ID,
		Title:      book.Title,
		Provider:   ProviderYihaiguanyao,
		Source:     "https://yihaiguanyao.com/book/" + book.SiteBookID + "-" + url.PathEscape(book.Title),
		Chapters:   entries,
		TotalChars: total,
	}, nil
}

type yhgyChapter struct {
	Num  string
	Name string
	Text string
}

func fetchYhgyPage(ctx context.Context, bookID string, page int) (string, error) {
	u := fmt.Sprintf("https://yihaiguanyao.com/read?id=%s&p=%d", bookID, page)
	body, err := httpGet(ctx, u)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func cleanYhgyPage(html string) string {
	m := reYhgyContent.FindStringSubmatch(html)
	if m == nil {
		return ""
	}
	inner := m[1]
	inner = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</div>`).ReplaceAllString(inner, "\n")
	inner = reAnyTag.ReplaceAllString(inner, "")
	inner = strings.ReplaceAll(inner, "&nbsp;", " ")
	inner = strings.ReplaceAll(inner, "&amp;", "&")
	inner = strings.ReplaceAll(inner, "&lt;", "<")
	inner = strings.ReplaceAll(inner, "&gt;", ">")
	inner = strings.ReplaceAll(inner, "&quot;", `"`)
	inner = strings.ReplaceAll(inner, "&#39;", "'")
	var keep []string
	for _, ln := range strings.Split(inner, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || ln == "/" || reYhgyLine.MatchString(ln) ||
			reYhgyPages.MatchString(ln) && len([]rune(ln)) < 12 ||
			reYhgyDup.MatchString(ln) {
			continue
		}
		keep = append(keep, ln)
	}
	return strings.Join(keep, "\n")
}

func splitYhgyChapters(full string, skipTranslation bool) ([]yhgyChapter, string) {
	lines := strings.Split(full, "\n")
	var chapters []yhgyChapter
	var preface []string
	cur := -1
	inTranslation := false
	for _, ln := range lines {
		if m := reYhgyChapter.FindStringSubmatch(ln); m != nil {
			cur++
			chapters = append(chapters, yhgyChapter{Num: m[1], Name: strings.TrimSpace(m[2])})
			inTranslation = false
			continue
		}
		if strings.Contains(ln, "以下是小雅译") {
			if skipTranslation {
				inTranslation = true
				continue
			}
			ln = "【译文】" + ln
		}
		if inTranslation {
			continue
		}
		if cur < 0 {
			if strings.TrimSpace(ln) != "" {
				preface = append(preface, ln)
			}
			continue
		}
		chapters[cur].Text += ln + "\n"
	}
	for i := range chapters {
		chapters[i].Text = strings.TrimSpace(chapters[i].Text)
	}
	var prefaceText string
	if len(preface) > 0 {
		prefaceText = squeezeBlankLines(strings.Join(preface, "\n"))
	}
	return chapters, prefaceText
}
