package service

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

type examPracticeService struct {
	questionRepo interfaces.ExamQuestionRepository
	practiceRepo interfaces.ExamPracticeRepository
	spaceService interfaces.ExamSpaceService
}

func NewExamPracticeService(
	questionRepo interfaces.ExamQuestionRepository,
	practiceRepo interfaces.ExamPracticeRepository,
	spaceService interfaces.ExamSpaceService,
) interfaces.ExamPracticeService {
	return &examPracticeService{
		questionRepo: questionRepo,
		practiceRepo: practiceRepo,
		spaceService: spaceService,
	}
}

func (s *examPracticeService) ListQuestionGroups(
	ctx context.Context,
	tenantID uint64,
	userID string,
	filter types.ListPracticeQuestionGroupsFilter,
) ([]*types.QuestionGroupPracticeSummary, error) {
	spaceIDs, err := s.resolveReadableSpaceIDs(ctx, tenantID, userID, strings.TrimSpace(filter.SpaceID))
	if err != nil {
		return nil, err
	}
	filter.SpaceID = strings.TrimSpace(filter.SpaceID)
	filter.DomainID = strings.TrimSpace(filter.DomainID)
	filter.SubjectID = strings.TrimSpace(filter.SubjectID)
	summaries, err := s.questionRepo.ListQuestionGroupPracticeSummaries(ctx, tenantID, spaceIDs, filter)
	if err != nil {
		return nil, err
	}
	groupIDs := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		if summary != nil && summary.Group != nil {
			groupIDs = append(groupIDs, summary.Group.ID)
		}
	}
	attempts, err := s.practiceRepo.ListLatestAttemptsByGroups(ctx, tenantID, userID, groupIDs)
	if err != nil {
		return nil, err
	}
	for _, summary := range summaries {
		if summary != nil && summary.Group != nil {
			summary.LastAttempt = attempts[summary.Group.ID]
		}
	}
	return summaries, nil
}

func (s *examPracticeService) GetQuestionGroupDetail(ctx context.Context, tenantID uint64, userID string, groupID string) (*types.QuestionGroupDetail, error) {
	detail, err := s.readableQuestionGroup(ctx, tenantID, userID, groupID)
	if err != nil {
		return nil, err
	}
	return practiceQuestionGroupView(detail), nil
}

func (s *examPracticeService) CreateAttempt(
	ctx context.Context,
	tenantID uint64,
	userID string,
	groupID string,
) (*types.CreatePracticeAttemptResult, error) {
	detail, err := s.readableQuestionGroup(ctx, tenantID, userID, groupID)
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.Group == nil {
		return nil, ErrExamNotFound
	}
	now := time.Now()
	attempt := &types.ExamPracticeAttempt{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		UserID:         userID,
		SpaceID:        detail.Group.SpaceID,
		QuestionBankID: detail.Group.QuestionBankID,
		GroupID:        detail.Group.ID,
		Status:         types.ExamPracticeAttemptStatusInProgress,
		QuestionCount:  len(detail.Questions),
		StartedAt:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.practiceRepo.CreateAttempt(ctx, attempt); err != nil {
		return nil, err
	}
	return &types.CreatePracticeAttemptResult{Attempt: attempt, Group: practiceQuestionGroupView(detail)}, nil
}

func (s *examPracticeService) SubmitAnswer(
	ctx context.Context,
	tenantID uint64,
	userID string,
	attemptID string,
	req *types.SubmitPracticeAnswerRequest,
) (*types.PracticeAnswerResult, error) {
	if req == nil || strings.TrimSpace(req.QuestionID) == "" {
		return nil, ErrExamInvalidRequest
	}
	attempt, err := s.ownedAttempt(ctx, tenantID, userID, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.Status == types.ExamPracticeAttemptStatusCompleted {
		return nil, ErrExamInvalidRequest
	}
	detail, err := s.readableQuestionGroup(ctx, tenantID, userID, attempt.GroupID)
	if err != nil {
		return nil, err
	}
	question := findPracticeQuestion(detail, strings.TrimSpace(req.QuestionID))
	if question == nil || question.Question == nil {
		return nil, ErrExamInvalidRequest
	}
	correctAnswers := practiceAnswerValues(question)
	isCorrect := isPracticeAnswerCorrect(req.AnswerText, correctAnswers)
	now := time.Now()
	answer := &types.ExamPracticeAnswer{
		ID:                  uuid.New().String(),
		TenantID:            tenantID,
		AttemptID:           attempt.ID,
		QuestionID:          question.Question.ID,
		QuestionNo:          question.Question.QuestionNo,
		AnswerText:          strings.TrimSpace(req.AnswerText),
		IsCorrect:           isCorrect,
		CorrectAnswer:       strings.Join(correctAnswers, "，"),
		QuestionSnapshot:    buildQuestionSnapshot(question),
		AnswerSnapshot:      mustPracticeJSON(question.Answers),
		ExplanationSnapshot: mustPracticeJSON(question.Explanations),
		AnsweredAt:          now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.practiceRepo.UpsertAnswer(ctx, answer); err != nil {
		return nil, err
	}
	if err := s.refreshAttemptStats(ctx, tenantID, attempt); err != nil {
		return nil, err
	}
	return &types.PracticeAnswerResult{
		Attempt:        attempt,
		Answer:         answer,
		CorrectAnswers: correctAnswers,
		Explanations:   question.Explanations,
		ChunkRefs:      question.ChunkRefs,
	}, nil
}

func (s *examPracticeService) CompleteAttempt(ctx context.Context, tenantID uint64, userID string, attemptID string) (*types.ExamPracticeAttempt, error) {
	attempt, err := s.ownedAttempt(ctx, tenantID, userID, attemptID)
	if err != nil {
		return nil, err
	}
	if err := s.refreshAttemptStats(ctx, tenantID, attempt); err != nil {
		return nil, err
	}
	now := time.Now()
	attempt.Status = types.ExamPracticeAttemptStatusCompleted
	attempt.CompletedAt = &now
	attempt.UpdatedAt = now
	if err := s.practiceRepo.UpdateAttempt(ctx, attempt); err != nil {
		return nil, err
	}
	return attempt, nil
}

func (s *examPracticeService) readableQuestionGroup(ctx context.Context, tenantID uint64, userID string, groupID string) (*types.QuestionGroupDetail, error) {
	if strings.TrimSpace(groupID) == "" {
		return nil, ErrExamInvalidRequest
	}
	detail, err := s.questionRepo.GetQuestionGroupDetailByIDAndTenant(ctx, tenantID, strings.TrimSpace(groupID))
	if err != nil {
		if errors.Is(err, repository.ErrQuestionNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	if detail == nil || detail.Group == nil {
		return nil, ErrExamNotFound
	}
	ok, err := s.spaceService.CanReadSpace(ctx, tenantID, userID, detail.Group.SpaceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrExamPermissionDenied
	}
	return detail, nil
}

func (s *examPracticeService) ownedAttempt(ctx context.Context, tenantID uint64, userID string, attemptID string) (*types.ExamPracticeAttempt, error) {
	if strings.TrimSpace(attemptID) == "" {
		return nil, ErrExamInvalidRequest
	}
	attempt, err := s.practiceRepo.GetAttemptByIDAndTenant(ctx, tenantID, strings.TrimSpace(attemptID))
	if err != nil {
		if errors.Is(err, repository.ErrExamPracticeAttemptNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	if attempt.UserID != userID {
		return nil, ErrExamPermissionDenied
	}
	return attempt, nil
}

func (s *examPracticeService) resolveReadableSpaceIDs(ctx context.Context, tenantID uint64, userID string, spaceID string) ([]string, error) {
	if spaceID != "" {
		ok, err := s.spaceService.CanReadSpace(ctx, tenantID, userID, spaceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrExamPermissionDenied
		}
		return []string{spaceID}, nil
	}
	spaces, err := s.spaceService.ListSpaces(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	spaceIDs := make([]string, 0, len(spaces))
	for _, space := range spaces {
		if space != nil {
			spaceIDs = append(spaceIDs, space.ID)
		}
	}
	return spaceIDs, nil
}

func (s *examPracticeService) refreshAttemptStats(ctx context.Context, tenantID uint64, attempt *types.ExamPracticeAttempt) error {
	answers, err := s.practiceRepo.ListAnswersByAttempt(ctx, tenantID, attempt.ID)
	if err != nil {
		return err
	}
	correct := 0
	for _, answer := range answers {
		if answer != nil && answer.IsCorrect {
			correct++
		}
	}
	attempt.AnsweredCount = len(answers)
	attempt.CorrectCount = correct
	attempt.UpdatedAt = time.Now()
	return s.practiceRepo.UpdateAttempt(ctx, attempt)
}

func findPracticeQuestion(group *types.QuestionGroupDetail, questionID string) *types.QuestionDetail {
	if group == nil {
		return nil
	}
	for _, question := range group.Questions {
		if question != nil && question.Question != nil && question.Question.ID == questionID {
			return question
		}
	}
	return nil
}

func practiceAnswerValues(question *types.QuestionDetail) []string {
	if question == nil {
		return nil
	}
	out := make([]string, 0, len(question.Answers))
	for _, answer := range question.Answers {
		if answer == nil || strings.TrimSpace(answer.AnswerText) == "" {
			continue
		}
		out = append(out, strings.TrimSpace(answer.AnswerText))
	}
	return out
}

func buildQuestionSnapshot(question *types.QuestionDetail) types.JSONMap {
	if question == nil || question.Question == nil {
		return types.JSONMap{}
	}
	return types.JSONMap{
		"id":                question.Question.ID,
		"question_no":       question.Question.QuestionNo,
		"stem":              question.Question.Stem,
		"options":           question.Options,
		"question_metadata": question.Question.QuestionMetadata,
	}
}

func practiceQuestionGroupView(detail *types.QuestionGroupDetail) *types.QuestionGroupDetail {
	if detail == nil {
		return nil
	}
	view := &types.QuestionGroupDetail{
		Group:     detail.Group,
		Assets:    append([]*types.QuestionGroupAsset{}, detail.Assets...),
		Questions: make([]*types.QuestionDetail, 0, len(detail.Questions)),
	}
	for _, question := range detail.Questions {
		if question == nil {
			view.Questions = append(view.Questions, nil)
			continue
		}
		view.Questions = append(view.Questions, &types.QuestionDetail{
			Question:     question.Question,
			Options:      append([]*types.QuestionOption{}, question.Options...),
			Answers:      []*types.QuestionAnswer{},
			Explanations: []*types.QuestionExplanation{},
			ChunkRefs:    []*types.QuestionChunkRef{},
		})
	}
	return view
}

func mustPracticeJSON(value any) types.JSON {
	data, err := json.Marshal(value)
	if err != nil {
		return types.JSON([]byte("[]"))
	}
	return types.JSON(data)
}

var answerTokenSplitter = regexp.MustCompile(`[\s,，、;；/]+`)

func isPracticeAnswerCorrect(submitted string, correctAnswers []string) bool {
	submittedTokens := normalizedPracticeAnswerTokens([]string{submitted})
	correctTokens := normalizedPracticeAnswerTokens(correctAnswers)
	if len(submittedTokens) == 0 || len(correctTokens) == 0 || len(submittedTokens) != len(correctTokens) {
		return false
	}
	for i := range submittedTokens {
		if submittedTokens[i] != correctTokens[i] {
			return false
		}
	}
	return true
}

func normalizedPracticeAnswerTokens(values []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, token := range answerTokenSplitter.Split(strings.ToUpper(value), -1) {
			token = strings.TrimSpace(token)
			if token == "" || seen[token] {
				continue
			}
			seen[token] = true
			out = append(out, token)
		}
	}
	sort.Strings(out)
	return out
}
