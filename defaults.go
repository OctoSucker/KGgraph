package knowledgegraph

import "os"

const (
	// Version is the current KGgraph release version, reported by MCP and CLI.
	Version                 = "0.4.0"
	DefaultIngestModel      = "gpt-4o-mini"
	DefaultIngestConfidence = 0.65
)

func IngestModelFromEnv() string {
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		return v
	}
	return DefaultIngestModel
}
