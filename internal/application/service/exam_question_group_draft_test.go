package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestExamQuestionGroupDraftService_ExtractRequiresWritableSpace(t *testing.T) {
	ctx := context.Background()
	repo := newReadyQuestionGroupDraftRepo()
	svc := newTestExamQuestionGroupDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: false}, nil, nil, nil)

	_, err := svc.ExtractDrafts(ctx, 10000, "student-1", "task-1", &types.ExtractExamQuestionGroupDraftsRequest{})

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("ExtractDrafts error = %v, want ErrExamPermissionDenied", err)
	}
	if len(repo.createdDrafts) != 0 {
		t.Fatalf("permission denied extraction created %d group drafts", len(repo.createdDrafts))
	}
}

func TestExamQuestionGroupDraftService_ExtractWritesGroupDrafts(t *testing.T) {
	ctx := context.Background()
	repo := newReadyQuestionGroupDraftRepo()
	extractor := &stubQuestionGroupExtractor{candidates: []*types.ExamQuestionGroupDraftCandidate{newReadingGroupCandidate()}}
	svc := newTestExamQuestionGroupDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: true}, &stubExamQuestionDraftChunkReader{
		chunks: []*types.Chunk{{ID: "chunk-1", KnowledgeID: "knowledge-1", Content: "Passage"}},
	}, extractor, nil)

	result, err := svc.ExtractDrafts(ctx, 10000, "teacher-1", "task-1", &types.ExtractExamQuestionGroupDraftsRequest{})

	if err != nil {
		t.Fatalf("ExtractDrafts returned error: %v", err)
	}
	if len(result.Drafts) != 1 {
		t.Fatalf("draft count = %d, want 1", len(result.Drafts))
	}
	if result.Drafts[0].GroupType != "reading_passage" {
		t.Fatalf("group type = %q, want reading_passage", result.Drafts[0].GroupType)
	}
	if repo.updatedStatus != types.ExamStructuringTaskStatusReviewing {
		t.Fatalf("task status = %s, want reviewing", repo.updatedStatus)
	}
}

func TestExamQuestionGroupDraftService_ExtractAllowsCompletedLegacyTaskWithoutGroupDrafts(t *testing.T) {
	ctx := context.Background()
	repo := newReadyQuestionGroupDraftRepo()
	repo.task.Status = types.ExamStructuringTaskStatusCompleted
	extractor := &stubQuestionGroupExtractor{candidates: []*types.ExamQuestionGroupDraftCandidate{newReadingGroupCandidate()}}
	svc := newTestExamQuestionGroupDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: true}, &stubExamQuestionDraftChunkReader{
		chunks: []*types.Chunk{{ID: "chunk-1", KnowledgeID: "knowledge-1", Content: "Passage"}},
	}, extractor, nil)

	result, err := svc.ExtractDrafts(ctx, 10000, "teacher-1", "task-1", &types.ExtractExamQuestionGroupDraftsRequest{})

	if err != nil {
		t.Fatalf("ExtractDrafts returned error: %v", err)
	}
	if len(result.Drafts) != 1 {
		t.Fatalf("draft count = %d, want 1", len(result.Drafts))
	}
	if repo.updatedStatus != types.ExamStructuringTaskStatusReviewing {
		t.Fatalf("task status = %s, want reviewing", repo.updatedStatus)
	}
}

func TestExamQuestionGroupDraftService_ExtractRejectsCompletedTaskWithStoredGroup(t *testing.T) {
	ctx := context.Background()
	repo := newReadyQuestionGroupDraftRepo()
	repo.task.Status = types.ExamStructuringTaskStatusCompleted
	questionRepo := &stubQuestionGroupWriter{
		created: []*types.QuestionGroupDetail{{
			Group: &types.QuestionGroup{ID: "group-1", QuestionBankID: "bank-1"},
		}},
	}
	svc := newTestExamQuestionGroupDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: true}, &stubExamQuestionDraftChunkReader{
		chunks: []*types.Chunk{{ID: "chunk-1", KnowledgeID: "knowledge-1", Content: "Passage"}},
	}, &stubQuestionGroupExtractor{candidates: []*types.ExamQuestionGroupDraftCandidate{newReadingGroupCandidate()}}, questionRepo)

	_, err := svc.ExtractDrafts(ctx, 10000, "teacher-1", "task-1", &types.ExtractExamQuestionGroupDraftsRequest{})

	if !errors.Is(err, ErrExamInvalidRequest) {
		t.Fatalf("ExtractDrafts error = %v, want ErrExamInvalidRequest", err)
	}
	if len(repo.createdDrafts) != 0 {
		t.Fatalf("created draft count = %d, want 0", len(repo.createdDrafts))
	}
}

func TestExamQuestionGroupDraftService_ApproveCreatesOfficialGroup(t *testing.T) {
	ctx := context.Background()
	repo := newReadyQuestionGroupDraftRepo()
	repo.drafts = []*types.ExamQuestionGroupDraft{newPendingQuestionGroupDraft("draft-1")}
	questionRepo := &stubQuestionGroupWriter{}
	svc := newTestExamQuestionGroupDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: true}, nil, nil, questionRepo)

	result, err := svc.ApproveDraft(ctx, 10000, "teacher-1", "draft-1")

	if err != nil {
		t.Fatalf("ApproveDraft returned error: %v", err)
	}
	if result.Draft.Status != types.ExamQuestionGroupDraftStatusApproved {
		t.Fatalf("draft status = %s, want approved", result.Draft.Status)
	}
	if result.Draft.ApprovedGroupID == "" {
		t.Fatalf("approved group id should not be empty")
	}
	if len(questionRepo.created) != 1 {
		t.Fatalf("created group count = %d, want 1", len(questionRepo.created))
	}
	if result.Draft.ApprovedGroupID != questionRepo.created[0].Group.ID {
		t.Fatalf("approved group id = %q, created group id = %q", result.Draft.ApprovedGroupID, questionRepo.created[0].Group.ID)
	}
	if questionRepo.created[0].Group.GroupType != "reading_passage" {
		t.Fatalf("created group type = %q, want reading_passage", questionRepo.created[0].Group.GroupType)
	}
	if len(questionRepo.created[0].Questions) != 3 {
		t.Fatalf("created question count = %d, want 3", len(questionRepo.created[0].Questions))
	}
	if repo.updatedStatus != types.ExamStructuringTaskStatusCompleted {
		t.Fatalf("task status = %s, want completed", repo.updatedStatus)
	}
}

func TestExamQuestionGroupDraftService_ApproveBackfillsMissingChoiceExplanation(t *testing.T) {
	ctx := context.Background()
	repo := newReadyQuestionGroupDraftRepo()
	draft := newPendingQuestionGroupDraft("draft-missing-explanation")
	draft.MaterialText = "Los Angeles Rams v Dallas Cowboys\nLos Angeles Rams v Houston Texans"
	questions, err := draftGroupQuestions(draft)
	if err != nil {
		t.Fatalf("draftGroupQuestions returned error: %v", err)
	}
	questions[0].Explanation = ""
	draft.QuestionsJSON = mustJSONForTest(questions)
	repo.drafts = []*types.ExamQuestionGroupDraft{draft}
	questionRepo := &stubQuestionGroupWriter{}
	svc := newTestExamQuestionGroupDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: true}, nil, nil, questionRepo)

	_, err = svc.ApproveDraft(ctx, 10000, "teacher-1", draft.ID)

	if err != nil {
		t.Fatalf("ApproveDraft returned error: %v", err)
	}
	createdQuestion := questionRepo.created[0].Questions[0]
	if len(createdQuestion.Explanations) != 1 {
		t.Fatalf("explanation count = %d, want 1", len(createdQuestion.Explanations))
	}
	explanation := createdQuestion.Explanations[0].ExplanationText
	if !strings.Contains(explanation, "Correct answer: A") {
		t.Fatalf("explanation should mention the correct answer, got %q", explanation)
	}
	if !strings.Contains(explanation, "Option A") {
		t.Fatalf("explanation should mention the correct option, got %q", explanation)
	}
	if createdQuestion.Explanations[0].SourceType != "auto_baseline" {
		t.Fatalf("source type = %q, want auto_baseline", createdQuestion.Explanations[0].SourceType)
	}
}
