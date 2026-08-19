package knowledgegraph

import (
	"context"
	"testing"
	"time"
)

func TestReopenEdgeExtendsValidityAndRestoresReasoning(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	createdAt := time.Now().UTC().Add(-72 * time.Hour)
	edgeID, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "oil supply",
		ToID:         "oil price up",
		GraphKind:    "knowledge",
		RelationType: "increases_probability_of",
		Polarity:     1,
		Confidence:   0.8,
		ObservedAt:   &createdAt,
	})
	if err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	retiredAt := createdAt.Add(24 * time.Hour)
	if _, err := svc.Call(ctx, ToolRetireEdge, map[string]any{
		"edge_id": edgeID,
		"as_of":   retiredAt.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("retire edge: %v", err)
	}
	queryBefore := createdAt.Add(48 * time.Hour)
	hitsBefore, err := svc.graph.ExpandReasoning("oil supply", "knowledge", queryBefore, 2, 5, 10, false, 0)
	if err != nil {
		t.Fatalf("expand before reopen: %v", err)
	}
	for _, h := range hitsBefore {
		if h.NodeID == "oil price up" {
			t.Fatalf("retired edge should not be reachable: %+v", hitsBefore)
		}
	}
	reopenedAt := createdAt.Add(96 * time.Hour)
	out, err := svc.Call(ctx, ToolReopenEdge, map[string]any{
		"edge_id": edgeID,
		"as_of":   reopenedAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("reopen edge: %v", err)
	}
	if out["open_ended"] != false {
		t.Fatalf("expected open_ended=false, got %#v", out["open_ended"])
	}
	hitsAfter, err := svc.graph.ExpandReasoning("oil supply", "knowledge", reopenedAt.Add(-time.Minute), 2, 5, 10, false, 0)
	if err != nil {
		t.Fatalf("expand after reopen: %v", err)
	}
	found := false
	for _, h := range hitsAfter {
		if h.NodeID == "oil price up" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("reopened edge should be reachable: %+v", hitsAfter)
	}
}

func TestReopenEdgeOpenEndedClearsValidity(t *testing.T) {
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
		RelationType: "causes",
		Polarity:     1,
		Confidence:   0.8,
	})
	if err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	if _, err := svc.Call(ctx, ToolRetireEdge, map[string]any{"edge_id": edgeID}); err != nil {
		t.Fatalf("retire edge: %v", err)
	}
	if _, err := svc.Call(ctx, ToolReopenEdge, map[string]any{
		"edge_id":    edgeID,
		"open_ended": true,
	}); err != nil {
		t.Fatalf("reopen edge open-ended: %v", err)
	}
	row, err := store.EdgeByID(edgeID)
	if err != nil {
		t.Fatalf("edge by id: %v", err)
	}
	if row.ValidUntil != nil {
		t.Fatalf("expected nil valid_until after open-ended reopen, got %v", row.ValidUntil)
	}
}

func TestReopenEdgeKeepsFutureValidity(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	later := now.Add(30 * 24 * time.Hour)
	edgeID, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "a",
		ToID:         "b",
		GraphKind:    "knowledge",
		RelationType: "causes",
		Polarity:     1,
		Confidence:   0.8,
		ValidUntil:   &later,
	})
	if err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	if _, err := svc.Call(ctx, ToolReopenEdge, map[string]any{
		"edge_id": edgeID,
		"as_of":   now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("reopen edge: %v", err)
	}
	row, err := store.EdgeByID(edgeID)
	if err != nil {
		t.Fatalf("edge by id: %v", err)
	}
	if row.ValidUntil == nil || !row.ValidUntil.Truncate(time.Second).Equal(later.Truncate(time.Second)) {
		t.Fatalf("expected valid_until unchanged at %v, got %v", later, row.ValidUntil)
	}
}
