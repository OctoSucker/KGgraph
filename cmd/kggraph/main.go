package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	kggraph "github.com/OctoSucker/KGgraph"
)

type commonFlags struct {
	workspace      string
	dbPath         string
	baseURL        string
	apiKey         string
	embeddingModel string
}

func main() {
	if len(os.Args) < 2 {
		writeJSONAndExit(2, map[string]any{"error": usage()})
	}
	ctx := context.Background()
	cmd := strings.TrimSpace(os.Args[1])
	args := os.Args[2:]

	switch cmd {
	case "upsert-node":
		runUpsertNode(ctx, args)
	case "add-fact-edge":
		runAddEdge(ctx, args, kggraph.ToolAddFactEdge)
	case "add-skill-edge":
		runAddEdge(ctx, args, kggraph.ToolAddSkillEdge)
	case "ingest-statement":
		runIngestStatement(ctx, args)
	case "ingest-preview":
		runIngestPreview(ctx, args)
	case "ingest-confirm":
		runIngestConfirm(ctx, args)
	case "record-decision":
		runRecordDecision(ctx, args)
	case "review-decision":
		runReviewDecision(ctx, args)
	case "evaluate-decision":
		runEvaluateDecision(ctx, args)
	case "strict-ask":
		runStrictAsk(ctx, args)
	case "pre-trade-check":
		runPreTradeCheck(ctx, args)
	case "decision-status":
		runDecisionStatus(ctx, args)
	case "decision-stats":
		runDecisionStats(ctx, args)
	case "attach-edge-evidence":
		runAttachEdgeEvidence(ctx, args)
	case "verify-edge":
		runVerifyEdge(ctx, args)
	case "retire-edge":
		runRetireEdge(ctx, args)
	case "reopen-edge":
		runReopenEdge(ctx, args)
	case "conflict-scan":
		runConflictScan(ctx, args)
	case "decision-audit":
		runDecisionAudit(ctx, args)
	case "lookup-context":
		runLookupContext(ctx, args)
	case "export-graph":
		runExportGraph(ctx, args)
	case "import-graph":
		runImportGraph(ctx, args)
	case "expand-reasoning":
		runExpandReasoning(ctx, args)
	case "lookup-node-exact":
		runLookup(ctx, args, kggraph.ToolLookupNodeExact)
	case "lookup-node-semantic":
		runLookup(ctx, args, kggraph.ToolLookupNodeSemantic)
	case "list-nodes":
		runList(ctx, args, kggraph.ToolListNodes)
	case "list-edges":
		runList(ctx, args, kggraph.ToolListEdges)
	case "call":
		runCall(ctx, args)
	case "serve-mcp":
		runServeMCP(ctx, args)
	case "graph-view":
		runGraphView(ctx, args)
	case "source":
		runSource(args)
	case "-h", "--help", "help":
		writeJSONAndExit(0, map[string]any{"usage": usage()})
	default:
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("unknown command %q", cmd), "usage": usage()})
	}
}

func usage() string {
	return "commands: upsert-node, add-fact-edge, add-skill-edge, ingest-statement, ingest-preview, ingest-confirm, record-decision, review-decision, evaluate-decision, strict-ask, pre-trade-check, decision-status, decision-stats, attach-edge-evidence, verify-edge, retire-edge, reopen-edge, conflict-scan, decision-audit, lookup-context, export-graph, import-graph, expand-reasoning, lookup-node-exact, lookup-node-semantic, list-nodes, list-edges, call, serve-mcp, graph-view, source (init|add|search|fetch|list|verify|clean)"
}

func addCommonFlags(fs *flag.FlagSet, c *commonFlags) {
	fs.StringVar(&c.workspace, "workspace", "", "workspace root containing data/knowledgegraph.sqlite")
	fs.StringVar(&c.dbPath, "db", os.Getenv("KG_DB_PATH"), "direct sqlite file path")
	fs.StringVar(&c.baseURL, "base-url", getenvDefault("OPENAI_BASE_URL", ""), "OpenAI base URL")
	fs.StringVar(&c.apiKey, "api-key", getenvDefault("OPENAI_API_KEY", ""), "OpenAI API key")
	fs.StringVar(&c.embeddingModel, "embedding-model", getenvDefault("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"), "OpenAI embedding model")
}

func openService(cfg commonFlags) (*kggraph.Service, error) {
	store, err := kggraph.OpenStore(kggraph.StoreConfig{
		WorkspaceRoot: cfg.workspace,
		DBPath:        cfg.dbPath,
	})
	if err != nil {
		return nil, err
	}
	var embedder kggraph.Embedder
	if strings.TrimSpace(cfg.embeddingModel) != "" && (strings.TrimSpace(cfg.apiKey) != "" || strings.TrimSpace(cfg.baseURL) != "") {
		embedder = kggraph.NewOpenAIEmbedder(kggraph.OpenAIConfig{
			BaseURL:        cfg.baseURL,
			APIKey:         cfg.apiKey,
			EmbeddingModel: cfg.embeddingModel,
		})
	}
	return kggraph.NewService(store, embedder)
}

func runUpsertNode(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("upsert-node", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var id, nodeType, status string
	var aliasesJSON string
	fs.StringVar(&id, "id", "", "node id")
	fs.StringVar(&nodeType, "node-type", "entity", "node type")
	fs.StringVar(&status, "status", "active", "node status")
	fs.StringVar(&aliasesJSON, "aliases-json", "[]", "JSON array of aliases")
	mustParse(fs, argv)
	var aliases []string
	if err := json.Unmarshal([]byte(aliasesJSON), &aliases); err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("parse aliases-json: %v", err)})
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolUpsertNode, map[string]any{
		"id":        id,
		"node_type": nodeType,
		"status":    status,
		"aliases":   anySlice(aliases),
	})
	writeResult(err, out)
}

func runAddEdge(ctx context.Context, argv []string, tool string) {
	fs := flag.NewFlagSet(tool, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var fromID, toID, relationType, conditionText, sourceType, sourceRef, observedAt, validFrom, validUntil, expiresAt, activationRule string
	var conflictPolicy string
	var polarity, decayHalfLifeDays int
	var confidence float64
	fs.StringVar(&fromID, "from-id", "", "source node id")
	fs.StringVar(&toID, "to-id", "", "target node id")
	fs.StringVar(&relationType, "relation-type", "", "relation type")
	fs.IntVar(&polarity, "polarity", 1, "relation polarity: -1, 0, 1")
	fs.Float64Var(&confidence, "confidence", 0.6, "edge confidence in [0,1]")
	fs.StringVar(&conditionText, "condition-text", "", "condition text")
	fs.StringVar(&sourceType, "source-type", "", "source type")
	fs.StringVar(&sourceRef, "source-ref", "", "source reference")
	fs.StringVar(&observedAt, "observed-at", "", "optional RFC3339 observed timestamp")
	fs.StringVar(&validFrom, "valid-from", "", "optional RFC3339 validity start")
	fs.StringVar(&validUntil, "valid-until", "", "optional RFC3339 validity end")
	fs.IntVar(&decayHalfLifeDays, "decay-half-life-days", 30, "time decay half-life in days")
	fs.StringVar(&expiresAt, "expires-at", "", "optional RFC3339 expiration")
	fs.StringVar(&activationRule, "activation-rule", "", "skill activation rule (required for add-skill-edge)")
	fs.StringVar(&conflictPolicy, "conflict-policy", "warn", "block, warn, or off")
	mustParse(fs, argv)
	args := map[string]any{
		"from_id":              fromID,
		"to_id":                toID,
		"relation_type":        relationType,
		"polarity":             polarity,
		"confidence":           confidence,
		"condition_text":       conditionText,
		"source_type":          sourceType,
		"source_ref":           sourceRef,
		"decay_half_life_days": decayHalfLifeDays,
		"conflict_policy":      conflictPolicy,
	}
	if strings.TrimSpace(expiresAt) != "" {
		args["expires_at"] = expiresAt
	}
	if strings.TrimSpace(observedAt) != "" {
		args["observed_at"] = observedAt
	}
	if strings.TrimSpace(validFrom) != "" {
		args["valid_from"] = validFrom
	}
	if strings.TrimSpace(validUntil) != "" {
		args["valid_until"] = validUntil
	}
	if tool == kggraph.ToolAddSkillEdge {
		args["activation_rule"] = activationRule
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, tool, args)
	writeResult(err, out)
}

func runIngestStatement(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("ingest-statement", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var statement, graphKind, sourceType, sourceRef, model string
	var conflictPolicy string
	var defaultConfidence float64
	fs.StringVar(&statement, "statement", "", "natural-language statement to ingest")
	fs.StringVar(&graphKind, "graph-kind", "knowledge", "graph kind filter")
	fs.StringVar(&sourceType, "source-type", "llm_extracted", "source type")
	fs.StringVar(&sourceRef, "source-ref", "", "source ref")
	fs.StringVar(&model, "model", getenvDefault("OPENAI_MODEL", kggraph.DefaultIngestModel), "LLM model used for extraction")
	fs.Float64Var(&defaultConfidence, "default-confidence", kggraph.DefaultIngestConfidence, "fallback confidence when LLM is uncertain")
	fs.StringVar(&conflictPolicy, "conflict-policy", "warn", "block, warn, or off")
	mustParse(fs, argv)
	args := map[string]any{
		"statement":          statement,
		"graph_kind":         graphKind,
		"source_type":        sourceType,
		"source_ref":         sourceRef,
		"model":              model,
		"default_confidence": defaultConfidence,
		"conflict_policy":    conflictPolicy,
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolIngestStatement, args)
	writeResult(err, out)
}

func runIngestPreview(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("ingest-preview", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var statement, graphKind, sourceType, sourceRef, model string
	var conflictPolicy string
	var defaultConfidence float64
	fs.StringVar(&statement, "statement", "", "natural-language statement to preview")
	fs.StringVar(&graphKind, "graph-kind", "knowledge", "graph kind filter")
	fs.StringVar(&sourceType, "source-type", "llm_extracted", "source type")
	fs.StringVar(&sourceRef, "source-ref", "", "source ref")
	fs.StringVar(&model, "model", getenvDefault("OPENAI_MODEL", kggraph.DefaultIngestModel), "LLM model used for extraction")
	fs.Float64Var(&defaultConfidence, "default-confidence", kggraph.DefaultIngestConfidence, "fallback confidence when LLM is uncertain")
	fs.StringVar(&conflictPolicy, "conflict-policy", "warn", "block, warn, or off")
	mustParse(fs, argv)
	args := map[string]any{
		"statement":          statement,
		"graph_kind":         graphKind,
		"source_type":        sourceType,
		"source_ref":         sourceRef,
		"model":              model,
		"default_confidence": defaultConfidence,
		"conflict_policy":    conflictPolicy,
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolIngestPreview, args)
	writeResult(err, out)
}

func runIngestConfirm(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("ingest-confirm", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var previewJSON, file, graphKind, sourceType, sourceRef, conflictPolicy string
	fs.StringVar(&previewJSON, "preview-json", "", "preview payload JSON from ingest-preview")
	fs.StringVar(&file, "file", "", "path to preview payload JSON file")
	fs.StringVar(&graphKind, "graph-kind", "knowledge", "graph kind")
	fs.StringVar(&sourceType, "source-type", "llm_extracted", "source type")
	fs.StringVar(&sourceRef, "source-ref", "", "source ref")
	fs.StringVar(&conflictPolicy, "conflict-policy", "warn", "block, warn, or off")
	mustParse(fs, argv)
	var raw []byte
	if strings.TrimSpace(file) != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("ingest-confirm: read file: %v", err)})
		}
		raw = data
	} else if strings.TrimSpace(previewJSON) != "" {
		raw = []byte(previewJSON)
	} else {
		writeJSONAndExit(2, map[string]any{"error": "ingest-confirm: --preview-json or --file is required"})
	}
	var preview map[string]any
	if err := json.Unmarshal(raw, &preview); err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("ingest-confirm: parse preview: %v", err)})
	}
	args := map[string]any{
		"preview":         preview,
		"graph_kind":      graphKind,
		"source_type":     sourceType,
		"source_ref":      sourceRef,
		"conflict_policy": conflictPolicy,
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolIngestConfirm, args)
	writeResult(err, out)
}

func runRecordDecision(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("record-decision", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var market, thesis, action, evidenceJSON, counterEvidenceJSON, failureConditionsJSON, nextTriggersJSON, positionRule, sourceRef string
	var confidence float64
	fs.StringVar(&market, "market", "", "market or decision object")
	fs.StringVar(&thesis, "thesis", "", "decision thesis to freeze")
	fs.StringVar(&action, "action", "", "buy, no-buy, hold, reduce, sell, wait, or watch")
	fs.Float64Var(&confidence, "confidence", kggraph.DefaultIngestConfidence, "decision confidence in [0,1]")
	fs.StringVar(&evidenceJSON, "evidence-json", "[]", "JSON array of supporting evidence strings")
	fs.StringVar(&counterEvidenceJSON, "counter-evidence-json", "[]", "JSON array of counter-evidence strings")
	fs.StringVar(&failureConditionsJSON, "failure-conditions-json", "[]", "JSON array of thesis failure conditions")
	fs.StringVar(&nextTriggersJSON, "next-triggers-json", "[]", "JSON array of review trigger strings")
	fs.StringVar(&positionRule, "position-rule", "", "position sizing or risk rule")
	fs.StringVar(&sourceRef, "source-ref", "", "source reference")
	mustParse(fs, argv)
	evidence := mustParseStringArrayFlag("evidence-json", evidenceJSON)
	counterEvidence := mustParseStringArrayFlag("counter-evidence-json", counterEvidenceJSON)
	failureConditions := mustParseStringArrayFlag("failure-conditions-json", failureConditionsJSON)
	nextTriggers := mustParseStringArrayFlag("next-triggers-json", nextTriggersJSON)
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolRecordDecision, map[string]any{
		"market":             market,
		"thesis":             thesis,
		"action":             action,
		"confidence":         confidence,
		"evidence":           anySlice(evidence),
		"counter_evidence":   anySlice(counterEvidence),
		"failure_conditions": anySlice(failureConditions),
		"next_triggers":      anySlice(nextTriggers),
		"position_rule":      positionRule,
		"source_ref":         sourceRef,
	})
	writeResult(err, out)
}

func runReviewDecision(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("review-decision", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var market, thesis, outcome, realizedResult, lessonsJSON, ruleUpdatesJSON, sourceRef string
	fs.StringVar(&market, "market", "", "market or decision object")
	fs.StringVar(&thesis, "thesis", "", "original thesis being reviewed")
	fs.StringVar(&outcome, "outcome", "", "correct, incorrect, mixed, invalidated, or unresolved")
	fs.StringVar(&realizedResult, "realized-result", "", "what actually happened")
	fs.StringVar(&lessonsJSON, "lessons-json", "[]", "JSON array of review lessons")
	fs.StringVar(&ruleUpdatesJSON, "rule-updates-json", "[]", "JSON array of rule updates")
	fs.StringVar(&sourceRef, "source-ref", "", "source reference")
	mustParse(fs, argv)
	lessons := mustParseStringArrayFlag("lessons-json", lessonsJSON)
	ruleUpdates := mustParseStringArrayFlag("rule-updates-json", ruleUpdatesJSON)
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolReviewDecision, map[string]any{
		"market":          market,
		"thesis":          thesis,
		"outcome":         outcome,
		"realized_result": realizedResult,
		"lessons":         anySlice(lessons),
		"rule_updates":    anySlice(ruleUpdates),
		"source_ref":      sourceRef,
	})
	writeResult(err, out)
}

func runEvaluateDecision(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("evaluate-decision", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var market, thesis, asOf, triggeredFailuresJSON, activeCounterEvidenceJSON string
	fs.StringVar(&market, "market", "", "market or decision object")
	fs.StringVar(&thesis, "thesis", "", "recorded thesis to evaluate")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 query time")
	fs.StringVar(&triggeredFailuresJSON, "triggered-failures-json", "[]", "JSON array of currently triggered failure conditions")
	fs.StringVar(&activeCounterEvidenceJSON, "active-counter-evidence-json", "[]", "JSON array of currently active counter-evidence")
	mustParse(fs, argv)
	triggeredFailures := mustParseStringArrayFlag("triggered-failures-json", triggeredFailuresJSON)
	activeCounterEvidence := mustParseStringArrayFlag("active-counter-evidence-json", activeCounterEvidenceJSON)
	args := map[string]any{
		"market":                  market,
		"thesis":                  thesis,
		"triggered_failures":      anySlice(triggeredFailures),
		"active_counter_evidence": anySlice(activeCounterEvidence),
	}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolEvaluateDecision, args)
	writeResult(err, out)
}

func runStrictAsk(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("strict-ask", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var market, thesis, question, language, asOf, triggeredFailuresJSON, activeCounterEvidenceJSON string
	fs.StringVar(&market, "market", "", "market or decision object")
	fs.StringVar(&thesis, "thesis", "", "recorded thesis to evaluate")
	fs.StringVar(&question, "question", "", "user-facing question; does not affect verdict")
	fs.StringVar(&language, "language", "en", "answer language: en or zh")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 query time")
	fs.StringVar(&triggeredFailuresJSON, "triggered-failures-json", "[]", "JSON array of currently triggered failure conditions")
	fs.StringVar(&activeCounterEvidenceJSON, "active-counter-evidence-json", "[]", "JSON array of currently active counter-evidence")
	mustParse(fs, argv)
	triggeredFailures := mustParseStringArrayFlag("triggered-failures-json", triggeredFailuresJSON)
	activeCounterEvidence := mustParseStringArrayFlag("active-counter-evidence-json", activeCounterEvidenceJSON)
	args := map[string]any{
		"market":                  market,
		"thesis":                  thesis,
		"question":                question,
		"language":                language,
		"triggered_failures":      anySlice(triggeredFailures),
		"active_counter_evidence": anySlice(activeCounterEvidence),
	}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolStrictAsk, args)
	writeResult(err, out)
}

func runPreTradeCheck(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("pre-trade-check", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var market, thesis, intendedAction, requestedSize, riskPlan, asOf, triggeredFailuresJSON, activeCounterEvidenceJSON string
	fs.StringVar(&market, "market", "", "market or decision object")
	fs.StringVar(&thesis, "thesis", "", "recorded thesis to check")
	fs.StringVar(&intendedAction, "intended-action", "", "buy, no-buy, hold, reduce, sell, wait, or watch")
	fs.StringVar(&requestedSize, "requested-size", "", "proposed size, unit, or exposure")
	fs.StringVar(&riskPlan, "risk-plan", "", "explicit risk limit or execution plan")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 query time")
	fs.StringVar(&triggeredFailuresJSON, "triggered-failures-json", "[]", "JSON array of currently triggered failure conditions")
	fs.StringVar(&activeCounterEvidenceJSON, "active-counter-evidence-json", "[]", "JSON array of currently active counter-evidence")
	mustParse(fs, argv)
	triggeredFailures := mustParseStringArrayFlag("triggered-failures-json", triggeredFailuresJSON)
	activeCounterEvidence := mustParseStringArrayFlag("active-counter-evidence-json", activeCounterEvidenceJSON)
	args := map[string]any{
		"market":                  market,
		"thesis":                  thesis,
		"intended_action":         intendedAction,
		"requested_size":          requestedSize,
		"risk_plan":               riskPlan,
		"triggered_failures":      anySlice(triggeredFailures),
		"active_counter_evidence": anySlice(activeCounterEvidence),
	}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolPreTradeCheck, args)
	writeResult(err, out)
}

func runDecisionStatus(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("decision-status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var market, asOf, triggeredFailuresJSON, activeCounterEvidenceJSON string
	var includeEvaluation bool
	fs.StringVar(&market, "market", "", "optional market filter")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 query time")
	fs.StringVar(&triggeredFailuresJSON, "triggered-failures-json", "[]", "JSON array of currently triggered failure conditions")
	fs.StringVar(&activeCounterEvidenceJSON, "active-counter-evidence-json", "[]", "JSON array of currently active counter-evidence")
	fs.BoolVar(&includeEvaluation, "include-evaluation", false, "include full deterministic evaluation payload")
	mustParse(fs, argv)
	triggeredFailures := mustParseStringArrayFlag("triggered-failures-json", triggeredFailuresJSON)
	activeCounterEvidence := mustParseStringArrayFlag("active-counter-evidence-json", activeCounterEvidenceJSON)
	args := map[string]any{
		"market":                  market,
		"triggered_failures":      anySlice(triggeredFailures),
		"active_counter_evidence": anySlice(activeCounterEvidence),
		"include_evaluation":      includeEvaluation,
	}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolDecisionStatus, args)
	writeResult(err, out)
}

func runDecisionStats(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("decision-stats", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var market, asOf string
	fs.StringVar(&market, "market", "", "optional market filter")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 query time")
	mustParse(fs, argv)
	args := map[string]any{"market": market}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolDecisionStats, args)
	writeResult(err, out)
}

func runAttachEdgeEvidence(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("attach-edge-evidence", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var edgeID int64
	var sourceType, sourceRef, snippet, observedAt string
	var supports bool
	var weight float64
	fs.Int64Var(&edgeID, "edge-id", 0, "edge id")
	fs.StringVar(&sourceType, "source-type", "", "source type")
	fs.StringVar(&sourceRef, "source-ref", "", "source ref")
	fs.StringVar(&snippet, "snippet", "", "evidence snippet")
	fs.Func("supports", "whether evidence supports the edge (true/false)", func(v string) error {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("supports must be true or false: %v", err)
		}
		supports = b
		return nil
	})
	fs.Float64Var(&weight, "weight", 1.0, "evidence weight")
	fs.StringVar(&observedAt, "observed-at", "", "optional RFC3339 observed timestamp")
	mustParse(fs, argv)
	args := map[string]any{
		"edge_id":     edgeID,
		"source_type": sourceType,
		"source_ref":  sourceRef,
		"snippet":     snippet,
		"supports":    supports,
		"weight":      weight,
	}
	if strings.TrimSpace(observedAt) != "" {
		args["observed_at"] = observedAt
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolAttachEdgeEvidence, args)
	writeResult(err, out)
}

func runVerifyEdge(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("verify-edge", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var edgeID int64
	var success bool
	var confidence float64
	var setConfidence bool
	var verifiedAt string
	fs.Int64Var(&edgeID, "edge-id", 0, "edge id")
	fs.Func("success", "verification result (true/false)", func(v string) error {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("success must be true or false: %v", err)
		}
		success = b
		return nil
	})
	fs.Float64Var(&confidence, "confidence", 0, "optional confidence value")
	fs.BoolVar(&setConfidence, "set-confidence", false, "whether to update confidence")
	fs.StringVar(&verifiedAt, "verified-at", "", "optional RFC3339 verified timestamp")
	mustParse(fs, argv)
	args := map[string]any{"edge_id": edgeID, "success": success}
	if setConfidence {
		args["confidence"] = confidence
	}
	if strings.TrimSpace(verifiedAt) != "" {
		args["verified_at"] = verifiedAt
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolVerifyEdge, args)
	writeResult(err, out)
}

func runRetireEdge(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("retire-edge", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var edgeID int64
	var asOf string
	fs.Int64Var(&edgeID, "edge-id", 0, "edge id to retire")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 retirement time; defaults to now")
	mustParse(fs, argv)
	args := map[string]any{"edge_id": edgeID}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolRetireEdge, args)
	writeResult(err, out)
}

func runReopenEdge(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("reopen-edge", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var edgeID int64
	var asOf string
	var openEnded bool
	fs.Int64Var(&edgeID, "edge-id", 0, "edge id to reopen")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 time the edge should be valid through; defaults to now")
	fs.BoolVar(&openEnded, "open-ended", false, "clear the validity end entirely")
	mustParse(fs, argv)
	args := map[string]any{"edge_id": edgeID, "open_ended": openEnded}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolReopenEdge, args)
	writeResult(err, out)
}

func runConflictScan(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("conflict-scan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var fromID, toID, relationType, graphKind, asOf string
	var polarity int
	fs.StringVar(&fromID, "from-id", "", "source node id of the candidate edge")
	fs.StringVar(&toID, "to-id", "", "target node id of the candidate edge")
	fs.StringVar(&relationType, "relation-type", "", "candidate relation type")
	fs.IntVar(&polarity, "polarity", 1, "candidate polarity: -1, 0, 1")
	fs.StringVar(&graphKind, "graph-kind", "knowledge", "graph kind filter")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 query time")
	mustParse(fs, argv)
	args := map[string]any{
		"from_id":       fromID,
		"to_id":         toID,
		"relation_type": relationType,
		"polarity":      polarity,
		"graph_kind":    graphKind,
	}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolConflictScan, args)
	writeResult(err, out)
}

func runDecisionAudit(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("decision-audit", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var market, thesis, asOf string
	fs.StringVar(&market, "market", "", "market or decision object")
	fs.StringVar(&thesis, "thesis", "", "recorded thesis to audit")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 query time")
	mustParse(fs, argv)
	args := map[string]any{"market": market, "thesis": thesis}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolDecisionAudit, args)
	writeResult(err, out)
}

func runLookupContext(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("lookup-context", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var term, graphKind, asOf string
	var maxDepth, maxBranch, maxResults int
	var minScore float64
	var includeNegative bool
	fs.StringVar(&term, "term", "", "term to resolve into graph context")
	fs.StringVar(&graphKind, "graph-kind", "knowledge", "graph kind filter")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 query time")
	fs.IntVar(&maxDepth, "max-depth", 3, "max reasoning depth")
	fs.IntVar(&maxBranch, "max-branch", 5, "max outgoing edges per step")
	fs.IntVar(&maxResults, "max-results", 10, "max context nodes")
	fs.Float64Var(&minScore, "min-score", 0, "minimum propagated score to keep")
	fs.BoolVar(&includeNegative, "include-negative", false, "whether to include negative-polarity edges")
	mustParse(fs, argv)
	args := map[string]any{
		"term":             term,
		"graph_kind":       graphKind,
		"max_depth":        maxDepth,
		"max_branch":       maxBranch,
		"max_results":      maxResults,
		"min_score":        minScore,
		"include_negative": includeNegative,
	}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolLookupContext, args)
	writeResult(err, out)
}

func runExportGraph(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("export-graph", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	mustParse(fs, argv)
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolExportGraph, map[string]any{})
	writeResult(err, out)
}

func runImportGraph(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("import-graph", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var file string
	fs.StringVar(&file, "file", "", "path to graph JSON export file")
	mustParse(fs, argv)
	if strings.TrimSpace(file) == "" {
		writeJSONAndExit(2, map[string]any{"error": "import-graph: --file is required"})
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("import-graph: read file: %v", err)})
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("import-graph: parse file: %v", err)})
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolImportGraph, map[string]any{"graph": payload})
	writeResult(err, out)
}

func runExpandReasoning(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("expand-reasoning", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var startID, graphKind, asOf string
	var maxDepth, maxBranch, maxResults int
	var includeNegative bool
	var minScore float64
	fs.StringVar(&startID, "start-id", "", "start node id")
	fs.StringVar(&graphKind, "graph-kind", "knowledge", "graph kind filter")
	fs.StringVar(&asOf, "as-of", "", "optional RFC3339 query time")
	fs.IntVar(&maxDepth, "max-depth", 3, "max reasoning depth")
	fs.IntVar(&maxBranch, "max-branch", 5, "max outgoing edges per step")
	fs.IntVar(&maxResults, "max-results", 10, "max result nodes")
	fs.BoolVar(&includeNegative, "include-negative", false, "whether to include negative-polarity edges")
	fs.Float64Var(&minScore, "min-score", 0, "minimum propagated score to keep")
	mustParse(fs, argv)
	args := map[string]any{
		"start_id":         startID,
		"graph_kind":       graphKind,
		"max_depth":        maxDepth,
		"max_branch":       maxBranch,
		"max_results":      maxResults,
		"include_negative": includeNegative,
		"min_score":        minScore,
	}
	if strings.TrimSpace(asOf) != "" {
		args["as_of"] = asOf
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, kggraph.ToolExpandReasoning, args)
	writeResult(err, out)
}

func runLookup(ctx context.Context, argv []string, tool string) {
	fs := flag.NewFlagSet(tool, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var term string
	fs.StringVar(&term, "term", "", "term to resolve")
	mustParse(fs, argv)
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, tool, map[string]any{"term": term})
	writeResult(err, out)
}

func runList(ctx context.Context, argv []string, tool string) {
	fs := flag.NewFlagSet(tool, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	mustParse(fs, argv)
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, tool, map[string]any{})
	writeResult(err, out)
}

func runCall(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("call", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	var tool string
	var argsJSON string
	fs.StringVar(&tool, "tool", "", "tool name")
	fs.StringVar(&argsJSON, "args-json", "{}", "tool arguments as JSON object")
	mustParse(fs, argv)

	var argsMap map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &argsMap); err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("parse args-json: %v", err)})
	}
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	out, err := svc.Call(ctx, tool, argsMap)
	writeResult(err, out)
}

func runServeMCP(ctx context.Context, argv []string) {
	fs := flag.NewFlagSet("serve-mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var cfg commonFlags
	addCommonFlags(fs, &cfg)
	mustParse(fs, argv)
	svc, err := openService(cfg)
	exitOnOpenError(err)
	defer svc.Close()
	if err := kggraph.RunMCPServer(ctx, svc); err != nil {
		writeJSONAndExit(1, map[string]any{"error": err.Error()})
	}
}

func mustParse(fs *flag.FlagSet, argv []string) {
	if err := fs.Parse(argv); err != nil {
		writeJSONAndExit(2, map[string]any{"error": err.Error()})
	}
}

func mustParseStringArrayFlag(name, raw string) []string {
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		writeJSONAndExit(2, map[string]any{"error": fmt.Sprintf("parse %s: %v", name, err)})
	}
	return out
}

func exitOnOpenError(err error) {
	if err != nil {
		writeJSONAndExit(1, map[string]any{"error": err.Error()})
	}
}

func writeResult(err error, out map[string]any) {
	if err != nil {
		writeJSONAndExit(1, map[string]any{"error": err.Error()})
	}
	writeJSONAndExit(0, out)
}

func writeJSONAndExit(code int, payload map[string]any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
	os.Exit(code)
}

func getenvDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func anySlice[T any](in []T) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}
