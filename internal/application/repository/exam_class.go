package repository

import (
	"context"
	"errors"
	"strings"
	"time"

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

type examClassMemberListRow struct {
	types.ExamClassMember `gorm:"embedded"`
	DisplayName           string `gorm:"column:display_name"`
	UserRegisteredAt      string `gorm:"column:user_registered_at"`
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

func (r *examClassRepository) GetByInviteCodeAndTenant(ctx context.Context, inviteCode string, tenantID uint64) (*types.ExamClass, error) {
	var class types.ExamClass
	err := r.db.WithContext(ctx).
		Where("UPPER(invite_code) = ? AND tenant_id = ? AND status = ?",
			strings.ToUpper(strings.TrimSpace(inviteCode)), tenantID, types.ExamClassStatusActive).
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

func (r *examClassRepository) GetAnyMember(ctx context.Context, classID string, tenantID uint64, userID string) (*types.ExamClassMember, error) {
	var member types.ExamClassMember
	err := r.db.WithContext(ctx).
		Where("class_id = ? AND tenant_id = ? AND user_id = ?",
			classID, tenantID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamClassMemberNotFound
		}
		return nil, err
	}
	return &member, nil
}

func (r *examClassRepository) ListMembers(ctx context.Context, classID string, tenantID uint64, statuses []types.ExamClassMemberStatus) ([]*types.ExamClassMember, error) {
	var rows []examClassMemberListRow
	q := r.db.WithContext(ctx).
		Table("exam_class_members").
		Select(`
			exam_class_members.*,
			COALESCE(NULLIF(users.username, ''), NULLIF(users.email, ''), exam_class_members.user_id) AS display_name,
			COALESCE(users.created_at, exam_class_members.created_at) AS user_registered_at
		`).
		Joins("LEFT JOIN users ON users.id = exam_class_members.user_id").
		Where("exam_class_members.class_id = ? AND exam_class_members.tenant_id = ?", classID, tenantID)
	if len(statuses) > 0 {
		q = q.Where("exam_class_members.status IN ?", statuses)
	}
	err := q.
		Order("COALESCE(users.created_at, exam_class_members.created_at) ASC").
		Order("exam_class_members.created_at ASC").
		Order("exam_class_members.user_id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	usedDisplayIDs := make(map[string]bool, len(rows))
	members := make([]*types.ExamClassMember, 0, len(rows))
	for _, row := range rows {
		member := row.ExamClassMember
		member.DisplayName = strings.TrimSpace(row.DisplayName)
		if member.DisplayName == "" {
			member.DisplayName = member.UserID
		}
		registeredAt := parseExamClassMemberDisplayTime(row.UserRegisteredAt)
		member.DisplayID = nextExamClassMemberDisplayID(registeredAt, member.CreatedAt, usedDisplayIDs)
		members = append(members, &member)
	}
	return members, nil
}

func parseExamClassMemberDisplayTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func nextExamClassMemberDisplayID(base time.Time, fallback time.Time, used map[string]bool) string {
	if base.IsZero() {
		base = fallback
	}
	if base.IsZero() {
		base = time.Now()
	}
	for {
		id := base.Format("20060102150405")
		if !used[id] {
			used[id] = true
			return id
		}
		base = base.Add(time.Second)
	}
}

func (r *examClassRepository) UpdateMemberStatus(ctx context.Context, classID string, tenantID uint64, userID string, status types.ExamClassMemberStatus) (*types.ExamClassMember, error) {
	var member types.ExamClassMember
	err := r.db.WithContext(ctx).
		Where("class_id = ? AND tenant_id = ? AND user_id = ?",
			classID, tenantID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamClassMemberNotFound
		}
		return nil, err
	}
	now := time.Now()
	member.Status = status
	member.UpdatedAt = now
	if status == types.ExamClassMemberStatusActive {
		member.JoinedAt = now
	}
	if err := r.db.WithContext(ctx).Save(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}
