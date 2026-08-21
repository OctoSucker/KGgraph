package knowledgegraph

import (
	"context"
	"testing"
)

func TestDecisionStatsAggregatesReviews(t *testing.T) {
	t.Parallel()
	store := mustOpenTestStore(t)
	defer store.Close()
	svc, err := NewService(store, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := context.Background()
	record := func(market, thesis, action string) {
		t.Helper()
		if _, err := svc.Call(ctx, ToolRecordDecision, map[string]any{
			"market":             market,
			"thesis":             thesis,
			"action":             action,
			"confidence":         0.7,
			"evidence":           []any{"evidence for " + thesis},
			"counter_evidence":   []any{"counter for " + thesis},
			"failure_conditions": []any{"failure for " + thesis},
		}); err != nil {
			t.Fatalf("record %s: %v", thesis, err)
		}
	}
	review := func(market, thesis, outcome string, lessons []any) {
		t.Helper()
		if _, err := svc.Call(ctx, ToolReviewDecision, map[string]any{
			"market":          market,
			"thesis":          thesis,
			"outcome":         outcome,
			"realized_result": "result of " + thesis,
			"lessons":         lessons,
		}); err != nil {
			t.Fatalf("review %s: %v", thesis, err)
		}
	}
	record("Market A", "Thesis A1", "buy")
	review("Market A", "Thesis A1", "correct", []any{"verify sources before acting"})
	record("Market A", "Thesis A2", "buy")
	review("Market A", "Thesis A2", "incorrect", []any{"verify sources before acting", "respect failure conditions"})
	record("Market B", "Thesis B1", "hold")
	review("Market B", "Thesis B1", "invalidated", []any{"respect failure conditions"})

	out, err := svc.Call(ctx, ToolDecisionStats, map[string]any{})
	if err != nil {
		t.Fatalf("decision stats: %v", err)
	}
	if int(mapFloat(out, "review_count")) != 3 {
		t.Fatalf("expected 3 reviews, got %#v", out["review_count"])
	}
	if int(mapFloat(out, "win_rate")) != 0 {
		t.Fatalf("expected win_rate 0.5, got %#v", out["win_rate"])
	}
	counts, ok := out["outcome_counts"].(map[string]any)
	if !ok {
		t.Fatalf("expected outcome_counts map, got %#v", out["outcome_counts"])
	}
	if int(mapFloat(counts, "correct")) != 1 || int(mapFloat(counts, "incorrect")) != 1 || int(mapFloat(counts, "invalidated")) != 1 {
		t.Fatalf("unexpected outcome counts: %#v", counts)
	}
	keyLessons, ok := out["key_lessons"].([]map[string]any)
	if !ok || len(keyLessons) != 2 {
		t.Fatalf("expected 2 key lessons, got %#v", out["key_lessons"])
	}
	lessonFreq, ok := out["lesson_frequency"].([]map[string]any)
	if !ok || len(lessonFreq) != 2 {
		t.Fatalf("expected 2 lesson entries, got %#v", out["lesson_frequency"])
	}
	for _, item := range lessonFreq {
		if item["lesson"] == "verify sources before acting" && int(mapFloat(item, "count")) != 2 {
			t.Fatalf("expected lesson frequency 2, got %#v", item)
		}
	}
	byMarket, ok := out["by_market"].([]map[string]any)
	if !ok || len(byMarket) != 2 {
		t.Fatalf("expected 2 market entries, got %#v", out["by_market"])
	}
	if byMarket[0]["market"] != "market a" || int(mapFloat(byMarket[0], "review_count")) != 2 {
		t.Fatalf("unexpected market aggregate: %#v", byMarket[0])
	}

	outA, err := svc.Call(ctx, ToolDecisionStats, map[string]any{"market": "Market A"})
	if err != nil {
		t.Fatalf("decision stats filtered: %v", err)
	}
	if int(mapFloat(outA, "review_count")) != 2 {
		t.Fatalf("expected 2 reviews for Market A, got %#v", outA["review_count"])
	}
}
