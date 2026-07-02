package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrExamResourceNotFound = errors.New("exam resource not found")

type examResourceRepository struct {
	db *gorm.DB
}

func NewExamResourceRepository(db *gorm.DB) interfaces.ExamResourceRepository {
	return &examResourceRepository{db: db}
}

func (r *examResourceRepository) Upsert(ctx context.Context, resource *types.ExamSpaceResource) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "resource_type"},
			{Name: "resource_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"space_id",
			"domain_id",
			"subject_id",
			"material_type",
			"review_status",
			"status",
			"updated_at",
		}),
	}).Create(resource).Error
}

func (r *examResourceRepository) GetByResource(ctx context.Context, tenantID uint64, resourceType types.ExamResourceType, resourceID string) (*types.ExamSpaceResource, error) {
	var resource types.ExamSpaceResource
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND resource_type = ? AND resource_id = ? AND status = ?",
			tenantID, resourceType, resourceID, types.ExamSpaceResourceStatusActive).
		First(&resource).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamResourceNotFound
		}
		return nil, err
	}
	return &resource, nil
}

func (r *examResourceRepository) List(ctx context.Context, tenantID uint64, filter types.ListExamResourcesFilter, spaceIDs []string) ([]*types.ExamSpaceResource, error) {
	if len(spaceIDs) == 0 {
		return []*types.ExamSpaceResource{}, nil
	}
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND space_id IN ? AND status = ?", tenantID, spaceIDs, types.ExamSpaceResourceStatusActive)
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.DomainID != "" {
		query = query.Where("domain_id = ?", filter.DomainID)
	}
	if filter.SubjectID != "" {
		query = query.Where("subject_id = ?", filter.SubjectID)
	}
	if filter.MaterialType != "" {
		query = query.Where("material_type = ?", filter.MaterialType)
	}

	var resources []*types.ExamSpaceResource
	err := query.Order("created_at DESC").Find(&resources).Error
	return resources, err
}
