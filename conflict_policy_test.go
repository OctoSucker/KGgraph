package knowledgegraph

import (
	"context"
	"strings"
	"testing"
)

func TestAddFactEdgeBlockPolicyRejectsConflict(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.Call(ctx, ToolAddFactEdge, map[string]any{
		"from_id":       "a",
		"to_id":         "b",
		"relation_type": "supports",
		"polarity":      1,
		"confidence":    0.8,
	}); err != nil {
		t.Fatalf("add first edge: %v", err)
	}
	_, err = svc.Call(ctx, ToolAddFactEdge, map[string]any{
		"from_id":         "a",
		"to_id":           "b",
		"relation_type":   "contradicts",
		"polarity":        -1,
		"confidence":      0.8,
		"conflict_policy": "block",
	})
	if err == nil {
		t.Fatalf("expected block policy to reject conflicting edge")
	}
	if !strings.Contains(err.Error(), "conflict_policy=block") {
		t.Fatalf("expected block error, got %v", err)
	}
}

func TestAddFactEdgeWarnPolicyWritesAndReportsConflicts(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.Call(ctx, ToolAddFactEdge, map[string]any{
		"from_id":       "a",
		"to_id":         "b",
		"relation_type": "supports",
		"polarity":      1,
		"confidence":    0.8,
	}); err != nil {
		t.Fatalf("add first edge: %v", err)
	}
	out, err := svc.Call(ctx, ToolAddFactEdge, map[string]any{
		"from_id":         "a",
		"to_id":           "b",
		"relation_type":   "contradicts",
		"polarity":        -1,
		"confidence":      0.8,
		"conflict_policy": "warn",
	})
	if err != nil {
		t.Fatalf("warn policy should allow write: %v", err)
	}
	conflicts, ok := out["conflicts"].([]map[string]any)
	if !ok || len(conflicts) != 1 {
		t.Fatalf("expected 1 reported conflict, got %#v", out["conflicts"])
	}
	edges, err := store.EdgesSelectAll()
	if err != nil {
		t.Fatalf("list edges: %v", err)
	}
	if len(edges) != 2 {
		t.Fatalf("expected both edges written, got %d", len(edges))
	}
}

func TestAddFactEdgeOffPolicySkipsConflictScan(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.Call(ctx, ToolAddFactEdge, map[string]any{
		"from_id":       "a",
		"to_id":         "b",
		"relation_type": "supports",
		"polarity":      1,
		"confidence":    0.8,
	}); err != nil {
		t.Fatalf("add first edge: %v", err)
	}
	out, err := svc.Call(ctx, ToolAddFactEdge, map[string]any{
		"from_id":         "a",
		"to_id":           "b",
		"relation_type":   "contradicts",
		"polarity":        -1,
		"confidence":      0.8,
		"conflict_policy": "off",
	})
	if err != nil {
		t.Fatalf("off policy should allow write: %v", err)
	}
	if _, ok := out["conflicts"]; ok {
		t.Fatalf("off policy must not report conflicts: %#v", out)
	}
}

func TestAddFactEdgeRejectsInvalidConflictPolicy(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolAddFactEdge, map[string]any{
		"from_id":         "a",
		"to_id":           "b",
		"relation_type":   "supports",
		"polarity":        1,
		"confidence":      0.8,
		"conflict_policy": "banana",
	})
	if err == nil {
		t.Fatalf("expected invalid conflict policy to fail")
	}
}
