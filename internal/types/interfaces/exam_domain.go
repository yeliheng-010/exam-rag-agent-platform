package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamDomainService interface {
	ListDomains(ctx context.Context) ([]*types.ExamDomain, error)
	ListSubjects(ctx context.Context, domainID string) ([]*types.ExamSubject, error)
}

type ExamDomainRepository interface {
	ListDomains(ctx context.Context) ([]*types.ExamDomain, error)
	GetDomainByID(ctx context.Context, id string) (*types.ExamDomain, error)
	ListSubjects(ctx context.Context, domainID string) ([]*types.ExamSubject, error)
	GetSubjectByID(ctx context.Context, id string) (*types.ExamSubject, error)
}
