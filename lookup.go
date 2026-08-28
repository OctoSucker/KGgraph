package knowledgegraph

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const LookupContextMode = "lookup_context"

type LookupContextInput struct {
	Term            string
	GraphKind       string
	AsOf            time.Time
	MaxDepth        int
	MaxBranch       int
	MaxResults      int
	MinScore        float64
	IncludeNegative bool
}

// LookupContext resolves a term to graph context using exact match, semantic
// match (when an embedder is configured), and weighted multi-hop expansion.
// It never calls an LLM. Without an embedder it degrades to exact + expansion.
func (s *Service) LookupContext(ctx context.Context, in LookupContextInput) (map[string]any, error) {
	if s == nil || s.graph == nil {
		return nil, fmt.Errorf("knowledgegraph: lookup context: service is nil")
	}
	term := strings.TrimSpace(in.Term)
	if term == "" {
		return nil, fmt.Errorf("knowledgegraph: lookup context: term is required")
	}
	graphKind := strings.TrimSpace(in.GraphKind)
	if graphKind == "" {
		graphKind = "knowledge"
	}
	asOf := in.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	asOf = asOf.UTC()
	maxDepth, maxBranch, maxResults := in.MaxDepth, in.MaxBranch, in.MaxResults
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if maxBranch <= 0 {
		maxBranch = 5
	}
	if maxResults <= 0 {
		maxResults = 10
	}
	minScore := in.MinScore
	if minScore < 0 {
		minScore = 0
	}

	type resolvedSeed struct {
		id        string
		matchType string
	}
	var seeds []resolvedSeed
	if canon, ok, err := s.graph.CanonicalFor(term); err != nil {
		return nil, err
	} else if ok {
		seeds = append(seeds, resolvedSeed{id: canon, matchType: "exact"})
	} else if s.graph.embedder != nil {
		if canon, ok, err := s.graph.CanonicalForContext(ctx, term); err != nil {
			return nil, err
		} else if ok {
			seeds = append(seeds, resolvedSeed{id: canon, matchType: "semantic"})
		}
	}
	empty := map[string]any{
		"mode":          LookupContextMode,
		"query":         term,
		"as_of":         asOf.Format(time.RFC3339),
		"graph_kind":    graphKind,
		"matched":       false,
		"resolved":      []map[string]any{},
		"context_nodes": []map[string]any{},
		"edges":         map[string]any{},
		"counts": map[string]any{
			"resolved":      0,
			"context_nodes": 0,
			"edges":         0,
			"evidence":      0,
		},
	}
	if len(seeds) == 0 {
		return empty, nil
	}

	nodes, err := s.graph.AllNodes()
	if err != nil {
		return nil, err
	}
	nodeRows := make(map[string]NodeRow, len(nodes))
	for _, n := range nodes {
		nodeRows[n.ID] = n
	}
	resolvedOut := make([]map[string]any, 0, len(seeds))
	for _, seed := range seeds {
		item := map[string]any{"node_id": seed.id, "match_type": seed.matchType}
		if row, ok := nodeRows[seed.id]; ok {
			item["node_type"] = row.NodeType
			item["status"] = row.Status
			if row.AliasesJSON != "" && row.AliasesJSON != "[]" {
				item["aliases_json"] = row.AliasesJSON
				item["domain_json"] = row.DomainJSON
			}
		}
		resolvedOut = append(resolvedOut, item)
	}

	edgeRows, err := s.graph.AllEdges()
	if err != nil {
		return nil, err
	}
	edgeByID := make(map[int64]EdgeRow, len(edgeRows))
	for _, e := range edgeRows {
		edgeByID[e.ID] = e
	}

	best := map[string]ReasoningHit{}
	for _, seed := range seeds {
		hits, err := s.graph.ExpandReasoning(seed.id, graphKind, asOf, maxDepth, maxBranch, maxResults, in.IncludeNegative, minScore)
		if err != nil {
			return nil, err
		}
		for _, h := range hits {
			if cur, ok := best[h.NodeID]; !ok || h.Score > cur.Score {
				best[h.NodeID] = h
			}
		}
	}
	contextHits := make([]ReasoningHit, 0, len(best))
	for _, h := range best {
		contextHits = append(contextHits, h)
	}
	sort.Slice(contextHits, func(i, j int) bool {
		if contextHits[i].Score != contextHits[j].Score {
			return contextHits[i].Score > contextHits[j].Score
		}
		return contextHits[i].NodeID < contextHits[j].NodeID
	})
	if len(contextHits) > maxResults {
		contextHits = contextHits[:maxResults]
	}

	contextOut := make([]map[string]any, 0, len(contextHits))
	edgeOut := map[string]any{}
	evidenceCount := 0
	for _, h := range contextHits {
		steps := make([]map[string]any, 0, len(h.Steps))
		for _, st := range h.Steps {
			stepMap := map[string]any{
				"edge_id":             st.EdgeID,
				"from_id":             st.FromID,
				"to_id":               st.ToID,
				"relation_type":       st.RelationType,
				"raw_confidence":      st.RawConfidence,
				"freshness_factor":    st.FreshnessFactor,
				"verification_factor": st.VerificationFactor,
				"final_weight":        st.FinalWeight,
			}
			steps = append(steps, stepMap)
			if _, seen := edgeOut[edgeKey(st.EdgeID)]; seen {
				continue
			}
			if row, ok := edgeByID[st.EdgeID]; ok {
				detail, count, err := edgeDetailMap(s, row, asOf, "out")
				if err != nil {
					return nil, err
				}
				evidenceCount += count
				edgeOut[edgeKey(st.EdgeID)] = detail
			}
		}
		contextOut = append(contextOut, map[string]any{
			"node_id": h.NodeID,
			"score":   h.Score,
			"depth":   h.Depth,
			"path":    h.Path,
			"steps":   steps,
		})
	}

	// Incoming context: edges pointing into the seed or any reached node.
	// These answer "what signals this state" and "what invalidates it".
	reached := map[string]int{}
	for _, seed := range seeds {
		reached[seed.id] = 0
	}
	for _, h := range contextHits {
		if _, ok := reached[h.NodeID]; !ok {
			reached[h.NodeID] = h.Depth
		}
	}
	incomingNodes := map[string]struct{}{}
	incomingEdgeCount := 0
	for node := range reached {
		for _, row := range edgeRows {
			if row.GraphKind != graphKind || row.ToID != node {
				continue
			}
			score, _ := edgeWeightWithDetail(row, asOf)
			if score <= 0 {
				continue
			}
			if _, seen := edgeOut[edgeKey(row.ID)]; !seen {
				detail, count, err := edgeDetailMap(s, row, asOf, "in")
				if err != nil {
					return nil, err
				}
				evidenceCount += count
				edgeOut[edgeKey(row.ID)] = detail
				incomingEdgeCount++
			}
			if _, ok := reached[row.FromID]; !ok {
				incomingNodes[row.FromID] = struct{}{}
			}
		}
	}
	incomingList := make([]string, 0, len(incomingNodes))
	for id := range incomingNodes {
		incomingList = append(incomingList, id)
	}
	sort.Strings(incomingList)

	return map[string]any{
		"mode":           LookupContextMode,
		"query":          term,
		"as_of":          asOf.Format(time.RFC3339),
		"graph_kind":     graphKind,
		"matched":        true,
		"resolved":       resolvedOut,
		"context_nodes":  contextOut,
		"edges":          edgeOut,
		"incoming_nodes": incomingList,
		"counts": map[string]any{
			"resolved":       len(resolvedOut),
			"context_nodes":  len(contextOut),
			"edges":          len(edgeOut),
			"evidence":       evidenceCount,
			"incoming_edges": incomingEdgeCount,
		},
	}, nil
}

func edgeDetailMap(s *Service, row EdgeRow, asOf time.Time, direction string) (map[string]any, int, error) {
	_, step := edgeWeightWithDetail(row, asOf)
	detail := map[string]any{
		"edge_id":             row.ID,
		"from_id":             row.FromID,
		"to_id":               row.ToID,
		"relation_type":       row.RelationType,
		"raw_confidence":      row.Confidence,
		"final_weight":        step.FinalWeight,
		"freshness_factor":    step.FreshnessFactor,
		"verification_factor": step.VerificationFactor,
		"graph_kind":          row.GraphKind,
		"polarity":            row.Polarity,
		"confidence":          row.Confidence,
		"source_type":         row.SourceType,
		"source_ref":          row.SourceRef,
		"condition_text":      row.ConditionText,
		"edge_kind":           row.EdgeKind,
		"condition_json":      row.ConditionJSON,
		"evidence_count":      row.EvidenceCount,
		"failed_count":        row.FailedCount,
		"direction":           direction,
	}
	if row.ValidFrom != nil {
		detail["valid_from"] = row.ValidFrom.UTC().Format(time.RFC3339)
	}
	if row.ValidUntil != nil {
		detail["valid_until"] = row.ValidUntil.UTC().Format(time.RFC3339)
	}
	if row.LastVerifiedAt != nil {
		detail["last_verified_at"] = row.LastVerifiedAt.UTC().Format(time.RFC3339)
	}
	evidenceRows, err := s.store.EdgeEvidenceList(row.ID)
	if err != nil {
		return nil, 0, err
	}
	evidenceCount := 0
	if len(evidenceRows) > 0 {
		evidence := make([]map[string]any, 0, len(evidenceRows))
		for _, ev := range evidenceRows {
			evidence = append(evidence, map[string]any{
				"evidence_id": ev.ID,
				"source_type": ev.SourceType,
				"source_ref":  ev.SourceRef,
				"snippet":     ev.Snippet,
				"observed_at": ev.ObservedAt.UTC().Format(time.RFC3339),
				"supports":    ev.Supports,
				"weight":      ev.Weight,
			})
			evidenceCount++
		}
		detail["evidence"] = evidence
	}
	return detail, evidenceCount, nil
}

func edgeKey(id int64) string {
	return fmt.Sprintf("%d", id)
}
