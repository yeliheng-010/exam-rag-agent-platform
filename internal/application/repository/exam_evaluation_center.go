package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrExamEvaluationRunNotCompleted    = errors.New("exam evaluation run is not completed")
	ErrExamEvaluationSetNotFound        = errors.New("exam evaluation set not found")
	ErrExamEvaluationSetVersionNotFound = errors.New("exam evaluation set version not found")
)

type examEvaluationCenterRepository struct {
	db *gorm.DB
}

func NewExamEvaluationCenterRepository(db *gorm.DB) interfaces.ExamEvaluationCenterRepository {
	return &examEvaluationCenterRepository{db: db}
}

func (r *examEvaluationCenterRepository) ListCenterQuestionBanks(
	ctx context.Context,
	tenantID uint64,
	bankIDs []string,
) ([]*types.QuestionBank, error) {
	if bankIDs != nil && len(bankIDs) == 0 {
		return []*types.QuestionBank{}, nil
	}
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, "active")
	if bankIDs != nil {
		query = query.Where("id IN ?", bankIDs)
	}
	var banks []*types.QuestionBank
	err := query.Order("created_at DESC").Find(&banks).Error
	return banks, err
}

func (r *examEvaluationCenterRepository) GetCenterRun(
	ctx context.Context,
	tenantID uint64,
	runID string,
) (*types.ExamRAGEvaluationRun, error) {
	var run types.ExamRAGEvaluationRun
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(runID)).
		First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrExamRAGEvaluationRunNotFound
	}
	return &run, err
}

func (r *examEvaluationCenterRepository) ListCenterRuns(
	ctx context.Context,
	tenantID uint64,
	bankIDs []string,
	filter types.ExamEvaluationCenterFilter,
) ([]*types.ExamRAGEvaluationRun, error) {
	if bankIDs != nil && len(bankIDs) == 0 {
		return []*types.ExamRAGEvaluationRun{}, nil
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	query = applyCenterRunScope(query, bankIDs, filter)
	var runs []*types.ExamRAGEvaluationRun
	err := query.Order("created_at DESC").Limit(limit).Find(&runs).Error
	return runs, err
}

func (r *examEvaluationCenterRepository) ListBaselines(
	ctx context.Context,
	tenantID uint64,
	bankIDs []string,
	kind types.ExamEvaluationKind,
	agentID string,
) ([]*types.ExamRAGEvaluationRun, error) {
	if bankIDs != nil && len(bankIDs) == 0 {
		return []*types.ExamRAGEvaluationRun{}, nil
	}
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND evaluation_kind = ? AND is_baseline = ?", tenantID, kind, true)
	if bankIDs != nil {
		query = query.Where("question_bank_id IN ?", bankIDs)
	}
	if value := strings.TrimSpace(agentID); value != "" {
		query = query.Where("agent_id = ?", value)
	}
	var runs []*types.ExamRAGEvaluationRun
	err := query.Order("created_at DESC").Find(&runs).Error
	return runs, err
}

func (r *examEvaluationCenterRepository) SetBaseline(ctx context.Context, tenantID uint64, runID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run types.ExamRAGEvaluationRun
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(runID)).
			First(&run).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrExamRAGEvaluationRunNotFound
		}
		if err != nil {
			return err
		}
		if run.Status != types.ExamRAGEvaluationRunStatusCompleted {
			return ErrExamEvaluationRunNotCompleted
		}
		if err := tx.Model(&types.ExamRAGEvaluationRun{}).
			Where(
				"tenant_id = ? AND question_bank_id = ? AND evaluation_kind = ? AND COALESCE(agent_id, '') = ? AND is_baseline = ?",
				tenantID, run.QuestionBankID, run.EvaluationKind, strings.TrimSpace(run.AgentID), true,
			).
			Update("is_baseline", false).Error; err != nil {
			return err
		}
		result := tx.Model(&types.ExamRAGEvaluationRun{}).
			Where("tenant_id = ? AND id = ? AND status = ?", tenantID, run.ID, types.ExamRAGEvaluationRunStatusCompleted).
			Update("is_baseline", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrExamEvaluationRunNotCompleted
		}
		return nil
	})
}

func (r *examEvaluationCenterRepository) AttachRunEvaluationSet(
	ctx context.Context,
	tenantID uint64,
	runID string,
	setID string,
	version int,
) error {
	result := r.db.WithContext(ctx).Model(&types.ExamRAGEvaluationRun{}).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(runID)).
		Updates(map[string]any{"evaluation_set_id": strings.TrimSpace(setID), "evaluation_set_version": version})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrExamRAGEvaluationRunNotFound
	}
	return nil
}

func (r *examEvaluationCenterRepository) CreateEvaluationSet(
	ctx context.Context,
	set *types.ExamEvaluationSet,
	version *types.ExamEvaluationSetVersion,
) error {
	if set == nil || version == nil || set.ID == "" || version.ID == "" {
		return ErrExamEvaluationSetNotFound
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(set).Error; err != nil {
			return err
		}
		version.TenantID = set.TenantID
		version.EvaluationSetID = set.ID
		version.Version = 1
		return tx.Create(version).Error
	})
}

func (r *examEvaluationCenterRepository) ListEvaluationSets(
	ctx context.Context,
	tenantID uint64,
	bankIDs []string,
	filter types.ExamEvaluationCenterFilter,
) ([]*types.ExamEvaluationSet, error) {
	if bankIDs != nil && len(bankIDs) == 0 {
		return []*types.ExamEvaluationSet{}, nil
	}
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, types.ExamEvaluationSetStatusActive)
	if bankIDs != nil {
		query = query.Where("question_bank_id IN ?", bankIDs)
	}
	if filter.Kind != "" {
		query = query.Where("evaluation_kind = ?", filter.Kind)
	}
	if value := strings.TrimSpace(filter.BankID); value != "" {
		query = query.Where("question_bank_id = ?", value)
	}
	if value := strings.TrimSpace(filter.AgentID); value != "" {
		query = query.Where("agent_id = ?", value)
	}
	var sets []*types.ExamEvaluationSet
	err := query.Order("updated_at DESC").Find(&sets).Error
	return sets, err
}

func (r *examEvaluationCenterRepository) GetEvaluationSet(
	ctx context.Context,
	tenantID uint64,
	setID string,
) (*types.ExamEvaluationSet, error) {
	var set types.ExamEvaluationSet
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ? AND status = ?", tenantID, strings.TrimSpace(setID), types.ExamEvaluationSetStatusActive).
		First(&set).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrExamEvaluationSetNotFound
	}
	return &set, err
}

func (r *examEvaluationCenterRepository) ListEvaluationSetVersions(
	ctx context.Context,
	tenantID uint64,
	setID string,
) ([]*types.ExamEvaluationSetVersion, error) {
	if _, err := r.GetEvaluationSet(ctx, tenantID, setID); err != nil {
		return nil, err
	}
	var versions []*types.ExamEvaluationSetVersion
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND evaluation_set_id = ?", tenantID, strings.TrimSpace(setID)).
		Order("version DESC").Find(&versions).Error
	return versions, err
}

func (r *examEvaluationCenterRepository) GetEvaluationSetVersion(
	ctx context.Context,
	tenantID uint64,
	setID string,
	version int,
) (*types.ExamEvaluationSetVersion, error) {
	var item types.ExamEvaluationSetVersion
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND evaluation_set_id = ? AND version = ?", tenantID, strings.TrimSpace(setID), version).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrExamEvaluationSetVersionNotFound
	}
	return &item, err
}

func (r *examEvaluationCenterRepository) CreateEvaluationSetVersion(
	ctx context.Context,
	tenantID uint64,
	setID string,
	version *types.ExamEvaluationSetVersion,
) (*types.ExamEvaluationSetVersion, error) {
	if version == nil || version.ID == "" {
		return nil, ErrExamEvaluationSetVersionNotFound
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var set types.ExamEvaluationSet
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND id = ? AND status = ?", tenantID, strings.TrimSpace(setID), types.ExamEvaluationSetStatusActive).
			First(&set).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrExamEvaluationSetNotFound
		}
		if err != nil {
			return err
		}
		version.TenantID = tenantID
		version.EvaluationSetID = set.ID
		version.Version = set.CurrentVersion + 1
		if err := tx.Create(version).Error; err != nil {
			return err
		}
		return tx.Model(&types.ExamEvaluationSet{}).
			Where("tenant_id = ? AND id = ?", tenantID, set.ID).
			Updates(map[string]any{"current_version": version.Version, "updated_at": version.CreatedAt}).Error
	})
	return version, err
}

func applyCenterRunScope(
	query *gorm.DB,
	bankIDs []string,
	filter types.ExamEvaluationCenterFilter,
) *gorm.DB {
	if bankIDs != nil {
		query = query.Where("question_bank_id IN ?", bankIDs)
	}
	if filter.Kind != "" {
		query = query.Where("evaluation_kind = ?", filter.Kind)
	}
	if value := strings.TrimSpace(filter.BankID); value != "" {
		query = query.Where("question_bank_id = ?", value)
	}
	if value := strings.TrimSpace(filter.AgentID); value != "" {
		query = query.Where("agent_id = ?", value)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	return query
}
