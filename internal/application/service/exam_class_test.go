package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

type fakeExamClassRepo struct {
	classes map[string]*types.ExamClass
	members map[string]*types.ExamClassMember
}

func newFakeExamClassRepo() *fakeExamClassRepo {
	return &fakeExamClassRepo{
		classes: map[string]*types.ExamClass{},
		members: map[string]*types.ExamClassMember{},
	}
}

func classMemberKey(classID, userID string) string {
	return classID + "|" + userID
}

func cloneExamClass(c *types.ExamClass) *types.ExamClass {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}

func cloneExamClassMember(m *types.ExamClassMember) *types.ExamClassMember {
	if m == nil {
		return nil
	}
	cp := *m
	return &cp
}

func (r *fakeExamClassRepo) CreateClass(_ context.Context, class *types.ExamClass) error {
	r.classes[class.ID] = cloneExamClass(class)
	return nil
}

func (r *fakeExamClassRepo) AddMember(_ context.Context, member *types.ExamClassMember) error {
	r.members[classMemberKey(member.ClassID, member.UserID)] = cloneExamClassMember(member)
	return nil
}

func (r *fakeExamClassRepo) ListByUser(_ context.Context, tenantID uint64, userID string) ([]*types.ExamClass, error) {
	out := []*types.ExamClass{}
	for _, member := range r.members {
		if member.TenantID != tenantID || member.UserID != userID || member.Status != types.ExamClassMemberStatusActive {
			continue
		}
		if class := r.classes[member.ClassID]; class != nil && class.Status == types.ExamClassStatusActive {
			out = append(out, cloneExamClass(class))
		}
	}
	return out, nil
}

func (r *fakeExamClassRepo) GetByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamClass, error) {
	class := r.classes[id]
	if class == nil || class.TenantID != tenantID || class.Status != types.ExamClassStatusActive {
		return nil, repository.ErrExamClassNotFound
	}
	return cloneExamClass(class), nil
}

func (r *fakeExamClassRepo) GetBySpaceIDAndTenant(_ context.Context, spaceID string, tenantID uint64) (*types.ExamClass, error) {
	for _, class := range r.classes {
		if class.SpaceID == spaceID && class.TenantID == tenantID && class.Status == types.ExamClassStatusActive {
			return cloneExamClass(class), nil
		}
	}
	return nil, repository.ErrExamClassNotFound
}

func (r *fakeExamClassRepo) GetByInviteCodeAndTenant(_ context.Context, inviteCode string, tenantID uint64) (*types.ExamClass, error) {
	for _, class := range r.classes {
		if class.InviteCode != nil && *class.InviteCode == inviteCode && class.TenantID == tenantID && class.Status == types.ExamClassStatusActive {
			return cloneExamClass(class), nil
		}
	}
	return nil, repository.ErrExamClassNotFound
}

func (r *fakeExamClassRepo) GetMember(_ context.Context, classID string, tenantID uint64, userID string) (*types.ExamClassMember, error) {
	member := r.members[classMemberKey(classID, userID)]
	if member == nil || member.TenantID != tenantID || member.Status != types.ExamClassMemberStatusActive {
		return nil, repository.ErrExamClassMemberNotFound
	}
	return cloneExamClassMember(member), nil
}

func (r *fakeExamClassRepo) GetAnyMember(_ context.Context, classID string, tenantID uint64, userID string) (*types.ExamClassMember, error) {
	member := r.members[classMemberKey(classID, userID)]
	if member == nil || member.TenantID != tenantID {
		return nil, repository.ErrExamClassMemberNotFound
	}
	return cloneExamClassMember(member), nil
}

func (r *fakeExamClassRepo) ListMembers(_ context.Context, classID string, tenantID uint64, statuses []types.ExamClassMemberStatus) ([]*types.ExamClassMember, error) {
	allowed := map[types.ExamClassMemberStatus]bool{}
	for _, status := range statuses {
		allowed[status] = true
	}
	out := []*types.ExamClassMember{}
	for _, member := range r.members {
		if member.ClassID != classID || member.TenantID != tenantID {
			continue
		}
		if len(allowed) > 0 && !allowed[member.Status] {
			continue
		}
		out = append(out, cloneExamClassMember(member))
	}
	return out, nil
}

func (r *fakeExamClassRepo) UpdateMemberStatus(_ context.Context, classID string, tenantID uint64, userID string, status types.ExamClassMemberStatus) (*types.ExamClassMember, error) {
	key := classMemberKey(classID, userID)
	member := r.members[key]
	if member == nil || member.TenantID != tenantID {
		return nil, repository.ErrExamClassMemberNotFound
	}
	cp := cloneExamClassMember(member)
	cp.Status = status
	cp.UpdatedAt = time.Now()
	r.members[key] = cp
	return cloneExamClassMember(cp), nil
}

type fakeExamSpaceRepo struct {
	created []*types.ExamSpace
}

func (r *fakeExamSpaceRepo) Create(_ context.Context, space *types.ExamSpace) error {
	cp := *space
	r.created = append(r.created, &cp)
	return nil
}

func (r *fakeExamSpaceRepo) ListByTenant(context.Context, uint64) ([]*types.ExamSpace, error) {
	return nil, nil
}

func (r *fakeExamSpaceRepo) GetByIDAndTenant(context.Context, string, uint64) (*types.ExamSpace, error) {
	return nil, repository.ErrExamSpaceNotFound
}

func (r *fakeExamSpaceRepo) GetPersonalByOwner(context.Context, uint64, string) (*types.ExamSpace, error) {
	return nil, repository.ErrExamSpaceNotFound
}

type fakeExamDomainRepo struct{}

func (r *fakeExamDomainRepo) ListDomains(context.Context) ([]*types.ExamDomain, error) {
	return nil, nil
}

func (r *fakeExamDomainRepo) GetDomainByID(_ context.Context, id string) (*types.ExamDomain, error) {
	return &types.ExamDomain{ID: id, Status: types.ExamDomainStatusActive}, nil
}

func (r *fakeExamDomainRepo) ListSubjects(context.Context, string) ([]*types.ExamSubject, error) {
	return nil, nil
}

func (r *fakeExamDomainRepo) GetSubjectByID(context.Context, string) (*types.ExamSubject, error) {
	return nil, repository.ErrExamSubjectNotFound
}

func seedExamClass(repo *fakeExamClassRepo, classID string, tenantID uint64, ownerID string, inviteCode string) *types.ExamClass {
	class := &types.ExamClass{
		ID:          classID,
		TenantID:    tenantID,
		OwnerUserID: ownerID,
		SpaceID:     "space-" + classID,
		Name:        "IELTS Class",
		InviteCode:  &inviteCode,
		MemberLimit: 50,
		Status:      types.ExamClassStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.classes[class.ID] = class
	return class
}

func seedExamClassMember(repo *fakeExamClassRepo, classID string, tenantID uint64, userID string, role types.ExamClassRole, status types.ExamClassMemberStatus) {
	member := &types.ExamClassMember{
		ID:        "member-" + userID,
		ClassID:   classID,
		UserID:    userID,
		TenantID:  tenantID,
		Role:      role,
		Status:    status,
		JoinedAt:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.members[classMemberKey(classID, userID)] = member
}

func TestExamClassJoinRequiresTeacherApproval(t *testing.T) {
	repo := newFakeExamClassRepo()
	seedExamClass(repo, "class-1", 1, "teacher-1", "CLASSCODE")
	seedExamClassMember(repo, "class-1", 1, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	svc := NewExamClassService(repo, &fakeExamSpaceRepo{}, &fakeExamDomainRepo{})

	member, err := svc.RequestJoinClass(context.Background(), 1, "student-1", &types.JoinExamClassRequest{
		InviteCode: " classcode ",
	})
	if err != nil {
		t.Fatalf("RequestJoinClass returned error: %v", err)
	}
	if member.Role != types.ExamClassRoleStudent || member.Status != types.ExamClassMemberStatusPending {
		t.Fatalf("join request member = role %s status %s, want student/pending", member.Role, member.Status)
	}

	if _, err := svc.GetClass(context.Background(), 1, "student-1", "class-1"); !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("pending student should not access class, got %v", err)
	}

	approved, err := svc.ApproveClassMember(context.Background(), 1, "teacher-1", "class-1", "student-1")
	if err != nil {
		t.Fatalf("ApproveClassMember returned error: %v", err)
	}
	if approved.Status != types.ExamClassMemberStatusActive {
		t.Fatalf("approved status = %s, want active", approved.Status)
	}

	if _, err := svc.GetClass(context.Background(), 1, "student-1", "class-1"); err != nil {
		t.Fatalf("approved student should access class, got %v", err)
	}
}

func TestExamClassOnlyTeacherOrAssistantCanReviewJoinRequests(t *testing.T) {
	repo := newFakeExamClassRepo()
	seedExamClass(repo, "class-1", 1, "teacher-1", "CLASSCODE")
	seedExamClassMember(repo, "class-1", 1, "student-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	seedExamClassMember(repo, "class-1", 1, "student-2", types.ExamClassRoleStudent, types.ExamClassMemberStatusPending)
	svc := NewExamClassService(repo, &fakeExamSpaceRepo{}, &fakeExamDomainRepo{})

	_, err := svc.ApproveClassMember(context.Background(), 1, "student-1", "class-1", "student-2")
	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("student reviewer should be denied, got %v", err)
	}
	member, err := repo.GetAnyMember(context.Background(), "class-1", 1, "student-2")
	if err != nil {
		t.Fatalf("expected pending member to remain: %v", err)
	}
	if member.Status != types.ExamClassMemberStatusPending {
		t.Fatalf("denied approval changed status to %s", member.Status)
	}
}
