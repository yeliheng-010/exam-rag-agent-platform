package service

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/application/repository"
)

func (s *examClassService) CanAccessClass(ctx context.Context, tenantID uint64, userID string, classID string) (bool, error) {
	_, err := s.GetClass(ctx, tenantID, userID, classID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrExamNotFound) || errors.Is(err, ErrExamPermissionDenied) {
		return false, nil
	}
	return false, err
}

func (s *examClassService) CanWriteClass(ctx context.Context, tenantID uint64, userID string, classID string) (bool, error) {
	class, err := s.classRepo.GetByIDAndTenant(ctx, classID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassNotFound) {
			return false, nil
		}
		return false, err
	}
	member, err := s.classRepo.GetMember(ctx, class.ID, tenantID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassMemberNotFound) {
			return false, nil
		}
		return false, err
	}
	return member.Role.CanWrite(), nil
}
