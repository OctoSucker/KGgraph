package knowledgegraph

import (
	"context"
	"testing"
)

func TestDecisionEvaluationIncludesSourceRefs(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.Call(ctx, ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
		"source_ref":         "research-note-2026-08-18",
	}); err != nil {
		t.Fatalf("record decision: %v", err)
	}
	out, err := svc.Call(ctx, ToolEvaluateDecision, map[string]any{
		"market": "Oil market",
		"thesis": "Escalation risk is underpriced",
	})
	if err != nil {
		t.Fatalf("evaluate decision: %v", err)
	}
	evidence, ok := out["supporting_evidence"].([]map[string]any)
	if !ok || len(evidence) == 0 {
		t.Fatalf("expected supporting evidence, got %#v", out["supporting_evidence"])
	}
	if ref, ok := evidence[0]["source_ref"].(string); !ok || ref != "research-note-2026-08-18" {
		t.Fatalf("expected source_ref on evidence item, got %#v", evidence[0])
	}
	conflicts, err := svc.Call(ctx, ToolConflictScan, map[string]any{
		"from_id":       "thesis:escalation risk is underpriced",
		"to_id":         "action:buy",
		"relation_type": "supports_action",
		"polarity":      -1,
		"graph_kind":    DecisionGraphKind,
	})
	if err != nil {
		t.Fatalf("conflict scan over decision graph: %v", err)
	}
	if count := conflicts["conflict_count"]; count != 1 {
		t.Fatalf("expected decision graph conflict on reversed polarity, got %#v", conflicts)
	}
}

func TestDecisionAuditIncludesTimelineEvidenceAndReviews(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.Call(ctx, ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
		"source_ref":         "research-note-2026-08-19",
	}); err != nil {
		t.Fatalf("record decision: %v", err)
	}
	eval, err := svc.Call(ctx, ToolEvaluateDecision, map[string]any{
		"market": "Oil market",
		"thesis": "Escalation risk is underpriced",
	})
	if err != nil {
		t.Fatalf("evaluate decision: %v", err)
	}
	evidence, ok := eval["supporting_evidence"].([]map[string]any)
	if !ok || len(evidence) == 0 {
		t.Fatalf("expected supporting evidence, got %#v", eval["supporting_evidence"])
	}
	edgeID, ok := evidence[0]["edge_id"].(int64)
	if !ok {
		t.Fatalf("expected int64 edge_id, got %#v", evidence[0]["edge_id"])
	}
	if _, err := svc.Call(ctx, ToolAttachEdgeEvidence, map[string]any{
		"edge_id":     edgeID,
		"supports":    true,
		"source_type": "research",
		"source_ref":  "shipping-report-2026-08",
		"snippet":     "insurance premiums rose for the third week",
	}); err != nil {
		t.Fatalf("attach edge evidence: %v", err)
	}
	if _, err := svc.Call(ctx, ToolReviewDecision, map[string]any{
		"market":          "Oil market",
		"thesis":          "Escalation risk is underpriced",
		"outcome":         "incorrect",
		"realized_result": "ceasefire confirmed, oil sold off",
		"lessons":         []any{"do not ignore ceasefire verification path"},
	}); err != nil {
		t.Fatalf("review decision: %v", err)
	}

	audit, err := svc.Call(ctx, ToolDecisionAudit, map[string]any{
		"market": "Oil market",
		"thesis": "Escalation risk is underpriced",
	})
	if err != nil {
		t.Fatalf("decision audit: %v", err)
	}
	if audit["verdict"] != DecisionVerdictInvalidated {
		t.Fatalf("expected invalidated verdict after negative review, got %#v", audit["verdict"])
	}
	timeline, ok := audit["timeline"].([]map[string]any)
	if !ok || len(timeline) == 0 {
		t.Fatalf("expected non-empty audit timeline, got %#v", audit["timeline"])
	}
	foundEvidence := false
	for _, item := range timeline {
		itemID, _ := item["edge_id"].(int64)
		if itemID != edgeID {
			continue
		}
		items, ok := item["evidence"].([]map[string]any)
		if !ok || len(items) == 0 {
			t.Fatalf("expected attached evidence in audit timeline for edge %d", edgeID)
		}
		if items[0]["source_ref"] != "shipping-report-2026-08" {
			t.Fatalf("expected evidence source_ref, got %#v", items[0])
		}
		foundEvidence = true
	}
	if !foundEvidence {
		t.Fatalf("audit timeline missing edge %d", edgeID)
	}
	reviews, ok := audit["reviews"].([]map[string]any)
	if !ok || len(reviews) != 1 {
		t.Fatalf("expected 1 review in audit, got %#v", audit["reviews"])
	}
	if reviews[0]["outcome"] != "incorrect" {
		t.Fatalf("expected review outcome incorrect, got %#v", reviews[0])
	}
}

func TestListEvidenceToolReturnsAttachedEvidence(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	edgeID, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "a",
		ToID:         "b",
		GraphKind:    "knowledge",
		RelationType: "supports",
		Polarity:     1,
		Confidence:   0.8,
	})
	if err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	if err := svc.graph.AttachEdgeEvidence(edgeID, "research", "doc-1", "snippet", "", true, 1.0, nil); err != nil {
		t.Fatalf("attach evidence: %v", err)
	}
	out, err := svc.Call(ctx, ToolListEvidence, map[string]any{})
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	evidence, ok := out["evidence"].([]map[string]any)
	if !ok || len(evidence) != 1 {
		t.Fatalf("expected 1 evidence row, got %#v", out["evidence"])
	}
	if evidence[0]["edge_id"] != edgeID || evidence[0]["source_ref"] != "doc-1" {
		t.Fatalf("unexpected evidence row: %#v", evidence[0])
	}
}
