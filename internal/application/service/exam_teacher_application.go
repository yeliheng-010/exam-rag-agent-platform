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

type examTeacherApplicationService struct {
	repo          interfaces.ExamTeacherApplicationRepository
	memberService interfaces.TenantMemberService
	userService   interfaces.UserService
}

func NewExamTeacherApplicationService(
	repo interfaces.ExamTeacherApplicationRepository,
	memberService interfaces.TenantMemberService,
	userService interfaces.UserService,
) interfaces.ExamTeacherApplicationService {
	return &examTeacherApplicationService{
		repo:          repo,
		memberService: memberService,
		userService:   userService,
	}
}

func (s *examTeacherApplicationService) Apply(ctx context.Context, tenantID uint64, userID string, req *types.ApplyExamTeacherRequest) (*types.ExamTeacherApplication, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrExamInvalidRequest
	}
	if req == nil {
		req = &types.ApplyExamTeacherRequest{}
	}
	existing, err := s.repo.GetByTenantAndUser(ctx, tenantID, userID)
	if err != nil && !errors.Is(err, repository.ErrExamTeacherApplicationNotFound) {
		return nil, err
	}
	if existing != nil && existing.Status == types.ExamTeacherApplicationStatusApproved {
		return existing, nil
	}
	now := time.Now()
	app := &types.ExamTeacherApplication{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		UserID:    userID,
		Status:    types.ExamTeacherApplicationStatusPending,
		Reason:    strings.TrimSpace(req.Reason),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.UpsertPending(ctx, app); err != nil {
		return nil, err
	}
	return s.repo.GetByTenantAndUser(ctx, tenantID, userID)
}

func (s *examTeacherApplicationService) GetMine(ctx context.Context, tenantID uint64, userID string) (*types.ExamTeacherApplication, error) {
	app, err := s.repo.GetByTenantAndUser(ctx, tenantID, userID)
	if errors.Is(err, repository.ErrExamTeacherApplicationNotFound) {
		return nil, ErrExamNotFound
	}
	return app, err
}

func (s *examTeacherApplicationService) List(ctx context.Context, tenantID uint64, status *types.ExamTeacherApplicationStatus) ([]*types.ExamTeacherApplicationView, error) {
	if !s.callerCanReview(ctx, tenantID) {
		return nil, ErrExamPermissionDenied
	}
	apps, err := s.repo.ListByTenant(ctx, tenantID, status)
	if err != nil {
		return nil, err
	}
	userIDs := make([]string, 0, len(apps)*2)
	for _, app := range apps {
		if app == nil {
			continue
		}
		userIDs = append(userIDs, app.UserID)
		if app.ReviewerID != nil {
			userIDs = append(userIDs, *app.ReviewerID)
		}
	}
	users, err := s.userService.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make([]*types.ExamTeacherApplicationView, 0, len(apps))
	for _, app := range apps {
		if app == nil {
			continue
		}
		view := &types.ExamTeacherApplicationView{ExamTeacherApplication: app}
		if user := users[app.UserID]; user != nil {
			view.UserEmail = user.Email
			view.Username = user.Username
		}
		if app.ReviewerID != nil {
			if reviewer := users[*app.ReviewerID]; reviewer != nil {
				view.ReviewerName = reviewer.Username
			}
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *examTeacherApplicationService) Approve(ctx context.Context, tenantID uint64, reviewerID string, applicationID string, req *types.ReviewExamTeacherApplicationRequest) (*types.ExamTeacherApplication, error) {
	return s.review(ctx, tenantID, reviewerID, applicationID, types.ExamTeacherApplicationStatusApproved, req)
}

func (s *examTeacherApplicationService) Reject(ctx context.Context, tenantID uint64, reviewerID string, applicationID string, req *types.ReviewExamTeacherApplicationRequest) (*types.ExamTeacherApplication, error) {
	return s.review(ctx, tenantID, reviewerID, applicationID, types.ExamTeacherApplicationStatusRejected, req)
}

func (s *examTeacherApplicationService) IsApprovedTeacher(ctx context.Context, tenantID uint64, userID string) (bool, error) {
	app, err := s.repo.GetByTenantAndUser(ctx, tenantID, userID)
	if errors.Is(err, repository.ErrExamTeacherApplicationNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return app.Status == types.ExamTeacherApplicationStatusApproved, nil
}

func (s *examTeacherApplicationService) review(
	ctx context.Context,
	tenantID uint64,
	reviewerID string,
	applicationID string,
	status types.ExamTeacherApplicationStatus,
	req *types.ReviewExamTeacherApplicationRequest,
) (*types.ExamTeacherApplication, error) {
	if strings.TrimSpace(applicationID) == "" || strings.TrimSpace(reviewerID) == "" {
		return nil, ErrExamInvalidRequest
	}
	if !s.callerCanReview(ctx, tenantID) {
		return nil, ErrExamPermissionDenied
	}
	if req == nil {
		req = &types.ReviewExamTeacherApplicationRequest{}
	}
	app, err := s.repo.GetByIDAndTenant(ctx, applicationID, tenantID)
	if errors.Is(err, repository.ErrExamTeacherApplicationNotFound) {
		return nil, ErrExamNotFound
	}
	if err != nil {
		return nil, err
	}
	reviewed, err := s.repo.UpdateReview(ctx, applicationID, tenantID, status, reviewerID, strings.TrimSpace(req.ReviewNote))
	if err != nil {
		return nil, err
	}
	if status == types.ExamTeacherApplicationStatusApproved {
		if err := s.ensureTeacherBaseRole(ctx, tenantID, app.UserID); err != nil {
			return nil, err
		}
	}
	return reviewed, nil
}

func (s *examTeacherApplicationService) callerCanReview(ctx context.Context, tenantID uint64) bool {
	if types.IsSystemAdminFromContext(ctx) {
		return true
	}
	role := types.TenantRoleFromContext(ctx)
	if role.HasPermission(types.TenantRoleAdmin) {
		return true
	}
	userID, ok := types.UserIDFromContext(ctx)
	if !ok || userID == "" || s.memberService == nil {
		return false
	}
	member, err := s.memberService.GetMembership(ctx, userID, tenantID)
	if err != nil || member == nil || member.Status != types.TenantMemberStatusActive {
		return false
	}
	return member.Role.HasPermission(types.TenantRoleAdmin)
}

func (s *examTeacherApplicationService) ensureTeacherBaseRole(ctx context.Context, tenantID uint64, userID string) error {
	if s.memberService == nil {
		return nil
	}
	member, err := s.memberService.GetMembership(ctx, userID, tenantID)
	if err != nil {
		return err
	}
	if member != nil && member.Role.HasPermission(types.TenantRoleContributor) {
		return nil
	}
	return s.memberService.UpdateRole(ctx, userID, tenantID, types.TenantRoleContributor)
}
