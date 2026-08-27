package source

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ChapterEntry is one saved text file of a book.
type ChapterEntry struct {
	File  string `json:"file"`
	Title string `json:"title,omitempty"`
	Chars int    `json:"chars"`
}

// BookManifest is the per-book manifest entry.
type BookManifest struct {
	Slug       string         `json:"slug"`
	Title      string         `json:"title"`
	Provider   string         `json:"provider"`
	Source     string         `json:"source"`
	Chapters   []ChapterEntry `json:"chapters"`
	TotalChars int            `json:"total_chars"`
	FetchedAt  string         `json:"fetched_at,omitempty"`
}

type Manifest struct {
	Books []BookManifest `json:"books"`
}

func loadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{Books: []BookManifest{}}, nil
		}
		return nil, err
	}
	// legacy: a bare array of book entries
	var legacy []BookManifest
	if err := json.Unmarshal(raw, &legacy); err == nil && legacy != nil {
		return &Manifest{Books: legacy}, nil
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if m.Books == nil {
		m.Books = []BookManifest{}
	}
	return &m, nil
}

func saveManifest(path string, m *Manifest) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func upsertManifest(path string, entry BookManifest) error {
	m, err := loadManifest(path)
	if err != nil {
		return err
	}
	entry.FetchedAt = time.Now().Format(time.RFC3339)
	replaced := false
	for i := range m.Books {
		if m.Books[i].Slug == entry.Slug {
			m.Books[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		m.Books = append(m.Books, entry)
	}
	return saveManifest(path, m)
}

func manifestEntryFromDisk(dir, slug string) (*BookManifest, bool) {
	entries := scanBookDir(dir, slug)
	if len(entries) == 0 {
		return nil, false
	}
	total := 0
	chs := make([]ChapterEntry, 0, len(entries))
	for _, e := range entries {
		total += e.Chars
		chs = append(chs, e)
	}
	return &BookManifest{
		Slug:       slug,
		Title:      slug,
		Chapters:   chs,
		TotalChars: total,
	}, true
}
