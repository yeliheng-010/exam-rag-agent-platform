package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

var (
	ErrExamNotFound            = errors.New("exam resource not found")
	ErrExamPermissionDenied    = errors.New("exam permission denied")
	ErrExamInvalidRequest      = errors.New("invalid exam request")
	ErrExamDraftQualityBlocked = errors.New("exam draft has blocking quality issues")
)

type examSpaceService struct {
	spaceRepo interfaces.ExamSpaceRepository
	classRepo interfaces.ExamClassRepository
}

func NewExamSpaceService(
	spaceRepo interfaces.ExamSpaceRepository,
	classRepo interfaces.ExamClassRepository,
) interfaces.ExamSpaceService {
	return &examSpaceService{spaceRepo: spaceRepo, classRepo: classRepo}
}

func (s *examSpaceService) ListSpaces(ctx context.Context, tenantID uint64, userID string) ([]*types.ExamSpace, error) {
	spaces, err := s.spaceRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	classes, err := s.classRepo.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	classSpaceIDs := make(map[string]bool, len(classes))
	for _, class := range classes {
		if class != nil {
			classSpaceIDs[class.SpaceID] = true
		}
	}

	out := make([]*types.ExamSpace, 0, len(spaces))
	for _, space := range spaces {
		if space == nil {
			continue
		}
		switch space.SpaceType {
		case types.ExamSpaceTypePublic:
			out = append(out, space)
		case types.ExamSpaceTypePersonal:
			if space.OwnerUserID != nil && *space.OwnerUserID == userID {
				out = append(out, space)
			}
		case types.ExamSpaceTypeClass:
			if classSpaceIDs[space.ID] {
				out = append(out, space)
			}
		}
	}
	return out, nil
}

func (s *examSpaceService) EnsurePersonalSpace(ctx context.Context, tenantID uint64, userID string) (*types.ExamSpace, error) {
	space, err := s.spaceRepo.GetPersonalByOwner(ctx, tenantID, userID)
	if err == nil {
		return space, nil
	}
	if !errors.Is(err, repository.ErrExamSpaceNotFound) {
		return nil, err
	}

	now := time.Now()
	space = &types.ExamSpace{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		OwnerUserID: &userID,
		SpaceType:   types.ExamSpaceTypePersonal,
		Name:        "个人学习空间",
		Description: "个人题库、资料和练习记录的默认空间",
		Status:      types.ExamSpaceStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if createErr := s.spaceRepo.Create(ctx, space); createErr != nil {
		existing, getErr := s.spaceRepo.GetPersonalByOwner(ctx, tenantID, userID)
		if getErr == nil {
			return existing, nil
		}
		return nil, createErr
	}
	return space, nil
}

func (s *examSpaceService) GetSpace(ctx context.Context, tenantID uint64, userID string, spaceID string) (*types.ExamSpace, error) {
	space, err := s.spaceRepo.GetByIDAndTenant(ctx, spaceID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamSpaceNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	ok, err := s.canUseSpace(ctx, tenantID, userID, space, false)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrExamPermissionDenied
	}
	return space, nil
}

func (s *examSpaceService) CanReadSpace(ctx context.Context, tenantID uint64, userID string, spaceID string) (bool, error) {
	space, err := s.spaceRepo.GetByIDAndTenant(ctx, spaceID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamSpaceNotFound) {
			return false, ErrExamNotFound
		}
		return false, err
	}
	return s.canUseSpace(ctx, tenantID, userID, space, false)
}

func (s *examSpaceService) CanWriteSpace(ctx context.Context, tenantID uint64, userID string, spaceID string) (bool, error) {
	space, err := s.spaceRepo.GetByIDAndTenant(ctx, spaceID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamSpaceNotFound) {
			return false, ErrExamNotFound
		}
		return false, err
	}
	return s.canUseSpace(ctx, tenantID, userID, space, true)
}

func (s *examSpaceService) canUseSpace(ctx context.Context, tenantID uint64, userID string, space *types.ExamSpace, write bool) (bool, error) {
	if space == nil || strings.TrimSpace(userID) == "" {
		return false, nil
	}
	switch space.SpaceType {
	case types.ExamSpaceTypePublic:
		return !write, nil
	case types.ExamSpaceTypePersonal:
		return space.OwnerUserID != nil && *space.OwnerUserID == userID, nil
	case types.ExamSpaceTypeClass:
		if types.TenantRoleFromContext(ctx).HasPermission(types.TenantRoleAdmin) {
			return true, nil
		}
		class, err := s.classRepo.GetBySpaceIDAndTenant(ctx, space.ID, tenantID)
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
		if write {
			return member.Role.CanWrite(), nil
		}
		return true, nil
	default:
		return false, nil
	}
}
