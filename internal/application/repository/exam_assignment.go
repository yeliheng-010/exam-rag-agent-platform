package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrExamClassAssignmentNotFound = errors.New("exam class assignment not found")

type examAssignmentRepository struct {
	db *gorm.DB
}

func NewExamAssignmentRepository(db *gorm.DB) interfaces.ExamAssignmentRepository {
	return &examAssignmentRepository{db: db}
}

func (r *examAssignmentRepository) CreateAssignment(ctx context.Context, assignment *types.ExamClassAssignment) error {
	return r.db.WithContext(ctx).Create(assignment).Error
}

func (r *examAssignmentRepository) GetAssignmentByIDAndTenant(ctx context.Context, tenantID uint64, assignmentID string) (*types.ExamClassAssignment, error) {
	var assignment types.ExamClassAssignment
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", assignmentID, tenantID).
		First(&assignment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamClassAssignmentNotFound
		}
		return nil, err
	}
	return &assignment, nil
}

func (r *examAssignmentRepository) ListAssignmentsByClass(ctx context.Context, tenantID uint64, classID string, limit int) ([]*types.ExamClassAssignment, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var assignments []*types.ExamClassAssignment
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND class_id = ? AND status = ?", tenantID, classID, types.ExamAssignmentStatusPublished).
		Order("created_at DESC").
		Limit(limit).
		Find(&assignments).Error
	return assignments, err
}

func (r *examAssignmentRepository) ListPublishedGroupsByClass(
	ctx context.Context,
	tenantID uint64,
	classID string,
) ([]*types.QuestionGroup, error) {
	var groups []*types.QuestionGroup
	err := r.db.WithContext(ctx).
		Model(&types.QuestionGroup{}).
		Distinct("question_groups.*").
		Joins("JOIN exam_class_assignments ON exam_class_assignments.group_id = question_groups.id").
		Where(
			"exam_class_assignments.tenant_id = ? AND exam_class_assignments.class_id = ? AND exam_class_assignments.status = ? AND question_groups.tenant_id = ?",
			tenantID,
			classID,
			types.ExamAssignmentStatusPublished,
			tenantID,
		).
		Find(&groups).Error
	return groups, err
}

func (r *examAssignmentRepository) ListAssignmentsByUserClasses(ctx context.Context, tenantID uint64, userID string, limit int) ([]*types.ExamClassAssignment, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var assignments []*types.ExamClassAssignment
	err := r.db.WithContext(ctx).
		Table("exam_class_assignments").
		Select("exam_class_assignments.*").
		Joins("JOIN exam_class_members ecm ON ecm.class_id = exam_class_assignments.class_id AND ecm.tenant_id = exam_class_assignments.tenant_id").
		Joins("JOIN exam_classes ec ON ec.id = exam_class_assignments.class_id AND ec.tenant_id = exam_class_assignments.tenant_id").
		Where("exam_class_assignments.tenant_id = ? AND ecm.user_id = ? AND ecm.status = ? AND ec.status = ? AND exam_class_assignments.status = ?",
			tenantID, userID, types.ExamClassMemberStatusActive, types.ExamClassStatusActive, types.ExamAssignmentStatusPublished).
		Order("exam_class_assignments.created_at DESC").
		Limit(limit).
		Find(&assignments).Error
	return assignments, err
}
