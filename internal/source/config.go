package source

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Book describes one corpus item and how to fetch it.
type Book struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Provider string `json:"provider"` // wikisource | ctext | yihaiguanyao

	// wikisource
	Page   string `json:"page,omitempty"`
	Prefix string `json:"prefix,omitempty"`

	// ctext
	ResID    string         `json:"res_id,omitempty"`
	Chapters []CtextChapter `json:"chapters,omitempty"`

	// yihaiguanyao
	SiteBookID      string `json:"site_book_id,omitempty"`
	SkipTranslation bool   `json:"skip_translation,omitempty"`
	ChapterPattern  string `json:"chapter_pattern,omitempty"`
	MaxPages        int    `json:"max_pages,omitempty"`
}

// CtextChapter is a chapter id + label pair for the ctext provider.
type CtextChapter struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Corpus is the topic-scoped configuration file (corpus.json).
type Corpus struct {
	Topic     string `json:"topic"`
	OutputDir string `json:"output_dir"`
	Books     []Book `json:"books"`
}

func loadCorpus(path string) (*Corpus, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Corpus
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &c, nil
}

func saveCorpus(path string, c *Corpus) error {
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func runInit(argv []string) {
	fs := flag.NewFlagSet("source init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", defaultCorpusPath(), "corpus directory (default gushu)")
	topic := fs.String("topic", "", "topic direction, e.g. 周易命理")
	force := fs.Bool("force", false, "overwrite existing corpus.json")
	mustParse(fs, argv)

	path := corpusPath(*dir)
	if _, err := os.Stat(path); err == nil && !*force {
		fail(2, map[string]any{"error": fmt.Sprintf("%s already exists (use --force to overwrite)", path)})
	}
	c := &Corpus{
		Topic:     *topic,
		OutputDir: *dir,
		Books:     []Book{},
	}
	if err := saveCorpus(path, c); err != nil {
		fail(1, map[string]any{"error": err.Error()})
	}
	ok(map[string]any{
		"corpus": path,
		"topic":  c.Topic,
		"books":  len(c.Books),
		"next":   "kggraph source add --id <book-id> --title <title> --provider wikisource --page <page>",
	})
}

func runAdd(argv []string) {
	fs := flag.NewFlagSet("source add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", defaultCorpusPath(), "corpus directory")
	var (
		id, title, provider, page, prefix, resID, siteBookID string
		chapterPattern                                       string
		skipTranslation                                      bool
	)
	fs.StringVar(&id, "id", "", "book id (slug)")
	fs.StringVar(&title, "title", "", "book title")
	fs.StringVar(&provider, "provider", "", "wikisource | ctext | yihaiguanyao")
	fs.StringVar(&page, "page", "", "wikisource page title")
	fs.StringVar(&prefix, "prefix", "", "wikisource subpage prefix (default <page>/)")
	fs.StringVar(&resID, "res-id", "", "ctext res id")
	fs.StringVar(&siteBookID, "site-book-id", "", "yihaiguanyao site book id")
	fs.StringVar(&chapterPattern, "chapter-pattern", "", "yihaiguanyao chapter heading regex")
	fs.BoolVar(&skipTranslation, "skip-translation", false, "yihaiguanyao: drop 小雅译 translation paragraphs")
	mustParse(fs, argv)

	if id == "" || title == "" || provider == "" {
		fail(2, map[string]any{"error": "--id, --title and --provider are required"})
	}
	switch provider {
	case ProviderWikisource:
		if page == "" {
			fail(2, map[string]any{"error": "--page is required for wikisource"})
		}
	case ProviderCtext:
		if resID == "" {
			fail(2, map[string]any{"error": "--res-id is required for ctext"})
		}
	case ProviderYihaiguanyao:
		if siteBookID == "" {
			fail(2, map[string]any{"error": "--site-book-id is required for yihaiguanyao"})
		}
	default:
		fail(2, map[string]any{"error": fmt.Sprintf("unknown provider %q", provider)})
	}

	path := corpusPath(*dir)
	c, err := loadCorpus(path)
	if err != nil {
		fail(1, map[string]any{"error": fmt.Sprintf("load corpus: %v (run `kggraph source init` first)", err)})
	}
	b := Book{
		ID:              id,
		Title:           title,
		Provider:        provider,
		Page:            page,
		Prefix:          prefix,
		ResID:           resID,
		SiteBookID:      siteBookID,
		SkipTranslation: skipTranslation,
		ChapterPattern:  chapterPattern,
	}
	if b.Prefix == "" && b.Page != "" {
		b.Prefix = strings.TrimRight(b.Page, "/") + "/"
	}
	for i := range c.Books {
		if c.Books[i].ID == id {
			c.Books[i] = b
			if err := saveCorpus(path, c); err != nil {
				fail(1, map[string]any{"error": err.Error()})
			}
			ok(map[string]any{"added": id, "updated": true})
			return
		}
	}
	c.Books = append(c.Books, b)
	if err := saveCorpus(path, c); err != nil {
		fail(1, map[string]any{"error": err.Error()})
	}
	ok(map[string]any{"added": id, "updated": false})
}
