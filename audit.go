package knowledgegraph

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const DecisionAuditMode = "decision_audit"

type DecisionAuditInput struct {
	Market string
	Thesis string
	AsOf   time.Time
}

// DecisionAudit returns the full provenance timeline of a recorded decision:
// every related edge (with scores and validity), attached evidence, and review
// history. It is deterministic and never calls an LLM.
func (s *Service) DecisionAudit(ctx context.Context, in DecisionAuditInput) (map[string]any, error) {
	_ = ctx
	if s == nil || s.graph == nil {
		return nil, fmt.Errorf("knowledgegraph: decision audit: service is nil")
	}
	market := strings.TrimSpace(in.Market)
	thesis := strings.TrimSpace(in.Thesis)
	if market == "" {
		return nil, fmt.Errorf("knowledgegraph: decision audit: market is required")
	}
	if thesis == "" {
		return nil, fmt.Errorf("knowledgegraph: decision audit: thesis is required")
	}
	asOf := in.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	asOf = asOf.UTC()
	marketID := canonicalizeNodeID(prefixedNodeID("market", market))
	thesisID := canonicalizeNodeID(prefixedNodeID("thesis", thesis))

	eval, err := s.EvaluateDecision(ctx, DecisionEvaluateInput{
		Market: market,
		Thesis: thesis,
		AsOf:   asOf,
	})
	if err != nil {
		return nil, err
	}
	rows, err := s.graph.AllEdges()
	if err != nil {
		return nil, err
	}
	actionIDs := map[string]struct{}{}
	reviewIDs := map[string]struct{}{}
	for _, e := range rows {
		if e.GraphKind != DecisionGraphKind {
			continue
		}
		if e.FromID == thesisID && e.RelationType == "supports_action" {
			actionIDs[e.ToID] = struct{}{}
		}
		if e.FromID == thesisID && e.RelationType == "reviewed_by" {
			reviewIDs[e.ToID] = struct{}{}
		}
	}
	inScope := func(e EdgeRow) bool {
		if e.FromID == marketID || e.ToID == marketID {
			return true
		}
		if e.FromID == thesisID || e.ToID == thesisID {
			return true
		}
		if _, ok := actionIDs[e.FromID]; ok {
			return true
		}
		if _, ok := actionIDs[e.ToID]; ok {
			return true
		}
		if _, ok := reviewIDs[e.FromID]; ok {
			return true
		}
		if _, ok := reviewIDs[e.ToID]; ok {
			return true
		}
		return false
	}

	timeline := make([]map[string]any, 0, 8)
	for _, e := range rows {
		if e.GraphKind != DecisionGraphKind || !inScope(e) {
			continue
		}
		score, _ := edgeWeightWithDetail(e, asOf)
		item := map[string]any{
			"edge_id":        e.ID,
			"from_id":        e.FromID,
			"to_id":          e.ToID,
			"from_text":      decisionNodeText(e.FromID),
			"to_text":        decisionNodeText(e.ToID),
			"relation_type":  e.RelationType,
			"polarity":       e.Polarity,
			"confidence":     e.Confidence,
			"created_at":     e.CreatedAt.UTC().Format(time.RFC3339),
			"evidence_count": e.EvidenceCount,
			"failed_count":   e.FailedCount,
			"score":          roundFloat(score, 4),
			"active":         score > 0,
			"source_ref":     e.SourceRef,
			"condition_text": e.ConditionText,
		}
		if e.ObservedAt != nil {
			item["observed_at"] = e.ObservedAt.UTC().Format(time.RFC3339)
		}
		if e.ValidFrom != nil {
			item["valid_from"] = e.ValidFrom.UTC().Format(time.RFC3339)
		}
		if e.ValidUntil != nil {
			item["valid_until"] = e.ValidUntil.UTC().Format(time.RFC3339)
		}
		if e.ExpiresAt != nil {
			item["expires_at"] = e.ExpiresAt.UTC().Format(time.RFC3339)
		}
		if e.LastVerifiedAt != nil {
			item["last_verified_at"] = e.LastVerifiedAt.UTC().Format(time.RFC3339)
		}
		evidenceRows, err := s.store.EdgeEvidenceList(e.ID)
		if err != nil {
			return nil, err
		}
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
			}
			item["evidence"] = evidence
		}
		timeline = append(timeline, item)
	}
	sort.Slice(timeline, func(i, j int) bool {
		ai := mapString(timeline[i], "created_at")
		aj := mapString(timeline[j], "created_at")
		if ai != aj {
			return ai < aj
		}
		return mapInt64(timeline[i], "edge_id") < mapInt64(timeline[j], "edge_id")
	})

	reviews := buildReviewSummary(rows, reviewIDs)
	return map[string]any{
		"mode":                     DecisionAuditMode,
		"market_id":                marketID,
		"market":                   decisionNodeText(marketID),
		"thesis_id":                thesisID,
		"thesis":                   decisionNodeText(thesisID),
		"as_of":                    asOf.Format(time.RFC3339),
		"verdict":                  mapString(eval, "verdict"),
		"execution_allowed":        mapBool(eval, "execution_allowed"),
		"deterministic_confidence": mapFloat(eval, "deterministic_confidence"),
		"counter_ratio":            mapFloat(eval, "counter_ratio"),
		"blocking_reasons":         mapStringSlice(eval, "blocking_reasons"),
		"edge_count":               len(timeline),
		"timeline":                 timeline,
		"review_count":             len(reviews),
		"reviews":                  reviews,
	}, nil
}

func buildReviewSummary(rows []EdgeRow, reviewIDs map[string]struct{}) []map[string]any {
	out := make([]map[string]any, 0, len(reviewIDs))
	for reviewID := range reviewIDs {
		entry := map[string]any{"review_id": reviewID}
		lessons := []string{}
		ruleUpdates := []string{}
		for _, e := range rows {
			if e.GraphKind != DecisionGraphKind {
				continue
			}
			if e.RelationType == "reviewed_by" && e.ToID == reviewID {
				entry["reviewed_at"] = e.CreatedAt.UTC().Format(time.RFC3339)
				continue
			}
			if e.FromID != reviewID {
				continue
			}
			switch e.RelationType {
			case "resolved_as":
				entry["outcome"] = decisionNodeText(e.ToID)
			case "realized_as":
				entry["realized_result"] = decisionNodeText(e.ToID)
			case "produced_lesson":
				lessons = append(lessons, decisionNodeText(e.ToID))
			case "updates_rule":
				ruleUpdates = append(ruleUpdates, decisionNodeText(e.ToID))
			}
		}
		if len(lessons) > 0 {
			sort.Strings(lessons)
			entry["lessons"] = lessons
		}
		if len(ruleUpdates) > 0 {
			sort.Strings(ruleUpdates)
			entry["rule_updates"] = ruleUpdates
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		return mapString(out[i], "reviewed_at") < mapString(out[j], "reviewed_at")
	})
	return out
}

func mapInt64(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}
