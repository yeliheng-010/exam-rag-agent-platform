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

const defaultExamClassMemberLimit = 50

type examClassService struct {
	classRepo  interfaces.ExamClassRepository
	spaceRepo  interfaces.ExamSpaceRepository
	domainRepo interfaces.ExamDomainRepository
}

func NewExamClassService(
	classRepo interfaces.ExamClassRepository,
	spaceRepo interfaces.ExamSpaceRepository,
	domainRepo interfaces.ExamDomainRepository,
) interfaces.ExamClassService {
	return &examClassService{
		classRepo:  classRepo,
		spaceRepo:  spaceRepo,
		domainRepo: domainRepo,
	}
}

func (s *examClassService) CreateClass(ctx context.Context, tenantID uint64, userID string, req *types.CreateExamClassRequest) (*types.ExamClass, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrExamInvalidRequest
	}
	memberLimit := defaultExamClassMemberLimit
	if req.MemberLimit != nil {
		if *req.MemberLimit < 0 {
			return nil, ErrExamInvalidRequest
		}
		memberLimit = *req.MemberLimit
	}
	if req.DomainID != nil && strings.TrimSpace(*req.DomainID) != "" {
		domainID := strings.TrimSpace(*req.DomainID)
		if _, err := s.domainRepo.GetDomainByID(ctx, domainID); err != nil {
			if errors.Is(err, repository.ErrExamDomainNotFound) {
				return nil, ErrExamNotFound
			}
			return nil, err
		}
		req.DomainID = &domainID
	} else {
		req.DomainID = nil
	}

	now := time.Now()
	spaceOwner := userID
	space := &types.ExamSpace{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		OwnerUserID: &spaceOwner,
		SpaceType:   types.ExamSpaceTypeClass,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Status:      types.ExamSpaceStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.spaceRepo.Create(ctx, space); err != nil {
		return nil, err
	}

	class := &types.ExamClass{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		OwnerUserID: userID,
		SpaceID:     space.ID,
		DomainID:    req.DomainID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		MemberLimit: memberLimit,
		Status:      types.ExamClassStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.classRepo.CreateClass(ctx, class); err != nil {
		return nil, err
	}
	member := &types.ExamClassMember{
		ID:        uuid.New().String(),
		ClassID:   class.ID,
		UserID:    userID,
		TenantID:  tenantID,
		Role:      types.ExamClassRoleTeacher,
		Status:    types.ExamClassMemberStatusActive,
		JoinedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.classRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}
	return class, nil
}

func (s *examClassService) ListClasses(ctx context.Context, tenantID uint64, userID string) ([]*types.ExamClass, error) {
	return s.classRepo.ListByUser(ctx, tenantID, userID)
}

func (s *examClassService) GetClass(ctx context.Context, tenantID uint64, userID string, classID string) (*types.ExamClass, error) {
	class, err := s.classRepo.GetByIDAndTenant(ctx, classID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	if _, err := s.classRepo.GetMember(ctx, class.ID, tenantID, userID); err != nil {
		if errors.Is(err, repository.ErrExamClassMemberNotFound) {
			return nil, ErrExamPermissionDenied
		}
		return nil, err
	}
	return class, nil
}

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
