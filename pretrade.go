package knowledgegraph

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	PreTradeGateAllow       = "allow"
	PreTradeGateObserveOnly = "observe_only"
	PreTradeGateReject      = "reject"
	PreTradeGateInvalidated = "invalidated"
)

type PreTradeCheckInput struct {
	Market                string
	Thesis                string
	IntendedAction        string
	RequestedSize         string
	RiskPlan              string
	AsOf                  time.Time
	TriggeredFailures     []string
	ActiveCounterEvidence []string
}

func (s *Service) PreTradeCheck(ctx context.Context, in PreTradeCheckInput) (map[string]any, error) {
	if s == nil {
		return nil, fmt.Errorf("knowledgegraph: pre-trade check: service is nil")
	}
	intendedAction := canonicalizeNodeID(in.IntendedAction)
	if intendedAction == "" {
		return nil, fmt.Errorf("knowledgegraph: pre-trade check: intended_action is required")
	}
	if !validDecisionAction(intendedAction) {
		return nil, fmt.Errorf("knowledgegraph: pre-trade check: intended_action must be buy, no-buy, hold, reduce, sell, wait, or watch")
	}
	evaluation, err := s.EvaluateDecision(ctx, DecisionEvaluateInput{
		Market:                in.Market,
		Thesis:                in.Thesis,
		AsOf:                  in.AsOf,
		TriggeredFailures:     in.TriggeredFailures,
		ActiveCounterEvidence: in.ActiveCounterEvidence,
	})
	if err != nil {
		return nil, err
	}

	recordedActions := mapStringSlice(evaluation, "recorded_actions")
	verdict := mapString(evaluation, "verdict")
	blockingReasons := append([]string{}, mapStringSlice(evaluation, "blocking_reasons")...)
	positionRules := mapItemTexts(evaluation, "position_rules", 10)
	riskPlan := strings.TrimSpace(in.RiskPlan)
	requestedSize := strings.TrimSpace(in.RequestedSize)
	actionMatched := stringSliceContainsCanonical(recordedActions, intendedAction)
	hasRiskConstraint := len(positionRules) > 0 || riskPlan != ""
	executableIntent := decisionActionExecutable([]string{intendedAction})
	recordedActionExecutable := decisionActionExecutable(recordedActions)

	gate := PreTradeGateAllow
	reasons := []string{}
	if verdict == DecisionVerdictInvalidated {
		gate = PreTradeGateInvalidated
		reasons = append(reasons, "thesis_invalidated")
	} else if verdict == DecisionVerdictBlocked {
		gate = PreTradeGateReject
		reasons = append(reasons, "decision_evaluation_blocked")
	} else if verdict == DecisionVerdictWatch {
		gate = PreTradeGateObserveOnly
		reasons = append(reasons, "decision_evaluation_requires_watch")
	}
	if !recordedActionExecutable {
		if executableIntent {
			gate = stricterPreTradeGate(gate, PreTradeGateReject)
		} else {
			gate = stricterPreTradeGate(gate, PreTradeGateObserveOnly)
		}
		reasons = append(reasons, "recorded_action_not_executable")
	}
	if !actionMatched {
		gate = stricterPreTradeGate(gate, PreTradeGateReject)
		reasons = append(reasons, "intended_action_not_supported_by_recorded_action")
	}
	if !executableIntent {
		gate = stricterPreTradeGate(gate, PreTradeGateObserveOnly)
		reasons = append(reasons, "intended_action_is_observation_only")
	}
	if executableIntent && !hasRiskConstraint {
		gate = stricterPreTradeGate(gate, PreTradeGateReject)
		reasons = append(reasons, "missing_position_or_risk_rule")
	}
	if len(blockingReasons) > 0 {
		reasons = append(reasons, blockingReasons...)
	}
	reasons = uniqueStrings(reasons)
	executionAllowed := gate == PreTradeGateAllow
	return map[string]any{
		"mode":                "pre_trade_check",
		"gate":                gate,
		"execution_allowed":   executionAllowed,
		"intended_action":     intendedAction,
		"recorded_actions":    recordedActions,
		"action_matched":      actionMatched,
		"requested_size":      requestedSize,
		"risk_plan":           riskPlan,
		"risk_plan_present":   riskPlan != "",
		"position_rules":      positionRules,
		"has_risk_constraint": hasRiskConstraint,
		"reasons":             reasons,
		"required_next_steps": preTradeNextSteps(gate, reasons),
		"evaluation":          evaluation,
	}, nil
}

func stricterPreTradeGate(current, next string) string {
	if preTradeGateRank(next) > preTradeGateRank(current) {
		return next
	}
	return current
}

func preTradeGateRank(gate string) int {
	switch gate {
	case PreTradeGateAllow:
		return 0
	case PreTradeGateObserveOnly:
		return 1
	case PreTradeGateReject:
		return 2
	case PreTradeGateInvalidated:
		return 3
	default:
		return 2
	}
}

func preTradeNextSteps(gate string, reasons []string) []string {
	switch gate {
	case PreTradeGateAllow:
		return []string{"execute only within recorded position/risk constraints", "review triggered failures before increasing size"}
	case PreTradeGateObserveOnly:
		return []string{"do not execute a new risk-increasing trade", "wait for review trigger or updated evidence"}
	case PreTradeGateInvalidated:
		return []string{"do not execute", "record a decision review before reusing this thesis"}
	default:
		steps := []string{"do not execute"}
		if stringSliceContainsCanonical(reasons, "missing_position_or_risk_rule") {
			steps = append(steps, "add an explicit position rule or risk plan")
		}
		if stringSliceContainsCanonical(reasons, "intended_action_not_supported_by_recorded_action") {
			steps = append(steps, "record a new decision if the intended action has changed")
		}
		return steps
	}
}

func stringSliceContainsCanonical(items []string, target string) bool {
	target = canonicalizeNodeID(target)
	for _, item := range items {
		if canonicalizeNodeID(item) == target {
			return true
		}
	}
	return false
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s := strings.TrimSpace(item)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
