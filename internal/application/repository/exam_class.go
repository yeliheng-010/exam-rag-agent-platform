package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var (
	ErrExamClassNotFound       = errors.New("exam class not found")
	ErrExamClassMemberNotFound = errors.New("exam class member not found")
)

type examClassRepository struct {
	db *gorm.DB
}

func NewExamClassRepository(db *gorm.DB) interfaces.ExamClassRepository {
	return &examClassRepository{db: db}
}

func (r *examClassRepository) CreateClass(ctx context.Context, class *types.ExamClass) error {
	return r.db.WithContext(ctx).Create(class).Error
}

func (r *examClassRepository) AddMember(ctx context.Context, member *types.ExamClassMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *examClassRepository) ListByUser(ctx context.Context, tenantID uint64, userID string) ([]*types.ExamClass, error) {
	var classes []*types.ExamClass
	err := r.db.WithContext(ctx).
		Joins("JOIN exam_class_members ecm ON ecm.class_id = exam_classes.id").
		Where("exam_classes.tenant_id = ? AND ecm.tenant_id = ? AND ecm.user_id = ? AND ecm.status = ? AND exam_classes.status = ?",
			tenantID, tenantID, userID, types.ExamClassMemberStatusActive, types.ExamClassStatusActive).
		Order("exam_classes.created_at DESC").
		Find(&classes).Error
	return classes, err
}

func (r *examClassRepository) GetByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamClass, error) {
	var class types.ExamClass
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status = ?", id, tenantID, types.ExamClassStatusActive).
		First(&class).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamClassNotFound
		}
		return nil, err
	}
	return &class, nil
}

func (r *examClassRepository) GetBySpaceIDAndTenant(ctx context.Context, spaceID string, tenantID uint64) (*types.ExamClass, error) {
	var class types.ExamClass
	err := r.db.WithContext(ctx).
		Where("space_id = ? AND tenant_id = ? AND status = ?", spaceID, tenantID, types.ExamClassStatusActive).
		First(&class).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamClassNotFound
		}
		return nil, err
	}
	return &class, nil
}

func (r *examClassRepository) GetMember(ctx context.Context, classID string, tenantID uint64, userID string) (*types.ExamClassMember, error) {
	var member types.ExamClassMember
	err := r.db.WithContext(ctx).
		Where("class_id = ? AND tenant_id = ? AND user_id = ? AND status = ?",
			classID, tenantID, userID, types.ExamClassMemberStatusActive).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamClassMemberNotFound
		}
		return nil, err
	}
	return &member, nil
}
