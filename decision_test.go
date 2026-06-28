package knowledgegraph

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRecordDecisionRequiresDisciplineFields(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market": "Oil market",
		"thesis": "Escalation risk is underpriced",
		"action": "buy",
	})
	if err == nil {
		t.Fatalf("expected missing discipline fields to fail")
	}
	if !strings.Contains(err.Error(), "evidence") {
		t.Fatalf("expected evidence-related error, got %v", err)
	}
}

func TestRecordDecisionWritesEvidenceCounterAndFailureEdges(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	out, err := svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
		"next_triggers":      []any{"opec emergency statement"},
		"position_rule":      "max 1u until settlement clarity",
	})
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	if out["graph_kind"] != DecisionGraphKind {
		t.Fatalf("unexpected graph kind: %#v", out["graph_kind"])
	}
	rows, err := svc.graph.AllEdges()
	if err != nil {
		t.Fatalf("list edges: %v", err)
	}
	relations := map[string]int{}
	for _, row := range rows {
		if row.GraphKind == DecisionGraphKind {
			relations[row.RelationType]++
		}
	}
	for _, relation := range []string{"has_thesis", "supports_action", "supported_by", "contradicted_by", "invalidated_by", "requires_review_on", "constrained_by"} {
		if relations[relation] == 0 {
			t.Fatalf("missing decision relation %q in %#v", relation, relations)
		}
	}
	hits, err := svc.graph.ExpandReasoning("market:oil market", DecisionGraphKind, time.Now().UTC(), 4, 10, 20, true, 0)
	if err != nil {
		t.Fatalf("expand decision graph: %v", err)
	}
	foundFailure := false
	for _, hit := range hits {
		if strings.HasPrefix(hit.NodeID, "failure:") {
			foundFailure = true
		}
	}
	if !foundFailure {
		t.Fatalf("expected decision expansion with negative edges to include failure condition, hits=%#v", hits)
	}
}

func TestReviewDecisionRequiresLessonsForBadOutcomes(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolReviewDecision, map[string]any{
		"market":          "Oil market",
		"thesis":          "Escalation risk is underpriced",
		"outcome":         "incorrect",
		"realized_result": "oil sold off after ceasefire",
	})
	if err == nil {
		t.Fatalf("expected missing lessons to fail for incorrect outcome")
	}
	if !strings.Contains(err.Error(), "lessons") {
		t.Fatalf("expected lessons error, got %v", err)
	}
}

func TestReviewDecisionWritesReviewAndFailsThesisActionEdge(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
	})
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	out, err := svc.Call(context.Background(), ToolReviewDecision, map[string]any{
		"market":          "Oil market",
		"thesis":          "Escalation risk is underpriced",
		"outcome":         "incorrect",
		"realized_result": "ceasefire confirmed and oil sold off",
		"lessons":         []any{"do not ignore ceasefire verification path"},
		"rule_updates":    []any{"reduce size when failure condition is imminent"},
	})
	if err != nil {
		t.Fatalf("review decision: %v", err)
	}
	verified, ok := out["verified_edges"].([]int64)
	if !ok || len(verified) == 0 {
		t.Fatalf("expected failed verified edges, got %#v", out["verified_edges"])
	}
	rows, err := svc.graph.AllEdges()
	if err != nil {
		t.Fatalf("list edges: %v", err)
	}
	foundReview := false
	foundFailedAction := false
	for _, row := range rows {
		if row.RelationType == "produced_lesson" || row.RelationType == "updates_rule" {
			foundReview = true
		}
		if row.RelationType == "supports_action" && row.FailedCount > 0 {
			foundFailedAction = true
		}
	}
	if !foundReview {
		t.Fatalf("expected review lesson/rule edges")
	}
	if !foundFailedAction {
		t.Fatalf("expected supports_action edge to be marked failed")
	}
}

func TestEvaluateDecisionIsDeterministicForSameAsOf(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising", "market not pricing weekend risk"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
		"position_rule":      "max 1u until settlement clarity",
	})
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	asOf := time.Now().UTC().Add(time.Second).Format(time.RFC3339)
	args := map[string]any{
		"market": "Oil market",
		"thesis": "Escalation risk is underpriced",
		"as_of":  asOf,
	}
	first, err := svc.Call(context.Background(), ToolEvaluateDecision, args)
	if err != nil {
		t.Fatalf("evaluate decision: %v", err)
	}
	second, err := svc.Call(context.Background(), ToolEvaluateDecision, args)
	if err != nil {
		t.Fatalf("evaluate decision again: %v", err)
	}
	if first["verdict"] != DecisionVerdictFollow {
		t.Fatalf("expected follow verdict, got %#v", first["verdict"])
	}
	if first["execution_allowed"] != true {
		t.Fatalf("expected execution allowed, got %#v", first["execution_allowed"])
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected deterministic output for same as_of\nfirst=%#v\nsecond=%#v", first, second)
	}
}

func TestEvaluateDecisionTriggeredFailureInvalidates(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
	})
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	out, err := svc.Call(context.Background(), ToolEvaluateDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"triggered_failures": []any{"confirmed ceasefire"},
	})
	if err != nil {
		t.Fatalf("evaluate decision: %v", err)
	}
	if out["verdict"] != DecisionVerdictInvalidated {
		t.Fatalf("expected invalidated verdict, got %#v", out["verdict"])
	}
	if out["execution_allowed"] != false {
		t.Fatalf("expected execution blocked, got %#v", out["execution_allowed"])
	}
	failures, ok := out["failure_conditions"].([]map[string]any)
	if !ok {
		t.Fatalf("unexpected failure_conditions type: %#v", out["failure_conditions"])
	}
	foundTriggered := false
	for _, item := range failures {
		if item["triggered"] == true {
			foundTriggered = true
		}
	}
	if !foundTriggered {
		t.Fatalf("expected triggered failure in output: %#v", failures)
	}
}

func TestEvaluateDecisionNegativeReviewInvalidates(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
	})
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolReviewDecision, map[string]any{
		"market":          "Oil market",
		"thesis":          "Escalation risk is underpriced",
		"outcome":         "incorrect",
		"realized_result": "ceasefire confirmed and oil sold off",
		"lessons":         []any{"do not ignore ceasefire verification path"},
	})
	if err != nil {
		t.Fatalf("review decision: %v", err)
	}
	out, err := svc.Call(context.Background(), ToolEvaluateDecision, map[string]any{
		"market": "Oil market",
		"thesis": "Escalation risk is underpriced",
	})
	if err != nil {
		t.Fatalf("evaluate decision: %v", err)
	}
	if out["verdict"] != DecisionVerdictInvalidated {
		t.Fatalf("expected invalidated verdict after negative review, got %#v", out["verdict"])
	}
}

func TestStrictAskQuestionToneDoesNotChangeAnswer(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising", "market not pricing weekend risk"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
	})
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	asOf := time.Now().UTC().Add(time.Second).Format(time.RFC3339)
	optimistic, err := svc.Call(context.Background(), ToolStrictAsk, map[string]any{
		"market":   "Oil market",
		"thesis":   "Escalation risk is underpriced",
		"question": "This looks great, should I buy aggressively?",
		"language": "en",
		"as_of":    asOf,
	})
	if err != nil {
		t.Fatalf("strict ask optimistic: %v", err)
	}
	fearful, err := svc.Call(context.Background(), ToolStrictAsk, map[string]any{
		"market":   "Oil market",
		"thesis":   "Escalation risk is underpriced",
		"question": "I am scared this is wrong, should I avoid it?",
		"language": "en",
		"as_of":    asOf,
	})
	if err != nil {
		t.Fatalf("strict ask fearful: %v", err)
	}
	if optimistic["answer"] != fearful["answer"] {
		t.Fatalf("question tone changed answer\noptimistic=%v\nfearful=%v", optimistic["answer"], fearful["answer"])
	}
	optEval := optimistic["evaluation"].(map[string]any)
	fearEval := fearful["evaluation"].(map[string]any)
	if optEval["verdict"] != fearEval["verdict"] {
		t.Fatalf("question tone changed verdict: %v vs %v", optEval["verdict"], fearEval["verdict"])
	}
}

func TestPreTradeCheckAllowsOnlyMatchingExecutableActionWithRiskRule(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising", "market not pricing weekend risk"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
		"position_rule":      "max 1u until settlement clarity",
	})
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	out, err := svc.Call(context.Background(), ToolPreTradeCheck, map[string]any{
		"market":          "Oil market",
		"thesis":          "Escalation risk is underpriced",
		"intended_action": "buy",
		"requested_size":  "0.5u",
	})
	if err != nil {
		t.Fatalf("pre-trade check: %v", err)
	}
	if out["gate"] != PreTradeGateAllow {
		t.Fatalf("expected allow gate, got %#v", out["gate"])
	}
	if out["execution_allowed"] != true {
		t.Fatalf("expected execution allowed, got %#v", out["execution_allowed"])
	}
}

func TestPreTradeCheckRejectsMissingRiskConstraint(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising", "market not pricing weekend risk"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
	})
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	out, err := svc.Call(context.Background(), ToolPreTradeCheck, map[string]any{
		"market":          "Oil market",
		"thesis":          "Escalation risk is underpriced",
		"intended_action": "buy",
	})
	if err != nil {
		t.Fatalf("pre-trade check: %v", err)
	}
	if out["gate"] != PreTradeGateReject {
		t.Fatalf("expected reject gate, got %#v", out["gate"])
	}
	reasons := out["reasons"].([]string)
	if !stringSliceContainsCanonical(reasons, "missing_position_or_risk_rule") {
		t.Fatalf("expected missing risk reason, got %#v", reasons)
	}
}

func TestPreTradeCheckRejectsActionMismatch(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
		"position_rule":      "max 1u until settlement clarity",
	})
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	out, err := svc.Call(context.Background(), ToolPreTradeCheck, map[string]any{
		"market":          "Oil market",
		"thesis":          "Escalation risk is underpriced",
		"intended_action": "sell",
	})
	if err != nil {
		t.Fatalf("pre-trade check: %v", err)
	}
	if out["gate"] != PreTradeGateReject {
		t.Fatalf("expected reject gate, got %#v", out["gate"])
	}
	if out["action_matched"] != false {
		t.Fatalf("expected action mismatch, got %#v", out["action_matched"])
	}
}

func TestPreTradeCheckInvalidatesTriggeredFailure(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
		"position_rule":      "max 1u until settlement clarity",
	})
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	out, err := svc.Call(context.Background(), ToolPreTradeCheck, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"intended_action":    "buy",
		"triggered_failures": []any{"confirmed ceasefire"},
	})
	if err != nil {
		t.Fatalf("pre-trade check: %v", err)
	}
	if out["gate"] != PreTradeGateInvalidated {
		t.Fatalf("expected invalidated gate, got %#v", out["gate"])
	}
	if out["execution_allowed"] != false {
		t.Fatalf("expected execution blocked, got %#v", out["execution_allowed"])
	}
}

func TestDecisionStatusListsAndFiltersRecordedTheses(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "Oil market",
		"thesis":             "Escalation risk is underpriced",
		"action":             "buy",
		"confidence":         0.72,
		"evidence":           []any{"shipping insurance rising"},
		"counter_evidence":   []any{"front-month oil already bid"},
		"failure_conditions": []any{"confirmed ceasefire"},
		"position_rule":      "max 1u until settlement clarity",
	})
	if err != nil {
		t.Fatalf("record oil decision: %v", err)
	}
	_, err = svc.Call(context.Background(), ToolRecordDecision, map[string]any{
		"market":             "AI market",
		"thesis":             "AI capex remains resilient",
		"action":             "wait",
		"confidence":         0.66,
		"evidence":           []any{"hyperscaler capex commentary remains firm"},
		"counter_evidence":   []any{"valuation already stretched"},
		"failure_conditions": []any{"guidance cut"},
	})
	if err != nil {
		t.Fatalf("record ai decision: %v", err)
	}
	out, err := svc.Call(context.Background(), ToolDecisionStatus, map[string]any{
		"triggered_failures": []any{"confirmed ceasefire"},
	})
	if err != nil {
		t.Fatalf("decision status: %v", err)
	}
	if out["count"] != 2 {
		t.Fatalf("expected 2 status items, got %#v", out["count"])
	}
	counts := out["counts"].(map[string]int)
	if counts[DecisionStatusInvalidated] != 1 {
		t.Fatalf("expected 1 invalidated thesis, counts=%#v", counts)
	}
	if counts[DecisionStatusWatch] != 1 {
		t.Fatalf("expected 1 watch thesis, counts=%#v", counts)
	}
	items := out["items"].([]map[string]any)
	if items[0]["status"] != DecisionStatusInvalidated {
		t.Fatalf("expected highest severity first, got %#v", items[0]["status"])
	}
	filtered, err := svc.Call(context.Background(), ToolDecisionStatus, map[string]any{
		"market": "AI market",
	})
	if err != nil {
		t.Fatalf("filtered decision status: %v", err)
	}
	if filtered["count"] != 1 {
		t.Fatalf("expected filtered count 1, got %#v", filtered["count"])
	}
	filteredItems := filtered["items"].([]map[string]any)
	if filteredItems[0]["market"] != "ai market" {
		t.Fatalf("unexpected filtered market: %#v", filteredItems[0]["market"])
	}
}
