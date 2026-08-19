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
