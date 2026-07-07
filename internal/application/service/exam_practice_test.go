package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

func TestExamPracticeService_CreateAttemptRequiresReadableGroup(t *testing.T) {
	ctx := context.Background()
	questionRepo := &stubQuestionGroupWriter{created: []*types.QuestionGroupDetail{newPracticeGroupDetail()}}
	svc := NewExamPracticeService(questionRepo, newStubPracticeRepo(), &stubExamQuestionDraftSpace{canRead: false}).(*examPracticeService)

	_, err := svc.CreateAttempt(ctx, 10000, "student-1", "group-1")

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("CreateAttempt error = %v, want ErrExamPermissionDenied", err)
	}
}

func TestExamPracticeService_SubmitAnswerGradesObjectiveQuestion(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestExamPracticeService()
	attempt, err := svc.CreateAttempt(ctx, 10000, "student-1", "group-1")
	if err != nil {
		t.Fatalf("CreateAttempt returned error: %v", err)
	}

	result, err := svc.SubmitAnswer(ctx, 10000, "student-1", attempt.Attempt.ID, &types.SubmitPracticeAnswerRequest{
		QuestionID: "question-1",
		AnswerText: "b",
	})

	if err != nil {
		t.Fatalf("SubmitAnswer returned error: %v", err)
	}
	if !result.Answer.IsCorrect {
		t.Fatalf("answer should be correct")
	}
	if result.Attempt.AnsweredCount != 1 || result.Attempt.CorrectCount != 1 {
		t.Fatalf("attempt stats = answered %d correct %d, want 1/1", result.Attempt.AnsweredCount, result.Attempt.CorrectCount)
	}
	if len(repo.answers) != 1 {
		t.Fatalf("answer count = %d, want 1", len(repo.answers))
	}
}

func TestExamPracticeService_CreateAttemptHidesAnswerMaterialUntilSubmit(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestExamPracticeService()
	attempt, err := svc.CreateAttempt(ctx, 10000, "student-1", "group-1")
	if err != nil {
		t.Fatalf("CreateAttempt returned error: %v", err)
	}
	if len(attempt.Group.Questions) != 1 {
		t.Fatalf("question count = %d, want 1", len(attempt.Group.Questions))
	}
	question := attempt.Group.Questions[0]
	if len(question.Options) != 2 {
		t.Fatalf("option count = %d, want 2", len(question.Options))
	}
	if len(question.Answers) != 0 || len(question.Explanations) != 0 || len(question.ChunkRefs) != 0 {
		t.Fatalf("practice group leaked answers=%d explanations=%d chunk_refs=%d", len(question.Answers), len(question.Explanations), len(question.ChunkRefs))
	}

	result, err := svc.SubmitAnswer(ctx, 10000, "student-1", attempt.Attempt.ID, &types.SubmitPracticeAnswerRequest{
		QuestionID: "question-1",
		AnswerText: "B",
	})
	if err != nil {
		t.Fatalf("SubmitAnswer returned error: %v", err)
	}
	if len(result.CorrectAnswers) != 1 || result.CorrectAnswers[0] != "B" {
		t.Fatalf("correct answers = %#v, want [B]", result.CorrectAnswers)
	}
	if len(result.Explanations) != 1 || len(result.ChunkRefs) != 1 {
		t.Fatalf("submit result explanations=%d chunk_refs=%d, want 1/1", len(result.Explanations), len(result.ChunkRefs))
	}
}

func TestExamPracticeService_SubmitAnswerUpdatesExistingQuestionAnswer(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestExamPracticeService()
	attempt, err := svc.CreateAttempt(ctx, 10000, "student-1", "group-1")
	if err != nil {
		t.Fatalf("CreateAttempt returned error: %v", err)
	}

	_, err = svc.SubmitAnswer(ctx, 10000, "student-1", attempt.Attempt.ID, &types.SubmitPracticeAnswerRequest{
		QuestionID: "question-1",
		AnswerText: "A",
	})
	if err != nil {
		t.Fatalf("first SubmitAnswer returned error: %v", err)
	}
	result, err := svc.SubmitAnswer(ctx, 10000, "student-1", attempt.Attempt.ID, &types.SubmitPracticeAnswerRequest{
		QuestionID: "question-1",
		AnswerText: "B",
	})

	if err != nil {
		t.Fatalf("second SubmitAnswer returned error: %v", err)
	}
	if len(repo.answers) != 1 {
		t.Fatalf("answer count = %d, want 1", len(repo.answers))
	}
	if result.Attempt.AnsweredCount != 1 || result.Attempt.CorrectCount != 1 {
		t.Fatalf("attempt stats = answered %d correct %d, want 1/1", result.Attempt.AnsweredCount, result.Attempt.CorrectCount)
	}
}

func TestExamPracticeService_SubmitAnswerRequiresOwnedAttempt(t *testing.T) {
	ctx := context.Background()
	svc, repo := newTestExamPracticeService()
	repo.attempts = append(repo.attempts, &types.ExamPracticeAttempt{
		ID:             "attempt-1",
		TenantID:       10000,
		UserID:         "other-user",
		SpaceID:        "space-1",
		QuestionBankID: "bank-1",
		GroupID:        "group-1",
		Status:         types.ExamPracticeAttemptStatusInProgress,
		QuestionCount:  1,
		StartedAt:      time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	_, err := svc.SubmitAnswer(ctx, 10000, "student-1", "attempt-1", &types.SubmitPracticeAnswerRequest{
		QuestionID: "question-1",
		AnswerText: "B",
	})

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("SubmitAnswer error = %v, want ErrExamPermissionDenied", err)
	}
}

func newTestExamPracticeService() (*examPracticeService, *stubPracticeRepo) {
	questionRepo := &stubQuestionGroupWriter{created: []*types.QuestionGroupDetail{newPracticeGroupDetail()}}
	practiceRepo := newStubPracticeRepo()
	svc := NewExamPracticeService(questionRepo, practiceRepo, &stubExamQuestionDraftSpace{canRead: true}).(*examPracticeService)
	return svc, practiceRepo
}

func newPracticeGroupDetail() *types.QuestionGroupDetail {
	groupID := "group-1"
	now := time.Now()
	return &types.QuestionGroupDetail{
		Group: &types.QuestionGroup{
			ID:              groupID,
			TenantID:        10000,
			SpaceID:         "space-1",
			QuestionBankID:  "bank-1",
			DomainID:        "gaokao",
			GroupType:       "reading_passage",
			Title:           "Reading A",
			MaterialText:    "Passage text",
			MaterialFormat:  "plain_text",
			ReviewStatus:    types.ExamReviewStatusPrivate,
			Status:          "active",
			CreatedByUserID: "teacher-1",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		Questions: []*types.QuestionDetail{{
			Question: &types.Question{
				ID:               "question-1",
				TenantID:         10000,
				QuestionBankID:   "bank-1",
				DomainID:         "gaokao",
				GroupID:          &groupID,
				QuestionNo:       "21",
				OrderInGroup:     1,
				Stem:             "What is the answer?",
				QuestionMetadata: types.JSONMap{},
				Difficulty:       "unknown",
				ReviewStatus:     types.ExamReviewStatusPrivate,
				Status:           "active",
				CreatedByUserID:  "teacher-1",
				CreatedAt:        now,
				UpdatedAt:        now,
			},
			Options: []*types.QuestionOption{
				{ID: "option-a", QuestionID: "question-1", OptionKey: "A", Content: "Alpha", SortOrder: 1},
				{ID: "option-b", QuestionID: "question-1", OptionKey: "B", Content: "Beta", SortOrder: 2},
			},
			Answers: []*types.QuestionAnswer{
				{ID: "answer-1", QuestionID: "question-1", AnswerText: "B", IsCorrect: true, CreatedAt: now},
			},
			Explanations: []*types.QuestionExplanation{
				{ID: "exp-1", QuestionID: "question-1", ExplanationText: "The passage says so.", SourceType: "manual", CreatedAt: now},
			},
			ChunkRefs: []*types.QuestionChunkRef{
				{QuestionID: "question-1", ChunkID: "chunk-1", RefType: "evidence", Confidence: 1, CreatedAt: now},
			},
		}},
	}
}

type stubPracticeRepo struct {
	attempts []*types.ExamPracticeAttempt
	answers  []*types.ExamPracticeAnswer
}

func newStubPracticeRepo() *stubPracticeRepo {
	return &stubPracticeRepo{}
}

func (r *stubPracticeRepo) CreateAttempt(_ context.Context, attempt *types.ExamPracticeAttempt) error {
	r.attempts = append(r.attempts, clonePracticeAttempt(attempt))
	return nil
}

func (r *stubPracticeRepo) GetAttemptByIDAndTenant(_ context.Context, tenantID uint64, attemptID string) (*types.ExamPracticeAttempt, error) {
	for _, attempt := range r.attempts {
		if attempt.ID == attemptID && attempt.TenantID == tenantID {
			return clonePracticeAttempt(attempt), nil
		}
	}
	return nil, repository.ErrExamPracticeAttemptNotFound
}

func (r *stubPracticeRepo) UpdateAttempt(_ context.Context, attempt *types.ExamPracticeAttempt) error {
	for i, existing := range r.attempts {
		if existing.ID == attempt.ID {
			r.attempts[i] = clonePracticeAttempt(attempt)
			return nil
		}
	}
	r.attempts = append(r.attempts, clonePracticeAttempt(attempt))
	return nil
}

func (r *stubPracticeRepo) UpsertAnswer(_ context.Context, answer *types.ExamPracticeAnswer) error {
	for i, existing := range r.answers {
		if existing.AttemptID == answer.AttemptID && existing.QuestionID == answer.QuestionID {
			cp := clonePracticeAnswer(answer)
			cp.ID = existing.ID
			cp.CreatedAt = existing.CreatedAt
			r.answers[i] = cp
			return nil
		}
	}
	r.answers = append(r.answers, clonePracticeAnswer(answer))
	return nil
}

func (r *stubPracticeRepo) ListAnswersByAttempt(_ context.Context, tenantID uint64, attemptID string) ([]*types.ExamPracticeAnswer, error) {
	out := []*types.ExamPracticeAnswer{}
	for _, answer := range r.answers {
		if answer.TenantID == tenantID && answer.AttemptID == attemptID {
			out = append(out, clonePracticeAnswer(answer))
		}
	}
	return out, nil
}

func (r *stubPracticeRepo) ListLatestAttemptsByGroups(_ context.Context, tenantID uint64, userID string, groupIDs []string) (map[string]*types.ExamPracticeAttempt, error) {
	allowed := make(map[string]bool, len(groupIDs))
	for _, groupID := range groupIDs {
		allowed[groupID] = true
	}
	out := map[string]*types.ExamPracticeAttempt{}
	for _, attempt := range r.attempts {
		if attempt.TenantID == tenantID && attempt.UserID == userID && allowed[attempt.GroupID] && out[attempt.GroupID] == nil {
			out[attempt.GroupID] = clonePracticeAttempt(attempt)
		}
	}
	return out, nil
}

func clonePracticeAttempt(attempt *types.ExamPracticeAttempt) *types.ExamPracticeAttempt {
	if attempt == nil {
		return nil
	}
	cp := *attempt
	return &cp
}

func clonePracticeAnswer(answer *types.ExamPracticeAnswer) *types.ExamPracticeAnswer {
	if answer == nil {
		return nil
	}
	cp := *answer
	return &cp
}
