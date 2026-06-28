package knowledgegraph

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	DecisionVerdictFollow      = "follow_recorded_action"
	DecisionVerdictWatch       = "watch"
	DecisionVerdictBlocked     = "blocked"
	DecisionVerdictInvalidated = "invalidated"
)

type DecisionEvaluateInput struct {
	Market                string
	Thesis                string
	AsOf                  time.Time
	TriggeredFailures     []string
	ActiveCounterEvidence []string
}

type decisionEvalItem struct {
	EdgeID        int64
	NodeID        string
	Text          string
	RelationType  string
	Score         float64
	Confidence    float64
	EvidenceCount int
	FailedCount   int
	Triggered     bool
	Active        bool
}

func (s *Service) EvaluateDecision(ctx context.Context, in DecisionEvaluateInput) (map[string]any, error) {
	_ = ctx
	if s == nil || s.graph == nil {
		return nil, fmt.Errorf("knowledgegraph: evaluate decision: service is nil")
	}
	market := strings.TrimSpace(in.Market)
	thesis := strings.TrimSpace(in.Thesis)
	if market == "" {
		return nil, fmt.Errorf("knowledgegraph: evaluate decision: market is required")
	}
	if thesis == "" {
		return nil, fmt.Errorf("knowledgegraph: evaluate decision: thesis is required")
	}
	asOf := in.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	asOf = asOf.UTC()

	nodes, err := s.graph.AllNodes()
	if err != nil {
		return nil, err
	}
	nodeStatus := make(map[string]string, len(nodes))
	for _, n := range nodes {
		nodeStatus[n.ID] = canonicalizeNodeID(n.Status)
	}

	rows, err := s.graph.AllEdges()
	if err != nil {
		return nil, err
	}
	marketID := canonicalizeNodeID(prefixedNodeID("market", market))
	thesisID := canonicalizeNodeID(prefixedNodeID("thesis", thesis))
	triggeredFailures := decisionMatchSet(in.TriggeredFailures, "failure")
	activeCounters := decisionMatchSet(in.ActiveCounterEvidence, "counter")

	var decisionFound bool
	var supportScore float64
	var counterScore float64
	var actionItems []decisionEvalItem
	var evidenceItems []decisionEvalItem
	var counterItems []decisionEvalItem
	var failureItems []decisionEvalItem
	var triggerItems []decisionEvalItem
	var reviewItems []decisionEvalItem
	blockingReasons := []string{}
	actionIDs := map[string]struct{}{}
	reviewIDs := map[string]struct{}{}
	negativeReview := false
	failedAction := false

	activeEdges := make([]EdgeRow, 0, len(rows))
	for _, e := range rows {
		if e.GraphKind != DecisionGraphKind {
			continue
		}
		weight, _ := edgeWeightWithDetail(e, asOf)
		if weight <= 0 {
			continue
		}
		activeEdges = append(activeEdges, e)
	}

	for _, e := range activeEdges {
		weight, _ := edgeWeightWithDetail(e, asOf)
		if e.FromID == marketID && e.ToID == thesisID && e.RelationType == "has_thesis" {
			decisionFound = true
		}
		if e.FromID != thesisID {
			continue
		}
		item := decisionItemFromEdge(e, weight, false, false)
		switch e.RelationType {
		case "supports_action":
			actionItems = append(actionItems, item)
			actionIDs[e.ToID] = struct{}{}
			supportScore += weight
			if e.FailedCount > e.EvidenceCount {
				failedAction = true
			}
		case "supported_by":
			evidenceItems = append(evidenceItems, item)
			supportScore += weight
		case "contradicted_by":
			active := decisionMatches(activeCounters, e.ToID)
			item.Active = active
			counterItems = append(counterItems, item)
			penalty := weight * 0.65
			if active {
				penalty = weight
			}
			counterScore += penalty
		case "invalidated_by":
			triggered := decisionMatches(triggeredFailures, e.ToID) || decisionNodeStatusTriggered(nodeStatus[e.ToID])
			item.Triggered = triggered
			failureItems = append(failureItems, item)
			if triggered {
				blockingReasons = append(blockingReasons, "failure_condition_triggered:"+decisionNodeText(e.ToID))
			}
		case "requires_review_on":
			triggerItems = append(triggerItems, item)
		case "reviewed_by":
			reviewItems = append(reviewItems, item)
			reviewIDs[e.ToID] = struct{}{}
			if e.Polarity < 0 {
				negativeReview = true
			}
		}
	}
	if !decisionFound {
		return nil, fmt.Errorf("knowledgegraph: evaluate decision: no recorded decision for market %q thesis %q", market, thesis)
	}

	var positionRules []decisionEvalItem
	for _, e := range activeEdges {
		weight, _ := edgeWeightWithDetail(e, asOf)
		if _, ok := actionIDs[e.FromID]; ok && e.RelationType == "constrained_by" {
			positionRules = append(positionRules, decisionItemFromEdge(e, weight, false, false))
		}
		if _, ok := reviewIDs[e.FromID]; ok && e.RelationType == "resolved_as" {
			reviewItems = append(reviewItems, decisionItemFromEdge(e, weight, false, false))
			switch strings.TrimPrefix(e.ToID, "outcome:") {
			case "incorrect", "invalidated":
				negativeReview = true
			}
		}
	}

	sortDecisionItems(actionItems)
	sortDecisionItems(evidenceItems)
	sortDecisionItems(counterItems)
	sortDecisionItems(failureItems)
	sortDecisionItems(triggerItems)
	sortDecisionItems(positionRules)
	sortDecisionItems(reviewItems)

	counterRatio := 0.0
	if supportScore > 0 {
		counterRatio = counterScore / supportScore
	}
	deterministicConfidence := bounded01(0.5 + 0.5*((supportScore-counterScore)/(supportScore+counterScore+0.000001)))
	verdict := DecisionVerdictFollow
	if len(actionItems) == 0 {
		verdict = DecisionVerdictBlocked
		blockingReasons = append(blockingReasons, "missing_recorded_action")
	}
	if len(evidenceItems) == 0 {
		verdict = DecisionVerdictBlocked
		blockingReasons = append(blockingReasons, "missing_supporting_evidence")
	}
	if failedAction {
		verdict = DecisionVerdictInvalidated
		blockingReasons = append(blockingReasons, "recorded_action_has_more_failures_than_successes")
	}
	if negativeReview {
		verdict = DecisionVerdictInvalidated
		blockingReasons = append(blockingReasons, "historical_review_invalidated_or_failed_this_thesis")
	}
	for _, item := range failureItems {
		if item.Triggered {
			verdict = DecisionVerdictInvalidated
			deterministicConfidence = math.Min(deterministicConfidence, 0.2)
			break
		}
	}
	if verdict == DecisionVerdictFollow {
		switch {
		case counterRatio >= 0.75:
			verdict = DecisionVerdictBlocked
			blockingReasons = append(blockingReasons, "counter_evidence_ratio_above_block_threshold")
		case deterministicConfidence < 0.58:
			verdict = DecisionVerdictBlocked
			blockingReasons = append(blockingReasons, "deterministic_confidence_below_threshold")
		case counterRatio >= 0.55:
			verdict = DecisionVerdictWatch
			blockingReasons = append(blockingReasons, "counter_evidence_ratio_requires_watch")
		}
	}

	recordedActions := decisionTexts(actionItems)
	executionAllowed := verdict == DecisionVerdictFollow && decisionActionExecutable(recordedActions)
	return map[string]any{
		"graph_kind":               DecisionGraphKind,
		"market_id":                marketID,
		"thesis_id":                thesisID,
		"as_of":                    asOf.Format(time.RFC3339),
		"verdict":                  verdict,
		"execution_allowed":        executionAllowed,
		"recorded_actions":         recordedActions,
		"deterministic_confidence": roundFloat(deterministicConfidence, 4),
		"support_score":            roundFloat(supportScore, 4),
		"counter_score":            roundFloat(counterScore, 4),
		"counter_ratio":            roundFloat(counterRatio, 4),
		"blocking_reasons":         blockingReasons,
		"recorded_action_edges":    decisionItemsAsMaps(actionItems),
		"rules": []string{
			"triggered failure condition => invalidated",
			"negative historical review or failed recorded action => invalidated",
			"missing action or evidence => blocked",
			"counter_ratio >= 0.75 => blocked",
			"deterministic_confidence < 0.58 => blocked",
			"counter_ratio >= 0.55 => watch",
			"otherwise follow recorded action",
		},
		"supporting_evidence":     decisionItemsAsMaps(evidenceItems),
		"counter_evidence":        decisionItemsAsMaps(counterItems),
		"failure_conditions":      decisionItemsAsMaps(failureItems),
		"review_triggers":         decisionItemsAsMaps(triggerItems),
		"position_rules":          decisionItemsAsMaps(positionRules),
		"review_history":          decisionItemsAsMaps(reviewItems),
		"active_counter_input":    cleanStringList(in.ActiveCounterEvidence),
		"triggered_failure_input": cleanStringList(in.TriggeredFailures),
	}, nil
}

func decisionItemFromEdge(e EdgeRow, score float64, triggered, active bool) decisionEvalItem {
	return decisionEvalItem{
		EdgeID:        e.ID,
		NodeID:        e.ToID,
		Text:          decisionNodeText(e.ToID),
		RelationType:  e.RelationType,
		Score:         score,
		Confidence:    e.Confidence,
		EvidenceCount: e.EvidenceCount,
		FailedCount:   e.FailedCount,
		Triggered:     triggered,
		Active:        active,
	}
}

func decisionItemsAsMaps(items []decisionEvalItem) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m := map[string]any{
			"edge_id":        item.EdgeID,
			"node_id":        item.NodeID,
			"text":           item.Text,
			"relation_type":  item.RelationType,
			"score":          roundFloat(item.Score, 4),
			"confidence":     item.Confidence,
			"evidence_count": item.EvidenceCount,
			"failed_count":   item.FailedCount,
		}
		if item.Triggered {
			m["triggered"] = true
		}
		if item.Active {
			m["active"] = true
		}
		out = append(out, m)
	}
	return out
}

func sortDecisionItems(items []decisionEvalItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		if items[i].Text != items[j].Text {
			return items[i].Text < items[j].Text
		}
		return items[i].EdgeID < items[j].EdgeID
	})
}

func decisionTexts(items []decisionEvalItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Text)
	}
	sort.Strings(out)
	return out
}

func decisionActionExecutable(actions []string) bool {
	for _, action := range actions {
		switch canonicalizeNodeID(action) {
		case "buy", "hold", "reduce", "sell":
			return true
		}
	}
	return false
}

func decisionNodeText(id string) string {
	for _, prefix := range []string{"market:", "thesis:", "action:", "evidence:", "counter:", "failure:", "trigger:", "position_rule:", "review:", "outcome:", "result:", "lesson:", "rule_update:"} {
		if strings.HasPrefix(id, prefix) {
			return strings.TrimPrefix(id, prefix)
		}
	}
	return id
}

func decisionMatchSet(items []string, prefix string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, item := range cleanStringList(items) {
		raw := canonicalizeNodeID(item)
		if raw == "" {
			continue
		}
		out[raw] = struct{}{}
		if !strings.Contains(raw, ":") {
			out[prefix+":"+raw] = struct{}{}
		}
	}
	return out
}

func decisionMatches(set map[string]struct{}, nodeID string) bool {
	if len(set) == 0 {
		return false
	}
	if _, ok := set[nodeID]; ok {
		return true
	}
	_, ok := set[decisionNodeText(nodeID)]
	return ok
}

func decisionNodeStatusTriggered(status string) bool {
	switch canonicalizeNodeID(status) {
	case "triggered", "hit", "true", "failed":
		return true
	default:
		return false
	}
}

func bounded01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func roundFloat(v float64, places int) float64 {
	if places <= 0 {
		return math.Round(v)
	}
	pow := math.Pow10(places)
	return math.Round(v*pow) / pow
}
