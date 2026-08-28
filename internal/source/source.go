// Package source implements the corpus-collection front end of kggraph:
// topic configuration, resource discovery, downloading, cleaning, manifest
// bookkeeping and quality verification for the reference corpus that feeds
// knowledge-graph ingestion.
package source

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ProviderWikisource   = "wikisource"
	ProviderCtext        = "ctext"
	ProviderYihaiguanyao = "yihaiguanyao"
)

// SearchHit is one candidate resource found by a provider.
type SearchHit struct {
	Provider string `json:"provider"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet,omitempty"`
}

// Run dispatches `kggraph source <subcommand>`.
func Run(argv []string) {
	if len(argv) < 1 {
		fail(2, map[string]any{"error": "missing source subcommand", "usage": subUsage()})
	}
	sub := argv[0]
	args := argv[1:]
	switch sub {
	case "init":
		runInit(args)
	case "add":
		runAdd(args)
	case "search":
		runSearch(args)
	case "fetch":
		runFetch(args)
	case "list":
		runList(args)
	case "verify":
		runVerify(args)
	case "verify-evidence":
		runVerifyEvidence(args)
	case "simplify":
		runSimplify(args)
	case "clean":
		runClean(args)
	case "-h", "--help", "help":
		ok(map[string]any{"usage": subUsage()})
	default:
		fail(2, map[string]any{"error": fmt.Sprintf("unknown source subcommand %q", sub), "usage": subUsage()})
	}
}

func subUsage() string {
	return "source subcommands: init, add, search, fetch, list, verify, verify-evidence, simplify, clean"
}

func fail(code int, payload map[string]any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
	os.Exit(code)
}

func ok(payload map[string]any) {
	fail(0, payload)
}

func mustParse(fs *flag.FlagSet, argv []string) {
	if err := fs.Parse(argv); err != nil {
		fail(2, map[string]any{"error": err.Error()})
	}
}

func defaultCorpusPath() string {
	if v := strings.TrimSpace(os.Getenv("KG_SOURCE_DIR")); v != "" {
		return v
	}
	return "gushu"
}

func corpusPath(dir string) string {
	if dir == "" {
		dir = defaultCorpusPath()
	}
	return filepath.Join(dir, "corpus.json")
}

func manifestPath(dir string) string {
	if dir == "" {
		dir = defaultCorpusPath()
	}
	return filepath.Join(dir, "manifest.json")
}
