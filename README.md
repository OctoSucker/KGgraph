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

## What it solves

KGgraph is built around three observable LLM failure modes:

- **Hallucination**: facts and decisions are stored with sources (`source_ref`, `attach-edge-evidence`) and can be verified or failed (`verify-edge`). An edge only contributes to reasoning while valid, weighted by confidence, freshness, and verification history.
- **Sycophancy (following the user's mood or opinion)**: `strict-ask` renders a fixed verdict from recorded rules and stored facts. Question wording, tone, and pressure never change the answer, because no LLM is consulted at decision time.
- **Inconsistency across time and context**: decisions are frozen as records and re-evaluated deterministically; facts can expire (`retire-edge`), and conflicting claims are surfaced before they enter the graph (`conflict-scan`).

## Real use cases

**1. Judgment-critical decisions (trading, operations, research theses)**

```text
record-decision -> decision-status -> strict-ask -> pre-trade-check -> review-decision
```

Each decision is frozen with supporting evidence, counter-evidence, failure conditions, review triggers, and a position/risk rule. Execution gates are deterministic, so the same inputs always produce the same verdict, regardless of who asks or how the question is phrased.

**2. Agent knowledge base / research memory**

Use `ingest-statement` (LLM-assisted extraction only) for low-risk claims, run `conflict-scan` before adding a claim that may contradict existing knowledge, attach evidence and verify edges as sources are checked, and `retire-edge` stale facts so reasoning reflects what is true now. `expand-reasoning` provides multi-hop context without giving the agent execution permission.

For retrieval, `lookup-context` resolves a term by exact match first, then semantic match (when embeddings are configured), then weighted multi-hop expansion — the response includes the reached nodes, the edges on the paths, and their attached evidence. Without an embedder it degrades to exact + expansion, so it works fully offline.

Graphs are portable: `export-graph` writes nodes, edges, and evidence as one JSON payload, and `import-graph` loads it into another workspace, re-attaching evidence by edge identity instead of raw ids.

**Try it in one command** (no API keys, fully local):

```bash
make demo
# or
bash examples/demo.sh
```

The demo builds a small AI-agent-memory knowledge graph and a recorded oil-escalation decision, then walks through conflict scanning, evidence, retirement, wording-proof answers, and the pre-trade gate.

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

For reviewable ingestion, split extraction from writing:

```bash
# 1) Extract and conflict-scan without writing anything
kggraph ingest-preview \
  --workspace ./workspace \
  --statement "战争升级通常会推高原油价格，并在市场未提前消化时压制美股大盘" \
  --conflict-policy warn > preview.json

# 2) Review preview.json, then write it (re-validated and re-scanned)
kggraph ingest-confirm \
  --workspace ./workspace \
  --file preview.json \
  --source-ref "research-note-2026-08"
```

`ingest-confirm` never trusts the preview blindly: it re-validates every node/edge and re-runs the conflict scan with the selected policy before writing, so what you reviewed is exactly what gets stored.

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
| `ingest-preview` | Extract and conflict-scan without writing; returns a reviewable payload | Yes |
| `ingest-confirm` | Write a preview payload after re-validating and re-scanning conflicts | No |
| `expand-reasoning` | Expand weighted graph paths from a start node | No |
| `add-fact-edge` | Manually add a knowledge edge | No |
| `add-skill-edge` | Manually add an executable skill/procedure edge | No |
| `upsert-node` | Manually add or update a node | No |
| `attach-edge-evidence` | Attach evidence to an existing edge | No |
| `verify-edge` | Mark an edge verified or failed | No |
| `retire-edge` | Close an edge's validity window; it stops contributing after the retirement time | No |
| `reopen-edge` | Extend an edge's validity, or clear its validity end entirely | No |
| `conflict-scan` | List active edges that deterministically contradict a candidate edge | No |
| `decision-audit` | Show the full provenance timeline of a recorded decision, including attached evidence and reviews | No |
| `lookup-context` | Resolve a term into graph context (exact/semantic match + multi-hop expansion, with evidence) | Uses embeddings for semantic match |
| `export-graph` | Serialize the whole graph as JSON for backup or migration | No |
| `import-graph` | Load a graph JSON export, re-attaching evidence by edge identity | No |
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

### Write-path conflict protection

`add-fact-edge`, `add-skill-edge`, and `ingest-statement` run a deterministic conflict scan before writing. The policy is set with `--conflict-policy` (or `conflict_policy` for MCP):

- `warn` (default): conflicting edges are written and reported in the response
- `block`: the write fails if any conflicting active edge exists
- `off`: skip the scan

For decision-critical knowledge, prefer `block` so a claim that contradicts a stored, still-valid fact cannot silently enter the graph.

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

Semantic lookups (`lookup-node-semantic`, `lookup-context`) keep an in-memory embedding cache once nodes are upserted, so repeated lookups avoid re-reading and re-decoding every embedding from SQLite.

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
- `kg_ingest_preview`
- `kg_ingest_confirm`
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
- `kg_retire_edge`
- `kg_reopen_edge`
- `kg_conflict_scan`
- `kg_decision_audit`
- `kg_lookup_context`
- `kg_export_graph`
- `kg_import_graph`
- `kg_list_evidence`

Example MCP client config: `examples/mcp-stdio.json`

## Graph viewer (manual refresh)

```bash
kggraph graph-view
# open http://127.0.0.1:8787
```

In the viewer, set `start-id` / `max-depth` / `graph-kind`, then click `Refresh` to reload from SQLite.

The viewer reflects the knowledge lifecycle:

- edges are drawn as **solid** when active, **gray/dashed** when retired or expired, **amber/dashed** when scheduled, and **red/dashed** when they conflict with another edge in view;
- the stats row shows `Retired/Expired` and `Conflicts` counts;
- clicking an edge opens a detail panel with validity windows, source reference, conflicting edge ids, and up to five attached evidence rows (source ref, snippet, support/refute, weight).

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
