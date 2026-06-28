package knowledgegraph

import (
	"strings"
	"testing"
)

func TestValidateExtractedEdgeRejectsUnsupportedRelation(t *testing.T) {
	t.Parallel()
	_, err := validateExtractedEdge(extractedEdge{
		From:       "A",
		To:         "B",
		Relation:   "predicts_trade",
		Polarity:   1,
		Confidence: 0.7,
	}, DefaultIngestConfidence)
	if err == nil {
		t.Fatalf("expected unsupported relation to fail")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected not allowed error, got %v", err)
	}
}

func TestValidateExtractedEdgeUsesDefaultConfidenceForMissingValue(t *testing.T) {
	t.Parallel()
	edge, err := validateExtractedEdge(extractedEdge{
		From:     "A",
		To:       "B",
		Relation: "supports",
		Polarity: 1,
	}, 0.64)
	if err != nil {
		t.Fatalf("validate edge: %v", err)
	}
	if edge.Confidence != 0.64 {
		t.Fatalf("expected default confidence, got %v", edge.Confidence)
	}
}

func TestValidateExtractedNodeRejectsUnsupportedType(t *testing.T) {
	t.Parallel()
	_, err := validateExtractedNode(extractedNode{
		ID:       "A",
		NodeType: "trade_signal",
	})
	if err == nil {
		t.Fatalf("expected unsupported node type to fail")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected not allowed error, got %v", err)
	}
}
