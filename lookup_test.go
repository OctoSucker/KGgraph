package knowledgegraph

import (
	"context"
	"testing"
)

type stubEmbedder struct {
	vecs map[string][]float32
}

func (e *stubEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if v, ok := e.vecs[text]; ok {
		return v, nil
	}
	return []float32{0, 0, 1}, nil
}

func TestLookupContextExactAndExpansionWithoutEmbedder(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "knowledge graph",
		ToID:         "multi-hop reasoning",
		GraphKind:    "knowledge",
		RelationType: "increases_probability_of",
		Polarity:     1,
		Confidence:   0.8,
	}); err != nil {
		t.Fatalf("upsert edge 1: %v", err)
	}
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "multi-hop reasoning",
		ToID:         "grounded answers",
		GraphKind:    "knowledge",
		RelationType: "supports",
		Polarity:     1,
		Confidence:   0.7,
	}); err != nil {
		t.Fatalf("upsert edge 2: %v", err)
	}
	out, err := svc.Call(ctx, ToolLookupContext, map[string]any{
		"term":       "knowledge graph",
		"graph_kind": "knowledge",
		"max_depth":  2,
	})
	if err != nil {
		t.Fatalf("lookup context: %v", err)
	}
	if out["matched"] != true {
		t.Fatalf("expected matched=true, got %#v", out["matched"])
	}
	resolved, ok := out["resolved"].([]map[string]any)
	if !ok || len(resolved) != 1 || resolved[0]["match_type"] != "exact" {
		t.Fatalf("expected exact resolved node, got %#v", out["resolved"])
	}
	nodes, ok := out["context_nodes"].([]map[string]any)
	if !ok {
		t.Fatalf("expected context_nodes, got %#v", out["context_nodes"])
	}
	found := map[string]bool{}
	for _, n := range nodes {
		id, _ := n["node_id"].(string)
		found[id] = true
	}
	if !found["multi-hop reasoning"] || !found["grounded answers"] {
		t.Fatalf("expected multi-hop expansion, got %#v", found)
	}
	edges, ok := out["edges"].(map[string]any)
	if !ok || len(edges) != 2 {
		t.Fatalf("expected 2 edges in context, got %#v", out["edges"])
	}
}

func TestLookupContextSemanticMatch(t *testing.T) {
	t.Parallel()
	embedder := &stubEmbedder{vecs: map[string][]float32{
		"graph memory":     {0.9, 0.1, 0},
		"graph memry":      {0.9, 0.1, 0},
		"grounded answers": {0, 1, 0},
	}}
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, embedder)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "graph memory",
		ToID:         "grounded answers",
		GraphKind:    "knowledge",
		RelationType: "supports",
		Polarity:     1,
		Confidence:   0.8,
	}); err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	out, err := svc.Call(ctx, ToolLookupContext, map[string]any{
		"term": "graph memry",
	})
	if err != nil {
		t.Fatalf("lookup context: %v", err)
	}
	if out["matched"] != true {
		t.Fatalf("expected semantic match, got %#v", out["matched"])
	}
	resolved, ok := out["resolved"].([]map[string]any)
	if !ok || len(resolved) != 1 {
		t.Fatalf("expected 1 resolved node, got %#v", out["resolved"])
	}
	if resolved[0]["node_id"] != "graph memory" || resolved[0]["match_type"] != "semantic" {
		t.Fatalf("expected semantic resolution to graph memory, got %#v", resolved[0])
	}
}

func TestLookupContextNoMatchReturnsEmptyWithoutError(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	out, err := svc.Call(context.Background(), ToolLookupContext, map[string]any{
		"term": "nothing here",
	})
	if err != nil {
		t.Fatalf("lookup context should not error on no match: %v", err)
	}
	if out["matched"] != false {
		t.Fatalf("expected matched=false, got %#v", out["matched"])
	}
}

func TestLookupContextIncludesEdgeEvidence(t *testing.T) {
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
		SourceRef:    "doc-1",
	})
	if err != nil {
		t.Fatalf("upsert edge: %v", err)
	}
	if err := svc.graph.AttachEdgeEvidence(edgeID, "research", "doc-1", "source snippet", "", true, 1.0, nil); err != nil {
		t.Fatalf("attach evidence: %v", err)
	}
	out, err := svc.Call(ctx, ToolLookupContext, map[string]any{"term": "a"})
	if err != nil {
		t.Fatalf("lookup context: %v", err)
	}
	edges, ok := out["edges"].(map[string]any)
	if !ok || len(edges) != 1 {
		t.Fatalf("expected 1 edge in context, got %#v", out["edges"])
	}
	var detail map[string]any
	for _, v := range edges {
		detail, _ = v.(map[string]any)
	}
	if detail == nil {
		t.Fatalf("expected edge detail, got %#v", edges)
	}
	if detail["source_ref"] != "doc-1" {
		t.Fatalf("expected source_ref in edge detail, got %#v", detail)
	}
	evidence, ok := detail["evidence"].([]map[string]any)
	if !ok || len(evidence) != 1 || evidence[0]["snippet"] != "source snippet" {
		t.Fatalf("expected evidence in edge detail, got %#v", detail["evidence"])
	}
}

func TestGraphEmbeddingCachePopulatedOnUpsert(t *testing.T) {
	t.Parallel()
	embedder := &stubEmbedder{vecs: map[string][]float32{
		"memory layer": {1, 0, 0},
		"memory layr":  {1, 0, 0},
	}}
	store := mustOpenTestStore(t)
	defer store.Close()
	g, err := NewGraph(store, embedder)
	if err != nil {
		t.Fatalf("new graph: %v", err)
	}
	ctx := context.Background()
	if err := g.UpsertNode(ctx, NodeUpsert{ID: "Memory Layer", NodeType: "concept", Status: "active"}); err != nil {
		t.Fatalf("upsert node: %v", err)
	}
	g.embedMu.RLock()
	_, ok := g.embedCache["memory layer"]
	g.embedMu.RUnlock()
	if !ok {
		t.Fatalf("expected embedding cache entry after upsert")
	}
	canon, ok, err := g.CanonicalForContext(ctx, "memory layr")
	if err != nil {
		t.Fatalf("semantic lookup: %v", err)
	}
	if !ok || canon != "memory layer" {
		t.Fatalf("expected semantic match to memory layer, got %q ok=%v", canon, ok)
	}
}

func TestLookupContextIncludesIncomingEdges(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "阶段高低点同步抬升",
		ToID:         "上升通道",
		GraphKind:    "knowledge",
		RelationType: "signals",
		Polarity:     1,
		Confidence:   0.8,
	}); err != nil {
		t.Fatalf("upsert incoming edge: %v", err)
	}
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "上升通道",
		ToID:         "择机做多",
		GraphKind:    "knowledge",
		RelationType: "enables",
		Polarity:     1,
		Confidence:   0.8,
	}); err != nil {
		t.Fatalf("upsert outgoing edge: %v", err)
	}
	if _, err := svc.graph.UpsertEdge(ctx, EdgeUpsert{
		FromID:       "放量跌破通道下沿",
		ToID:         "上升通道",
		GraphKind:    "knowledge",
		RelationType: "invalidates",
		Polarity:     -1,
		Confidence:   0.9,
	}); err != nil {
		t.Fatalf("upsert invalidation edge: %v", err)
	}
	out, err := svc.Call(ctx, ToolLookupContext, map[string]any{
		"term": "上升通道", "graph_kind": "knowledge", "max_depth": 2,
	})
	if err != nil {
		t.Fatalf("lookup context: %v", err)
	}
	edges, ok := out["edges"].(map[string]any)
	if !ok || len(edges) != 3 {
		t.Fatalf("expected 3 edges in context, got %#v", out["edges"])
	}
	incoming, ok := out["incoming_nodes"].([]string)
	if !ok {
		t.Fatalf("expected incoming_nodes list, got %#v", out["incoming_nodes"])
	}
	wantIncoming := map[string]bool{"阶段高低点同步抬升": true, "放量跌破通道下沿": true}
	if len(incoming) != 2 {
		t.Fatalf("expected 2 incoming source nodes, got %#v", incoming)
	}
	for _, id := range incoming {
		if !wantIncoming[id] {
			t.Fatalf("unexpected incoming node %q", id)
		}
	}
	directions := map[string]int{}
	for _, v := range edges {
		m, _ := v.(map[string]any)
		if dir, ok := m["direction"].(string); ok {
			directions[dir]++
		}
	}
	if directions["in"] != 2 || directions["out"] != 1 {
		t.Fatalf("expected 2 in / 1 out edges, got %#v", directions)
	}
}
