package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type questionGroupBatchChat struct {
	responses []string
	prompts   []string
}

func (m *questionGroupBatchChat) Chat(_ context.Context, messages []chat.Message, _ *chat.ChatOptions) (*types.ChatResponse, error) {
	if len(messages) > 1 {
		m.prompts = append(m.prompts, messages[1].Content)
	}
	index := len(m.prompts) - 1
	if index < 0 || index >= len(m.responses) {
		return nil, fmt.Errorf("unexpected model call %d", index+1)
	}
	return &types.ChatResponse{Content: m.responses[index]}, nil
}

func (m *questionGroupBatchChat) ChatStream(context.Context, []chat.Message, *chat.ChatOptions) (<-chan types.StreamResponse, error) {
	return nil, nil
}

func (m *questionGroupBatchChat) GetModelName() string { return "batch-test" }
func (m *questionGroupBatchChat) GetModelID() string   { return "batch-test" }

type questionGroupBatchModelService struct {
	interfaces.ModelService
	chatModel chat.Chat
}

func (s *questionGroupBatchModelService) ListModels(context.Context) ([]*types.Model, error) {
	return []*types.Model{{
		ID:     "chat-1",
		Type:   types.ModelTypeKnowledgeQA,
		Status: types.ModelStatusActive,
	}}, nil
}

func (s *questionGroupBatchModelService) GetChatModel(context.Context, string) (chat.Chat, error) {
	return s.chatModel, nil
}

func TestMathQuestionGroupExtractorBatchesKeepQuestionOwnershipDistinct(t *testing.T) {
	model := &questionGroupBatchChat{responses: []string{
		mathQuestionGroupResponse("1", "short"),
		mathQuestionGroupResponse("4", "fourth question") + "\n",
	}}
	extractor := &examQuestionGroupExtractor{
		modelService: &questionGroupBatchModelService{chatModel: model},
		registry:     NewQuestionGroupStrategyRegistry(),
	}
	subjectID := "math"
	chunks := []*types.Chunk{
		{ID: "body-1", ChunkIndex: 0, Content: "1. Q1"},
		{ID: "body-2", ChunkIndex: 1, Content: "2. Q2"},
		{ID: "body-3", ChunkIndex: 2, Content: "3. Q3"},
		{ID: "body-4", ChunkIndex: 3, Content: "4. Q4"},
	}

	candidates, _, err := extractor.Extract(context.Background(), &types.ExamMaterial{
		DomainID:  "gaokao",
		SubjectID: &subjectID,
	}, &types.ExamStructuringTask{}, chunks)

	if err != nil {
		t.Fatalf("expected batched extraction success, got %v", err)
	}
	if len(model.prompts) != 2 {
		t.Fatalf("expected two model calls, got %d", len(model.prompts))
	}
	if len(candidates) != 4 {
		t.Fatalf("expected model output plus source fallback to cover four questions, got %d candidates", len(candidates))
	}
	if candidates[3].Questions[0].QuestionNo != "4" {
		t.Fatalf("expected final candidate to own question 4, got %q", candidates[3].Questions[0].QuestionNo)
	}
}

func TestMathQuestionGroupExtractorReportsBatchProgress(t *testing.T) {
	model := &questionGroupBatchChat{responses: []string{
		mathQuestionGroupResponse("1", "first"),
		mathQuestionGroupResponse("4", "second"),
	}}
	extractor := &examQuestionGroupExtractor{
		modelService: &questionGroupBatchModelService{chatModel: model},
		registry:     NewQuestionGroupStrategyRegistry(),
	}
	subjectID := "math"
	chunks := []*types.Chunk{
		{ID: "body-1", ChunkIndex: 0, Content: "1. Q1"},
		{ID: "body-2", ChunkIndex: 1, Content: "2. Q2"},
		{ID: "body-3", ChunkIndex: 2, Content: "3. Q3"},
		{ID: "body-4", ChunkIndex: 3, Content: "4. Q4"},
	}
	progress := []interfaces.ExamQuestionGroupBatchProgress{}

	_, _, err := extractor.ExtractWithProgress(
		context.Background(),
		&types.ExamMaterial{DomainID: "gaokao", SubjectID: &subjectID},
		&types.ExamStructuringTask{},
		chunks,
		func(item interfaces.ExamQuestionGroupBatchProgress) error {
			progress = append(progress, item)
			return nil
		},
	)

	if err != nil {
		t.Fatalf("ExtractWithProgress returned error: %v", err)
	}
	if len(progress) != 4 {
		t.Fatalf("progress events = %#v", progress)
	}
	if progress[0].Completed != 0 || progress[1].Completed != 1 || progress[3].Completed != 2 {
		t.Fatalf("progress events = %#v", progress)
	}
}

func TestMathQuestionGroupExtractorSkipsInvalidBatchAndContinues(t *testing.T) {
	invalid := `{"question_groups":[{"group_no":"4","group_type":"math_problem","questions":[{"question_no":"4","stem":""}]}]}`
	model := &questionGroupBatchChat{responses: []string{
		mathQuestionGroupResponse("1", "first"),
		invalid,
		mathQuestionGroupResponse("7", "third"),
	}}
	extractor := &examQuestionGroupExtractor{
		modelService: &questionGroupBatchModelService{chatModel: model},
		registry:     NewQuestionGroupStrategyRegistry(),
	}
	subjectID := "math"
	chunks := []*types.Chunk{
		{ID: "body-1", ChunkIndex: 0, Content: "1. Q1"},
		{ID: "body-2", ChunkIndex: 1, Content: "2. Q2"},
		{ID: "body-3", ChunkIndex: 2, Content: "3. Q3"},
		{ID: "body-4", ChunkIndex: 3, Content: "4. Q4"},
		{ID: "body-5", ChunkIndex: 4, Content: "5. Q5"},
		{ID: "body-6", ChunkIndex: 5, Content: "6. Q6"},
		{ID: "body-7", ChunkIndex: 6, Content: "7. Q7"},
	}

	candidates, _, err := extractor.Extract(
		context.Background(),
		&types.ExamMaterial{DomainID: "gaokao", SubjectID: &subjectID},
		&types.ExamStructuringTask{},
		chunks,
	)

	if err != nil {
		t.Fatalf("expected extraction to survive one invalid batch, got %v", err)
	}
	if len(model.prompts) != 3 {
		t.Fatalf("expected all three batches to run, got %d calls", len(model.prompts))
	}
	if len(candidates) != 7 {
		t.Fatalf("expected invalid model batch to be recovered from source, got %d candidates", len(candidates))
	}
}

func TestMathQuestionGroupExtractorFailsWhenAllBatchesInvalid(t *testing.T) {
	invalid := `{"question_groups":[{"group_no":"1","group_type":"math_problem","questions":[{"question_no":"1","stem":""}]}]}`
	model := &questionGroupBatchChat{responses: []string{invalid, invalid}}
	extractor := &examQuestionGroupExtractor{
		modelService: &questionGroupBatchModelService{chatModel: model},
		registry:     NewQuestionGroupStrategyRegistry(),
	}
	subjectID := "math"
	chunks := []*types.Chunk{
		{ID: "body-1", ChunkIndex: 0, Content: "unstructured formula fragment"},
		{ID: "body-2", ChunkIndex: 1, Content: "another fragment"},
		{ID: "body-3", ChunkIndex: 2, Content: "continued fragment"},
		{ID: "body-4", ChunkIndex: 3, Content: "final fragment"},
	}

	_, _, err := extractor.Extract(
		context.Background(),
		&types.ExamMaterial{DomainID: "gaokao", SubjectID: &subjectID},
		&types.ExamStructuringTask{},
		chunks,
	)

	if !errors.Is(err, errNoValidQuestionGroupDrafts) {
		t.Fatalf("expected all-invalid extraction to fail with sentinel error, got %v", err)
	}
}

func TestMathQuestionGroupExtractionReservesEnoughOutputTokens(t *testing.T) {
	if got := questionGroupMaxTokens(gaokaoMathQuestionGroupStrategy, 14); got != 6144 {
		t.Fatalf("expected 6144 output tokens for batched math extraction, got %d", got)
	}
}

func TestMergeQuestionGroupCandidatesMergesCompleteSubquestionIntoMainGroup(t *testing.T) {
	mainGroup := &types.ExamQuestionGroupDraftCandidate{
		GroupNo: "15",
		Questions: []types.ExamQuestionGroupDraftQuestionCandidate{
			{QuestionNo: "15(1)", Stem: "prove", Answer: types.JSONMap{"value": "proof"}},
			{QuestionNo: "15(2)", Stem: "distance", Answer: types.JSONMap{"value": ""}},
		},
	}
	standalone := &types.ExamQuestionGroupDraftCandidate{
		GroupNo: "15(2)",
		Questions: []types.ExamQuestionGroupDraftQuestionCandidate{
			{QuestionNo: "15(2)", Stem: "distance", Answer: types.JSONMap{"value": "1"}, Explanation: "distance is 1"},
		},
	}

	merged := mergeQuestionGroupCandidates([]*types.ExamQuestionGroupDraftCandidate{mainGroup, standalone})

	if len(merged) != 1 {
		t.Fatalf("expected one merged problem group, got %d", len(merged))
	}
	if len(merged[0].Questions) != 2 {
		t.Fatalf("expected both subquestions to remain, got %d", len(merged[0].Questions))
	}
	if got := merged[0].Questions[1].Answer["value"]; got != "1" {
		t.Fatalf("expected complete subquestion answer to be merged, got %v", got)
	}
}

func mathQuestionGroupResponse(no string, explanation string) string {
	return fmt.Sprintf(`{"question_groups":[{"group_no":%q,"group_type":"math_problem","title":%q,"material_text":"","material_format":"latex","source_chunk_ids":[],"assets":[],"questions":[{"question_no":%q,"question_type_code":"math_problem","stem":%q,"options":[],"answer":{"value":"1"},"explanation":%q,"evidence":[],"metadata":{},"difficulty":"unknown","confidence":0.8,"order_in_group":1,"source_chunk_ids":[]}],"confidence":0.8}]}`,
		no, "Question "+no, no, "Stem "+no, explanation)
}
