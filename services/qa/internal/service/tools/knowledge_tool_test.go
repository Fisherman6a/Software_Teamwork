package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/contextutil"
)

type listToolsRetrieverStub struct{}

func (listToolsRetrieverStub) Retrieve(context.Context, string, RetrievalTestInput) ([]RetrievalTestResult, error) {
	return nil, nil
}

type captureKnowledgeRetriever struct {
	input RetrievalTestInput
}

func (r *captureKnowledgeRetriever) Retrieve(_ context.Context, _ string, input RetrievalTestInput) ([]RetrievalTestResult, error) {
	r.input = input
	return []RetrievalTestResult{{DocumentID: "doc-1", DocumentName: "Doc", ContentPreview: "content"}}, nil
}

func TestKnowledgeToolClientListsCitationSourceTool(t *testing.T) {
	client, err := NewKnowledgeToolClient(KnowledgeToolConfig{RetrievalClient: listToolsRetrieverStub{}})
	if err != nil {
		t.Fatal(err)
	}

	definitions, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, definition := range definitions {
		names[definition.Function.Name] = true
	}

	if !names[ToolSearchKnowledge] || !names[ToolGetCitationSource] {
		t.Fatalf("tool names=%#v, want search and citation source tools", names)
	}
}

func TestKnowledgeToolDoesNotRaiseConfiguredScoreThreshold(t *testing.T) {
	retriever := &captureKnowledgeRetriever{}
	client, err := NewKnowledgeToolClient(KnowledgeToolConfig{RetrievalClient: retriever})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithDefaultKnowledgeBaseIDs(ctx, []string{"kb-1"})
	ctx = contextutil.WithRetrievalSettings(ctx, contextutil.RetrievalSettings{
		TopK:                     5,
		ScoreThreshold:           0,
		ScoreThresholdConfigured: true,
	})
	args := json.RawMessage(`{"query":"支持向量机","score_threshold":0.2}`)

	result, err := client.CallTool(ctx, ToolSearchKnowledge, args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool failure: %+v", result)
	}
	if retriever.input.Retrieval.ScoreThreshold != 0 {
		t.Fatalf("scoreThreshold=%v, want configured zero", retriever.input.Retrieval.ScoreThreshold)
	}
	if !retriever.input.Retrieval.ScoreThresholdConfigured {
		t.Fatal("scoreThreshold should be marked configured")
	}
}

func TestKnowledgeToolKeepsEmptyKnowledgeBaseIDsForGlobalSearch(t *testing.T) {
	retriever := &captureKnowledgeRetriever{}
	client, err := NewKnowledgeToolClient(KnowledgeToolConfig{RetrievalClient: retriever})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithDefaultKnowledgeBaseIDs(ctx, []string{"stale-default-kb"})
	ctx = contextutil.WithRetrievalSettings(ctx, contextutil.RetrievalSettings{TopK: 5})

	result, err := client.CallTool(ctx, ToolSearchKnowledge, json.RawMessage(`{"query":"支持向量机"}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool failure: %+v", result)
	}
	if len(retriever.input.KnowledgeBaseIDs) != 0 {
		t.Fatalf("knowledgeBaseIds=%+v, want empty for global search", retriever.input.KnowledgeBaseIDs)
	}
}

func TestKnowledgeToolAllowsLowerScoreThreshold(t *testing.T) {
	retriever := &captureKnowledgeRetriever{}
	client, err := NewKnowledgeToolClient(KnowledgeToolConfig{RetrievalClient: retriever})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithDefaultKnowledgeBaseIDs(ctx, []string{"kb-1"})
	ctx = contextutil.WithRetrievalSettings(ctx, contextutil.RetrievalSettings{
		TopK:                     5,
		ScoreThreshold:           0.7,
		ScoreThresholdConfigured: true,
	})
	args := json.RawMessage(`{"query":"支持向量机","score_threshold":0.2}`)

	result, err := client.CallTool(ctx, ToolSearchKnowledge, args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool failure: %+v", result)
	}
	if retriever.input.Retrieval.ScoreThreshold != 0.2 {
		t.Fatalf("scoreThreshold=%v, want model-lowered threshold", retriever.input.Retrieval.ScoreThreshold)
	}
	if !retriever.input.Retrieval.ScoreThresholdConfigured {
		t.Fatal("scoreThreshold should be marked configured")
	}
}

func TestKnowledgeToolUsesModelScoreThresholdWhenNoDefaultConfigured(t *testing.T) {
	retriever := &captureKnowledgeRetriever{}
	client, err := NewKnowledgeToolClient(KnowledgeToolConfig{RetrievalClient: retriever})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithDefaultKnowledgeBaseIDs(ctx, []string{"kb-1"})
	ctx = contextutil.WithRetrievalSettings(ctx, contextutil.RetrievalSettings{TopK: 5})
	args := json.RawMessage(`{"query":"支持向量机","score_threshold":0.8}`)

	result, err := client.CallTool(ctx, ToolSearchKnowledge, args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool failure: %+v", result)
	}
	if retriever.input.Retrieval.ScoreThreshold != 0.8 {
		t.Fatalf("scoreThreshold=%v, want model threshold", retriever.input.Retrieval.ScoreThreshold)
	}
	if !retriever.input.Retrieval.ScoreThresholdConfigured {
		t.Fatal("scoreThreshold should be marked configured")
	}
}

func TestKnowledgeToolAllowsAllAccessibleBasesWhenDefaultListEmpty(t *testing.T) {
	retriever := &captureKnowledgeRetriever{}
	client, err := NewKnowledgeToolClient(KnowledgeToolConfig{RetrievalClient: retriever})
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextutil.WithUserID(context.Background(), "user-1")
	ctx = contextutil.WithDefaultKnowledgeBaseIDs(ctx, []string{})
	args := json.RawMessage(`{"query":"继电保护","knowledge_base_ids":["kb-any"]}`)

	result, err := client.CallTool(ctx, ToolSearchKnowledge, args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool failure: %+v", result)
	}
	if len(retriever.input.KnowledgeBaseIDs) != 1 || retriever.input.KnowledgeBaseIDs[0] != "kb-any" {
		t.Fatalf("knowledgeBaseIDs=%+v, want model-provided KB when default list is empty", retriever.input.KnowledgeBaseIDs)
	}
}

func TestGenerateSearchSummaryDeduplicatesRepeatedChunkContent(t *testing.T) {
	repeated := "a）发电厂和有人值班变电站内的变压器，一般每天一次，每周进行一次夜间巡视： b）无人值班变电站内一般每10天一次。5.1.3 在下列情况下应对变压器进行特殊巡视检查，增加巡视频次。"
	results := []RetrievalTestResult{{
		RankNo: 1, KnowledgeBaseID: "kb-1", DocumentID: "doc-1", DocumentName: "DL 572.pdf", ChunkID: "chunk-1", ContentPreview: repeated, Score: 0.37,
	}, {
		RankNo: 2, KnowledgeBaseID: "kb-2", DocumentID: "doc-2", DocumentName: "DL 572 copy.pdf", ChunkID: "chunk-2", ContentPreview: repeated, Score: 0.37,
	}, {
		RankNo: 3, KnowledgeBaseID: "kb-1", DocumentID: "doc-1", DocumentName: "DL 572.pdf", ChunkID: "chunk-3", ContentPreview: "6.4.1 变压器跳闸后，应立即查明原因。如综合判断证明变压器跳闸不是由于内部故障所引起，可重新投入运行。", Score: 0.32,
	}}

	summary := generateSearchSummary(results, 1)
	var decoded struct {
		HitCount int `json:"hit_count"`
		Returned int `json:"returned"`
		Results  []struct {
			ChunkID string `json:"chunk_id"`
			Preview string `json:"preview"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(summary), &decoded); err != nil {
		t.Fatalf("decode summary: %v\n%s", err, summary)
	}
	if decoded.HitCount != 2 || decoded.Returned != 2 || len(decoded.Results) != 2 {
		t.Fatalf("summary=%s", summary)
	}
	if decoded.Results[0].ChunkID != "chunk-1" || decoded.Results[1].ChunkID != "chunk-3" {
		t.Fatalf("deduped result order/chunks=%+v", decoded.Results)
	}
}
