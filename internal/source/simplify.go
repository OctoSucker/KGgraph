package source

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/liuzl/gocc"
	_ "modernc.org/sqlite"
)

var t2sOnce struct {
	c   *gocc.OpenCC
	err error
}

// SimplifyChinese converts traditional Chinese text to simplified Chinese.
// Used as a front-end cleaning step so corpus text and evidence are stored in
// one script consistently.
func SimplifyChinese(text string) (string, error) {
	if t2sOnce.c == nil && t2sOnce.err == nil {
		t2sOnce.c, t2sOnce.err = gocc.New("t2s")
	}
	if t2sOnce.err != nil {
		return "", t2sOnce.err
	}
	out, err := t2sOnce.c.Convert(text)
	if err != nil {
		return "", err
	}
	return out, nil
}

func runSimplify(argv []string) {
	fs := flag.NewFlagSet("source simplify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", defaultCorpusPath(), "corpus directory (convert all *.txt)")
	db := fs.String("db", "", "sqlite db path: convert evidence snippets in place")
	mustParse(fs, argv)

	changed := 0
	if *dir != "" {
		books, err := os.ReadDir(*dir)
		if err != nil {
			fail(1, map[string]any{"error": err.Error()})
		}
		for _, b := range books {
			if !b.IsDir() {
				continue
			}
			bookDir := filepath.Join(*dir, b.Name())
			files, err := os.ReadDir(bookDir)
			if err != nil {
				continue
			}
			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".txt") {
					continue
				}
				p := filepath.Join(bookDir, f.Name())
				raw, err := os.ReadFile(p)
				if err != nil {
					continue
				}
				sim, err := SimplifyChinese(string(raw))
				if err != nil {
					fail(1, map[string]any{"error": err.Error()})
				}
				if sim != string(raw) {
					if err := os.WriteFile(p, []byte(sim), 0o644); err != nil {
						fail(1, map[string]any{"error": err.Error()})
					}
					changed++
				}
			}
		}
	}
	if *db != "" {
		sqlDB, err := sql.Open("sqlite", "file:"+filepath.Clean(*db))
		if err != nil {
			fail(1, map[string]any{"error": err.Error()})
		}
		defer sqlDB.Close()
		rows, err := sqlDB.Query(`SELECT id, snippet FROM kg_edge_evidence`)
		if err != nil {
			fail(1, map[string]any{"error": err.Error()})
		}
		type row struct {
			id      int64
			snippet string
		}
		var rowsData []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.snippet); err != nil {
				fail(1, map[string]any{"error": err.Error()})
			}
			rowsData = append(rowsData, r)
		}
		rows.Close()
		for _, r := range rowsData {
			sim, err := SimplifyChinese(r.snippet)
			if err != nil {
				fail(1, map[string]any{"error": err.Error()})
			}
			if sim != r.snippet {
				if _, err := sqlDB.Exec(`UPDATE kg_edge_evidence SET snippet=? WHERE id=?`, sim, r.id); err != nil {
					fail(1, map[string]any{"error": err.Error()})
				}
				changed++
			}
		}
	}
	ok(map[string]any{"changed_files": changed,
		"note": "corpus text and evidence snippets now stored in simplified Chinese"})
}

var _ = fmt.Sprint
