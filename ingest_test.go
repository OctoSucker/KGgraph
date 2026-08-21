package knowledgegraph

import (
	"context"
	"strings"
	"testing"
)

func TestValidateExtractedEdgeRejectsUnsupportedRelation(t *testing.T) {
	t.Parallel()
	_, err := validateExtractedEdge(extractedEdge{
		From:       "A",
		To:         "B",
		Relation:   "predicts_trade",
		Polarity:   1,
		Confidence: 0.7,
	}, DefaultIngestConfidence)
	if err == nil {
		t.Fatalf("expected unsupported relation to fail")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected not allowed error, got %v", err)
	}
}

func TestValidateExtractedEdgeUsesDefaultConfidenceForMissingValue(t *testing.T) {
	t.Parallel()
	edge, err := validateExtractedEdge(extractedEdge{
		From:     "A",
		To:       "B",
		Relation: "supports",
		Polarity: 1,
	}, 0.64)
	if err != nil {
		t.Fatalf("validate edge: %v", err)
	}
	if edge.Confidence != 0.64 {
		t.Fatalf("expected default confidence, got %v", edge.Confidence)
	}
}

func TestValidateExtractedNodeRejectsUnsupportedType(t *testing.T) {
	t.Parallel()
	_, err := validateExtractedNode(extractedNode{
		ID:       "A",
		NodeType: "trade_signal",
	})
	if err == nil {
		t.Fatalf("expected unsupported node type to fail")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected not allowed error, got %v", err)
	}
}

func TestIngestConfirmWritesFromPreview(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	preview := map[string]any{
		"statement": "memory layer supports grounded answers",
		"nodes": []any{
			map[string]any{"id": "memory layer", "node_type": "concept", "aliases": []any{"mem"}},
			map[string]any{"id": "grounded answers", "node_type": "concept"},
		},
		"edges": []any{
			map[string]any{"from": "memory layer", "to": "grounded answers", "relation": "supports", "polarity": 1, "confidence": 0.8},
		},
		"ingest_model": "test-model",
	}
	out, err := svc.Call(context.Background(), ToolIngestConfirm, map[string]any{
		"preview":     preview,
		"graph_kind":  "knowledge",
		"source_ref":  "doc-1",
		"source_type": "research",
	})
	if err != nil {
		t.Fatalf("ingest confirm: %v", err)
	}
	if out["edge_count"] != 1 || out["node_count"] != 2 {
		t.Fatalf("unexpected counts: %#v", out)
	}
	edges, err := svc.graph.AllEdges()
	if err != nil {
		t.Fatalf("list edges: %v", err)
	}
	if len(edges) != 1 || edges[0].SourceRef != "doc-1" || edges[0].SourceType != "research" {
		t.Fatalf("edge source metadata not applied: %#v", edges)
	}
}

func TestIngestConfirmBlockPolicyRejectsConflict(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "a",
		ToID:         "b",
		GraphKind:    "knowledge",
		RelationType: "supports",
		Polarity:     1,
		Confidence:   0.8,
	}); err != nil {
		t.Fatalf("pre-add edge: %v", err)
	}
	preview := map[string]any{
		"statement": "a contradicts b",
		"nodes":     []any{},
		"edges": []any{
			map[string]any{"from": "a", "to": "b", "relation": "contradicts", "polarity": -1, "confidence": 0.8},
		},
	}
	_, err = svc.Call(ctx, ToolIngestConfirm, map[string]any{
		"preview":         preview,
		"conflict_policy": "block",
	})
	if err == nil {
		t.Fatalf("expected block policy to reject conflicting preview")
	}
	if !strings.Contains(err.Error(), "conflict_policy=block") {
		t.Fatalf("expected block error, got %v", err)
	}
	edges, err := svc.graph.AllEdges()
	if err != nil {
		t.Fatalf("list edges: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("conflicting edge must not be written, got %d edges", len(edges))
	}
}

func TestIngestConfirmRejectsInvalidPreview(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	preview := map[string]any{
		"statement": "bad relation",
		"nodes":     []any{},
		"edges": []any{
			map[string]any{"from": "a", "to": "b", "relation": "predicts_trade", "polarity": 1, "confidence": 0.8},
		},
	}
	if _, err := svc.Call(context.Background(), ToolIngestConfirm, map[string]any{
		"preview": preview,
	}); err == nil {
		t.Fatalf("expected invalid preview relation to fail")
	}
}

func TestIngestPreviewRequiresEmbedder(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolIngestPreview, map[string]any{
		"statement": "anything",
	})
	if err == nil {
		t.Fatalf("expected preview to fail without embedder")
	}
	if !strings.Contains(err.Error(), "embedder") {
		t.Fatalf("expected embedder error, got %v", err)
	}
}

func TestParseIngestPreviewRoundTrip(t *testing.T) {
	t.Parallel()
	preview := map[string]any{
		"nodes": []any{
			map[string]any{"id": "Memory Layer", "node_type": "concept", "aliases": []any{"mem"}},
		},
		"edges": []any{
			map[string]any{"from": "Memory Layer", "to": "Grounded Answers", "relation": "supports", "polarity": 1, "confidence": 0.8},
		},
	}
	nodes, edges, err := parseIngestPreview(preview)
	if err != nil {
		t.Fatalf("parse preview: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "Memory Layer" || nodes[0].NodeType != "concept" {
		t.Fatalf("unexpected nodes: %#v", nodes)
	}
	if len(edges) != 1 || edges[0].FromID != "Memory Layer" || edges[0].RelationType != "supports" {
		t.Fatalf("unexpected edges: %#v", edges)
	}
}
