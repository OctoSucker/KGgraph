package knowledgegraph

import (
	"context"
	"testing"
	"time"
)

func TestRetireEdgeRemovesEdgeFromReasoningAfterAsOf(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	edgeID, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "oil supply",
		ToID:         "oil price up",
		GraphKind:    "knowledge",
		RelationType: "increases_probability_of",
		Polarity:     1,
		Confidence:   0.8,
	})
	if err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	retiredAt := time.Now().UTC().Add(-time.Hour)
	out, err := svc.Call(ctx, ToolRetireEdge, map[string]any{
		"edge_id": edgeID,
		"as_of":   retiredAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("retire edge: %v", err)
	}
	if out["retired"] != true {
		t.Fatalf("expected retired=true, got %#v", out["retired"])
	}
	row, err := store.EdgeByID(edgeID)
	if err != nil {
		t.Fatalf("edge by id: %v", err)
	}
	if row.ValidUntil == nil || !row.ValidUntil.Truncate(time.Second).Equal(retiredAt.Truncate(time.Second)) {
		t.Fatalf("expected valid_until=%v, got %v", retiredAt, row.ValidUntil)
	}
	hits, err := svc.graph.ExpandReasoning("oil supply", "knowledge", time.Now().UTC(), 2, 5, 10, false, 0)
	if err != nil {
		t.Fatalf("expand reasoning: %v", err)
	}
	for _, h := range hits {
		if h.NodeID == "oil price up" {
			t.Fatalf("retired edge should not contribute to reasoning: %+v", hits)
		}
	}
}

func TestRetireEdgeClosesWindowEarlierThanScheduled(t *testing.T) {
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
	if _, err := svc.Call(ctx, ToolRetireEdge, map[string]any{
		"edge_id": edgeID,
		"as_of":   now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("retire edge: %v", err)
	}
	row, err := store.EdgeByID(edgeID)
	if err != nil {
		t.Fatalf("edge by id: %v", err)
	}
	if row.ValidUntil == nil || !row.ValidUntil.Truncate(time.Second).Equal(now.Truncate(time.Second)) {
		t.Fatalf("expected valid_until closed at %v, got %v", now, row.ValidUntil)
	}
}

func TestRetireEdgeDoesNotExtendAlreadyExpiredEdge(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	expiredFrom := now.Add(-48 * time.Hour)
	expired := now.Add(-24 * time.Hour)
	edgeID, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "a",
		ToID:         "b",
		GraphKind:    "knowledge",
		RelationType: "causes",
		Polarity:     1,
		Confidence:   0.8,
		ValidFrom:    &expiredFrom,
		ValidUntil:   &expired,
	})
	if err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	if _, err := svc.Call(ctx, ToolRetireEdge, map[string]any{
		"edge_id": edgeID,
		"as_of":   now.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("retire edge: %v", err)
	}
	row, err := store.EdgeByID(edgeID)
	if err != nil {
		t.Fatalf("edge by id: %v", err)
	}
	if row.ValidUntil == nil || !row.ValidUntil.Truncate(time.Second).Equal(expired.Truncate(time.Second)) {
		t.Fatalf("expected valid_until unchanged at %v, got %v", expired, row.ValidUntil)
	}
}

func TestRetireEdgeRequiresPositiveID(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if _, err := svc.Call(context.Background(), ToolRetireEdge, map[string]any{
		"edge_id": 0,
	}); err == nil {
		t.Fatalf("expected retire edge to fail with edge_id 0")
	}
}
