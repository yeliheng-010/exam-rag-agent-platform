package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

func (s *examQuestionDraftService) prepareExtraction(ctx context.Context, tenantID uint64, userID string, taskID string) (*types.ExamStructuringTask, *types.ExamMaterial, error) {
	task, err := s.getTaskForWrite(ctx, tenantID, userID, taskID)
	if err != nil {
		return nil, nil, err
	}
	if task.Status != types.ExamStructuringTaskStatusReadyForReview && task.Status != types.ExamStructuringTaskStatusFailed {
		return nil, nil, ErrExamInvalidRequest
	}
	material, err := s.materialRepo.GetMaterialByIDAndTenant(ctx, task.MaterialID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamMaterialNotFound) {
			return nil, nil, ErrExamNotFound
		}
		return nil, nil, err
	}
	if material.IngestStatus != types.ExamMaterialIngestStatusCompleted {
		return nil, nil, ErrExamInvalidRequest
	}
	return task, material, nil
}

func (s *examQuestionDraftService) getTaskForRead(ctx context.Context, tenantID uint64, userID string, taskID string) (*types.ExamStructuringTask, error) {
	task, err := s.loadTask(ctx, tenantID, taskID)
	if err != nil {
		return nil, err
	}
	ok, err := s.spaceService.CanReadSpace(ctx, tenantID, userID, task.SpaceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrExamPermissionDenied
	}
	return task, nil
}

func (s *examQuestionDraftService) getTaskForWrite(ctx context.Context, tenantID uint64, userID string, taskID string) (*types.ExamStructuringTask, error) {
	task, err := s.loadTask(ctx, tenantID, taskID)
	if err != nil {
		return nil, err
	}
	ok, err := s.spaceService.CanWriteSpace(ctx, tenantID, userID, task.SpaceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrExamPermissionDenied
	}
	return task, nil
}

func (s *examQuestionDraftService) loadTask(ctx context.Context, tenantID uint64, taskID string) (*types.ExamStructuringTask, error) {
	if strings.TrimSpace(taskID) == "" {
		return nil, ErrExamInvalidRequest
	}
	task, err := s.materialRepo.GetStructuringTaskByIDAndTenant(ctx, strings.TrimSpace(taskID), tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamStructuringTaskNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	return task, nil
}

func (s *examQuestionDraftService) getDraftForWrite(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ExamQuestionDraft, error) {
	if strings.TrimSpace(draftID) == "" {
		return nil, ErrExamInvalidRequest
	}
	draft, err := s.draftRepo.GetDraftByIDAndTenant(ctx, strings.TrimSpace(draftID), tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamQuestionDraftNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	ok, err := s.spaceService.CanWriteSpace(ctx, tenantID, userID, draft.SpaceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrExamPermissionDenied
	}
	return draft, nil
}

func (s *examQuestionDraftService) updateTaskStatus(ctx context.Context, task *types.ExamStructuringTask, status types.ExamStructuringTaskStatus, count int, message string) (*types.ExamStructuringTask, error) {
	task.Status = status
	task.ErrorMessage = message
	task.UpdatedAt = time.Now()
	if count >= 0 {
		task.StructuredQuestionCount = count
	}
	if err := s.materialRepo.UpdateStructuringTask(context.WithoutCancel(ctx), task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *examQuestionDraftService) markDraftReviewed(ctx context.Context, draft *types.ExamQuestionDraft, status types.ExamQuestionDraftStatus, userID string, questionID string, message string) error {
	now := time.Now()
	draft.Status = status
	draft.ApprovedQuestionID = questionID
	draft.ReviewedByUserID = userID
	draft.ReviewedAt = &now
	draft.ErrorMessage = message
	draft.UpdatedAt = now
	return s.draftRepo.UpdateDraft(ctx, draft)
}

func (s *examQuestionDraftService) refreshTaskAfterReview(ctx context.Context, taskID string, tenantID uint64) error {
	task, err := s.materialRepo.GetStructuringTaskByIDAndTenant(ctx, taskID, tenantID)
	if err != nil {
		return err
	}
	stats, err := s.draftRepo.CountDraftsByTask(ctx, tenantID, taskID)
	if err != nil {
		return err
	}
	status := types.ExamStructuringTaskStatusReviewing
	if stats.Total > 0 && stats.PendingReview == 0 {
		status = types.ExamStructuringTaskStatusCompleted
	}
	return s.updateReviewedTask(ctx, task, status, stats.Approved)
}

func (s *examQuestionDraftService) updateReviewedTask(ctx context.Context, task *types.ExamStructuringTask, status types.ExamStructuringTaskStatus, approved int) error {
	task.Status = status
	task.StructuredQuestionCount = approved
	task.ErrorMessage = ""
	task.UpdatedAt = time.Now()
	return s.materialRepo.UpdateStructuringTask(ctx, task)
}
