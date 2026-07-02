package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type registerUserRepo struct {
	interfaces.UserRepository
	created *types.User
}

func (r *registerUserRepo) GetUserByEmail(context.Context, string) (*types.User, error) {
	return nil, nil
}

func (r *registerUserRepo) GetUserByUsername(context.Context, string) (*types.User, error) {
	return nil, nil
}

func (r *registerUserRepo) CreateUser(_ context.Context, user *types.User) error {
	cp := *user
	r.created = &cp
	return nil
}

type registerTenantService struct {
	interfaces.TenantService
	tenant *types.Tenant
}

func (s *registerTenantService) CreateTenant(_ context.Context, tenant *types.Tenant) (*types.Tenant, error) {
	cp := *tenant
	cp.ID = 42
	s.tenant = &cp
	return &cp, nil
}

type registerMemberService struct {
	interfaces.TenantMemberService
	addCalls []struct {
		userID   string
		tenantID uint64
		role     types.TenantRole
	}
	ensureOwnerCalls int
}

func (s *registerMemberService) AddMember(
	_ context.Context,
	userID string,
	tenantID uint64,
	role types.TenantRole,
	_ *string,
) (*types.TenantMember, error) {
	s.addCalls = append(s.addCalls, struct {
		userID   string
		tenantID uint64
		role     types.TenantRole
	}{userID: userID, tenantID: tenantID, role: role})
	return &types.TenantMember{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		Status:   types.TenantMemberStatusActive,
	}, nil
}

func (s *registerMemberService) EnsureOwner(
	context.Context,
	string,
	uint64,
) (*types.TenantMember, error) {
	s.ensureOwnerCalls++
	return &types.TenantMember{Role: types.TenantRoleOwner}, nil
}

func TestRegisterCreatesViewerMembershipForPublicSignup(t *testing.T) {
	userRepo := &registerUserRepo{}
	tenantSvc := &registerTenantService{}
	memberSvc := &registerMemberService{}
	svc := NewUserService(nil, userRepo, nil, tenantSvc, memberSvc)

	user, err := svc.Register(context.Background(), &types.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "secret-password",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if user == nil {
		t.Fatal("Register returned nil user")
	}
	if user.TenantID != 42 {
		t.Fatalf("registered user TenantID = %d, want 42", user.TenantID)
	}
	if userRepo.created == nil {
		t.Fatal("Register did not create user")
	}
	if tenantSvc.tenant == nil {
		t.Fatal("Register did not create tenant")
	}
	if memberSvc.ensureOwnerCalls != 0 {
		t.Fatalf("public signup must not call EnsureOwner, got %d calls", memberSvc.ensureOwnerCalls)
	}
	if len(memberSvc.addCalls) != 1 {
		t.Fatalf("expected one AddMember call, got %+v", memberSvc.addCalls)
	}
	call := memberSvc.addCalls[0]
	if call.userID != user.ID || call.tenantID != 42 || call.role != types.TenantRoleViewer {
		t.Fatalf("AddMember call = %+v, want user=%s tenant=42 role=viewer", call, user.ID)
	}
}
