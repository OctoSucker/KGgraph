package knowledgegraph

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const DecisionStatsMode = "decision_stats"

type DecisionStatsInput struct {
	Market string
	AsOf   time.Time
}

// DecisionStats aggregates recorded decision reviews into per-market and
// overall outcome statistics, win rate, and lesson reuse. It is deterministic
// and never calls an LLM.
func (s *Service) DecisionStats(ctx context.Context, in DecisionStatsInput) (map[string]any, error) {
	_ = ctx
	if s == nil || s.graph == nil {
		return nil, fmt.Errorf("knowledgegraph: decision stats: service is nil")
	}
	asOf := in.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	asOf = asOf.UTC()
	marketFilter := strings.TrimSpace(in.Market)

	rows, err := s.graph.AllEdges()
	if err != nil {
		return nil, err
	}
	type reviewInfo struct {
		reviewID   string
		market     string
		thesis     string
		outcome    string
		result     string
		lessons    []string
		rules      []string
		reviewedAt string
	}
	reviews := map[string]*reviewInfo{}
	getReview := func(id string) *reviewInfo {
		if ri, ok := reviews[id]; ok {
			return ri
		}
		ri := &reviewInfo{reviewID: id}
		reviews[id] = ri
		return ri
	}
	for _, e := range rows {
		if e.GraphKind != DecisionGraphKind {
			continue
		}
		switch e.RelationType {
		case "has_review":
			getReview(e.ToID).market = decisionNodeText(e.FromID)
		case "reviewed_by":
			ri := getReview(e.ToID)
			ri.thesis = decisionNodeText(e.FromID)
			ri.reviewedAt = e.CreatedAt.UTC().Format(time.RFC3339)
		case "resolved_as":
			getReview(e.FromID).outcome = decisionNodeText(e.ToID)
		case "realized_as":
			getReview(e.FromID).result = decisionNodeText(e.ToID)
		case "produced_lesson":
			ri := getReview(e.FromID)
			ri.lessons = append(ri.lessons, decisionNodeText(e.ToID))
		case "updates_rule":
			ri := getReview(e.FromID)
			ri.rules = append(ri.rules, decisionNodeText(e.ToID))
		}
	}

	type marketAgg struct {
		counts map[string]int
	}
	allCounts := map[string]int{}
	marketAggs := map[string]*marketAgg{}
	lessonCount := map[string]int{}
	keyLessonCount := map[string]int{}
	orderedReviews := make([]*reviewInfo, 0, len(reviews))
	for _, ri := range reviews {
		if marketFilter != "" && !strings.EqualFold(ri.market, marketFilter) {
			continue
		}
		orderedReviews = append(orderedReviews, ri)
		outcome := canonicalizeNodeID(ri.outcome)
		if outcome == "" {
			outcome = "unresolved"
		}
		allCounts[outcome]++
		agg := marketAggs[ri.market]
		if agg == nil {
			agg = &marketAgg{counts: map[string]int{}}
			marketAggs[ri.market] = agg
		}
		agg.counts[outcome]++
		for _, lesson := range ri.lessons {
			lessonCount[lesson]++
			if outcome == "incorrect" || outcome == "mixed" || outcome == "invalidated" {
				keyLessonCount[lesson]++
			}
		}
	}
	sort.Slice(orderedReviews, func(i, j int) bool {
		return orderedReviews[i].reviewedAt < orderedReviews[j].reviewedAt
	})

	overall := outcomeCountsMap(allCounts)
	markets := make([]map[string]any, 0, len(marketAggs))
	for market, agg := range marketAggs {
		markets = append(markets, map[string]any{
			"market":         market,
			"review_count":   reviewCount(agg.counts),
			"outcome_counts": outcomeCountsMap(agg.counts),
			"win_rate":       winRate(agg.counts),
		})
	}
	sort.Slice(markets, func(i, j int) bool {
		ci := mapInt64(markets[i], "review_count")
		cj := mapInt64(markets[j], "review_count")
		if ci != cj {
			return ci > cj
		}
		return mapString(markets[i], "market") < mapString(markets[j], "market")
	})
	return map[string]any{
		"mode":             DecisionStatsMode,
		"as_of":            asOf.Format(time.RFC3339),
		"market_filter":    marketFilter,
		"review_count":     len(orderedReviews),
		"outcome_counts":   overall,
		"win_rate":         winRate(allCounts),
		"key_lessons":      sortedLessonFreq(keyLessonCount, 10),
		"lesson_frequency": sortedLessonFreq(lessonCount, 10),
		"by_market":        markets,
	}, nil
}

func reviewCount(counts map[string]int) int {
	total := 0
	for _, c := range counts {
		total += c
	}
	return total
}

func outcomeCountsMap(counts map[string]int) map[string]any {
	return map[string]any{
		"correct":     counts["correct"],
		"incorrect":   counts["incorrect"],
		"mixed":       counts["mixed"],
		"invalidated": counts["invalidated"],
		"unresolved":  counts["unresolved"],
	}
}

func winRate(counts map[string]int) float64 {
	correct := counts["correct"]
	incorrect := counts["incorrect"]
	if correct+incorrect == 0 {
		return 0
	}
	return roundFloat(float64(correct)/float64(correct+incorrect), 4)
}

func sortedLessonFreq(freq map[string]int, limit int) []map[string]any {
	out := make([]map[string]any, 0, len(freq))
	for lesson, count := range freq {
		out = append(out, map[string]any{"lesson": lesson, "count": count})
	}
	sort.Slice(out, func(i, j int) bool {
		ci := int(mapFloat(out[i], "count"))
		cj := int(mapFloat(out[j], "count"))
		if ci != cj {
			return ci > cj
		}
		return mapString(out[i], "lesson") < mapString(out[j], "lesson")
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
