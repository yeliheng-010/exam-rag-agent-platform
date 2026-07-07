package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrExamPracticeAttemptNotFound = errors.New("exam practice attempt not found")

type examPracticeRepository struct {
	db *gorm.DB
}

func NewExamPracticeRepository(db *gorm.DB) interfaces.ExamPracticeRepository {
	return &examPracticeRepository{db: db}
}

func (r *examPracticeRepository) CreateAttempt(ctx context.Context, attempt *types.ExamPracticeAttempt) error {
	return r.db.WithContext(ctx).Create(attempt).Error
}

func (r *examPracticeRepository) GetAttemptByIDAndTenant(ctx context.Context, tenantID uint64, attemptID string) (*types.ExamPracticeAttempt, error) {
	var attempt types.ExamPracticeAttempt
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", attemptID, tenantID).
		First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamPracticeAttemptNotFound
		}
		return nil, err
	}
	return &attempt, nil
}

func (r *examPracticeRepository) UpdateAttempt(ctx context.Context, attempt *types.ExamPracticeAttempt) error {
	return r.db.WithContext(ctx).Save(attempt).Error
}

func (r *examPracticeRepository) UpsertAnswer(ctx context.Context, answer *types.ExamPracticeAnswer) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "attempt_id"},
			{Name: "question_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"question_no",
			"answer_text",
			"is_correct",
			"correct_answer",
			"question_snapshot",
			"answer_snapshot",
			"explanation_snapshot",
			"answered_at",
			"updated_at",
		}),
	}).Create(answer).Error
}

func (r *examPracticeRepository) ListAnswersByAttempt(ctx context.Context, tenantID uint64, attemptID string) ([]*types.ExamPracticeAnswer, error) {
	var answers []*types.ExamPracticeAnswer
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND attempt_id = ?", tenantID, attemptID).
		Order("answered_at ASC").
		Find(&answers).Error
	return answers, err
}

func (r *examPracticeRepository) ListLatestAttemptsByGroups(
	ctx context.Context,
	tenantID uint64,
	userID string,
	groupIDs []string,
) (map[string]*types.ExamPracticeAttempt, error) {
	out := make(map[string]*types.ExamPracticeAttempt, len(groupIDs))
	if len(groupIDs) == 0 {
		return out, nil
	}
	var attempts []*types.ExamPracticeAttempt
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND group_id IN ?", tenantID, userID, groupIDs).
		Order("created_at DESC").
		Find(&attempts).Error; err != nil {
		return nil, err
	}
	for _, attempt := range attempts {
		if attempt == nil || out[attempt.GroupID] != nil {
			continue
		}
		out[attempt.GroupID] = attempt
	}
	return out, nil
}
