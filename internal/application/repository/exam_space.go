package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrExamSpaceNotFound = errors.New("exam space not found")

type examSpaceRepository struct {
	db *gorm.DB
}

func NewExamSpaceRepository(db *gorm.DB) interfaces.ExamSpaceRepository {
	return &examSpaceRepository{db: db}
}

func (r *examSpaceRepository) Create(ctx context.Context, space *types.ExamSpace) error {
	return r.db.WithContext(ctx).Create(space).Error
}

func (r *examSpaceRepository) ListByTenant(ctx context.Context, tenantID uint64) ([]*types.ExamSpace, error) {
	var spaces []*types.ExamSpace
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, types.ExamSpaceStatusActive).
		Order("created_at DESC").
		Find(&spaces).Error
	return spaces, err
}

func (r *examSpaceRepository) GetByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamSpace, error) {
	var space types.ExamSpace
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status = ?", id, tenantID, types.ExamSpaceStatusActive).
		First(&space).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamSpaceNotFound
		}
		return nil, err
	}
	return &space, nil
}

func (r *examSpaceRepository) GetPersonalByOwner(ctx context.Context, tenantID uint64, userID string) (*types.ExamSpace, error) {
	var space types.ExamSpace
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND owner_user_id = ? AND space_type = ? AND status = ?",
			tenantID, userID, types.ExamSpaceTypePersonal, types.ExamSpaceStatusActive).
		First(&space).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamSpaceNotFound
		}
		return nil, err
	}
	return &space, nil
}
