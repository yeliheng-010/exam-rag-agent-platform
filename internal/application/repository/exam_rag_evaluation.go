package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrExamRAGEvaluationRunNotFound = errors.New("exam rag evaluation run not found")

type examRAGEvaluationRepository struct {
	db *gorm.DB
}

func NewExamRAGEvaluationRepository(db *gorm.DB) interfaces.ExamRAGEvaluationRepository {
	return &examRAGEvaluationRepository{db: db}
}

func (r *examRAGEvaluationRepository) CreateRun(ctx context.Context, run *types.ExamRAGEvaluationRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *examRAGEvaluationRepository) GetRun(
	ctx context.Context,
	tenantID uint64,
	bankID string,
	runID string,
) (*types.ExamRAGEvaluationRun, error) {
	var run types.ExamRAGEvaluationRun
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND question_bank_id = ?", runID, tenantID, bankID).
		First(&run).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamRAGEvaluationRunNotFound
		}
		return nil, err
	}
	return &run, nil
}

func (r *examRAGEvaluationRepository) GetRunForTask(
	ctx context.Context,
	runID string,
) (*types.ExamRAGEvaluationRun, error) {
	var run types.ExamRAGEvaluationRun
	err := r.db.WithContext(ctx).Where("id = ?", runID).First(&run).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamRAGEvaluationRunNotFound
		}
		return nil, err
	}
	return &run, nil
}

func (r *examRAGEvaluationRepository) ListRuns(
	ctx context.Context,
	tenantID uint64,
	bankID string,
	limit int,
) ([]*types.ExamRAGEvaluationRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var runs []*types.ExamRAGEvaluationRun
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND question_bank_id = ?", tenantID, bankID).
		Order("created_at DESC").
		Limit(limit).
		Find(&runs).Error
	return runs, err
}

func (r *examRAGEvaluationRepository) UpdateRun(
	ctx context.Context,
	tenantID uint64,
	bankID string,
	run *types.ExamRAGEvaluationRun,
) error {
	if run == nil {
		return ErrExamRAGEvaluationRunNotFound
	}
	tx := r.db.WithContext(ctx).
		Model(&types.ExamRAGEvaluationRun{}).
		Where("id = ? AND tenant_id = ? AND question_bank_id = ?", run.ID, tenantID, bankID).
		Updates(map[string]interface{}{
			"status":           run.Status,
			"progress":         run.Progress,
			"request_snapshot": run.RequestSnapshot,
			"result_snapshot":  run.ResultSnapshot,
			"error_message":    run.ErrorMessage,
			"started_at":       run.StartedAt,
			"completed_at":     run.CompletedAt,
			"updated_at":       run.UpdatedAt,
		})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrExamRAGEvaluationRunNotFound
	}
	return nil
}
