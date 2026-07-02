package service

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type examDomainService struct {
	domainRepo interfaces.ExamDomainRepository
}

func NewExamDomainService(domainRepo interfaces.ExamDomainRepository) interfaces.ExamDomainService {
	return &examDomainService{domainRepo: domainRepo}
}

func (s *examDomainService) ListDomains(ctx context.Context) ([]*types.ExamDomain, error) {
	return s.domainRepo.ListDomains(ctx)
}

func (s *examDomainService) ListSubjects(ctx context.Context, domainID string) ([]*types.ExamSubject, error) {
	if _, err := s.domainRepo.GetDomainByID(ctx, domainID); err != nil {
		return nil, err
	}
	return s.domainRepo.ListSubjects(ctx, domainID)
}
