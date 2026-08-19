package knowledgegraph

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	ConflictScanMode    = "conflict_scan"
	ConflictPolicyWarn  = "warn"
	ConflictPolicyBlock = "block"
	ConflictPolicyOff   = "off"
)

// checkEdgeConflictPolicy runs ConflictScan for a candidate edge before it is
// written. Policies:
//   - warn (default): conflicts are returned but the write proceeds;
//   - block: the write fails when any conflicting active edge exists;
//   - off: no scan is performed.
func checkEdgeConflictPolicy(ctx context.Context, s *Service, in EdgeUpsert, policy string) ([]map[string]any, error) {
	policy = canonicalizeNodeID(policy)
	if policy == "" {
		policy = ConflictPolicyWarn
	}
	if !validConflictPolicy(policy) {
		return nil, fmt.Errorf("knowledgegraph: conflict_policy must be block, warn, or off")
	}
	if policy == ConflictPolicyOff {
		return nil, nil
	}
	out, err := s.ConflictScan(ctx, ConflictScanInput{
		FromID:       in.FromID,
		ToID:         in.ToID,
		RelationType: in.RelationType,
		Polarity:     in.Polarity,
		GraphKind:    in.GraphKind,
	})
	if err != nil {
		return nil, err
	}
	conflicts, _ := out["conflicts"].([]map[string]any)
	if policy == ConflictPolicyBlock && len(conflicts) > 0 {
		return nil, fmt.Errorf("knowledgegraph: conflict_policy=block rejected %d conflicting edge(s)", len(conflicts))
	}
	return conflicts, nil
}

func validConflictPolicy(policy string) bool {
	switch policy {
	case ConflictPolicyWarn, ConflictPolicyBlock, ConflictPolicyOff:
		return true
	default:
		return false
	}
}

// ConflictScanInput describes a candidate edge. ConflictScan lists existing
// active edges that contradict it, without calling an LLM.
type ConflictScanInput struct {
	FromID       string
	ToID         string
	RelationType string
	Polarity     int
	GraphKind    string
	AsOf         time.Time
}

type ConflictHit struct {
	EdgeID        int64
	FromID        string
	ToID          string
	RelationType  string
	Polarity      int
	Confidence    float64
	Score         float64
	EvidenceCount int
	FailedCount   int
	Reason        string
}

// ConflictScan returns edges that currently contradict the candidate edge:
//   - same endpoints with opposite nonzero polarity;
//   - same endpoints with an antonym relation (supports<->contradicts,
//     increases_probability_of<->decreases_probability_of, requires<->blocks);
//   - an existing "contradicts" edge in the reverse direction.
func (s *Service) ConflictScan(ctx context.Context, in ConflictScanInput) (map[string]any, error) {
	_ = ctx
	if s == nil || s.graph == nil {
		return nil, fmt.Errorf("knowledgegraph: conflict scan: service is nil")
	}
	fromID := canonicalizeNodeID(in.FromID)
	toID := canonicalizeNodeID(in.ToID)
	relation := canonicalizeNodeID(in.RelationType)
	if fromID == "" || toID == "" {
		return nil, fmt.Errorf("knowledgegraph: conflict scan: from_id and to_id are required")
	}
	if relation == "" {
		return nil, fmt.Errorf("knowledgegraph: conflict scan: relation_type is required")
	}
	if in.Polarity < -1 || in.Polarity > 1 {
		return nil, fmt.Errorf("knowledgegraph: conflict scan: polarity must be -1, 0, or 1")
	}
	if strings.TrimSpace(in.GraphKind) == "" {
		in.GraphKind = "knowledge"
	}
	asOf := in.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	asOf = asOf.UTC()
	antonym := oppositeRelation(relation)

	rows, err := s.graph.AllEdges()
	if err != nil {
		return nil, err
	}
	seen := map[int64]struct{}{}
	hits := make([]ConflictHit, 0, 4)
	for _, e := range rows {
		if e.GraphKind != in.GraphKind {
			continue
		}
		score, _ := edgeWeightWithDetail(e, asOf)
		if score <= 0 {
			continue
		}
		sameDirection := e.FromID == fromID && e.ToID == toID
		reverseContradiction := e.RelationType == "contradicts" && e.FromID == toID && e.ToID == fromID
		if !sameDirection && !reverseContradiction {
			continue
		}
		var reason string
		switch {
		case sameDirection && e.RelationType == relation && e.Polarity != 0 && in.Polarity != 0 && e.Polarity != in.Polarity:
			reason = "opposite_polarity"
		case sameDirection && antonym != "" && e.RelationType == antonym:
			reason = "antonym_relation"
		case reverseContradiction:
			reason = "reverse_contradicts_edge"
		}
		if reason == "" {
			continue
		}
		if _, ok := seen[e.ID]; ok {
			continue
		}
		seen[e.ID] = struct{}{}
		hits = append(hits, ConflictHit{
			EdgeID:        e.ID,
			FromID:        e.FromID,
			ToID:          e.ToID,
			RelationType:  e.RelationType,
			Polarity:      e.Polarity,
			Confidence:    e.Confidence,
			Score:         roundFloat(score, 4),
			EvidenceCount: e.EvidenceCount,
			FailedCount:   e.FailedCount,
			Reason:        reason,
		})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		if hits[i].EdgeID != hits[j].EdgeID {
			return hits[i].EdgeID < hits[j].EdgeID
		}
		return hits[i].RelationType < hits[j].RelationType
	})
	out := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		out = append(out, map[string]any{
			"edge_id":        h.EdgeID,
			"from_id":        h.FromID,
			"to_id":          h.ToID,
			"relation_type":  h.RelationType,
			"polarity":       h.Polarity,
			"confidence":     h.Confidence,
			"score":          h.Score,
			"evidence_count": h.EvidenceCount,
			"failed_count":   h.FailedCount,
			"reason":         h.Reason,
		})
	}
	return map[string]any{
		"mode":           ConflictScanMode,
		"from_id":        fromID,
		"to_id":          toID,
		"relation_type":  relation,
		"polarity":       in.Polarity,
		"graph_kind":     in.GraphKind,
		"as_of":          asOf.Format(time.RFC3339),
		"conflict_count": len(hits),
		"conflicts":      out,
	}, nil
}

// oppositeRelation maps a relation to its deterministic antonym. Unknown
// relations return "".
func oppositeRelation(relation string) string {
	switch relation {
	case "supports":
		return "contradicts"
	case "contradicts":
		return "supports"
	case "increases_probability_of":
		return "decreases_probability_of"
	case "decreases_probability_of":
		return "increases_probability_of"
	case "requires":
		return "blocks"
	case "blocks":
		return "requires"
	default:
		return ""
	}
}
