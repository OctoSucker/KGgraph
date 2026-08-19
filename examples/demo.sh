#!/usr/bin/env bash
#
# KGgraph deterministic demo
#
# Builds a small knowledge graph (AI agent memory domain) and a recorded
# decision (oil escalation), then walks through the deterministic workflow.
# No API keys or LLM calls are needed; every command below is local.
#
# Usage:
#   bash examples/demo.sh [workspace-dir]
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${1:-$REPO_ROOT/examples/demo-workspace}"
BUILD_DIR="${TMPDIR:-/tmp}/kggraph-demo"
BIN="$BUILD_DIR/kggraph"

# Keep the demo deterministic: never call embeddings or the LLM.
export OPENAI_API_KEY=
export OPENAI_BASE_URL=
export OPENAI_EMBEDDING_MODEL=
export OPENAI_MODEL=

echo "==> Building kggraph (no network needed)"
GOCACHE="${GOCACHE:-$REPO_ROOT/.gocache}" go build -o "$BIN" ./cmd/kggraph

rm -rf "$WORKSPACE"
mkdir -p "$WORKSPACE"

kg() {
	local cmd="$1"
	shift
	"$BIN" "$cmd" --workspace "$WORKSPACE" "$@"
}

echo
echo "==> 1. Knowledge graph: AI agent memory domain"
kg add-fact-edge \
  --from-id "knowledge graph" --to-id "multi-hop reasoning" \
  --relation-type "increases_probability_of" --confidence 0.8 \
  --source-type research --source-ref "survey-2025-acl" \
  --condition-text "KG integration improves multi-hop retrieval"
kg add-fact-edge \
  --from-id "knowledge graph" --to-id "grounded answers" \
  --relation-type "supports" --confidence 0.75 \
  --source-type research --source-ref "graphrag-paper"
kg add-fact-edge \
  --from-id "knowledge graph" --to-id "audit trail" \
  --relation-type "supports" --confidence 0.7 \
  --source-type experience --source-ref "kggraph-demo"
kg add-fact-edge \
  --from-id "vector memory" --to-id "grounded answers" \
  --relation-type "supports" --confidence 0.6 \
  --source-type research --source-ref "rag-survey"

echo
echo "==> Multi-hop reasoning from \"knowledge graph\""
kg expand-reasoning \
  --start-id "knowledge graph" --max-depth 2 --max-results 6

echo
echo "==> Conflict scan: candidate claim \"knowledge graph decreases multi-hop reasoning\""
kg conflict-scan \
  --from-id "knowledge graph" --to-id "multi-hop reasoning" \
  --relation-type "increases_probability_of" --polarity -1

echo
echo "==> Write-path protection: block policy rejects a contradicting claim"
if kg add-fact-edge --conflict-policy block \
  --from-id "knowledge graph" --to-id "multi-hop reasoning" \
  --relation-type "decreases_probability_of" --confidence 0.8; then
  echo "unexpected: block policy allowed a conflicting edge"
else
  echo "(block policy rejected the conflicting edge as expected)"
fi

echo
echo "==> Provenance: attach evidence to an edge, then verify it"
EDGE_ID="$(kg list-edges | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d["edges"][0]["id"])')"
kg attach-edge-evidence \
  --edge-id "$EDGE_ID" --supports true --weight 1.0 \
  --source-type research --source-ref "survey-2025-acl" \
  --snippet "KG integration can significantly improve LLM performance on benchmark datasets"
kg verify-edge --edge-id "$EDGE_ID" --success true

echo
echo "==> Fact lifecycle: retire a stale edge and confirm it leaves reasoning"
STALE_EDGE="$(kg list-edges | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d["edges"][1]["id"])')"
kg retire-edge --edge-id "$STALE_EDGE"
kg expand-reasoning \
  --start-id "knowledge graph" --max-depth 2 --max-results 6

echo
echo "==> Fact lifecycle: reopen the retired edge and confirm it returns"
kg reopen-edge --edge-id "$STALE_EDGE" --open-ended
kg expand-reasoning \
  --start-id "knowledge graph" --max-depth 2 --max-results 6

echo
echo "==> Hybrid retrieval: lookup-context (exact match + multi-hop expansion)"
kg lookup-context \
  --term "knowledge graph" --max-depth 2 --max-results 6

echo
echo "==> Graph portability: export and import into a fresh workspace"
kg export-graph > "$BUILD_DIR/export.json"
"$BIN" import-graph --workspace "$WORKSPACE-copy" --file "$BUILD_DIR/export.json"

echo
echo "==> 2. Decision discipline: oil escalation market"
kg record-decision \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --action buy --confidence 0.72 \
  --evidence-json '["shipping insurance rising", "market not pricing weekend headline risk"]' \
  --counter-evidence-json '["front-month oil already bid"]' \
  --failure-conditions-json '["confirmed ceasefire"]' \
  --next-triggers-json '["official military statement"]' \
  --position-rule "max 1u until settlement clarity"

echo
echo "==> Decision status"
kg decision-status

echo
echo "==> Strict ask is wording-proof (two different questions, same verdict)"
kg strict-ask \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --question "现在是不是应该买？" --language zh
kg strict-ask \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --question "Are you sure buying is a good idea?" --language en

echo
echo "==> Pre-trade gate"
kg pre-trade-check \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --intended-action buy --requested-size "0.5u" --risk-plan "stop at -0.15u"

echo
echo "==> Failure condition triggers invalidation"
kg pre-trade-check \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --intended-action buy --requested-size "0.5u" \
  --triggered-failures-json '["confirmed ceasefire"]'

echo
echo "==> Decision audit: full provenance timeline"
kg decision-audit \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced"

echo
echo "Demo finished. Workspace: $WORKSPACE"
echo "Explore it with: kggraph graph-view --workspace $WORKSPACE"
