package source

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func runSearch(argv []string) {
	fs := flag.NewFlagSet("source search", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var query string
	var provider string
	fs.StringVar(&query, "query", "", "search query (or first positional arg)")
	fs.StringVar(&provider, "provider", "", "wikisource | ctext | yihaiguanyao (default all)")
	mustParse(fs, argv)
	if query == "" && fs.NArg() > 0 {
		query = fs.Arg(0)
	}
	if query == "" {
		fail(2, map[string]any{"error": "--query is required"})
	}
	ctx := context.Background()
	var all []SearchHit
	if provider == "" || provider == ProviderWikisource {
		hits, err := SearchWikisource(ctx, query)
		if err != nil {
			fail(1, map[string]any{"error": err.Error()})
		}
		all = append(all, hits...)
	}
	if provider == "" || provider == ProviderCtext {
		hits, err := SearchCtext(ctx, query)
		if err != nil {
			fail(1, map[string]any{"error": err.Error()})
		}
		all = append(all, hits...)
	}
	if provider == "" || provider == ProviderYihaiguanyao {
		hits, err := SearchYihaiguanyao(ctx, query)
		if err != nil {
			fail(1, map[string]any{"error": err.Error()})
		}
		all = append(all, hits...)
	}
	ok(map[string]any{"query": query, "provider": provider, "hits": all, "count": len(all)})
}

func runFetch(argv []string) {
	fs := flag.NewFlagSet("source fetch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", defaultCorpusPath(), "corpus directory")
	bookID := fs.String("book-id", "", "book id from corpus.json")
	all := fs.Bool("all", false, "fetch every book in corpus.json")
	force := fs.Bool("force", false, "re-download even when files match")
	keepTranslation := fs.Bool("keep-translation", false, "keep 小雅译 paragraphs (yihaiguanyao)")
	mustParse(fs, argv)

	path := corpusPath(*dir)
	c, err := loadCorpus(path)
	if err != nil {
		fail(1, map[string]any{"error": fmt.Sprintf("load corpus: %v", err)})
	}
	var books []Book
	if *all {
		books = c.Books
	} else if *bookID != "" {
		for _, b := range c.Books {
			if b.ID == *bookID {
				books = []Book{b}
				break
			}
		}
		if len(books) == 0 {
			fail(2, map[string]any{"error": fmt.Sprintf("book %q not in corpus", *bookID),
				"hint": "kggraph source add --id ..."})
		}
	} else {
		fail(2, map[string]any{"error": "need --book-id or --all"})
	}

	ctx := context.Background()
	var results []BookManifest
	for _, b := range books {
		if *keepTranslation {
			b.SkipTranslation = false
		} else {
			b.SkipTranslation = true
		}
		entry, err := fetchBook(ctx, b, *dir, *force)
		if err != nil {
			fail(1, map[string]any{"error": fmt.Sprintf("fetch %s: %v", b.ID, err)})
		}
		if err := upsertManifest(manifestPath(*dir), *entry); err != nil {
			fail(1, map[string]any{"error": err.Error()})
		}
		results = append(results, *entry)
	}
	ok(map[string]any{"fetched": results})
}

func fetchBook(ctx context.Context, b Book, dir string, force bool) (*BookManifest, error) {
	switch b.Provider {
	case ProviderWikisource:
		return FetchWikisource(ctx, b, dir, force)
	case ProviderCtext:
		return FetchCtext(ctx, b, dir, force)
	case ProviderYihaiguanyao:
		return FetchYihaiguanyao(ctx, b, dir, force)
	default:
		return nil, fmt.Errorf("unknown provider %q", b.Provider)
	}
}

func runList(argv []string) {
	fs := flag.NewFlagSet("source list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", defaultCorpusPath(), "corpus directory")
	mustParse(fs, argv)
	m, err := loadManifest(manifestPath(*dir))
	if err != nil {
		fail(1, map[string]any{"error": err.Error()})
	}
	ok(map[string]any{"books": m.Books, "count": len(m.Books)})
}

func runVerify(argv []string) {
	fs := flag.NewFlagSet("source verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", defaultCorpusPath(), "corpus directory")
	bookID := fs.String("book-id", "", "book id to verify (default all)")
	mustParse(fs, argv)

	m, err := loadManifest(manifestPath(*dir))
	if err != nil {
		fail(1, map[string]any{"error": err.Error()})
	}
	type report struct {
		Book   string  `json:"book"`
		Issues []Issue `json:"issues"`
	}
	var reports []report
	for _, bm := range m.Books {
		if *bookID != "" && bm.Slug != *bookID {
			continue
		}
		var issues []Issue
		for _, ch := range bm.Chapters {
			raw, err := os.ReadFile(filepath.Join(*dir, bm.Slug, ch.File))
			if err != nil {
				continue
			}
			found := VerifyText(string(raw))
			for i := range found {
				found[i].Snippet = fmt.Sprintf("[%s] %s", ch.File, found[i].Snippet)
			}
			issues = append(issues, found...)
		}
		reports = append(reports, report{Book: bm.Slug, Issues: issues})
	}
	ok(map[string]any{"reports": reports})
}

func runClean(argv []string) {
	fs := flag.NewFlagSet("source clean", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", defaultCorpusPath(), "corpus directory")
	bookID := fs.String("book-id", "", "book id to clean (default all)")
	mustParse(fs, argv)

	m, err := loadManifest(manifestPath(*dir))
	if err != nil {
		fail(1, map[string]any{"error": err.Error()})
	}
	cleaned := 0
	for _, bm := range m.Books {
		if *bookID != "" && bm.Slug != *bookID {
			continue
		}
		bookDir := filepath.Join(*dir, bm.Slug)
		for _, ch := range bm.Chapters {
			p := filepath.Join(bookDir, ch.File)
			raw, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			out := CleanWikitext(string(raw))
			if strings.TrimSpace(out) != strings.TrimSpace(string(raw)) {
				if err := os.WriteFile(p, []byte(out+"\n"), 0o644); err != nil {
					fail(1, map[string]any{"error": err.Error()})
				}
				cleaned++
			}
		}
		entries := scanBookDir(*dir, bm.Slug)
		_ = writeFull(bookDir, entries, true)
	}
	ok(map[string]any{"cleaned_files": cleaned})
}

func sortedBooks(m *Manifest) []BookManifest {
	out := append([]BookManifest(nil), m.Books...)
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}
