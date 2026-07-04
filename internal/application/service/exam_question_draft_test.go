package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestExamQuestionDraftService_ExtractRequiresWritableSpace(t *testing.T) {
	ctx := context.Background()
	repo := newReadyDraftRepo()
	svc := newTestExamQuestionDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: false}, nil, nil)

	_, err := svc.ExtractDrafts(ctx, 10000, "student-1", "task-1", &types.ExtractExamQuestionDraftsRequest{})

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("ExtractDrafts error = %v, want ErrExamPermissionDenied", err)
	}
	if len(repo.createdDrafts) != 0 {
		t.Fatalf("permission denied extraction created %d drafts", len(repo.createdDrafts))
	}
}

func TestExamQuestionDraftService_ExtractWritesDraftsFromModelJSON(t *testing.T) {
	ctx := context.Background()
	repo := newReadyDraftRepo()
	space := &stubExamQuestionDraftSpace{canRead: true, canWrite: true}
	chunks := &stubExamQuestionDraftChunkReader{chunks: []*types.Chunk{
		{ID: "chunk-1", KnowledgeID: "knowledge-1", Content: "21. What does the man suggest?\nA. Stay home\nB. Go out\n答案 A"},
	}}
	extractor := &stubExamQuestionExtractor{candidates: []*types.ExamQuestionDraftCandidate{{
		QuestionNo:       "21",
		QuestionTypeCode: "single_choice",
		Stem:             "What does the man suggest?",
		Options: []types.ExamQuestionDraftOption{
			{Key: "A", Content: "Stay home"},
			{Key: "B", Content: "Go out"},
		},
		Answer:         types.JSONMap{"value": "A"},
		Explanation:    "The answer is stated in the dialogue.",
		Confidence:     0.9,
		SourceChunkIDs: []string{"chunk-1"},
	}}}
	svc := newTestExamQuestionDraftService(repo, space, chunks, extractor)

	result, err := svc.ExtractDrafts(ctx, 10000, "teacher-1", "task-1", &types.ExtractExamQuestionDraftsRequest{})

	if err != nil {
		t.Fatalf("ExtractDrafts returned error: %v", err)
	}
	if len(result.Drafts) != 1 {
		t.Fatalf("draft count = %d, want 1", len(result.Drafts))
	}
	if repo.updatedStatus != types.ExamStructuringTaskStatusReviewing {
		t.Fatalf("task status = %s, want reviewing", repo.updatedStatus)
	}
	if result.Drafts[0].QuestionNo != "21" {
		t.Fatalf("question no = %q, want 21", result.Drafts[0].QuestionNo)
	}
}

func TestExamQuestionDraftService_ExtractInvalidModelOutputFailsTask(t *testing.T) {
	ctx := context.Background()
	repo := newReadyDraftRepo()
	space := &stubExamQuestionDraftSpace{canRead: true, canWrite: true}
	chunks := &stubExamQuestionDraftChunkReader{chunks: []*types.Chunk{{ID: "chunk-1", KnowledgeID: "knowledge-1", Content: "bad"}}}
	extractor := &stubExamQuestionExtractor{err: errors.New("invalid model json")}
	svc := newTestExamQuestionDraftService(repo, space, chunks, extractor)

	_, err := svc.ExtractDrafts(ctx, 10000, "teacher-1", "task-1", &types.ExtractExamQuestionDraftsRequest{})

	if err == nil {
		t.Fatalf("ExtractDrafts returned nil error, want invalid model output error")
	}
	if repo.updatedStatus != types.ExamStructuringTaskStatusFailed {
		t.Fatalf("task status = %s, want failed", repo.updatedStatus)
	}
	if len(repo.createdDrafts) != 0 {
		t.Fatalf("invalid extraction created %d drafts", len(repo.createdDrafts))
	}
}

func TestExamQuestionDraftService_ApproveCreatesOfficialQuestion(t *testing.T) {
	ctx := context.Background()
	repo := newReadyDraftRepo()
	repo.drafts = []*types.ExamQuestionDraft{newPendingDraft("draft-1")}
	space := &stubExamQuestionDraftSpace{canRead: true, canWrite: true}
	questionRepo := &stubExamQuestionWriter{}
	svc := newTestExamQuestionDraftServiceWithQuestionRepo(repo, space, nil, nil, questionRepo)

	result, err := svc.ApproveDraft(ctx, 10000, "teacher-1", "draft-1")

	if err != nil {
		t.Fatalf("ApproveDraft returned error: %v", err)
	}
	if result.Draft.Status != types.ExamQuestionDraftStatusApproved {
		t.Fatalf("draft status = %s, want approved", result.Draft.Status)
	}
	if result.Draft.ApprovedQuestionID == "" {
		t.Fatalf("approved question id should not be empty")
	}
	if len(questionRepo.created) != 1 {
		t.Fatalf("created official question count = %d, want 1", len(questionRepo.created))
	}
	if len(questionRepo.created[0].Options) != 2 {
		t.Fatalf("created option count = %d, want 2", len(questionRepo.created[0].Options))
	}
	if len(questionRepo.created[0].Answers) != 1 {
		t.Fatalf("created answer count = %d, want 1", len(questionRepo.created[0].Answers))
	}
}

func TestBuildQuestionDetailFromDraftSupportsStringValueAnswers(t *testing.T) {
	draft := newPendingDraft("draft-1")
	draft.AnswerJSON = types.JSONMap{"values": []string{"A", "C"}}
	draft.Confidence = 95

	detail, err := buildQuestionDetailFromDraft(draft, "teacher-1")

	if err != nil {
		t.Fatalf("buildQuestionDetailFromDraft returned error: %v", err)
	}
	if got := detail.Answers[0].AnswerText; got != "A,C" {
		t.Fatalf("answer text = %q, want A,C", got)
	}
}

func TestBuildDraftFromCandidateNormalizesConfidence(t *testing.T) {
	repo := newReadyDraftRepo()
	candidate := &types.ExamQuestionDraftCandidate{
		Stem:       "Question",
		Answer:     types.JSONMap{"value": "A"},
		Confidence: 95,
	}

	draft, err := buildDraftFromCandidate(repo.task, repo.material, candidate, "", time.Now())

	if err != nil {
		t.Fatalf("buildDraftFromCandidate returned error: %v", err)
	}
	if draft.Confidence != 1 {
		t.Fatalf("confidence = %v, want 1", draft.Confidence)
	}
}
