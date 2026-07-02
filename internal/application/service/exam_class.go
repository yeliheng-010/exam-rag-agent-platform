package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

const defaultExamClassMemberLimit = 50

type examClassService struct {
	classRepo            interfaces.ExamClassRepository
	spaceRepo            interfaces.ExamSpaceRepository
	domainRepo           interfaces.ExamDomainRepository
	teacherAccessService interfaces.ExamTeacherAccessService
}

func NewExamClassService(
	classRepo interfaces.ExamClassRepository,
	spaceRepo interfaces.ExamSpaceRepository,
	domainRepo interfaces.ExamDomainRepository,
	teacherAccessService interfaces.ExamTeacherAccessService,
) interfaces.ExamClassService {
	return &examClassService{
		classRepo:            classRepo,
		spaceRepo:            spaceRepo,
		domainRepo:           domainRepo,
		teacherAccessService: teacherAccessService,
	}
}

func (s *examClassService) CreateClass(ctx context.Context, tenantID uint64, userID string, req *types.CreateExamClassRequest) (*types.ExamClass, error) {
	if s.teacherAccessService != nil {
		approved, err := s.teacherAccessService.IsApprovedTeacher(ctx, tenantID, userID)
		if err != nil {
			return nil, err
		}
		if !approved {
			return nil, ErrExamPermissionDenied
		}
	}
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

	inviteCode, err := s.generateInviteCode(ctx, tenantID)
	if err != nil {
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
		InviteCode:  &inviteCode,
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

func (s *examClassService) RequestJoinClass(ctx context.Context, tenantID uint64, userID string, req *types.JoinExamClassRequest) (*types.ExamClassMember, error) {
	if req == nil {
		return nil, ErrExamInvalidRequest
	}
	inviteCode := strings.ToUpper(strings.TrimSpace(req.InviteCode))
	if inviteCode == "" {
		return nil, ErrExamInvalidRequest
	}
	class, err := s.classRepo.GetByInviteCodeAndTenant(ctx, inviteCode, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	if class.InviteCodeExpiresAt != nil && time.Now().After(*class.InviteCodeExpiresAt) {
		return nil, ErrExamInvalidRequest
	}
	if existing, err := s.classRepo.GetAnyMember(ctx, class.ID, tenantID, userID); err == nil {
		if existing.Status == types.ExamClassMemberStatusRemoved {
			return s.classRepo.UpdateMemberStatus(ctx, class.ID, tenantID, userID, types.ExamClassMemberStatusPending)
		}
		return existing, nil
	} else if !errors.Is(err, repository.ErrExamClassMemberNotFound) {
		return nil, err
	}

	now := time.Now()
	member := &types.ExamClassMember{
		ID:        uuid.New().String(),
		ClassID:   class.ID,
		UserID:    userID,
		TenantID:  tenantID,
		Role:      types.ExamClassRoleStudent,
		Status:    types.ExamClassMemberStatusPending,
		JoinedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.classRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}
	return member, nil
}

func (s *examClassService) ListClassMembers(ctx context.Context, tenantID uint64, userID string, classID string) ([]*types.ExamClassMember, error) {
	if err := s.ensureCanReviewClassMembers(ctx, tenantID, userID, classID); err != nil {
		return nil, err
	}
	return s.classRepo.ListMembers(ctx, classID, tenantID, []types.ExamClassMemberStatus{
		types.ExamClassMemberStatusPending,
		types.ExamClassMemberStatusActive,
	})
}

func (s *examClassService) ApproveClassMember(ctx context.Context, tenantID uint64, reviewerID string, classID string, targetUserID string) (*types.ExamClassMember, error) {
	if strings.TrimSpace(targetUserID) == "" {
		return nil, ErrExamInvalidRequest
	}
	if err := s.ensureCanReviewClassMembers(ctx, tenantID, reviewerID, classID); err != nil {
		return nil, err
	}
	member, err := s.classRepo.GetAnyMember(ctx, classID, tenantID, targetUserID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassMemberNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	if member.Role != types.ExamClassRoleStudent || member.Status != types.ExamClassMemberStatusPending {
		return nil, ErrExamInvalidRequest
	}
	return s.classRepo.UpdateMemberStatus(ctx, classID, tenantID, targetUserID, types.ExamClassMemberStatusActive)
}

func (s *examClassService) RejectClassMember(ctx context.Context, tenantID uint64, reviewerID string, classID string, targetUserID string) (*types.ExamClassMember, error) {
	if strings.TrimSpace(targetUserID) == "" {
		return nil, ErrExamInvalidRequest
	}
	if err := s.ensureCanReviewClassMembers(ctx, tenantID, reviewerID, classID); err != nil {
		return nil, err
	}
	member, err := s.classRepo.GetAnyMember(ctx, classID, tenantID, targetUserID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassMemberNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	if member.Role != types.ExamClassRoleStudent || member.Status != types.ExamClassMemberStatusPending {
		return nil, ErrExamInvalidRequest
	}
	return s.classRepo.UpdateMemberStatus(ctx, classID, tenantID, targetUserID, types.ExamClassMemberStatusRemoved)
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

func (s *examClassService) ensureCanReviewClassMembers(ctx context.Context, tenantID uint64, userID string, classID string) error {
	if strings.TrimSpace(classID) == "" {
		return ErrExamInvalidRequest
	}
	class, err := s.classRepo.GetByIDAndTenant(ctx, classID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassNotFound) {
			return ErrExamNotFound
		}
		return err
	}
	member, err := s.classRepo.GetMember(ctx, class.ID, tenantID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassMemberNotFound) {
			return ErrExamPermissionDenied
		}
		return err
	}
	if !member.Role.CanWrite() {
		return ErrExamPermissionDenied
	}
	return nil
}

func (s *examClassService) generateInviteCode(ctx context.Context, tenantID uint64) (string, error) {
	for i := 0; i < 8; i++ {
		code, err := randomExamInviteCode()
		if err != nil {
			return "", err
		}
		if _, err := s.classRepo.GetByInviteCodeAndTenant(ctx, code, tenantID); errors.Is(err, repository.ErrExamClassNotFound) {
			return code, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", errors.New("failed to generate unique class invite code")
}

func randomExamInviteCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}
	for i, b := range buf {
		buf[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(buf), nil
}
