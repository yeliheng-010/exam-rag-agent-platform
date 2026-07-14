package service

import (
	"context"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type examQuestionGroupDraftService struct {
	draftRepo    interfaces.ExamQuestionGroupDraftRepository
	materialRepo interfaces.ExamMaterialRepository
	questionRepo interfaces.ExamQuestionRepository
	spaceService interfaces.ExamSpaceService
	chunkReader  interfaces.ExamMaterialChunkReader
	extractor    interfaces.ExamQuestionGroupExtractor
	taskEnqueuer interfaces.TaskEnqueuer
}

func NewExamQuestionGroupDraftService(
	draftRepo interfaces.ExamQuestionGroupDraftRepository,
	materialRepo interfaces.ExamMaterialRepository,
	questionRepo interfaces.ExamQuestionRepository,
	spaceService interfaces.ExamSpaceService,
	chunkReader interfaces.ExamMaterialChunkReader,
	extractor interfaces.ExamQuestionGroupExtractor,
	taskEnqueuer interfaces.TaskEnqueuer,
) interfaces.ExamQuestionGroupDraftService {
	return &examQuestionGroupDraftService{
		draftRepo:    draftRepo,
		materialRepo: materialRepo,
		questionRepo: questionRepo,
		spaceService: spaceService,
		chunkReader:  chunkReader,
		extractor:    extractor,
		taskEnqueuer: taskEnqueuer,
	}
}

func (s *examQuestionGroupDraftService) ExtractDrafts(ctx context.Context, tenantID uint64, userID string, taskID string, req *types.ExtractExamQuestionGroupDraftsRequest) (*types.ExamQuestionGroupDraftExtractionResult, error) {
	if req == nil {
		req = &types.ExtractExamQuestionGroupDraftsRequest{}
	}
	task, material, err := s.prepareExtraction(ctx, tenantID, userID, taskID, req.Force)
	if err != nil {
		return nil, err
	}
	return s.runQuestionGroupExtraction(ctx, tenantID, task, material, req.Force)
}

func (s *examQuestionGroupDraftService) ListDrafts(ctx context.Context, tenantID uint64, userID string, taskID string) (*types.ListExamQuestionGroupDraftsResult, error) {
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
	summarizeQuestionGroupQuality(drafts)
	return &types.ListExamQuestionGroupDraftsResult{Task: task, Drafts: drafts, Stats: stats}, nil
}

func (s *examQuestionGroupDraftService) UpdateDraft(ctx context.Context, tenantID uint64, userID string, draftID string, req *types.UpdateExamQuestionGroupDraftRequest) (*types.ExamQuestionGroupDraft, error) {
	if req == nil || len(req.Questions) == 0 || strings.TrimSpace(req.GroupType) == "" {
		return nil, ErrExamInvalidRequest
	}
	draft, err := s.getDraftForWrite(ctx, tenantID, userID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status != types.ExamQuestionGroupDraftStatusPendingReview {
		return nil, ErrExamInvalidRequest
	}
	if err := applyQuestionGroupDraftUpdate(draft, req); err != nil {
		return nil, err
	}
	if err := s.draftRepo.UpdateDraft(ctx, draft); err != nil {
		return nil, err
	}
	draft.QualityReport = evaluateQuestionGroupDraftQuality(draft)
	return draft, nil
}

func (s *examQuestionGroupDraftService) ApproveDraft(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ApproveExamQuestionGroupDraftResult, error) {
	draft, err := s.getDraftForWrite(ctx, tenantID, userID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status == types.ExamQuestionGroupDraftStatusApproved && draft.ApprovedGroupID != "" {
		group, _ := s.questionRepo.GetQuestionGroupDetailByIDAndTenant(ctx, tenantID, draft.ApprovedGroupID)
		return &types.ApproveExamQuestionGroupDraftResult{Draft: draft, Group: group}, nil
	}
	if draft.Status != types.ExamQuestionGroupDraftStatusPendingReview {
		return nil, ErrExamInvalidRequest
	}
	report := evaluateQuestionGroupDraftQuality(draft)
	draft.QualityReport = report
	if report.Blocking {
		return nil, ErrExamDraftQualityBlocked
	}
	group, err := buildQuestionGroupDetailFromDraft(draft, userID)
	if err != nil {
		return nil, err
	}
	if err := s.questionRepo.CreateQuestionGroupDetail(ctx, group); err != nil {
		return nil, err
	}
	if err := s.markDraftReviewed(ctx, draft, types.ExamQuestionGroupDraftStatusApproved, userID, group.Group.ID, ""); err != nil {
		return nil, err
	}
	if err := s.refreshTaskAfterReview(ctx, draft.TaskID, tenantID); err != nil {
		return nil, err
	}
	return &types.ApproveExamQuestionGroupDraftResult{Draft: draft, Group: group}, nil
}

func (s *examQuestionGroupDraftService) RejectDraft(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ExamQuestionGroupDraft, error) {
	draft, err := s.getDraftForWrite(ctx, tenantID, userID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status != types.ExamQuestionGroupDraftStatusPendingReview {
		return nil, ErrExamInvalidRequest
	}
	if err := s.markDraftReviewed(ctx, draft, types.ExamQuestionGroupDraftStatusRejected, userID, "", ""); err != nil {
		return nil, err
	}
	if err := s.refreshTaskAfterReview(ctx, draft.TaskID, tenantID); err != nil {
		return nil, err
	}
	return draft, nil
}
