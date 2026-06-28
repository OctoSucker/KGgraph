# KGgraph

`KGgraph` is a **deterministic judgment memory and reasoning discipline tool** for AI agents and humans.

Use it to:
- freeze decisions as thesis, evidence, counter-evidence, failure conditions, and reviews
- evaluate recorded decisions with deterministic rules, without calling an LLM
- expand weighted multi-hop graph paths (`A -> B -> C`) when exploratory graph context is useful

It provides:
- **CLI** (`kggraph ...`)
- **MCP stdio server** (`kggraph serve-mcp`)
- **Local graph viewer** (`kggraph graph-view`)
- **Decision records** (`kggraph record-decision`) for freezing thesis, evidence, counter-evidence, failure conditions, and review triggers
- **Deterministic decision evaluation** (`kggraph evaluate-decision`) for repeatable `follow / watch / blocked / invalidated` verdicts
- **Strict answers** (`kggraph strict-ask`) rendered from deterministic evaluation, not from a fresh LLM judgment
- **Pre-trade gates** (`kggraph pre-trade-check`) for deterministic `allow / observe_only / reject / invalidated` execution checks
- **Decision status scans** (`kggraph decision-status`) for listing all recorded theses as `usable / watch / blocked / invalidated`

For judgment-critical workflows, start with `record-decision`. Use `ingest-statement` only for low-risk graph extraction.

## Install

Homebrew (recommended):

```bash
brew tap 0xfakeSpike/tap
brew install kggraph
```

## Quick start: deterministic decision workflow

```bash
# 1) Freeze a decision. Evidence, counter-evidence, and failure conditions are required.
kggraph record-decision \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --action buy \
  --confidence 0.72 \
  --evidence-json '["shipping insurance rising", "market not pricing weekend headline risk"]' \
  --counter-evidence-json '["front-month oil already bid"]' \
  --failure-conditions-json '["confirmed ceasefire", "contract rules exclude proxy damage"]' \
  --position-rule "max 1u until settlement clarity"

# 2) Scan all recorded theses before acting.
kggraph decision-status \
  --workspace ./workspace

# 3) Ask for a fixed answer. The question wording does not change the verdict.
kggraph strict-ask \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --question "现在是不是应该买？" \
  --language zh

# 4) Run the execution gate before placing risk.
kggraph pre-trade-check \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --intended-action buy \
  --requested-size "0.5u"

# 5) If a listed failure condition becomes true, it overrides the old thesis.
kggraph pre-trade-check \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --intended-action buy \
  --triggered-failures-json '["confirmed ceasefire"]'
```

## How to use it

### Trading or decision discipline

Use this flow when the goal is stable judgment rather than open-ended brainstorming:

```text
record-decision -> decision-status -> strict-ask -> pre-trade-check -> review-decision
```

- `record-decision`: freezes the thesis, action, evidence, counter-evidence, failure conditions, and risk rule
- `decision-status`: lists every active thesis as `usable`, `watch`, `blocked`, or `invalidated`
- `strict-ask`: renders a human-readable answer from deterministic evaluation
- `pre-trade-check`: returns the execution gate: `allow`, `observe_only`, `reject`, or `invalidated`
- `review-decision`: writes the realized outcome and lessons back into the graph

### Agent or MCP usage

For an AI agent, do not ask the model to re-decide freely. Route judgment-heavy questions through the deterministic tools first:

```text
kg_decision_status
kg_evaluate_decision
kg_strict_ask
kg_pre_trade_check
kg_review_decision
```

The agent can use the returned JSON to explain the result, but it should not override `verdict`, `gate`, `blocking_reasons`, or `failure_conditions`.

### Exploratory graph usage

Use graph expansion when you want context, not execution permission:

```bash
kggraph ingest-statement \
  --workspace ./workspace \
  --statement "战争升级通常会推高原油价格，并在市场未提前消化时压制美股大盘"

kggraph expand-reasoning \
  --workspace ./workspace \
  --start-id "战争升级" \
  --max-depth 3
```

`ingest-statement` uses the LLM only as a constrained extractor. It is not the right entrypoint for trades or other high-stakes decisions.

## CLI command reference

| Command | Purpose | Calls LLM |
|---|---|---:|
| `record-decision` | Freeze a decision with evidence, counter-evidence, failure conditions, and optional risk rule | No |
| `evaluate-decision` | Recompute the deterministic verdict for one recorded thesis | No |
| `strict-ask` | Render a fixed human-readable answer from deterministic evaluation | No |
| `pre-trade-check` | Run the execution gate for an intended action | No |
| `decision-status` | List status for all recorded theses | No |
| `review-decision` | Record realized outcome, lessons, and rule updates | No |
| `ingest-statement` | Extract low-risk graph nodes/edges from natural language | Yes |
| `expand-reasoning` | Expand weighted graph paths from a start node | No |
| `add-fact-edge` | Manually add a knowledge edge | No |
| `add-skill-edge` | Manually add an executable skill/procedure edge | No |
| `upsert-node` | Manually add or update a node | No |
| `attach-edge-evidence` | Attach evidence to an existing edge | No |
| `verify-edge` | Mark an edge verified or failed | No |
| `lookup-node-exact` | Resolve an exact node id | No |
| `lookup-node-semantic` | Resolve a node by embedding similarity | Uses embeddings |
| `list-nodes` | List nodes | No |
| `list-edges` | List edges | No |
| `serve-mcp` | Start MCP stdio server | No |
| `graph-view` | Start local graph viewer | No |

## Common CLI flags

Most commands accept the same storage flags:

- `--workspace ./workspace`: stores data at `./workspace/data/knowledgegraph.sqlite`
- `--db /path/to/knowledgegraph.sqlite`: uses an explicit SQLite file
- `KG_DB_PATH=/path/to/knowledgegraph.sqlite`: environment override when `--db` is not passed

LLM and embedding flags are only needed for `ingest-statement` and semantic lookup:

- `--api-key`: OpenAI API key, or `OPENAI_API_KEY`
- `--base-url`: OpenAI-compatible API base URL, or `OPENAI_BASE_URL`
- `--embedding-model`: embedding model, or `OPENAI_EMBEDDING_MODEL`
- `--model`: ingest extraction model, or `OPENAI_MODEL`

Current facts are passed at query time and do not rewrite the graph:

- `--triggered-failures-json '["confirmed ceasefire"]'`
- `--active-counter-evidence-json '["front-month oil already bid"]'`
- `--as-of 2026-06-28T12:00:00Z`

Use these flags with `evaluate-decision`, `strict-ask`, `pre-trade-check`, and `decision-status` when market conditions change.

## Manual graph mode (optional)

If you want explicit control, add nodes/edges directly:

```bash
kggraph upsert-node --workspace ./workspace --id "战争升级" --node-type event
kggraph upsert-node --workspace ./workspace --id "原油上涨" --node-type event
kggraph add-fact-edge --workspace ./workspace --from-id "战争升级" --to-id "原油上涨" --relation-type increases_probability_of --confidence 0.72
```

## Decision discipline mode

Use `record-decision` when KGgraph is used for trading or other judgment-heavy workflows. This command refuses to write a decision unless it includes supporting evidence, counter-evidence, and failure conditions.

```bash
kggraph record-decision \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --action buy \
  --confidence 0.72 \
  --evidence-json '["shipping insurance rising", "market not pricing weekend headline risk"]' \
  --counter-evidence-json '["front-month oil already bid"]' \
  --failure-conditions-json '["confirmed ceasefire", "contract rules exclude proxy damage"]' \
  --next-triggers-json '["official military statement", "settlement source update"]' \
  --position-rule "max 1u until settlement clarity"
```

Decision records are stored in `graph_kind=decision`. To inspect the full judgment, include negative edges:

```bash
kggraph expand-reasoning \
  --workspace ./workspace \
  --graph-kind decision \
  --start-id "market:Oil escalation market" \
  --include-negative \
  --max-depth 4
```

After the trade or event resolves, write the review back into the graph. Incorrect, mixed, or invalidated outcomes require lessons.

```bash
kggraph review-decision \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --outcome incorrect \
  --realized-result "Ceasefire confirmed and oil sold off" \
  --lessons-json '["do not ignore ceasefire verification path"]' \
  --rule-updates-json '["reduce size when a listed failure condition is imminent"]'
```

To force a repeatable judgment, evaluate the recorded decision without an LLM:

```bash
kggraph evaluate-decision \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced"
```

If a listed failure condition is now true, pass it explicitly. The verdict becomes `invalidated` regardless of how persuasive the original thesis sounded.

```bash
kggraph evaluate-decision \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --triggered-failures-json '["confirmed ceasefire"]'
```

`evaluate-decision` returns structured JSON:
- `verdict`: `follow_recorded_action`, `watch`, `blocked`, or `invalidated`
- `execution_allowed`: whether the recorded executable action may be followed
- `blocking_reasons`: exact rule hits that stopped the action
- `supporting_evidence`, `counter_evidence`, `failure_conditions`, `review_history`
- deterministic scores computed from stored edge confidence, freshness, and verification history

For a human-readable answer that still cannot be swayed by wording, use `strict-ask`. The `question` is recorded for context but does not change the verdict.

```bash
kggraph strict-ask \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --question "现在是不是应该买？" \
  --language zh
```

`strict-ask` does not call an LLM. It calls the deterministic evaluator and renders the result from a fixed template.

Before placing a trade, use `pre-trade-check`. It is stricter than `strict-ask`: the intended action must match the recorded action, the thesis must not be blocked or invalidated, and an executable action must have a position rule or explicit risk plan.

```bash
kggraph pre-trade-check \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --thesis "Escalation risk is underpriced" \
  --intended-action buy \
  --requested-size "0.5u"
```

Possible gates:
- `allow`: execution is allowed only within recorded position/risk constraints
- `observe_only`: do not add risk; wait for new evidence or review trigger
- `reject`: do not execute; the action, risk plan, or evaluation is not acceptable
- `invalidated`: do not execute; the thesis must be reviewed before reuse

For a daily or pre-open scan, use `decision-status`. It does not require an intended action; it lists every active recorded thesis and its current deterministic status.

```bash
kggraph decision-status \
  --workspace ./workspace
```

You can filter to one market or pass currently triggered failure conditions:

```bash
kggraph decision-status \
  --workspace ./workspace \
  --market "Oil escalation market" \
  --triggered-failures-json '["confirmed ceasefire"]'
```

## Defaults (no extra setup needed)

By default, KGgraph auto-fills edge time fields internally:
- `observed_at = now`
- `valid_from = observed_at`
- `valid_until = null` (open-ended)
- reasoning `as_of = now`

You only need to provide extra time fields when you want strict time-window behavior.

For `ingest-statement`, KGgraph uses the LLM only as a constrained extractor:
- it must return strict JSON
- confidence means extraction certainty, not real-world truth probability
- it must not judge whether a claim is true, tradable, important, or likely
- unsupported node types, relations, polarity values, confidence values, and unknown JSON fields are rejected before write
- edge time fields are accepted only when explicitly present in the statement; internal defaults are used only when the model cannot infer them

## MCP usage

```bash
kggraph serve-mcp --workspace ./workspace
```

Main tools:
- `kg_upsert_node`
- `kg_add_fact_edge`
- `kg_add_skill_edge`
- `kg_ingest_statement`
- `kg_record_decision`
- `kg_review_decision`
- `kg_evaluate_decision`
- `kg_strict_ask`
- `kg_pre_trade_check`
- `kg_decision_status`
- `kg_expand_reasoning`
- `kg_lookup_node_exact`
- `kg_lookup_node_semantic`
- `kg_list_nodes`
- `kg_list_edges`
- `kg_attach_edge_evidence`
- `kg_verify_edge`

Example MCP client config: `examples/mcp-stdio.json`

## Graph viewer (manual refresh)

```bash
kggraph graph-view
# open http://127.0.0.1:8787
```

In the viewer, set `start-id` / `max-depth` / `graph-kind`, then click `Refresh` to reload from SQLite.

## Design boundaries

- KGgraph is not a graph database replacement. SQLite is the local persistence layer.
- KGgraph is not an unconstrained chatbot. Its decision mode is intentionally strict and rule-bound.
- LLM ingestion can still create noisy nodes/edges; use `record-decision` for judgment-critical workflows.
- Decision quality still depends on the user-supplied evidence and failure conditions.
- semantic lookup requires embeddings
- SQLite target is local/small-to-medium agent memory

## Data location

DB path resolution order:
1. `--db /path/to/knowledgegraph.sqlite`
2. `KG_DB_PATH`
3. `--workspace WORKSPACE` -> `WORKSPACE/data/knowledgegraph.sqlite`
4. User data directory:
   - macOS: `~/Library/Application Support/kggraph/knowledgegraph.sqlite`
   - Linux: `${XDG_DATA_HOME}/kggraph/knowledgegraph.sqlite` or `~/.local/share/kggraph/knowledgegraph.sqlite`
   - Windows: `%APPDATA%\kggraph\knowledgegraph.sqlite`

For agent or MCP usage, prefer `--workspace` or `--db` to avoid mixing unrelated projects in one graph.
