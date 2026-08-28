package source

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	reCtextRes  = regexp.MustCompile(`href="([^"]*wiki\.pl\?[^"]*res=(\d+)[^"]*)"[^>]*>([^<]{1,60})`)
	reCtextCell = regexp.MustCompile(`<td class="ctext">(.*?)</td>`)
)

// SearchCtext returns candidate book entries from ctext's book search.
func SearchCtext(ctx context.Context, query string) ([]SearchHit, error) {
	u := "https://ctext.org/searchbooks.pl?if=gb&searchu=" + url.QueryEscape(query)
	body, err := httpGet(ctx, u)
	if err != nil {
		return nil, err
	}
	html := string(body)
	seen := map[string]bool{}
	var hits []SearchHit
	for _, m := range reCtextRes.FindAllStringSubmatch(html, -1) {
		label := strings.TrimSpace(stripTags(m[3]))
		if label == "" || seen[m[2]] {
			continue
		}
		seen[m[2]] = true
		hits = append(hits, SearchHit{
			Provider: ProviderCtext,
			Title:    label,
			URL:      "https://ctext.org/wiki.pl?if=gb&res=" + m[2],
			Snippet:  "ctext res=" + m[2],
		})
	}
	return hits, nil
}

// FetchCtext downloads the configured chapters of a ctext book.
func FetchCtext(ctx context.Context, book Book, dir string, force bool) (*BookManifest, error) {
	if book.ResID == "" {
		return nil, fmt.Errorf("ctext book %s missing res id", book.ID)
	}
	if len(book.Chapters) == 0 {
		return nil, fmt.Errorf("ctext book %s has no chapters configured (need chapter id/label pairs)", book.ID)
	}
	bookDir := filepath.Join(dir, book.ID)
	if err := os.MkdirAll(bookDir, 0o755); err != nil {
		return nil, err
	}
	var entries []ChapterEntry
	for _, ch := range book.Chapters {
		text, err := fetchCtextChapter(ctx, ch.ID)
		if err != nil {
			return nil, fmt.Errorf("fetch %s chapter %s: %w", book.ID, ch.Label, err)
		}
		fname := slugify(ch.Label) + ".txt"
		if err := writeIfChanged(filepath.Join(bookDir, fname), text, force); err != nil {
			return nil, err
		}
		entries = append(entries, ChapterEntry{File: fname, Title: ch.Label, Chars: len([]rune(text))})
		time.Sleep(800 * time.Millisecond)
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
		Provider:   ProviderCtext,
		Source:     "https://ctext.org/wiki.pl?if=gb&res=" + book.ResID,
		Chapters:   entries,
		TotalChars: total,
	}, nil
}

func fetchCtextChapter(ctx context.Context, chapterID string) (string, error) {
	u := "https://ctext.org/wiki.pl?if=gb&chapter=" + url.QueryEscape(chapterID)
	body, err := httpGet(ctx, u)
	if err != nil {
		return "", err
	}
	html := string(body)
	var parts []string
	for _, m := range reCtextCell.FindAllStringSubmatch(html, -1) {
		text := stripTags(m[1])
		text = strings.TrimSpace(text)
		if text != "" {
			parts = append(parts, text)
		}
	}
	text := squeezeBlankLines(strings.Join(parts, "\n\n"))
	sim, err := SimplifyChinese(text)
	if err != nil {
		return "", err
	}
	return sim, nil
}

func stripTags(s string) string {
	s = reAnyTag.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	return s
}
