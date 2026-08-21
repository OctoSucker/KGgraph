package knowledgegraph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openai "github.com/openai/openai-go"
)

const (
	IngestPreviewMode = "ingest_preview"
	IngestConfirmMode = "ingest_confirm"
)

type extractedGraphPayload struct {
	Nodes []extractedNode `json:"nodes"`
	Edges []extractedEdge `json:"edges"`
}

type extractedNode struct {
	ID       string   `json:"id"`
	NodeType string   `json:"node_type"`
	Aliases  []string `json:"aliases"`
}

type extractedEdge struct {
	From       string  `json:"from"`
	To         string  `json:"to"`
	Relation   string  `json:"relation"`
	Polarity   int     `json:"polarity"`
	Confidence float64 `json:"confidence"`
	Condition  string  `json:"condition"`
	ObservedAt string  `json:"observed_at"`
	ValidFrom  string  `json:"valid_from"`
	ValidUntil string  `json:"valid_until"`
}

type ingestDraft struct {
	nodes           []NodeUpsert
	edges           []EdgeUpsert
	conflictsByEdge []map[string]any
}

// IngestStatement extracts with the LLM, applies the conflict policy, and
// writes the graph in one call.
func (s *Service) IngestStatement(ctx context.Context, statement, graphKind, sourceType, sourceRef, model string, defaultConfidence float64, conflictPolicy string) (map[string]any, error) {
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return nil, fmt.Errorf("knowledgegraph: ingest statement: empty statement")
	}
	if strings.TrimSpace(graphKind) == "" {
		graphKind = "knowledge"
	}
	if strings.TrimSpace(sourceType) == "" {
		sourceType = "llm_extracted"
	}
	if strings.TrimSpace(model) == "" {
		model = IngestModelFromEnv()
	}
	if defaultConfidence <= 0 || defaultConfidence > 1 {
		defaultConfidence = DefaultIngestConfidence
	}
	policy, err := normalizeIngestPolicy(conflictPolicy)
	if err != nil {
		return nil, err
	}
	nodes, edges, err := s.extractWithLLM(ctx, statement, model, defaultConfidence)
	if err != nil {
		return nil, err
	}
	draft, err := s.prepareIngestDraft(ctx, nodes, edges, graphKind, policy)
	if err != nil {
		return nil, err
	}
	if len(draft.nodes) == 0 && len(draft.edges) == 0 {
		return nil, fmt.Errorf("knowledgegraph: ingest statement: no nodes/edges extracted")
	}
	if policy == ConflictPolicyBlock && len(draft.conflictsByEdge) > 0 {
		return nil, fmt.Errorf("knowledgegraph: ingest statement: conflict_policy=block rejected %d conflicting edge(s)", len(draft.conflictsByEdge))
	}
	return s.finishIngest(ctx, statement, graphKind, sourceType, sourceRef, model, policy, draft)
}

// IngestPreview runs extraction and conflict scanning but writes nothing. The
// returned payload can be reviewed and then passed to IngestConfirm.
func (s *Service) IngestPreview(ctx context.Context, statement, graphKind, sourceType, sourceRef, model string, defaultConfidence float64, conflictPolicy string) (map[string]any, error) {
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return nil, fmt.Errorf("knowledgegraph: ingest preview: empty statement")
	}
	if strings.TrimSpace(graphKind) == "" {
		graphKind = "knowledge"
	}
	if strings.TrimSpace(sourceType) == "" {
		sourceType = "llm_extracted"
	}
	if strings.TrimSpace(model) == "" {
		model = IngestModelFromEnv()
	}
	if defaultConfidence <= 0 || defaultConfidence > 1 {
		defaultConfidence = DefaultIngestConfidence
	}
	policy, err := normalizeIngestPolicy(conflictPolicy)
	if err != nil {
		return nil, err
	}
	nodes, edges, err := s.extractWithLLM(ctx, statement, model, defaultConfidence)
	if err != nil {
		return nil, err
	}
	draft, err := s.prepareIngestDraft(ctx, nodes, edges, graphKind, policy)
	if err != nil {
		return nil, err
	}
	if len(draft.nodes) == 0 && len(draft.edges) == 0 {
		return nil, fmt.Errorf("knowledgegraph: ingest preview: no nodes/edges extracted")
	}
	return ingestPreviewMap(statement, graphKind, sourceType, sourceRef, model, policy, draft), nil
}

// IngestConfirm writes a payload previously produced by IngestPreview. The
// extracted nodes/edges are re-validated and conflict-scanned before any write.
func (s *Service) IngestConfirm(ctx context.Context, preview map[string]any, graphKind, sourceType, sourceRef string, conflictPolicy string) (map[string]any, error) {
	if s == nil || s.graph == nil {
		return nil, fmt.Errorf("knowledgegraph: ingest confirm: service is nil")
	}
	if preview == nil {
		return nil, fmt.Errorf("knowledgegraph: ingest confirm: preview is required")
	}
	if strings.TrimSpace(graphKind) == "" {
		graphKind = "knowledge"
	}
	if strings.TrimSpace(sourceType) == "" {
		sourceType = "llm_extracted"
	}
	policy, err := normalizeIngestPolicy(conflictPolicy)
	if err != nil {
		return nil, err
	}
	statement := strings.TrimSpace(mapString(preview, "statement"))
	nodes, edges, err := parseIngestPreview(preview)
	if err != nil {
		return nil, fmt.Errorf("knowledgegraph: ingest confirm: %w", err)
	}
	draft, err := s.prepareIngestDraft(ctx, nodes, edges, graphKind, policy)
	if err != nil {
		return nil, err
	}
	if policy == ConflictPolicyBlock && len(draft.conflictsByEdge) > 0 {
		return nil, fmt.Errorf("knowledgegraph: ingest confirm: conflict_policy=block rejected %d conflicting edge(s)", len(draft.conflictsByEdge))
	}
	model := mapString(preview, "ingest_model")
	return s.finishIngest(ctx, statement, graphKind, sourceType, sourceRef, model, policy, draft)
}

func normalizeIngestPolicy(raw string) (string, error) {
	policy := canonicalizeNodeID(raw)
	if policy == "" {
		policy = ConflictPolicyWarn
	}
	if !validConflictPolicy(policy) {
		return "", fmt.Errorf("knowledgegraph: conflict_policy must be block, warn, or off")
	}
	return policy, nil
}

// prepareIngestDraft normalizes and validates extracted nodes/edges and runs
// the deterministic conflict scan. It never writes to the store.
func (s *Service) prepareIngestDraft(ctx context.Context, nodes []NodeUpsert, edges []EdgeUpsert, graphKind, policy string) (*ingestDraft, error) {
	if s == nil || s.graph == nil {
		return nil, fmt.Errorf("knowledgegraph: ingest: service is nil")
	}
	draft := &ingestDraft{}
	for _, n := range nodes {
		id := strings.TrimSpace(n.ID)
		if id == "" {
			continue
		}
		nodeType := strings.TrimSpace(n.NodeType)
		if nodeType == "" {
			nodeType = "entity"
		}
		draft.nodes = append(draft.nodes, NodeUpsert{
			ID:       id,
			NodeType: nodeType,
			Aliases:  cleanStringList(n.Aliases),
			Status:   "active",
		})
	}
	pending := make([]EdgeUpsert, 0, len(edges))
	for _, e := range edges {
		fromID := strings.TrimSpace(e.FromID)
		toID := strings.TrimSpace(e.ToID)
		if fromID == "" || toID == "" {
			continue
		}
		relation := canonicalizeNodeID(e.RelationType)
		if relation == "" {
			return nil, fmt.Errorf("knowledgegraph: ingest: edge relation is required")
		}
		if e.Confidence <= 0 || e.Confidence > 1 {
			return nil, fmt.Errorf("knowledgegraph: ingest: edge confidence must be in (0,1]")
		}
		if e.Polarity < -1 || e.Polarity > 1 {
			return nil, fmt.Errorf("knowledgegraph: ingest: edge polarity must be -1, 0, or 1")
		}
		e.FromID = fromID
		e.ToID = toID
		e.RelationType = relation
		e.GraphKind = graphKind
		pending = append(pending, e)
	}
	draft.edges = pending
	if policy != ConflictPolicyOff {
		conflictsByEdge := make([]map[string]any, 0, len(pending))
		for _, e := range pending {
			out, err := s.ConflictScan(ctx, ConflictScanInput{
				FromID:       e.FromID,
				ToID:         e.ToID,
				RelationType: e.RelationType,
				Polarity:     e.Polarity,
				GraphKind:    graphKind,
			})
			if err != nil {
				return nil, err
			}
			conflicts, _ := out["conflicts"].([]map[string]any)
			if len(conflicts) == 0 {
				continue
			}
			conflictsByEdge = append(conflictsByEdge, map[string]any{
				"candidate": map[string]any{
					"from_id":       e.FromID,
					"to_id":         e.ToID,
					"relation_type": e.RelationType,
					"polarity":      e.Polarity,
				},
				"conflict_count": len(conflicts),
				"conflicts":      conflicts,
			})
		}
		draft.conflictsByEdge = conflictsByEdge
	}
	return draft, nil
}

// finishIngest writes a prepared draft and returns the ingest result payload.
func (s *Service) finishIngest(ctx context.Context, statement, graphKind, sourceType, sourceRef, model, policy string, draft *ingestDraft) (map[string]any, error) {
	nodeSet := map[string]struct{}{}
	for _, n := range draft.nodes {
		if err := s.graph.UpsertNode(ctx, n); err != nil {
			return nil, err
		}
		nodeSet[n.ID] = struct{}{}
	}
	addedEdges := make([]map[string]any, 0, len(draft.edges))
	for _, e := range draft.edges {
		if _, ok := nodeSet[e.FromID]; !ok {
			if err := s.graph.UpsertNode(ctx, NodeUpsert{ID: e.FromID, NodeType: "entity", Status: "active"}); err != nil {
				return nil, err
			}
			nodeSet[e.FromID] = struct{}{}
		}
		if _, ok := nodeSet[e.ToID]; !ok {
			if err := s.graph.UpsertNode(ctx, NodeUpsert{ID: e.ToID, NodeType: "entity", Status: "active"}); err != nil {
				return nil, err
			}
			nodeSet[e.ToID] = struct{}{}
		}
		e.SourceType = sourceType
		e.SourceRef = sourceRef
		edgeID, err := s.graph.UpsertEdge(ctx, e)
		if err != nil {
			return nil, err
		}
		addedEdges = append(addedEdges, map[string]any{
			"edge_id":       edgeID,
			"from_id":       e.FromID,
			"to_id":         e.ToID,
			"relation_type": e.RelationType,
			"confidence":    e.Confidence,
		})
	}
	nodeIDs := make([]string, 0, len(nodeSet))
	for id := range nodeSet {
		nodeIDs = append(nodeIDs, id)
	}
	return map[string]any{
		"statement":         statement,
		"graph_kind":        graphKind,
		"source_type":       sourceType,
		"node_count":        len(nodeIDs),
		"edge_count":        len(addedEdges),
		"node_ids":          nodeIDs,
		"edges":             addedEdges,
		"ingest_model":      model,
		"conflict_policy":   policy,
		"conflicts_by_edge": draft.conflictsByEdge,
	}, nil
}

func ingestPreviewMap(statement, graphKind, sourceType, sourceRef, model, policy string, draft *ingestDraft) map[string]any {
	nodes := make([]map[string]any, 0, len(draft.nodes))
	for _, n := range draft.nodes {
		nodes = append(nodes, map[string]any{
			"id":        n.ID,
			"node_type": n.NodeType,
			"aliases":   n.Aliases,
		})
	}
	edges := make([]map[string]any, 0, len(draft.edges))
	for _, e := range draft.edges {
		item := map[string]any{
			"from":       e.FromID,
			"to":         e.ToID,
			"relation":   e.RelationType,
			"polarity":   e.Polarity,
			"confidence": e.Confidence,
			"condition":  e.ConditionText,
		}
		if e.ObservedAt != nil {
			item["observed_at"] = e.ObservedAt.UTC().Format(time.RFC3339)
		}
		if e.ValidFrom != nil {
			item["valid_from"] = e.ValidFrom.UTC().Format(time.RFC3339)
		}
		if e.ValidUntil != nil {
			item["valid_until"] = e.ValidUntil.UTC().Format(time.RFC3339)
		}
		edges = append(edges, item)
	}
	return map[string]any{
		"mode":              IngestPreviewMode,
		"statement":         statement,
		"graph_kind":        graphKind,
		"source_type":       sourceType,
		"source_ref":        sourceRef,
		"ingest_model":      model,
		"conflict_policy":   policy,
		"node_count":        len(nodes),
		"edge_count":        len(edges),
		"nodes":             nodes,
		"edges":             edges,
		"conflicts_by_edge": draft.conflictsByEdge,
		"written":           false,
	}
}

// parseIngestPreview converts a preview payload back into validated
// NodeUpsert/EdgeUpsert values using the same validators as LLM extraction.
func parseIngestPreview(preview map[string]any) ([]NodeUpsert, []EdgeUpsert, error) {
	nodesRaw, err := toMapSlice(preview["nodes"])
	if err != nil {
		return nil, nil, fmt.Errorf("nodes: %w", err)
	}
	edgesRaw, err := toMapSlice(preview["edges"])
	if err != nil {
		return nil, nil, fmt.Errorf("edges: %w", err)
	}
	nodes := make([]NodeUpsert, 0, len(nodesRaw))
	for i, n := range nodesRaw {
		extracted, err := validateExtractedNode(extractedNode{
			ID:       strAny(n["id"]),
			NodeType: strAny(n["node_type"]),
			Aliases:  stringSliceFromAny(n["aliases"]),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("invalid node %d: %w", i, err)
		}
		nodes = append(nodes, extracted)
	}
	edges := make([]EdgeUpsert, 0, len(edgesRaw))
	for i, e := range edgesRaw {
		extracted, err := validateExtractedEdge(extractedEdge{
			From:       strAny(e["from"]),
			To:         strAny(e["to"]),
			Relation:   strAny(e["relation"]),
			Polarity:   intFromAny(e["polarity"], 0),
			Confidence: floatFromAny(e["confidence"], 0),
			Condition:  strAny(e["condition"]),
			ObservedAt: strAny(e["observed_at"]),
			ValidFrom:  strAny(e["valid_from"]),
			ValidUntil: strAny(e["valid_until"]),
		}, DefaultIngestConfidence)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid edge %d: %w", i, err)
		}
		edges = append(edges, extracted)
	}
	return nodes, edges, nil
}

func strAny(raw any) string {
	if s, ok := raw.(string); ok {
		return s
	}
	return ""
}

func stringSliceFromAny(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func (s *Service) extractWithLLM(ctx context.Context, statement, model string, defaultConfidence float64) ([]NodeUpsert, []EdgeUpsert, error) {
	if s == nil || s.graph == nil || s.graph.embedder == nil {
		return nil, nil, fmt.Errorf("knowledgegraph: ingest statement: OpenAI embedder is required")
	}
	emb, ok := s.graph.embedder.(*OpenAIEmbedder)
	if !ok || emb == nil {
		return nil, nil, fmt.Errorf("knowledgegraph: ingest statement: only OpenAI embedder supports llm extraction")
	}
	systemPrompt := "Extract a compact knowledge graph from the user statement. Return STRICT JSON only. " +
		`Schema: {"nodes":[{"id":"string","node_type":"entity|event|concept","aliases":["string"]}],"edges":[{"from":"string","to":"string","relation":"related_to|causes|increases_probability_of|decreases_probability_of|requires|blocks|supports|contradicts|part_of|example_of","polarity":-1|0|1,"confidence":0..1,"condition":"string","observed_at":"RFC3339 or empty","valid_from":"RFC3339 or empty","valid_until":"RFC3339 or empty"}]}. ` +
		"Your task is extraction only, not judgment. Do not decide whether a claim is true, important, tradable, actionable, or likely to happen. " +
		"Extract only relationships explicitly stated by the text or grammatically unavoidable from the text. Omit weak implications. Do not add outside knowledge. " +
		"Use short canonical node IDs in the original language; prefer noun phrases, events, or concepts, not full sentences. Merge duplicate concepts within the statement. " +
		"Use relation only from the allowed enum. Use polarity 1 for supporting/positive relations, -1 for opposing/negative relations, 0 for neutral structural relations. " +
		"Confidence is extraction certainty that the statement expresses the relation, not real-world truth probability. Use 0.75-0.90 for direct relations and 0.55-0.70 only when the relation is grammatically unavoidable. Omit weak/speculative claims. " +
		"Only fill observed_at, valid_from, or valid_until when the statement explicitly contains a date/time or validity window. Never invent dates. Use empty strings otherwise."
	userPrompt := fmt.Sprintf("Default confidence if uncertain: %.2f\nStatement:\n%s", defaultConfidence, statement)
	resp, err := emb.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(systemPrompt),
					},
				},
			},
			openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(userPrompt),
					},
				},
			},
		},
		Temperature: openai.Float(0),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("knowledgegraph: ingest statement llm call: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, nil, fmt.Errorf("knowledgegraph: ingest statement: empty model response")
	}
	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	if raw == "" {
		return nil, nil, fmt.Errorf("knowledgegraph: ingest statement: empty model content")
	}
	var payload extractedGraphPayload
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return nil, nil, fmt.Errorf("knowledgegraph: ingest statement json parse: %w", err)
	}
	nodes := make([]NodeUpsert, 0, len(payload.Nodes))
	for i, n := range payload.Nodes {
		node, err := validateExtractedNode(n)
		if err != nil {
			return nil, nil, fmt.Errorf("knowledgegraph: ingest statement: invalid node %d: %w", i, err)
		}
		nodes = append(nodes, node)
	}
	edges := make([]EdgeUpsert, 0, len(payload.Edges))
	for i, e := range payload.Edges {
		edge, err := validateExtractedEdge(e, defaultConfidence)
		if err != nil {
			return nil, nil, fmt.Errorf("knowledgegraph: ingest statement: invalid edge %d: %w", i, err)
		}
		edges = append(edges, edge)
	}
	return nodes, edges, nil
}

func validateExtractedNode(n extractedNode) (NodeUpsert, error) {
	id := strings.TrimSpace(n.ID)
	if id == "" {
		return NodeUpsert{}, fmt.Errorf("id is required")
	}
	nodeType := canonicalizeNodeID(n.NodeType)
	if nodeType == "" {
		nodeType = "entity"
	}
	if !allowedExtractedNodeType(nodeType) {
		return NodeUpsert{}, fmt.Errorf("node_type %q is not allowed", n.NodeType)
	}
	return NodeUpsert{
		ID:       id,
		NodeType: nodeType,
		Aliases:  cleanStringList(n.Aliases),
		Status:   "active",
	}, nil
}

func validateExtractedEdge(e extractedEdge, defaultConfidence float64) (EdgeUpsert, error) {
	fromID := strings.TrimSpace(e.From)
	toID := strings.TrimSpace(e.To)
	if fromID == "" || toID == "" {
		return EdgeUpsert{}, fmt.Errorf("from and to are required")
	}
	relation := canonicalizeNodeID(e.Relation)
	if relation == "" {
		return EdgeUpsert{}, fmt.Errorf("relation is required")
	}
	if !allowedExtractedRelation(relation) {
		return EdgeUpsert{}, fmt.Errorf("relation %q is not allowed", e.Relation)
	}
	if e.Polarity < -1 || e.Polarity > 1 {
		return EdgeUpsert{}, fmt.Errorf("polarity must be -1, 0, or 1")
	}
	conf := e.Confidence
	if conf == 0 {
		conf = defaultConfidence
	}
	if conf <= 0 || conf > 1 {
		return EdgeUpsert{}, fmt.Errorf("confidence must be in (0,1]")
	}
	observedAt, err := parseLLMTime(e.ObservedAt)
	if err != nil {
		return EdgeUpsert{}, fmt.Errorf("invalid observed_at %q: %w", e.ObservedAt, err)
	}
	validFrom, err := parseLLMTime(e.ValidFrom)
	if err != nil {
		return EdgeUpsert{}, fmt.Errorf("invalid valid_from %q: %w", e.ValidFrom, err)
	}
	validUntil, err := parseLLMTime(e.ValidUntil)
	if err != nil {
		return EdgeUpsert{}, fmt.Errorf("invalid valid_until %q: %w", e.ValidUntil, err)
	}
	return EdgeUpsert{
		FromID:        fromID,
		ToID:          toID,
		RelationType:  relation,
		Polarity:      e.Polarity,
		Confidence:    conf,
		ConditionText: strings.TrimSpace(e.Condition),
		ObservedAt:    observedAt,
		ValidFrom:     validFrom,
		ValidUntil:    validUntil,
	}, nil
}

func allowedExtractedNodeType(nodeType string) bool {
	switch nodeType {
	case "entity", "event", "concept":
		return true
	default:
		return false
	}
}

func allowedExtractedRelation(relation string) bool {
	switch relation {
	case "related_to", "causes", "increases_probability_of", "decreases_probability_of", "requires", "blocks", "supports", "contradicts", "part_of", "example_of":
		return true
	default:
		return false
	}
}

func parseLLMTime(raw string) (*time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	tt := t.UTC()
	return &tt, nil
}
