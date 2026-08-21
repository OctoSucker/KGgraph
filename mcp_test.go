package knowledgegraph

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPServerEndToEnd(t *testing.T) {
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := newMCPServer(svc)
	ss, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer ss.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "kggraph-test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	toolsResult, err := cs.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range toolsResult.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{
		ToolRecordDecision, ToolStrictAsk, ToolConflictScan,
		ToolRetireEdge, ToolReopenEdge, ToolDecisionAudit,
		ToolLookupContext, ToolExportGraph, ToolImportGraph, ToolListEvidence,
	} {
		if !names[want] {
			t.Fatalf("missing tool %q in %v", want, names)
		}
	}

	mcpCall := func(name string, args map[string]any) map[string]any {
		t.Helper()
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("call %s: %v", name, err)
		}
		if res.IsError {
			t.Fatalf("tool %s returned error: %v", name, contentText(res))
		}
		m, err := structuredMap(res)
		if err != nil {
			t.Fatalf("tool %s structured content: %v", name, err)
		}
		return m
	}

	if out := mcpCall(ToolAddFactEdge, map[string]any{
		"from_id": "a", "to_id": "b", "relation_type": "supports",
		"polarity": 1, "confidence": 0.8,
	}); out["edge_id"] == nil {
		t.Fatalf("add fact edge missing edge_id: %#v", out)
	}
	conflicts := mcpCall(ToolConflictScan, map[string]any{
		"from_id": "a", "to_id": "b", "relation_type": "supports", "polarity": -1,
	})
	if int(mapFloat(conflicts, "conflict_count")) != 1 {
		t.Fatalf("expected conflict_count 1, got %#v", conflicts)
	}

	record := mcpCall(ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []string{"shipping insurance rising"},
		"counter_evidence":   []string{"front-month oil already bid"},
		"failure_conditions": []string{"confirmed ceasefire"},
	})
	if record["edge_count"] == nil {
		t.Fatalf("record decision missing edge_count: %#v", record)
	}
	strict := mcpCall(ToolStrictAsk, map[string]any{
		"market": "Oil market", "thesis": "Escalation risk is underpriced",
		"question": "现在是不是应该买？", "language": "zh",
	})
	if !strings.Contains(mapString(strict, "answer"), "跟随已记录动作") {
		t.Fatalf("unexpected strict answer: %#v", strict["answer"])
	}
	audit := mcpCall(ToolDecisionAudit, map[string]any{
		"market": "Oil market", "thesis": "Escalation risk is underpriced",
	})
	if mapInt64(audit, "edge_count") < 5 {
		t.Fatalf("expected decision audit timeline, got %#v", audit)
	}
	lookup := mcpCall(ToolLookupContext, map[string]any{
		"term": "a", "graph_kind": "knowledge", "max_depth": 2,
	})
	if lookup["matched"] != true {
		t.Fatalf("expected lookup-context match, got %#v", lookup)
	}
	export := mcpCall(ToolExportGraph, map[string]any{})
	if mapInt64(export, "node_count") < 3 || mapInt64(export, "edge_count") < 2 {
		t.Fatalf("unexpected export counts: %#v", export)
	}

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: ToolLookupNodeExact, Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("expected tool-level error result, got transport error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError for missing arguments")
	}
}

func contentText(res *mcp.CallToolResult) string {
	var parts []string
	for _, c := range res.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func structuredMap(res *mcp.CallToolResult) (map[string]any, error) {
	switch v := res.StructuredContent.(type) {
	case map[string]any:
		return v, nil
	case json.RawMessage:
		var m map[string]any
		if err := json.Unmarshal(v, &m); err != nil {
			return nil, err
		}
		return m, nil
	default:
		return nil, nil
	}
}
