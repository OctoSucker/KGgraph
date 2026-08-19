package knowledgegraph

import (
	"context"
	"testing"
)

func TestExportImportGraphRoundTrip(t *testing.T) {
	t.Parallel()
	src := mustOpenTestStore(t)
	defer src.Close()
	svc1, err := NewService(src, nil)
	if err != nil {
		t.Fatalf("new source service: %v", err)
	}
	ctx := context.Background()
	if err := svc1.graph.UpsertNode(ctx, NodeUpsert{ID: "research note", NodeType: "document", Aliases: []string{"note-1"}, Status: "active"}); err != nil {
		t.Fatalf("upsert node: %v", err)
	}
	edgeID, err := svc1.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "research note",
		ToID:         "grounded answers",
		GraphKind:    "knowledge",
		RelationType: "supports",
		Polarity:     1,
		Confidence:   0.8,
		SourceType:   "research",
		SourceRef:    "doc-2026-08",
	})
	if err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	if err := svc1.graph.AttachEdgeEvidence(edgeID, "research", "doc-2026-08", "evidence snippet", true, 1.0, nil); err != nil {
		t.Fatalf("attach evidence: %v", err)
	}
	export, err := svc1.Call(ctx, ToolExportGraph, map[string]any{})
	if err != nil {
		t.Fatalf("export graph: %v", err)
	}

	dst := mustOpenTestStore(t)
	defer dst.Close()
	svc2, err := NewService(dst, nil)
	if err != nil {
		t.Fatalf("new destination service: %v", err)
	}
	imported, err := svc2.Call(ctx, ToolImportGraph, map[string]any{"graph": export})
	if err != nil {
		t.Fatalf("import graph: %v", err)
	}
	if imported["node_count"] != 2 || imported["edge_count"] != 1 || imported["evidence_count"] != 1 {
		t.Fatalf("unexpected import counts: %#v", imported)
	}
	if imported["skipped_evidence"] != 0 {
		t.Fatalf("expected no skipped evidence, got %#v", imported["skipped_evidence"])
	}
	nodes, err := svc2.graph.AllNodes()
	if err != nil {
		t.Fatalf("list imported nodes: %v", err)
	}
	foundDocument := false
	for _, n := range nodes {
		if n.ID == "research note" {
			foundDocument = true
			if n.NodeType != "document" || n.AliasesJSON != `["note-1"]` {
				t.Fatalf("node metadata not preserved: %#v", n)
			}
		}
	}
	if !foundDocument {
		t.Fatalf("imported graph missing research note node")
	}
	edges, err := svc2.graph.AllEdges()
	if err != nil {
		t.Fatalf("list imported edges: %v", err)
	}
	if len(edges) != 1 || edges[0].SourceRef != "doc-2026-08" || edges[0].EvidenceCount != 1 {
		t.Fatalf("edge metadata or evidence count not preserved: %#v", edges)
	}
	evidence, err := dst.EdgeEvidenceList(edges[0].ID)
	if err != nil {
		t.Fatalf("list imported evidence: %v", err)
	}
	if len(evidence) != 1 || evidence[0].Snippet != "evidence snippet" || !evidence[0].Supports {
		t.Fatalf("evidence not preserved: %#v", evidence)
	}
}

func TestImportGraphSkipsEvidenceWithoutEdgeMapping(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	payload := map[string]any{
		"nodes": []any{},
		"edges": []any{},
		"evidence": []any{
			map[string]any{
				"edge_id":     999,
				"source_type": "research",
				"source_ref":  "doc-x",
				"snippet":     "orphaned",
				"supports":    true,
				"weight":      1.0,
			},
		},
	}
	out, err := svc.Call(context.Background(), ToolImportGraph, map[string]any{"graph": payload})
	if err != nil {
		t.Fatalf("import graph: %v", err)
	}
	if out["skipped_evidence"] != 1 || out["evidence_count"] != 0 {
		t.Fatalf("expected orphaned evidence skipped, got %#v", out)
	}
}
