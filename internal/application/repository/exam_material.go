package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrExamMaterialNotFound        = errors.New("exam material not found")
	ErrExamStructuringTaskNotFound = errors.New("exam structuring task not found")
)

type examMaterialRepository struct {
	db *gorm.DB
}

func NewExamMaterialRepository(db *gorm.DB) interfaces.ExamMaterialRepository {
	return &examMaterialRepository{db: db}
}

func (r *examMaterialRepository) UpsertMaterial(ctx context.Context, material *types.ExamMaterial) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "knowledge_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"space_id",
			"knowledge_base_id",
			"domain_id",
			"subject_id",
			"material_type",
			"title",
			"description",
			"source_year",
			"source_region",
			"paper_type",
			"ingest_status",
			"review_status",
			"status",
			"updated_at",
		}),
	}).Create(material).Error
}

func (r *examMaterialRepository) GetMaterialByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamMaterial, error) {
	var material types.ExamMaterial
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status = ?", id, tenantID, "active").
		First(&material).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamMaterialNotFound
		}
		return nil, err
	}
	return &material, nil
}

func (r *examMaterialRepository) GetMaterialByKnowledge(ctx context.Context, tenantID uint64, knowledgeID string) (*types.ExamMaterial, error) {
	var material types.ExamMaterial
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND knowledge_id = ? AND status = ?", tenantID, knowledgeID, "active").
		First(&material).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamMaterialNotFound
		}
		return nil, err
	}
	return &material, nil
}

func (r *examMaterialRepository) ListMaterials(ctx context.Context, tenantID uint64, filter types.ListExamMaterialsFilter, spaceIDs []string) ([]*types.ExamMaterial, error) {
	if len(spaceIDs) == 0 {
		return []*types.ExamMaterial{}, nil
	}
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND space_id IN ? AND status = ?", tenantID, spaceIDs, "active")
	if filter.DomainID != "" {
		query = query.Where("domain_id = ?", filter.DomainID)
	}
	if filter.SubjectID != "" {
		query = query.Where("subject_id = ?", filter.SubjectID)
	}
	if filter.MaterialType != "" {
		query = query.Where("material_type = ?", filter.MaterialType)
	}

	var materials []*types.ExamMaterial
	err := query.Order("created_at DESC").Find(&materials).Error
	return materials, err
}

func (r *examMaterialRepository) CreateStructuringTask(ctx context.Context, task *types.ExamStructuringTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *examMaterialRepository) GetStructuringTaskByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamStructuringTask, error) {
	var task types.ExamStructuringTask
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamStructuringTaskNotFound
		}
		return nil, err
	}
	return &task, nil
}

func (r *examMaterialRepository) UpdateStructuringTask(ctx context.Context, task *types.ExamStructuringTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *examMaterialRepository) ListStructuringTasks(ctx context.Context, tenantID uint64, filter types.ListExamStructuringTasksFilter, spaceIDs []string) ([]*types.ExamStructuringTask, error) {
	if len(spaceIDs) == 0 {
		return []*types.ExamStructuringTask{}, nil
	}
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND space_id IN ?", tenantID, spaceIDs)
	if filter.MaterialID != "" {
		query = query.Where("material_id = ?", filter.MaterialID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var tasks []*types.ExamStructuringTask
	err := query.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}
