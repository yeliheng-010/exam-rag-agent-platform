package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

const (
	maxExamClassNameCharacters        = 255
	maxExamClassDescriptionCharacters = 2000
	maxExamClassMemberLimit           = 1000
)

func (s *examClassService) ListClasses(
	ctx context.Context,
	tenantID uint64,
	userID string,
	filter types.ListExamClassesFilter,
) ([]*types.ExamClass, error) {
	if !filter.IncludeArchived {
		return s.classRepo.ListByUser(ctx, tenantID, userID)
	}
	return s.classRepo.ListByUserWithStatuses(ctx, tenantID, userID, []types.ExamClassStatus{
		types.ExamClassStatusActive,
		types.ExamClassStatusArchived,
	})
}

func (s *examClassService) GetClass(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
) (*types.ExamClass, error) {
	class, err := s.classRepo.GetByIDAndTenantIncludingArchived(ctx, classID, tenantID)
	if err != nil {
		return nil, mapExamClassSettingsRepositoryError(err)
	}
	if _, err := s.classRepo.GetMember(ctx, class.ID, tenantID, userID); err != nil {
		if errors.Is(err, repository.ErrExamClassMemberNotFound) {
			return nil, ErrExamPermissionDenied
		}
		return nil, err
	}
	return class, nil
}

func (s *examClassService) UpdateClass(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	req *types.UpdateExamClassRequest,
) (*types.ExamClass, error) {
	classID = strings.TrimSpace(classID)
	name, description, err := normalizeExamClassUpdate(classID, req)
	if err != nil {
		return nil, err
	}
	class, err := s.ensureExamClassOwner(ctx, tenantID, userID, classID)
	if err != nil {
		return nil, err
	}
	if class.Status != types.ExamClassStatusActive {
		return nil, ErrExamStateConflict
	}
	if err := s.classRepo.UpdateClassMetadata(
		ctx, tenantID, classID, userID, name, description, req.MemberLimit, time.Now(),
	); err != nil {
		return nil, mapExamClassSettingsRepositoryError(err)
	}
	return s.classRepo.GetByIDAndTenantIncludingArchived(ctx, classID, tenantID)
}

func (s *examClassService) ArchiveClass(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
) (*types.ExamClass, error) {
	return s.transitionClassStatus(
		ctx, tenantID, userID, classID, types.ExamClassStatusActive, types.ExamClassStatusArchived,
	)
}

func (s *examClassService) RestoreClass(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
) (*types.ExamClass, error) {
	return s.transitionClassStatus(
		ctx, tenantID, userID, classID, types.ExamClassStatusArchived, types.ExamClassStatusActive,
	)
}

func (s *examClassService) transitionClassStatus(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	expected types.ExamClassStatus,
	next types.ExamClassStatus,
) (*types.ExamClass, error) {
	classID = strings.TrimSpace(classID)
	if classID == "" {
		return nil, ErrExamInvalidRequest
	}
	class, err := s.ensureExamClassOwner(ctx, tenantID, userID, classID)
	if err != nil {
		return nil, err
	}
	if class.Status != expected {
		return nil, ErrExamStateConflict
	}
	if err := s.classRepo.TransitionClassStatus(ctx, tenantID, classID, userID, expected, next, time.Now()); err != nil {
		return nil, mapExamClassSettingsRepositoryError(err)
	}
	return s.classRepo.GetByIDAndTenantIncludingArchived(ctx, classID, tenantID)
}

func (s *examClassService) ensureExamClassOwner(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
) (*types.ExamClass, error) {
	class, err := s.classRepo.GetByIDAndTenantIncludingArchived(ctx, classID, tenantID)
	if err != nil {
		return nil, mapExamClassSettingsRepositoryError(err)
	}
	if class.OwnerUserID != userID {
		return nil, ErrExamPermissionDenied
	}
	if _, err := s.classRepo.GetMember(ctx, classID, tenantID, userID); err != nil {
		if errors.Is(err, repository.ErrExamClassMemberNotFound) {
			return nil, ErrExamPermissionDenied
		}
		return nil, err
	}
	return class, nil
}

func normalizeExamClassUpdate(classID string, req *types.UpdateExamClassRequest) (string, string, error) {
	if classID == "" || req == nil {
		return "", "", ErrExamInvalidRequest
	}
	name := strings.TrimSpace(req.Name)
	description := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(name) == 0 || utf8.RuneCountInString(name) > maxExamClassNameCharacters {
		return "", "", ErrExamInvalidRequest
	}
	if utf8.RuneCountInString(description) > maxExamClassDescriptionCharacters {
		return "", "", ErrExamInvalidRequest
	}
	if req.MemberLimit < 0 || req.MemberLimit > maxExamClassMemberLimit {
		return "", "", ErrExamInvalidRequest
	}
	return name, description, nil
}

func mapExamClassSettingsRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrExamClassNotFound),
		errors.Is(err, repository.ErrExamClassMemberNotFound):
		return ErrExamNotFound
	case errors.Is(err, repository.ErrExamClassStateConflict),
		errors.Is(err, repository.ErrExamClassMemberLimitConflict):
		return ErrExamStateConflict
	default:
		return err
	}
}
