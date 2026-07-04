package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type examQuestionDraftService struct {
	draftRepo    interfaces.ExamQuestionDraftRepository
	materialRepo interfaces.ExamMaterialRepository
	questionRepo interfaces.ExamQuestionRepository
	spaceService interfaces.ExamSpaceService
	chunkReader  interfaces.ExamMaterialChunkReader
	extractor    interfaces.ExamQuestionExtractor
}

func NewExamQuestionDraftService(
	draftRepo interfaces.ExamQuestionDraftRepository,
	materialRepo interfaces.ExamMaterialRepository,
	questionRepo interfaces.ExamQuestionRepository,
	spaceService interfaces.ExamSpaceService,
	chunkReader interfaces.ExamMaterialChunkReader,
	extractor interfaces.ExamQuestionExtractor,
) interfaces.ExamQuestionDraftService {
	return &examQuestionDraftService{
		draftRepo:    draftRepo,
		materialRepo: materialRepo,
		questionRepo: questionRepo,
		spaceService: spaceService,
		chunkReader:  chunkReader,
		extractor:    extractor,
	}
}

func (s *examQuestionDraftService) ExtractDrafts(ctx context.Context, tenantID uint64, userID string, taskID string, req *types.ExtractExamQuestionDraftsRequest) (*types.ExamQuestionDraftExtractionResult, error) {
	task, material, err := s.prepareExtraction(ctx, tenantID, userID, taskID)
	if err != nil {
		return nil, err
	}
	if req == nil {
		req = &types.ExtractExamQuestionDraftsRequest{}
	}
	if req.Force {
		if err := s.draftRepo.DeleteDraftsByTask(ctx, tenantID, task.ID); err != nil {
			return nil, err
		}
	}
	if err := s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusExtracting, -1, ""); err != nil {
		return nil, err
	}
	chunks, err := s.chunkReader.ListChunksByKnowledgeID(ctx, material.KnowledgeID)
	if err != nil {
		_ = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusFailed, -1, err.Error())
		return nil, err
	}
	if len(chunks) == 0 {
		err = errors.New("exam material has no available chunks")
		_ = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusFailed, 0, err.Error())
		return nil, err
	}
	candidates, rawOutput, err := s.extractor.Extract(context.WithValue(ctx, types.TenantIDContextKey, tenantID), material, task, chunks)
	if err != nil {
		_ = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusFailed, 0, err.Error())
		return nil, err
	}
	drafts, err := s.buildDrafts(task, material, candidates, rawOutput)
	if err != nil {
		_ = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusFailed, 0, err.Error())
		return nil, err
	}
	if err := s.draftRepo.CreateDrafts(ctx, drafts); err != nil {
		_ = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusFailed, 0, err.Error())
		return nil, err
	}
	task, err = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusReviewing, len(drafts), "")
	if err != nil {
		return nil, err
	}
	stats, err := s.draftRepo.CountDraftsByTask(ctx, tenantID, task.ID)
	if err != nil {
		return nil, err
	}
	return &types.ExamQuestionDraftExtractionResult{Task: task, Drafts: drafts, Stats: stats}, nil
}

func (s *examQuestionDraftService) ListDrafts(ctx context.Context, tenantID uint64, userID string, taskID string) (*types.ListExamQuestionDraftsResult, error) {
	task, err := s.getTaskForRead(ctx, tenantID, userID, taskID)
	if err != nil {
		return nil, err
	}
	drafts, err := s.draftRepo.ListDraftsByTask(ctx, tenantID, task.ID)
	if err != nil {
		return nil, err
	}
	stats, err := s.draftRepo.CountDraftsByTask(ctx, tenantID, task.ID)
	if err != nil {
		return nil, err
	}
	return &types.ListExamQuestionDraftsResult{Task: task, Drafts: drafts, Stats: stats}, nil
}

func (s *examQuestionDraftService) UpdateDraft(ctx context.Context, tenantID uint64, userID string, draftID string, req *types.UpdateExamQuestionDraftRequest) (*types.ExamQuestionDraft, error) {
	if req == nil || strings.TrimSpace(req.Stem) == "" {
		return nil, ErrExamInvalidRequest
	}
	draft, err := s.getDraftForWrite(ctx, tenantID, userID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status != types.ExamQuestionDraftStatusPendingReview {
		return nil, ErrExamInvalidRequest
	}
	if err := applyDraftUpdate(draft, req); err != nil {
		return nil, err
	}
	if err := s.draftRepo.UpdateDraft(ctx, draft); err != nil {
		return nil, err
	}
	return draft, nil
}

func (s *examQuestionDraftService) ApproveDraft(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ApproveExamQuestionDraftResult, error) {
	draft, err := s.getDraftForWrite(ctx, tenantID, userID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status == types.ExamQuestionDraftStatusApproved && draft.ApprovedQuestionID != "" {
		question, _ := s.questionRepo.GetQuestionDetailByIDAndTenant(ctx, tenantID, draft.ApprovedQuestionID)
		return &types.ApproveExamQuestionDraftResult{Draft: draft, Question: question}, nil
	}
	if draft.Status != types.ExamQuestionDraftStatusPendingReview {
		return nil, ErrExamInvalidRequest
	}
	question, err := buildQuestionDetailFromDraft(draft, userID)
	if err != nil {
		return nil, err
	}
	if err := s.questionRepo.CreateQuestionDetail(ctx, question); err != nil {
		return nil, err
	}
	if err := s.markDraftReviewed(ctx, draft, types.ExamQuestionDraftStatusApproved, userID, question.Question.ID, ""); err != nil {
		return nil, err
	}
	if err := s.refreshTaskAfterReview(ctx, draft.TaskID, tenantID); err != nil {
		return nil, err
	}
	return &types.ApproveExamQuestionDraftResult{Draft: draft, Question: question}, nil
}

func (s *examQuestionDraftService) RejectDraft(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ExamQuestionDraft, error) {
	draft, err := s.getDraftForWrite(ctx, tenantID, userID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status != types.ExamQuestionDraftStatusPendingReview {
		return nil, ErrExamInvalidRequest
	}
	if err := s.markDraftReviewed(ctx, draft, types.ExamQuestionDraftStatusRejected, userID, "", ""); err != nil {
		return nil, err
	}
	if err := s.refreshTaskAfterReview(ctx, draft.TaskID, tenantID); err != nil {
		return nil, err
	}
	return draft, nil
}
