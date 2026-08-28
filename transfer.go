package knowledgegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	ExportGraphMode          = "export_graph"
	ImportGraphMode          = "import_graph"
	GraphExportSchemaVersion = 1
)

// ExportGraph serializes the whole graph (nodes, edges, evidence) as a JSON
// payload that can be moved to another machine or backed up.
func (s *Service) ExportGraph(ctx context.Context) (map[string]any, error) {
	_ = ctx
	if s == nil || s.graph == nil {
		return nil, fmt.Errorf("knowledgegraph: export graph: service is nil")
	}
	nodes, err := s.graph.AllNodes()
	if err != nil {
		return nil, err
	}
	edges, err := s.graph.AllEdges()
	if err != nil {
		return nil, err
	}
	evidence, err := s.store.EvidenceSelectAll()
	if err != nil {
		return nil, err
	}
	nodeOut := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		nodeOut = append(nodeOut, map[string]any{
			"id":           n.ID,
			"node_type":    n.NodeType,
			"aliases_json": n.AliasesJSON,
			"domain_json":  n.DomainJSON,
			"status":       n.Status,
			"updated_at":   n.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	edgeOut := make([]map[string]any, 0, len(edges))
	for _, e := range edges {
		edgeOut = append(edgeOut, edgeRowToMap(e))
	}
	evidenceOut := make([]map[string]any, 0, len(evidence))
	for _, ev := range evidence {
		evidenceOut = append(evidenceOut, map[string]any{
			"id":          ev.ID,
			"edge_id":     ev.EdgeID,
			"source_type": ev.SourceType,
			"source_ref":  ev.SourceRef,
			"snippet":     ev.Snippet,
			"translation": ev.Translation,
			"observed_at": ev.ObservedAt.UTC().Format(time.RFC3339),
			"supports":    ev.Supports,
			"weight":      ev.Weight,
		})
	}
	return map[string]any{
		"mode":           ExportGraphMode,
		"schema_version": GraphExportSchemaVersion,
		"exported_at":    time.Now().UTC().Format(time.RFC3339),
		"node_count":     len(nodeOut),
		"edge_count":     len(edgeOut),
		"evidence_count": len(evidenceOut),
		"nodes":          nodeOut,
		"edges":          edgeOut,
		"evidence":       evidenceOut,
	}, nil
}

// ImportGraph loads a payload produced by ExportGraph (or an equivalent shape
// with nodes/edges/evidence arrays). Evidence is re-attached by matching edges
// on their identity fields, so the new database does not need the same ids.
func (s *Service) ImportGraph(ctx context.Context, payload map[string]any) (map[string]any, error) {
	_ = ctx
	if s == nil || s.graph == nil {
		return nil, fmt.Errorf("knowledgegraph: import graph: service is nil")
	}
	nodes, err := toMapSlice(payload["nodes"])
	if err != nil {
		return nil, fmt.Errorf("knowledgegraph: import graph: nodes: %w", err)
	}
	edges, err := toMapSlice(payload["edges"])
	if err != nil {
		return nil, fmt.Errorf("knowledgegraph: import graph: edges: %w", err)
	}
	evidence, err := toMapSlice(payload["evidence"])
	if err != nil {
		return nil, fmt.Errorf("knowledgegraph: import graph: evidence: %w", err)
	}

	nodeCount := 0
	for _, n := range nodes {
		id, _ := n["id"].(string)
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("knowledgegraph: import graph: node without id")
		}
		nodeType, _ := n["node_type"].(string)
		status, _ := n["status"].(string)
		var aliases []string
		if raw, ok := n["aliases_json"].(string); ok && strings.TrimSpace(raw) != "" && strings.TrimSpace(raw) != "[]" {
			if err := json.Unmarshal([]byte(raw), &aliases); err != nil {
				return nil, fmt.Errorf("knowledgegraph: import graph: node %q aliases_json: %w", id, err)
			}
		} else if rawAliases, ok := n["aliases"].([]any); ok {
			for _, a := range rawAliases {
				if s, ok := a.(string); ok && strings.TrimSpace(s) != "" {
					aliases = append(aliases, strings.TrimSpace(s))
				}
			}
		}
		var domain []string
		if raw, ok := n["domain_json"].(string); ok && strings.TrimSpace(raw) != "" && strings.TrimSpace(raw) != "[]" {
			if err := json.Unmarshal([]byte(raw), &domain); err != nil {
				return nil, fmt.Errorf("knowledgegraph: import graph: node %q domain_json: %w", id, err)
			}
		} else if rawDomain, ok := n["domain"].([]any); ok {
			for _, d := range rawDomain {
				if s, ok := d.(string); ok && strings.TrimSpace(s) != "" {
					domain = append(domain, strings.TrimSpace(s))
				}
			}
		}
		if err := s.graph.UpsertNode(ctx, NodeUpsert{ID: id, NodeType: nodeType, Aliases: aliases, Domain: domain, Status: status}); err != nil {
			return nil, err
		}
		nodeCount++
	}

	edgeIDMap := map[int64]int64{}
	edgeCount := 0
	for _, e := range edges {
		fromID, _ := e["from_id"].(string)
		toID, _ := e["to_id"].(string)
		relation, _ := e["relation_type"].(string)
		if fromID == "" || toID == "" || relation == "" {
			return nil, fmt.Errorf("knowledgegraph: import graph: edge requires from_id, to_id, relation_type")
		}
		graphKind, _ := e["graph_kind"].(string)
		if graphKind == "" {
			graphKind = "knowledge"
		}
		polarity := intFromAny(e["polarity"], 1)
		confidence := floatFromAny(e["confidence"], 0.6)
		conditionText, _ := e["condition_text"].(string)
		edgeKind, _ := e["edge_kind"].(string)
		conditionJSON, _ := e["condition_json"].(string)
		sourceType, _ := e["source_type"].(string)
		sourceRef, _ := e["source_ref"].(string)
		decay := intFromAny(e["decay_half_life_days"], 30)
		isExecutable := boolFromAny(e["is_executable"], false)
		activationRule, _ := e["activation_rule"].(string)
		observedAt, err := exportTime(e["observed_at"])
		if err != nil {
			return nil, fmt.Errorf("knowledgegraph: import graph: observed_at: %w", err)
		}
		validFrom, err := exportTime(e["valid_from"])
		if err != nil {
			return nil, fmt.Errorf("knowledgegraph: import graph: valid_from: %w", err)
		}
		validUntil, err := exportTime(e["valid_until"])
		if err != nil {
			return nil, fmt.Errorf("knowledgegraph: import graph: valid_until: %w", err)
		}
		expiresAt, err := exportTime(e["expires_at"])
		if err != nil {
			return nil, fmt.Errorf("knowledgegraph: import graph: expires_at: %w", err)
		}
		newID, err := s.graph.UpsertEdge(ctx, EdgeUpsert{
			FromID:            fromID,
			ToID:              toID,
			GraphKind:         graphKind,
			RelationType:      relation,
			Polarity:          polarity,
			Confidence:        confidence,
			ConditionText:     conditionText,
			EdgeKind:          edgeKind,
			ConditionJSON:     conditionJSON,
			SourceType:        sourceType,
			SourceRef:         sourceRef,
			ObservedAt:        observedAt,
			ValidFrom:         validFrom,
			ValidUntil:        validUntil,
			DecayHalfLifeDays: decay,
			ExpiresAt:         expiresAt,
			IsExecutable:      isExecutable,
			ActivationRule:    activationRule,
		})
		if err != nil {
			return nil, err
		}
		if oldID := int64FromAny(e["id"], 0); oldID > 0 {
			edgeIDMap[oldID] = newID
		}
		edgeCount++
	}

	evidenceCount := 0
	skippedEvidence := 0
	for _, ev := range evidence {
		oldEdgeID := int64FromAny(ev["edge_id"], 0)
		newEdgeID, ok := edgeIDMap[oldEdgeID]
		if !ok {
			skippedEvidence++
			continue
		}
		sourceType, _ := ev["source_type"].(string)
		sourceRef, _ := ev["source_ref"].(string)
		snippet, _ := ev["snippet"].(string)
		translation, _ := ev["translation"].(string)
		supports := boolFromAny(ev["supports"], true)
		weight := floatFromAny(ev["weight"], 1.0)
		observedAt, err := exportTime(ev["observed_at"])
		if err != nil {
			return nil, fmt.Errorf("knowledgegraph: import graph: evidence observed_at: %w", err)
		}
		if err := s.graph.AttachEdgeEvidence(newEdgeID, sourceType, sourceRef, snippet, translation, supports, weight, observedAt); err != nil {
			return nil, err
		}
		evidenceCount++
	}
	return map[string]any{
		"mode":             ImportGraphMode,
		"schema_version":   GraphExportSchemaVersion,
		"node_count":       nodeCount,
		"edge_count":       edgeCount,
		"evidence_count":   evidenceCount,
		"skipped_evidence": skippedEvidence,
	}, nil
}

func toMapSlice(raw any) ([]map[string]any, error) {
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case []map[string]any:
		return v, nil
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected object entries")
			}
			out = append(out, m)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected array of objects")
	}
}

func exportTime(raw any) (*time.Time, error) {
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil, nil
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, err
		}
		tt := t.UTC()
		return &tt, nil
	default:
		return nil, fmt.Errorf("expected RFC3339 string")
	}
}

func intFromAny(raw any, fallback int) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return fallback
	}
}

func int64FromAny(raw any, fallback int64) int64 {
	switch v := raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return fallback
	}
}

func floatFromAny(raw any, fallback float64) float64 {
	switch v := raw.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return fallback
	}
}

func boolFromAny(raw any, fallback bool) bool {
	if v, ok := raw.(bool); ok {
		return v
	}
	return fallback
}
