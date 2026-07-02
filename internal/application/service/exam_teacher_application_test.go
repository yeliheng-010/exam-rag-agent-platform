package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

type fakeTeacherApplicationRepo struct {
	applications map[string]*types.ExamTeacherApplication
}

func newFakeTeacherApplicationRepo() *fakeTeacherApplicationRepo {
	return &fakeTeacherApplicationRepo{
		applications: map[string]*types.ExamTeacherApplication{},
	}
}

func teacherApplicationKey(tenantID uint64, userID string) string {
	return strconv.FormatUint(tenantID, 10) + "|" + userID
}

func cloneTeacherApplication(app *types.ExamTeacherApplication) *types.ExamTeacherApplication {
	if app == nil {
		return nil
	}
	cp := *app
	return &cp
}

func (r *fakeTeacherApplicationRepo) UpsertPending(_ context.Context, app *types.ExamTeacherApplication) error {
	key := teacherApplicationKey(app.TenantID, app.UserID)
	if existing := r.applications[key]; existing != nil {
		cp := cloneTeacherApplication(existing)
		cp.Status = types.ExamTeacherApplicationStatusPending
		cp.Reason = app.Reason
		cp.ReviewerID = nil
		cp.ReviewNote = ""
		cp.ReviewedAt = nil
		cp.UpdatedAt = app.UpdatedAt
		r.applications[key] = cp
		return nil
	}
	r.applications[key] = cloneTeacherApplication(app)
	return nil
}

func (r *fakeTeacherApplicationRepo) GetByTenantAndUser(_ context.Context, tenantID uint64, userID string) (*types.ExamTeacherApplication, error) {
	app := r.applications[teacherApplicationKey(tenantID, userID)]
	if app == nil {
		return nil, repository.ErrExamTeacherApplicationNotFound
	}
	return cloneTeacherApplication(app), nil
}

func (r *fakeTeacherApplicationRepo) GetByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamTeacherApplication, error) {
	for _, app := range r.applications {
		if app.ID == id && app.TenantID == tenantID {
			return cloneTeacherApplication(app), nil
		}
	}
	return nil, repository.ErrExamTeacherApplicationNotFound
}

func (r *fakeTeacherApplicationRepo) ListByTenant(_ context.Context, tenantID uint64, status *types.ExamTeacherApplicationStatus) ([]*types.ExamTeacherApplication, error) {
	out := []*types.ExamTeacherApplication{}
	for _, app := range r.applications {
		if app.TenantID != tenantID {
			continue
		}
		if status != nil && app.Status != *status {
			continue
		}
		out = append(out, cloneTeacherApplication(app))
	}
	return out, nil
}

func (r *fakeTeacherApplicationRepo) UpdateReview(_ context.Context, id string, tenantID uint64, status types.ExamTeacherApplicationStatus, reviewerID string, reviewNote string) (*types.ExamTeacherApplication, error) {
	for key, app := range r.applications {
		if app.ID != id || app.TenantID != tenantID {
			continue
		}
		cp := cloneTeacherApplication(app)
		cp.Status = status
		cp.ReviewerID = &reviewerID
		cp.ReviewNote = reviewNote
		now := time.Now()
		cp.ReviewedAt = &now
		cp.UpdatedAt = now
		r.applications[key] = cp
		return cloneTeacherApplication(cp), nil
	}
	return nil, repository.ErrExamTeacherApplicationNotFound
}

type fakeTeacherMemberService struct {
	members map[string]*types.TenantMember
}

func newFakeTeacherMemberService() *fakeTeacherMemberService {
	return &fakeTeacherMemberService{members: map[string]*types.TenantMember{}}
}

func (s *fakeTeacherMemberService) setRole(userID string, tenantID uint64, role types.TenantRole) {
	s.members[teacherApplicationKey(tenantID, userID)] = &types.TenantMember{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		Status:   types.TenantMemberStatusActive,
		JoinedAt: time.Now(),
	}
}

func (s *fakeTeacherMemberService) AddMember(context.Context, string, uint64, types.TenantRole, *string) (*types.TenantMember, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherMemberService) EnsureOwner(context.Context, string, uint64) (*types.TenantMember, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherMemberService) GetMembership(_ context.Context, userID string, tenantID uint64) (*types.TenantMember, error) {
	member := s.members[teacherApplicationKey(tenantID, userID)]
	if member == nil {
		return nil, nil
	}
	cp := *member
	return &cp, nil
}

func (s *fakeTeacherMemberService) ListByUser(context.Context, string) ([]*types.TenantMember, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherMemberService) ListByTenant(context.Context, uint64) ([]*types.TenantMember, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherMemberService) ListMembersPage(context.Context, uint64, string, int, int) ([]*types.TenantMember, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *fakeTeacherMemberService) HasAnyMembers(context.Context, uint64) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *fakeTeacherMemberService) UpdateRole(_ context.Context, userID string, tenantID uint64, newRole types.TenantRole) error {
	member := s.members[teacherApplicationKey(tenantID, userID)]
	if member == nil {
		s.setRole(userID, tenantID, newRole)
		return nil
	}
	member.Role = newRole
	return nil
}

func (s *fakeTeacherMemberService) RemoveMember(context.Context, string, uint64) error {
	return errors.New("not implemented")
}

type fakeTeacherUserService struct {
	users map[string]*types.User
}

func newFakeTeacherUserService() *fakeTeacherUserService {
	return &fakeTeacherUserService{users: map[string]*types.User{}}
}

func (s *fakeTeacherUserService) addUser(id string, username string, email string) {
	s.users[id] = &types.User{ID: id, Username: username, Email: email, IsActive: true}
}

func (s *fakeTeacherUserService) Register(context.Context, *types.RegisterRequest) (*types.User, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) Login(context.Context, *types.LoginRequest) (*types.LoginResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) RequestPasswordReset(context.Context, string) (*types.PasswordResetRequestResult, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) ResetPassword(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *fakeTeacherUserService) GetOIDCAuthorizationURL(context.Context, string) (*types.OIDCAuthURLResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) LoginWithOIDC(context.Context, string, string) (*types.OIDCCallbackResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) GetUserByID(_ context.Context, id string) (*types.User, error) {
	user := s.users[id]
	if user == nil {
		return nil, repository.ErrUserNotFound
	}
	cp := *user
	return &cp, nil
}

func (s *fakeTeacherUserService) GetUsersByIDs(_ context.Context, ids []string) (map[string]*types.User, error) {
	out := map[string]*types.User{}
	for _, id := range ids {
		if user := s.users[id]; user != nil {
			cp := *user
			out[id] = &cp
		}
	}
	return out, nil
}

func (s *fakeTeacherUserService) GetUserByEmail(context.Context, string) (*types.User, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) GetUserByUsername(context.Context, string) (*types.User, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) GetUserByTenantID(context.Context, uint64) (*types.User, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) UpdateUser(context.Context, *types.User) error {
	return errors.New("not implemented")
}

func (s *fakeTeacherUserService) DeleteUser(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *fakeTeacherUserService) ChangePassword(context.Context, string, string, string) error {
	return errors.New("not implemented")
}

func (s *fakeTeacherUserService) ValidatePassword(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *fakeTeacherUserService) GenerateTokens(context.Context, *types.User) (string, string, error) {
	return "", "", errors.New("not implemented")
}

func (s *fakeTeacherUserService) BuildLoginMemberships(context.Context, *types.User, *types.Tenant) []types.Membership {
	return nil
}

func (s *fakeTeacherUserService) SwitchTenant(context.Context, *types.User, uint64, string) (*types.LoginResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) ValidateToken(context.Context, string) (*types.User, uint64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *fakeTeacherUserService) RefreshToken(context.Context, string) (string, string, error) {
	return "", "", errors.New("not implemented")
}

func (s *fakeTeacherUserService) RevokeToken(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *fakeTeacherUserService) GetCurrentUser(context.Context) (*types.User, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) SearchUsers(context.Context, string, int) ([]*types.User, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) ListSystemAdmins(context.Context, int, int) ([]*types.User, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *fakeTeacherUserService) RevokeSystemAdmin(context.Context, string, string) (*types.User, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeTeacherUserService) UpdateUserPreferences(context.Context, string, types.UserPreferences) (types.UserPreferences, error) {
	return types.UserPreferences{}, errors.New("not implemented")
}

func newTeacherApplicationServiceForTest(repo *fakeTeacherApplicationRepo, members *fakeTeacherMemberService, users *fakeTeacherUserService) *examTeacherApplicationService {
	return NewExamTeacherApplicationService(repo, members, users).(*examTeacherApplicationService)
}

func examTeacherCallerContext(userID string, role types.TenantRole) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
	ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
	return ctx
}

func TestExamTeacherApplicationApprovalGrantsTeacherCapability(t *testing.T) {
	repo := newFakeTeacherApplicationRepo()
	members := newFakeTeacherMemberService()
	members.setRole("teacher-1", 1, types.TenantRoleViewer)
	members.setRole("admin-1", 1, types.TenantRoleAdmin)
	users := newFakeTeacherUserService()
	users.addUser("teacher-1", "Teacher One", "teacher@example.com")
	svc := newTeacherApplicationServiceForTest(repo, members, users)

	applied, err := svc.Apply(context.Background(), 1, "teacher-1", &types.ApplyExamTeacherRequest{
		Reason: "I teach IELTS reading.",
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if applied.Status != types.ExamTeacherApplicationStatusPending {
		t.Fatalf("application status = %s, want pending", applied.Status)
	}
	if ok, err := svc.IsApprovedTeacher(context.Background(), 1, "teacher-1"); err != nil || ok {
		t.Fatalf("IsApprovedTeacher before approval = %v, %v; want false, nil", ok, err)
	}

	reviewed, err := svc.Approve(examTeacherCallerContext("admin-1", types.TenantRoleAdmin), 1, "admin-1", applied.ID, &types.ReviewExamTeacherApplicationRequest{
		ReviewNote: "approved",
	})
	if err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	if reviewed.Status != types.ExamTeacherApplicationStatusApproved {
		t.Fatalf("reviewed status = %s, want approved", reviewed.Status)
	}
	if ok, err := svc.IsApprovedTeacher(context.Background(), 1, "teacher-1"); err != nil || !ok {
		t.Fatalf("IsApprovedTeacher after approval = %v, %v; want true, nil", ok, err)
	}
	member, err := members.GetMembership(context.Background(), "teacher-1", 1)
	if err != nil {
		t.Fatalf("GetMembership returned error: %v", err)
	}
	if member.Role != types.TenantRoleContributor {
		t.Fatalf("approved teacher tenant role = %s, want contributor", member.Role)
	}
}

func TestExamTeacherApplicationApprovedTeacherCannotResetToPending(t *testing.T) {
	repo := newFakeTeacherApplicationRepo()
	members := newFakeTeacherMemberService()
	members.setRole("teacher-1", 1, types.TenantRoleViewer)
	members.setRole("admin-1", 1, types.TenantRoleAdmin)
	users := newFakeTeacherUserService()
	users.addUser("teacher-1", "Teacher One", "teacher@example.com")
	svc := newTeacherApplicationServiceForTest(repo, members, users)

	applied, err := svc.Apply(context.Background(), 1, "teacher-1", &types.ApplyExamTeacherRequest{
		Reason: "I teach IELTS reading.",
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	reviewed, err := svc.Approve(examTeacherCallerContext("admin-1", types.TenantRoleAdmin), 1, "admin-1", applied.ID, &types.ReviewExamTeacherApplicationRequest{})
	if err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}

	reapplied, err := svc.Apply(context.Background(), 1, "teacher-1", &types.ApplyExamTeacherRequest{
		Reason: "Trying to edit my approved application.",
	})
	if err != nil {
		t.Fatalf("Apply after approval returned error: %v", err)
	}
	if reapplied.Status != types.ExamTeacherApplicationStatusApproved {
		t.Fatalf("reapplied status = %s, want approved", reapplied.Status)
	}
	if reapplied.ID != reviewed.ID {
		t.Fatalf("reapplied id = %s, want existing approved id %s", reapplied.ID, reviewed.ID)
	}
	if ok, err := svc.IsApprovedTeacher(context.Background(), 1, "teacher-1"); err != nil || !ok {
		t.Fatalf("IsApprovedTeacher after reapply = %v, %v; want true, nil", ok, err)
	}
}

func TestExamTeacherApplicationNonAdminCannotReview(t *testing.T) {
	repo := newFakeTeacherApplicationRepo()
	members := newFakeTeacherMemberService()
	members.setRole("teacher-1", 1, types.TenantRoleViewer)
	members.setRole("student-1", 1, types.TenantRoleViewer)
	users := newFakeTeacherUserService()
	users.addUser("teacher-1", "Teacher One", "teacher@example.com")
	svc := newTeacherApplicationServiceForTest(repo, members, users)

	applied, err := svc.Apply(context.Background(), 1, "teacher-1", &types.ApplyExamTeacherRequest{
		Reason: "I teach Gaokao math.",
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	_, err = svc.Approve(examTeacherCallerContext("student-1", types.TenantRoleViewer), 1, "student-1", applied.ID, &types.ReviewExamTeacherApplicationRequest{})
	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("non-admin reviewer error = %v, want ErrExamPermissionDenied", err)
	}
	if ok, err := svc.IsApprovedTeacher(context.Background(), 1, "teacher-1"); err != nil || ok {
		t.Fatalf("IsApprovedTeacher after denied review = %v, %v; want false, nil", ok, err)
	}
}
