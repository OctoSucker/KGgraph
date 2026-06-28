package knowledgegraph

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const StrictAskMode = "strict_decision_answer"

type StrictAskInput struct {
	Market                string
	Thesis                string
	Question              string
	Language              string
	AsOf                  time.Time
	TriggeredFailures     []string
	ActiveCounterEvidence []string
}

func (s *Service) StrictAsk(ctx context.Context, in StrictAskInput) (map[string]any, error) {
	if s == nil {
		return nil, fmt.Errorf("knowledgegraph: strict ask: service is nil")
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
	language := canonicalizeNodeID(in.Language)
	if language == "" {
		language = "en"
	}
	answer := renderStrictAnswer(evaluation, language)
	return map[string]any{
		"mode":       StrictAskMode,
		"question":   strings.TrimSpace(in.Question),
		"answer":     answer,
		"evaluation": evaluation,
		"guardrails": []string{
			"question text is not used to change the verdict",
			"no LLM is called by strict-ask",
			"answer is rendered from deterministic evaluation output",
		},
	}, nil
}

func renderStrictAnswer(eval map[string]any, language string) string {
	verdict := mapString(eval, "verdict")
	executionAllowed := mapBool(eval, "execution_allowed")
	actions := mapStringSlice(eval, "recorded_actions")
	blockingReasons := mapStringSlice(eval, "blocking_reasons")
	conf := mapFloat(eval, "deterministic_confidence")
	counterRatio := mapFloat(eval, "counter_ratio")
	evidence := mapItemTexts(eval, "supporting_evidence", 3)
	counter := mapItemTexts(eval, "counter_evidence", 3)
	failures := mapItemTexts(eval, "failure_conditions", 3)
	positionRules := mapItemTexts(eval, "position_rules", 2)

	if language == "zh" || language == "cn" || language == "zh-cn" {
		var b strings.Builder
		b.WriteString("结论: ")
		b.WriteString(localizedVerdict(verdict, language))
		if executionAllowed {
			b.WriteString("，允许执行已记录动作")
		} else {
			b.WriteString("，不允许执行已记录动作")
		}
		if len(actions) > 0 {
			b.WriteString("。已记录动作: ")
			b.WriteString(strings.Join(actions, ", "))
		}
		b.WriteString(fmt.Sprintf("。确定性置信度: %.4f，反证比例: %.4f", conf, counterRatio))
		if len(blockingReasons) > 0 {
			b.WriteString("。阻断原因: ")
			b.WriteString(strings.Join(blockingReasons, "; "))
		}
		appendChineseSentence(&b, "支持证据", evidence)
		appendChineseSentence(&b, "反证", counter)
		appendChineseSentence(&b, "失效条件", failures)
		appendChineseSentence(&b, "仓位/执行约束", positionRules)
		b.WriteString("。纪律: 这是基于已固化图谱和固定规则的输出，不是新的 LLM 主观判断。")
		return b.String()
	}

	var b strings.Builder
	b.WriteString("Verdict: ")
	b.WriteString(verdict)
	if executionAllowed {
		b.WriteString("; execution allowed for the recorded action")
	} else {
		b.WriteString("; execution not allowed for the recorded action")
	}
	if len(actions) > 0 {
		b.WriteString(". Recorded action: ")
		b.WriteString(strings.Join(actions, ", "))
	}
	b.WriteString(fmt.Sprintf(". Deterministic confidence: %.4f; counter ratio: %.4f", conf, counterRatio))
	if len(blockingReasons) > 0 {
		b.WriteString(". Blocking reasons: ")
		b.WriteString(strings.Join(blockingReasons, "; "))
	}
	appendSentence(&b, "Supporting evidence", evidence)
	appendSentence(&b, "Counter-evidence", counter)
	appendSentence(&b, "Failure conditions", failures)
	appendSentence(&b, "Position rules", positionRules)
	b.WriteString(". Discipline: this is rendered from stored graph facts and fixed rules, not a fresh LLM judgment.")
	return b.String()
}

func appendSentence(b *strings.Builder, label string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString(". ")
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(strings.Join(items, "; "))
}

func appendChineseSentence(b *strings.Builder, label string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("。")
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(strings.Join(items, "; "))
}

func localizedVerdict(verdict, language string) string {
	if language != "zh" && language != "cn" && language != "zh-cn" {
		return verdict
	}
	switch verdict {
	case DecisionVerdictFollow:
		return "跟随已记录动作"
	case DecisionVerdictWatch:
		return "只观察"
	case DecisionVerdictBlocked:
		return "阻断"
	case DecisionVerdictInvalidated:
		return "已失效"
	default:
		return verdict
	}
}

func mapString(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func mapBool(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func mapFloat(m map[string]any, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	default:
		return 0
	}
}

func mapStringSlice(m map[string]any, key string) []string {
	switch items := m[key].(type) {
	case []string:
		out := make([]string, len(items))
		copy(out, items)
		return out
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

func mapItemTexts(m map[string]any, key string, maxItems int) []string {
	items, ok := m[key].([]map[string]any)
	if !ok {
		return nil
	}
	if maxItems <= 0 || maxItems > len(items) {
		maxItems = len(items)
	}
	out := make([]string, 0, maxItems)
	for _, item := range items[:maxItems] {
		if text, ok := item["text"].(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}
