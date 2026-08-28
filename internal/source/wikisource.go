package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const wikisourceAPI = "https://zh.wikisource.org/w/api.php"

type wsQuery struct {
	Query struct {
		Pages map[string]struct {
			Title     string `json:"title"`
			Revisions []struct {
				Slots struct {
					Main struct {
						Content string `json:"*"`
					} `json:"main"`
				} `json:"slots"`
			} `json:"revisions"`
		} `json:"pages"`
		Search []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"search"`
	} `json:"query"`
	Continue map[string]any `json:"continue"`
}

// SearchWikisource returns candidate page titles for a query.
func SearchWikisource(ctx context.Context, query string) ([]SearchHit, error) {
	u := wikisourceAPI + "?action=query&list=search&srnamespace=0&srlimit=20&format=json&srsearch=" +
		url.QueryEscape(query)
	body, err := httpGet(ctx, u)
	if err != nil {
		return nil, err
	}
	var q wsQuery
	if err := json.Unmarshal(body, &q); err != nil {
		return nil, err
	}
	hits := make([]SearchHit, 0, len(q.Query.Search))
	for _, s := range q.Query.Search {
		hits = append(hits, SearchHit{
			Provider: ProviderWikisource,
			Title:    s.Title,
			URL:      "https://zh.wikisource.org/wiki/" + url.PathEscape(s.Title),
			Snippet:  cleanSnippet(s.Snippet),
		})
	}
	return hits, nil
}

func cleanSnippet(s string) string {
	s = strings.ReplaceAll(s, `<span class="searchmatch">`, "")
	s = strings.ReplaceAll(s, `</span>`, "")
	return s
}

// FetchWikisource downloads a book (main page + all subpages) into dir/slug.
func FetchWikisource(ctx context.Context, book Book, dir string, force bool) (*BookManifest, error) {
	if book.Page == "" {
		return nil, fmt.Errorf("wikisource book %s missing page title", book.ID)
	}
	prefix := book.Prefix
	if prefix == "" {
		prefix = strings.TrimRight(book.Page, "/") + "/"
	}
	pages := map[string]string{}
	cont := map[string]any{}
	for {
		q := url.Values{}
		q.Set("action", "query")
		q.Set("format", "json")
		q.Set("generator", "allpages")
		q.Set("gapnamespace", "0")
		q.Set("gaplimit", "20")
		q.Set("gapprefix", prefix)
		q.Set("prop", "revisions")
		q.Set("rvprop", "content")
		q.Set("rvslots", "main")
		for k, v := range cont {
			q.Set(k, fmt.Sprint(v))
		}
		body, err := httpGet(ctx, wikisourceAPI+"?"+q.Encode())
		if err != nil {
			return nil, err
		}
		var wq wsQuery
		if err := json.Unmarshal(body, &wq); err != nil {
			return nil, err
		}
		for _, p := range wq.Query.Pages {
			if len(p.Revisions) == 0 {
				continue
			}
			content := p.Revisions[0].Slots.Main.Content
			pages[p.Title] = content
		}
		if len(wq.Continue) == 0 {
			break
		}
		cont = wq.Continue
	}
	// main page
	u := wikisourceAPI + "?action=query&format=json&titles=" + url.QueryEscape(book.Page) +
		"&prop=revisions&rvprop=content&rvslots=main"
	body, err := httpGet(ctx, u)
	if err != nil {
		return nil, err
	}
	var wq wsQuery
	if err := json.Unmarshal(body, &wq); err != nil {
		return nil, err
	}
	for _, p := range wq.Query.Pages {
		if len(p.Revisions) > 0 {
			pages[p.Title] = p.Revisions[0].Slots.Main.Content
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no content found for %s", book.Page)
	}

	bookDir := filepath.Join(dir, book.ID)
	if err := os.MkdirAll(bookDir, 0o755); err != nil {
		return nil, err
	}
	titles := make([]string, 0, len(pages))
	for t := range pages {
		titles = append(titles, t)
	}
	sort.Strings(titles)
	var entries []ChapterEntry
	for _, t := range titles {
		text, err := SimplifyChinese(CleanWikitext(pages[t]))
		if err != nil {
			return nil, fmt.Errorf("simplify %s: %w", t, err)
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		name := t
		if i := strings.LastIndex(t, "/"); i >= 0 {
			name = t[i+1:]
		}
		fname := slugify(name) + ".txt"
		if err := writeIfChanged(filepath.Join(bookDir, fname), text, force); err != nil {
			return nil, err
		}
		entries = append(entries, ChapterEntry{File: fname, Title: name, Chars: len([]rune(text))})
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
		Provider:   ProviderWikisource,
		Source:     "https://zh.wikisource.org/wiki/" + url.PathEscape(book.Page),
		Chapters:   entries,
		TotalChars: total,
	}, nil
}

func slugify(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r >= 0x4E00 && r <= 0x9FFF, r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	s := b.String()
	s = strings.Trim(s, "-")
	return strings.ReplaceAll(s, "--", "-")
}

func writeIfChanged(path, text string, force bool) error {
	if !force {
		if raw, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(raw)) == strings.TrimSpace(text) {
			return nil
		}
	}
	return os.WriteFile(path, []byte(text+"\n"), 0o644)
}

func writeFull(bookDir string, entries []ChapterEntry, force bool) error {
	var b strings.Builder
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(bookDir, e.File))
		if err != nil {
			return err
		}
		b.WriteString("\n===== " + e.Title + " =====\n\n")
		b.Write(raw)
	}
	return os.WriteFile(filepath.Join(bookDir, "full.txt"), []byte(b.String()), 0o644)
}
