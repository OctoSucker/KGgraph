package main

import (
	"github.com/OctoSucker/KGgraph/internal/source"
)

func runSource(argv []string) {
	// `source` is the corpus-collection front end; it does not need a graph
	// database, so it stays outside the common flags.
	source.Run(argv)
}
