package knowledgegraph

import (
	"context"
	"testing"
	"time"
)

func conflictOut(t *testing.T, svc *Service, args map[string]any) map[string]any {
	t.Helper()
	out, err := svc.Call(context.Background(), ToolConflictScan, args)
	if err != nil {
		t.Fatalf("conflict scan: %v", err)
	}
	return out
}

func conflictCount(t *testing.T, out map[string]any) int {
	t.Helper()
	count, ok := out["conflict_count"].(int)
	if !ok {
		t.Fatalf("conflict_count has unexpected type: %#v", out["conflict_count"])
	}
	return count
}

func firstConflictReason(t *testing.T, out map[string]any) string {
	t.Helper()
	conflicts, ok := out["conflicts"].([]map[string]any)
	if !ok || len(conflicts) == 0 {
		t.Fatalf("expected conflicts, got %#v", out["conflicts"])
	}
	reason, ok := conflicts[0]["reason"].(string)
	if !ok {
		t.Fatalf("reason has unexpected type: %#v", conflicts[0]["reason"])
	}
	return reason
}

func TestConflictScanFindsOppositePolarity(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "war escalation",
		ToID:         "oil price up",
		GraphKind:    "knowledge",
		RelationType: "increases_probability_of",
		Polarity:     1,
		Confidence:   0.8,
	}); err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	out := conflictOut(t, svc, map[string]any{
		"from_id":       "war escalation",
		"to_id":         "oil price up",
		"relation_type": "increases_probability_of",
		"polarity":      -1,
	})
	if conflictCount(t, out) != 1 {
		t.Fatalf("expected 1 conflict, got %#v", out)
	}
	if reason := firstConflictReason(t, out); reason != "opposite_polarity" {
		t.Fatalf("expected opposite_polarity, got %q", reason)
	}
}

func TestConflictScanFindsAntonymRelation(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "new memory layer",
		ToID:         "hallucination rate down",
		GraphKind:    "knowledge",
		RelationType: "supports",
		Polarity:     1,
		Confidence:   0.7,
	}); err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	out := conflictOut(t, svc, map[string]any{
		"from_id":       "new memory layer",
		"to_id":         "hallucination rate down",
		"relation_type": "contradicts",
		"polarity":      -1,
	})
	if conflictCount(t, out) != 1 {
		t.Fatalf("expected 1 conflict, got %#v", out)
	}
	if reason := firstConflictReason(t, out); reason != "antonym_relation" {
		t.Fatalf("expected antonym_relation, got %q", reason)
	}
}

func TestConflictScanFindsIncreasesDecreasesAntonym(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "利率上升",
		ToID:         "债券价格",
		GraphKind:    "knowledge",
		RelationType: "decreases",
		Polarity:     1,
		Confidence:   0.9,
	}); err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	out := conflictOut(t, svc, map[string]any{
		"from_id":       "利率上升",
		"to_id":         "债券价格",
		"relation_type": "increases",
		"polarity":      1,
	})
	if conflictCount(t, out) != 1 {
		t.Fatalf("expected 1 conflict, got %#v", out)
	}
	if reason := firstConflictReason(t, out); reason != "antonym_relation" {
		t.Fatalf("expected antonym_relation, got %q", reason)
	}
}

func TestConflictScanFindsReverseContradictsEdge(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "oil price up",
		ToID:         "war escalation",
		GraphKind:    "knowledge",
		RelationType: "contradicts",
		Polarity:     -1,
		Confidence:   0.8,
	}); err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	out := conflictOut(t, svc, map[string]any{
		"from_id":       "war escalation",
		"to_id":         "oil price up",
		"relation_type": "increases_probability_of",
		"polarity":      1,
	})
	if conflictCount(t, out) != 1 {
		t.Fatalf("expected 1 conflict, got %#v", out)
	}
	if reason := firstConflictReason(t, out); reason != "reverse_contradicts_edge" {
		t.Fatalf("expected reverse_contradicts_edge, got %q", reason)
	}
}

func TestConflictScanIgnoresMatchingAndInactiveEdges(t *testing.T) {
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
	same := conflictOut(t, svc, map[string]any{
		"from_id":       "a",
		"to_id":         "b",
		"relation_type": "supports",
		"polarity":      1,
	})
	if conflictCount(t, same) != 0 {
		t.Fatalf("matching edge should not be a conflict: %#v", same)
	}
	if _, err := svc.graph.RetireEdge(edgeID, ctxTime()); err != nil {
		t.Fatalf("retire edge: %v", err)
	}
	opposite := conflictOut(t, svc, map[string]any{
		"from_id":       "a",
		"to_id":         "b",
		"relation_type": "supports",
		"polarity":      -1,
	})
	if conflictCount(t, opposite) != 0 {
		t.Fatalf("inactive edge should not be a conflict: %#v", opposite)
	}
}

func TestConflictScanFiltersByGraphKind(t *testing.T) {
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
		t.Fatalf("upsert edge: %v", err)
	}
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "a",
		ToID:         "b",
		GraphKind:    "skill",
		RelationType: "supports",
		Polarity:     1,
		Confidence:   0.8,
	}); err != nil {
		t.Fatalf("upsert skill edge: %v", err)
	}
	out := conflictOut(t, svc, map[string]any{
		"from_id":       "a",
		"to_id":         "b",
		"relation_type": "supports",
		"polarity":      -1,
		"graph_kind":    "knowledge",
	})
	if conflictCount(t, out) != 1 {
		t.Fatalf("expected 1 conflict in knowledge kind, got %#v", out)
	}
	out = conflictOut(t, svc, map[string]any{
		"from_id":       "a",
		"to_id":         "b",
		"relation_type": "supports",
		"polarity":      -1,
		"graph_kind":    "skill",
	})
	if conflictCount(t, out) != 1 {
		t.Fatalf("expected 1 conflict in skill kind, got %#v", out)
	}
}

func ctxTime() time.Time {
	return time.Now().UTC().Add(-time.Hour)
}
