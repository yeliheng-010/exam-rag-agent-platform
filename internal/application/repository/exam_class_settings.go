package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrExamClassStateConflict       = errors.New("exam class state conflict")
	ErrExamClassMemberLimitConflict = errors.New("exam class member limit conflict")
)

func (r *examClassRepository) ListByUserWithStatuses(
	ctx context.Context,
	tenantID uint64,
	userID string,
	statuses []types.ExamClassStatus,
) ([]*types.ExamClass, error) {
	if len(statuses) == 0 {
		return []*types.ExamClass{}, nil
	}
	var classes []*types.ExamClass
	err := r.db.WithContext(ctx).
		Joins("JOIN exam_class_members ecm ON ecm.class_id = exam_classes.id").
		Where("exam_classes.tenant_id = ? AND ecm.tenant_id = ? AND ecm.user_id = ? AND ecm.status = ?",
			tenantID, tenantID, userID, types.ExamClassMemberStatusActive).
		Where("exam_classes.status IN ?", statuses).
		Order("exam_classes.created_at DESC").
		Find(&classes).Error
	return classes, err
}

func (r *examClassRepository) GetByIDAndTenantIncludingArchived(
	ctx context.Context,
	id string,
	tenantID uint64,
) (*types.ExamClass, error) {
	var class types.ExamClass
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&class).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamClassNotFound
		}
		return nil, err
	}
	return &class, nil
}

func (r *examClassRepository) ListByIDsAndTenantIncludingArchived(
	ctx context.Context,
	tenantID uint64,
	ids []string,
) ([]*types.ExamClass, error) {
	if len(ids) == 0 {
		return []*types.ExamClass{}, nil
	}
	var classes []*types.ExamClass
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, ids).
		Find(&classes).Error
	return classes, err
}

func (r *examClassRepository) UpdateClassMetadata(
	ctx context.Context,
	tenantID uint64,
	classID string,
	ownerUserID string,
	name string,
	description string,
	memberLimit int,
	updatedAt time.Time,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		class, err := lockExamClass(tx, tenantID, classID)
		if err != nil {
			return err
		}
		if class.OwnerUserID != ownerUserID || class.Status != types.ExamClassStatusActive {
			return ErrExamClassStateConflict
		}
		activeMembers, err := countActiveExamClassMembers(tx, tenantID, classID)
		if err != nil {
			return err
		}
		if memberLimit != 0 && activeMembers > int64(memberLimit) {
			return ErrExamClassMemberLimitConflict
		}

		result := tx.Model(&types.ExamClass{}).
			Where("id = ? AND tenant_id = ? AND owner_user_id = ? AND status = ?",
				classID, tenantID, ownerUserID, types.ExamClassStatusActive).
			Updates(map[string]any{
				"name": name, "description": description, "member_limit": memberLimit, "updated_at": updatedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrExamClassStateConflict
		}
		return nil
	})
}

func (r *examClassRepository) TransitionClassStatus(
	ctx context.Context,
	tenantID uint64,
	classID string,
	ownerUserID string,
	expected types.ExamClassStatus,
	next types.ExamClassStatus,
	updatedAt time.Time,
) error {
	result := r.db.WithContext(ctx).
		Model(&types.ExamClass{}).
		Where("id = ? AND tenant_id = ? AND owner_user_id = ? AND status = ?",
			classID, tenantID, ownerUserID, expected).
		Updates(map[string]any{"status": next, "updated_at": updatedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&types.ExamClass{}).
		Where("id = ? AND tenant_id = ?", classID, tenantID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrExamClassNotFound
	}
	return ErrExamClassStateConflict
}

func (r *examClassRepository) ApproveMemberWithinLimit(
	ctx context.Context,
	tenantID uint64,
	classID string,
	userID string,
	joinedAt time.Time,
) (*types.ExamClassMember, error) {
	var approved *types.ExamClassMember
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		class, err := lockExamClass(tx, tenantID, classID)
		if err != nil {
			return err
		}
		if class.Status != types.ExamClassStatusActive {
			return ErrExamClassStateConflict
		}

		var member types.ExamClassMember
		err = tx.Where("class_id = ? AND tenant_id = ? AND user_id = ?", classID, tenantID, userID).
			First(&member).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrExamClassMemberNotFound
			}
			return err
		}
		if member.Status != types.ExamClassMemberStatusPending {
			return ErrExamClassStateConflict
		}

		activeMembers, err := countActiveExamClassMembers(tx, tenantID, classID)
		if err != nil {
			return err
		}
		if class.MemberLimit != 0 && activeMembers >= int64(class.MemberLimit) {
			return ErrExamClassMemberLimitConflict
		}

		result := tx.Model(&types.ExamClassMember{}).
			Where("id = ? AND status = ?", member.ID, types.ExamClassMemberStatusPending).
			Updates(map[string]any{
				"status": types.ExamClassMemberStatusActive, "joined_at": joinedAt, "updated_at": joinedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrExamClassStateConflict
		}
		member.Status = types.ExamClassMemberStatusActive
		member.JoinedAt = joinedAt
		member.UpdatedAt = joinedAt
		approved = &member
		return nil
	})
	return approved, err
}

func lockExamClass(tx *gorm.DB, tenantID uint64, classID string) (*types.ExamClass, error) {
	var class types.ExamClass
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND tenant_id = ?", classID, tenantID).
		First(&class).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamClassNotFound
		}
		return nil, err
	}
	return &class, nil
}

func countActiveExamClassMembers(tx *gorm.DB, tenantID uint64, classID string) (int64, error) {
	var count int64
	err := tx.Model(&types.ExamClassMember{}).
		Where("class_id = ? AND tenant_id = ? AND status = ?",
			classID, tenantID, types.ExamClassMemberStatusActive).
		Count(&count).Error
	return count, err
}
