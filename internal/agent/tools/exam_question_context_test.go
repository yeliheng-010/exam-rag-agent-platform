package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type stubExamQuestionContextRepo struct {
	interfaces.ExamQuestionRepository
	detail    *types.QuestionGroupDetail
	gotTenant uint64
	gotIDs    []string
}

type stubExamQuestionContextKBService struct {
	interfaces.KnowledgeBaseService
	results   []*types.SearchResult
	gotID     string
	gotParams types.SearchParams
	calls     int
}

func (s *stubExamQuestionContextKBService) HybridSearch(_ context.Context, id string, params types.SearchParams) ([]*types.SearchResult, error) {
	s.calls++
	s.gotID = id
	s.gotParams = params
	return s.results, nil
}

func (r *stubExamQuestionContextRepo) FindQuestionGroupDetailByChunkIDs(_ context.Context, tenantID uint64, chunkIDs []string) (*types.QuestionGroupDetail, error) {
	r.gotTenant = tenantID
	r.gotIDs = append([]string{}, chunkIDs...)
	return r.detail, nil
}

func TestExamQuestionContextToolExecuteByChunkIDs(t *testing.T) {
	repo := &stubExamQuestionContextRepo{detail: newToolStructuredReadingGroup()}
	tool := NewExamQuestionContextTool(repo, nil, types.SearchTargets{
		{KnowledgeBaseID: "kb-1", TenantID: 10000},
	})

	args, err := json.Marshal(map[string]interface{}{
		"query":              "first reading question 21 answer and explanation",
		"knowledge_base_ids": []string{"kb-1"},
		"chunk_ids":          []string{"chunk-21", ""},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("expected successful result, got %#v", result)
	}
	if !strings.Contains(result.Output, `<exam_question_context`) {
		t.Fatalf("missing root element:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "[Exam Structured Context]") {
		t.Fatalf("missing structured context:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "B. Los Angeles Rams") {
		t.Fatalf("missing option content:\n%s", result.Output)
	}
	if got := strings.Join(repo.gotIDs, ","); got != "chunk-21" {
		t.Fatalf("repo chunk IDs = %q", got)
	}
	if repo.gotTenant != 10000 {
		t.Fatalf("tenant = %d", repo.gotTenant)
	}
	if result.Data["display_type"] != "exam_question_context" {
		t.Fatalf("display_type = %#v", result.Data["display_type"])
	}
	if result.Data["group_id"] != "group-reading-a" {
		t.Fatalf("group_id = %#v", result.Data["group_id"])
	}
}

func TestExamQuestionContextToolExecuteByQueryRetrievesCandidateChunks(t *testing.T) {
	repo := &stubExamQuestionContextRepo{detail: newToolStructuredReadingGroup()}
	kbService := &stubExamQuestionContextKBService{
		results: []*types.SearchResult{
			{
				ID:              "chunk-21",
				KnowledgeBaseID: "kb-1",
				SubChunkID:      []string{"chunk-child-21", "chunk-21"},
			},
		},
	}
	tool := NewExamQuestionContextTool(repo, kbService, types.SearchTargets{
		{Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "kb-1", TenantID: 10000},
	})

	args, err := json.Marshal(map[string]interface{}{
		"query":              "first reading passage answer",
		"knowledge_base_ids": []string{"kb-1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("expected successful result, got %#v", result)
	}
	if kbService.calls != 1 {
		t.Fatalf("hybrid search calls = %d", kbService.calls)
	}
	if kbService.gotID != "kb-1" {
		t.Fatalf("hybrid search kb = %q", kbService.gotID)
	}
	if kbService.gotParams.QueryText != "first reading passage answer" {
		t.Fatalf("query text = %q", kbService.gotParams.QueryText)
	}
	if kbService.gotParams.MatchCount == 0 {
		t.Fatal("expected non-zero match count")
	}
	if got := strings.Join(repo.gotIDs, ","); got != "chunk-21,chunk-child-21" {
		t.Fatalf("repo chunk IDs = %q", got)
	}
	if !strings.Contains(result.Output, "[Exam Structured Context]") {
		t.Fatalf("missing structured context:\n%s", result.Output)
	}
}

func TestExamQuestionContextToolRejectsOutOfScopeKnowledgeBase(t *testing.T) {
	tool := NewExamQuestionContextTool(&stubExamQuestionContextRepo{}, nil, types.SearchTargets{
		{KnowledgeBaseID: "kb-1", TenantID: 10000},
	})

	args := json.RawMessage(`{
		"query": "first reading passage",
		"knowledge_base_ids": ["kb-other"],
		"chunk_ids": ["chunk-21"]
	}`)
	result, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected out-of-scope error")
	}
	if result == nil || result.Success {
		t.Fatalf("expected failed result, got %#v", result)
	}
	if !strings.Contains(result.Error, "not accessible") {
		t.Fatalf("unexpected error: %s", result.Error)
	}
}

func TestExamQuestionContextToolInDefaultDefinitions(t *testing.T) {
	var found bool
	for _, tool := range AvailableToolDefinitions() {
		if tool.Name == ToolExamQuestionContext {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("%s missing from AvailableToolDefinitions()", ToolExamQuestionContext)
	}

	defaults := strings.Join(DefaultAllowedTools(), ",")
	if !strings.Contains(defaults, ToolExamQuestionContext) {
		t.Fatalf("%s missing from DefaultAllowedTools(): %s", ToolExamQuestionContext, defaults)
	}
}
