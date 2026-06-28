package knowledgegraph

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	DecisionStatusUsable      = "usable"
	DecisionStatusWatch       = "watch"
	DecisionStatusBlocked     = "blocked"
	DecisionStatusInvalidated = "invalidated"
)

type DecisionStatusInput struct {
	Market                string
	AsOf                  time.Time
	TriggeredFailures     []string
	ActiveCounterEvidence []string
	IncludeEvaluation     bool
}

type decisionStatusTarget struct {
	marketID string
	thesisID string
	market   string
	thesis   string
}

func (s *Service) DecisionStatus(ctx context.Context, in DecisionStatusInput) (map[string]any, error) {
	if s == nil || s.graph == nil {
		return nil, fmt.Errorf("knowledgegraph: decision status: service is nil")
	}
	asOf := in.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	asOf = asOf.UTC()
	targets, err := s.decisionStatusTargets(in.Market)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(targets))
	counts := map[string]int{
		DecisionStatusUsable:      0,
		DecisionStatusWatch:       0,
		DecisionStatusBlocked:     0,
		DecisionStatusInvalidated: 0,
	}
	for _, target := range targets {
		eval, err := s.EvaluateDecision(ctx, DecisionEvaluateInput{
			Market:                target.market,
			Thesis:                target.thesis,
			AsOf:                  asOf,
			TriggeredFailures:     in.TriggeredFailures,
			ActiveCounterEvidence: in.ActiveCounterEvidence,
		})
		if err != nil {
			return nil, err
		}
		item := decisionStatusItem(target, eval, in.IncludeEvaluation)
		status := mapString(item, "status")
		counts[status]++
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		ri := decisionStatusRank(mapString(items[i], "status"))
		rj := decisionStatusRank(mapString(items[j], "status"))
		if ri != rj {
			return ri > rj
		}
		ci := mapFloat(items[i], "deterministic_confidence")
		cj := mapFloat(items[j], "deterministic_confidence")
		if ci != cj {
			return ci < cj
		}
		if mapString(items[i], "market") != mapString(items[j], "market") {
			return mapString(items[i], "market") < mapString(items[j], "market")
		}
		return mapString(items[i], "thesis") < mapString(items[j], "thesis")
	})
	return map[string]any{
		"mode":                    "decision_status",
		"as_of":                   asOf.Format(time.RFC3339),
		"market_filter":           strings.TrimSpace(in.Market),
		"count":                   len(items),
		"counts":                  counts,
		"items":                   items,
		"triggered_failure_input": cleanStringList(in.TriggeredFailures),
		"active_counter_input":    cleanStringList(in.ActiveCounterEvidence),
	}, nil
}

func (s *Service) decisionStatusTargets(marketFilter string) ([]decisionStatusTarget, error) {
	rows, err := s.graph.AllEdges()
	if err != nil {
		return nil, err
	}
	nodes, err := s.graph.AllNodes()
	if err != nil {
		return nil, err
	}
	nodeStatus := make(map[string]string, len(nodes))
	for _, node := range nodes {
		nodeStatus[node.ID] = canonicalizeNodeID(node.Status)
	}
	marketFilterID := ""
	if strings.TrimSpace(marketFilter) != "" {
		marketFilterID = canonicalizeNodeID(prefixedNodeID("market", marketFilter))
	}
	seen := map[string]struct{}{}
	targets := make([]decisionStatusTarget, 0)
	for _, row := range rows {
		if row.GraphKind != DecisionGraphKind || row.RelationType != "has_thesis" {
			continue
		}
		if !strings.HasPrefix(row.FromID, "market:") || !strings.HasPrefix(row.ToID, "thesis:") {
			continue
		}
		if marketFilterID != "" && row.FromID != marketFilterID {
			continue
		}
		if !decisionStatusNodeActive(nodeStatus[row.FromID]) || !decisionStatusNodeActive(nodeStatus[row.ToID]) {
			continue
		}
		key := row.FromID + "\x00" + row.ToID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		targets = append(targets, decisionStatusTarget{
			marketID: row.FromID,
			thesisID: row.ToID,
			market:   decisionNodeText(row.FromID),
			thesis:   decisionNodeText(row.ToID),
		})
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].market != targets[j].market {
			return targets[i].market < targets[j].market
		}
		return targets[i].thesis < targets[j].thesis
	})
	return targets, nil
}

func decisionStatusItem(target decisionStatusTarget, eval map[string]any, includeEvaluation bool) map[string]any {
	status, statusReason := decisionStatusFromEvaluation(eval)
	item := map[string]any{
		"market_id":                target.marketID,
		"market":                   target.market,
		"thesis_id":                target.thesisID,
		"thesis":                   target.thesis,
		"status":                   status,
		"status_reason":            statusReason,
		"verdict":                  mapString(eval, "verdict"),
		"execution_allowed":        mapBool(eval, "execution_allowed"),
		"recorded_actions":         mapStringSlice(eval, "recorded_actions"),
		"deterministic_confidence": mapFloat(eval, "deterministic_confidence"),
		"counter_ratio":            mapFloat(eval, "counter_ratio"),
		"blocking_reasons":         mapStringSlice(eval, "blocking_reasons"),
		"supporting_evidence":      mapItemTexts(eval, "supporting_evidence", 3),
		"counter_evidence":         mapItemTexts(eval, "counter_evidence", 3),
		"failure_conditions":       mapItemTexts(eval, "failure_conditions", 3),
		"position_rules":           mapItemTexts(eval, "position_rules", 3),
	}
	if includeEvaluation {
		item["evaluation"] = eval
	}
	return item
}

func decisionStatusFromEvaluation(eval map[string]any) (string, string) {
	switch mapString(eval, "verdict") {
	case DecisionVerdictInvalidated:
		return DecisionStatusInvalidated, "decision_evaluation_invalidated"
	case DecisionVerdictBlocked:
		return DecisionStatusBlocked, "decision_evaluation_blocked"
	case DecisionVerdictWatch:
		return DecisionStatusWatch, "decision_evaluation_requires_watch"
	case DecisionVerdictFollow:
		if mapBool(eval, "execution_allowed") {
			return DecisionStatusUsable, "recorded_action_executable"
		}
		return DecisionStatusWatch, "recorded_action_observation_only"
	default:
		return DecisionStatusBlocked, "unknown_decision_verdict"
	}
}

func decisionStatusRank(status string) int {
	switch status {
	case DecisionStatusInvalidated:
		return 4
	case DecisionStatusBlocked:
		return 3
	case DecisionStatusWatch:
		return 2
	case DecisionStatusUsable:
		return 1
	default:
		return 3
	}
}

func decisionStatusNodeActive(status string) bool {
	switch canonicalizeNodeID(status) {
	case "", "active":
		return true
	default:
		return false
	}
}
