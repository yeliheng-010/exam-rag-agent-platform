package service

import (
	"context"
	"testing"
	"time"

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
	tenant      *types.Tenant
	tenants     []*types.Tenant
	createCalls int
	nextID      uint64
}

func (s *registerTenantService) ListTenants(context.Context) ([]*types.Tenant, error) {
	out := make([]*types.Tenant, 0, len(s.tenants))
	for _, tenant := range s.tenants {
		cp := *tenant
		out = append(out, &cp)
	}
	return out, nil
}

func (s *registerTenantService) CreateTenant(_ context.Context, tenant *types.Tenant) (*types.Tenant, error) {
	cp := *tenant
	if s.nextID == 0 {
		s.nextID = 42
	}
	cp.ID = s.nextID
	cp.CreatedAt = time.Now()
	cp.UpdatedAt = cp.CreatedAt
	s.createCalls++
	s.tenant = &cp
	s.tenants = append(s.tenants, &cp)
	return &cp, nil
}

type registerMemberService struct {
	interfaces.TenantMemberService
	addCalls []struct {
		userID   string
		tenantID uint64
		role     types.TenantRole
	}
	ensureOwnerCalls []struct {
		userID   string
		tenantID uint64
	}
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
	_ context.Context,
	userID string,
	tenantID uint64,
) (*types.TenantMember, error) {
	s.ensureOwnerCalls = append(s.ensureOwnerCalls, struct {
		userID   string
		tenantID uint64
	}{userID: userID, tenantID: tenantID})
	return &types.TenantMember{UserID: userID, TenantID: tenantID, Role: types.TenantRoleOwner}, nil
}

func TestRegisterBootstrapsSharedSchoolTenantOwnerForFirstSignup(t *testing.T) {
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
	if tenantSvc.tenant.Name != defaultExamSchoolTenantName {
		t.Fatalf("created tenant name = %q, want %q", tenantSvc.tenant.Name, defaultExamSchoolTenantName)
	}
	if len(memberSvc.addCalls) != 0 {
		t.Fatalf("first signup must not add viewer membership, got %+v", memberSvc.addCalls)
	}
	if len(memberSvc.ensureOwnerCalls) != 1 {
		t.Fatalf("expected one EnsureOwner call, got %+v", memberSvc.ensureOwnerCalls)
	}
	call := memberSvc.ensureOwnerCalls[0]
	if call.userID != user.ID || call.tenantID != 42 {
		t.Fatalf("EnsureOwner call = %+v, want user=%s tenant=42", call, user.ID)
	}
}

func TestRegisterJoinsExistingSchoolTenantAsViewer(t *testing.T) {
	userRepo := &registerUserRepo{}
	tenantSvc := &registerTenantService{
		tenants: []*types.Tenant{
			{ID: 200, Name: "Newer Workspace", Status: "active", CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
			{ID: 100, Name: "Exam RAG School", Status: "active", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	memberSvc := &registerMemberService{}
	svc := NewUserService(nil, userRepo, nil, tenantSvc, memberSvc)

	user, err := svc.Register(context.Background(), &types.RegisterRequest{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "secret-password",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if user.TenantID != 100 {
		t.Fatalf("registered user TenantID = %d, want oldest school tenant 100", user.TenantID)
	}
	if tenantSvc.createCalls != 0 {
		t.Fatalf("existing school registration must not create tenant, got %d calls", tenantSvc.createCalls)
	}
	if len(memberSvc.ensureOwnerCalls) != 0 {
		t.Fatalf("existing school registration must not call EnsureOwner, got %+v", memberSvc.ensureOwnerCalls)
	}
	if len(memberSvc.addCalls) != 1 {
		t.Fatalf("expected one AddMember call, got %+v", memberSvc.addCalls)
	}
	call := memberSvc.addCalls[0]
	if call.userID != user.ID || call.tenantID != 100 || call.role != types.TenantRoleViewer {
		t.Fatalf("AddMember call = %+v, want user=%s tenant=100 role=viewer", call, user.ID)
	}
}
