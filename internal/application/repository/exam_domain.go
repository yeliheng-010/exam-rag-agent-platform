package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var (
	ErrExamDomainNotFound  = errors.New("exam domain not found")
	ErrExamSubjectNotFound = errors.New("exam subject not found")
)

type examDomainRepository struct {
	db *gorm.DB
}

func NewExamDomainRepository(db *gorm.DB) interfaces.ExamDomainRepository {
	return &examDomainRepository{db: db}
}

func (r *examDomainRepository) ListDomains(ctx context.Context) ([]*types.ExamDomain, error) {
	var domains []*types.ExamDomain
	err := r.db.WithContext(ctx).
		Where("status = ?", types.ExamDomainStatusActive).
		Order("created_at ASC").
		Find(&domains).Error
	return domains, err
}

func (r *examDomainRepository) GetDomainByID(ctx context.Context, id string) (*types.ExamDomain, error) {
	var domain types.ExamDomain
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&domain).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamDomainNotFound
		}
		return nil, err
	}
	return &domain, nil
}

func (r *examDomainRepository) ListSubjects(ctx context.Context, domainID string) ([]*types.ExamSubject, error) {
	var subjects []*types.ExamSubject
	err := r.db.WithContext(ctx).
		Where("domain_id = ? AND status = ?", domainID, types.ExamDomainStatusActive).
		Order("sort_order ASC, created_at ASC").
		Find(&subjects).Error
	return subjects, err
}

func (r *examDomainRepository) GetSubjectByID(ctx context.Context, id string) (*types.ExamSubject, error) {
	var subject types.ExamSubject
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&subject).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamSubjectNotFound
		}
		return nil, err
	}
	return &subject, nil
}
