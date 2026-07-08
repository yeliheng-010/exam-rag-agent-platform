package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

func TestExamResourceBindAllowsClassWriterToBindTenantKnowledgeBase(t *testing.T) {
	ctx := examResourceCallerContext(types.TenantRoleViewer)
	resourceRepo := newFakeExamResourceRepo()
	space := &types.ExamSpace{ID: "space-class", TenantID: 10000, SpaceType: types.ExamSpaceTypeClass}
	svc := NewExamResourceService(
		resourceRepo,
		&stubExamResourceSpaceService{space: space, canWrite: true},
		&fakeExamDomainRepo{},
		&stubExamResourceKBService{kb: &types.KnowledgeBase{
			ID:        "kb-1",
			TenantID:  10000,
			CreatorID: "admin-1",
		}},
	)

	resource, err := svc.BindKnowledgeBase(ctx, 10000, "teacher-1", "kb-1", &types.BindKnowledgeBaseResourceRequest{
		SpaceID:      "space-class",
		DomainID:     "gaokao",
		MaterialType: types.ExamMaterialTypeExamPaper,
	})

	if err != nil {
		t.Fatalf("BindKnowledgeBase returned error: %v", err)
	}
	if resource.SpaceID != "space-class" || resource.CreatedByUserID != "teacher-1" {
		t.Fatalf("resource binding = space %s creator %s, want space-class/teacher-1", resource.SpaceID, resource.CreatedByUserID)
	}
	if resource.ReviewStatus != types.ExamReviewStatusPrivate {
		t.Fatalf("class space review status = %s, want private", resource.ReviewStatus)
	}
}

func TestExamResourceBindRejectsClassSpaceWithoutWritePermission(t *testing.T) {
	ctx := examResourceCallerContext(types.TenantRoleViewer)
	resourceRepo := newFakeExamResourceRepo()
	space := &types.ExamSpace{ID: "space-class", TenantID: 10000, SpaceType: types.ExamSpaceTypeClass}
	svc := NewExamResourceService(
		resourceRepo,
		&stubExamResourceSpaceService{space: space, canWrite: false},
		&fakeExamDomainRepo{},
		&stubExamResourceKBService{kb: &types.KnowledgeBase{
			ID:        "kb-1",
			TenantID:  10000,
			CreatorID: "admin-1",
		}},
	)

	_, err := svc.BindKnowledgeBase(ctx, 10000, "student-1", "kb-1", &types.BindKnowledgeBaseResourceRequest{
		SpaceID:  "space-class",
		DomainID: "gaokao",
	})

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("BindKnowledgeBase error = %v, want ErrExamPermissionDenied", err)
	}
	if resourceRepo.saved != nil {
		t.Fatalf("student created resource binding: %#v", resourceRepo.saved)
	}
}

func TestExamResourceBindRejectsCrossTenantKnowledgeBase(t *testing.T) {
	ctx := examResourceCallerContext(types.TenantRoleOwner)
	resourceRepo := newFakeExamResourceRepo()
	space := &types.ExamSpace{ID: "space-class", TenantID: 10000, SpaceType: types.ExamSpaceTypeClass}
	svc := NewExamResourceService(
		resourceRepo,
		&stubExamResourceSpaceService{space: space, canWrite: true},
		&fakeExamDomainRepo{},
		&stubExamResourceKBService{kb: &types.KnowledgeBase{
			ID:        "kb-1",
			TenantID:  20000,
			CreatorID: "teacher-1",
		}},
	)

	_, err := svc.BindKnowledgeBase(ctx, 10000, "teacher-1", "kb-1", &types.BindKnowledgeBaseResourceRequest{
		SpaceID:  "space-class",
		DomainID: "gaokao",
	})

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("BindKnowledgeBase error = %v, want ErrExamPermissionDenied", err)
	}
	if resourceRepo.saved != nil {
		t.Fatalf("cross-tenant resource binding was saved: %#v", resourceRepo.saved)
	}
}

func TestExamResourceBindAllowsTenantAdminToBindClassKnowledgeBase(t *testing.T) {
	ctx := examResourceCallerContext(types.TenantRoleAdmin)
	classRepo := newFakeExamClassRepo()
	class := seedExamClass(classRepo, "class-1", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, class.ID, 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	spaceSvc := NewExamSpaceService(&fakeExamResourceSpaceRepo{
		spaces: map[string]*types.ExamSpace{
			class.SpaceID: {
				ID:        class.SpaceID,
				TenantID:  10000,
				SpaceType: types.ExamSpaceTypeClass,
				Name:      class.Name,
				Status:    types.ExamSpaceStatusActive,
			},
		},
	}, classRepo)
	resourceRepo := newFakeExamResourceRepo()
	svc := NewExamResourceService(
		resourceRepo,
		spaceSvc,
		&fakeExamDomainRepo{},
		&stubExamResourceKBService{kb: &types.KnowledgeBase{
			ID:        "kb-1",
			TenantID:  10000,
			CreatorID: "teacher-1",
		}},
	)

	resource, err := svc.BindKnowledgeBase(ctx, 10000, "admin-1", "kb-1", &types.BindKnowledgeBaseResourceRequest{
		SpaceID:  class.SpaceID,
		DomainID: "gaokao",
	})

	if err != nil {
		t.Fatalf("tenant admin BindKnowledgeBase returned error: %v", err)
	}
	if resource.SpaceID != class.SpaceID || resource.CreatedByUserID != "admin-1" {
		t.Fatalf("admin resource binding = space %s creator %s, want %s/admin-1", resource.SpaceID, resource.CreatedByUserID, class.SpaceID)
	}
}

func examResourceCallerContext(role types.TenantRole) context.Context {
	return context.WithValue(context.Background(), types.TenantRoleContextKey, role)
}

type fakeExamResourceRepo struct {
	saved     *types.ExamSpaceResource
	resources []*types.ExamSpaceResource
}

func newFakeExamResourceRepo() *fakeExamResourceRepo {
	return &fakeExamResourceRepo{}
}

func (r *fakeExamResourceRepo) Upsert(_ context.Context, resource *types.ExamSpaceResource) error {
	r.saved = cloneExamSpaceResource(resource)
	for i, existing := range r.resources {
		if existing.TenantID == resource.TenantID &&
			existing.SpaceID == resource.SpaceID &&
			existing.ResourceType == resource.ResourceType &&
			existing.ResourceID == resource.ResourceID {
			r.resources[i] = cloneExamSpaceResource(resource)
			return nil
		}
	}
	r.resources = append(r.resources, cloneExamSpaceResource(resource))
	return nil
}

func (r *fakeExamResourceRepo) GetByResource(_ context.Context, tenantID uint64, resourceType types.ExamResourceType, resourceID string) (*types.ExamSpaceResource, error) {
	for _, resource := range r.resources {
		if resource.TenantID == tenantID && resource.ResourceType == resourceType && resource.ResourceID == resourceID {
			return cloneExamSpaceResource(resource), nil
		}
	}
	return nil, repository.ErrExamResourceNotFound
}

func (r *fakeExamResourceRepo) GetBySpaceResource(_ context.Context, tenantID uint64, spaceID string, resourceType types.ExamResourceType, resourceID string) (*types.ExamSpaceResource, error) {
	for _, resource := range r.resources {
		if resource.TenantID == tenantID && resource.SpaceID == spaceID && resource.ResourceType == resourceType && resource.ResourceID == resourceID {
			return cloneExamSpaceResource(resource), nil
		}
	}
	return nil, repository.ErrExamResourceNotFound
}

func (r *fakeExamResourceRepo) ListByResource(_ context.Context, tenantID uint64, resourceType types.ExamResourceType, resourceID string) ([]*types.ExamSpaceResource, error) {
	out := []*types.ExamSpaceResource{}
	for _, resource := range r.resources {
		if resource.TenantID == tenantID && resource.ResourceType == resourceType && resource.ResourceID == resourceID {
			out = append(out, cloneExamSpaceResource(resource))
		}
	}
	return out, nil
}

func (r *fakeExamResourceRepo) List(_ context.Context, tenantID uint64, filter types.ListExamResourcesFilter, spaceIDs []string) ([]*types.ExamSpaceResource, error) {
	allowed := map[string]bool{}
	for _, spaceID := range spaceIDs {
		allowed[spaceID] = true
	}
	out := []*types.ExamSpaceResource{}
	for _, resource := range r.resources {
		if resource.TenantID != tenantID || !allowed[resource.SpaceID] {
			continue
		}
		if filter.ResourceType != "" && resource.ResourceType != filter.ResourceType {
			continue
		}
		out = append(out, cloneExamSpaceResource(resource))
	}
	return out, nil
}

type stubExamResourceSpaceService struct {
	space    *types.ExamSpace
	canWrite bool
}

func (s *stubExamResourceSpaceService) ListSpaces(context.Context, uint64, string) ([]*types.ExamSpace, error) {
	return nil, nil
}

func (s *stubExamResourceSpaceService) EnsurePersonalSpace(context.Context, uint64, string) (*types.ExamSpace, error) {
	return nil, nil
}

func (s *stubExamResourceSpaceService) CanReadSpace(context.Context, uint64, string, string) (bool, error) {
	return true, nil
}

func (s *stubExamResourceSpaceService) CanWriteSpace(_ context.Context, _ uint64, _ string, spaceID string) (bool, error) {
	return s.space != nil && s.space.ID == spaceID && s.canWrite, nil
}

func (s *stubExamResourceSpaceService) GetSpace(_ context.Context, tenantID uint64, _ string, spaceID string) (*types.ExamSpace, error) {
	if s.space == nil || s.space.ID != spaceID || s.space.TenantID != tenantID {
		return nil, ErrExamNotFound
	}
	cp := *s.space
	return &cp, nil
}

type stubExamResourceKBService struct {
	interfaces.KnowledgeBaseService
	kb *types.KnowledgeBase
}

func (s *stubExamResourceKBService) GetKnowledgeBaseByID(_ context.Context, id string) (*types.KnowledgeBase, error) {
	if s.kb == nil || s.kb.ID != id {
		return nil, repository.ErrKnowledgeBaseNotFound
	}
	cp := *s.kb
	return &cp, nil
}

type fakeExamResourceSpaceRepo struct {
	spaces map[string]*types.ExamSpace
}

func (r *fakeExamResourceSpaceRepo) Create(_ context.Context, space *types.ExamSpace) error {
	r.spaces[space.ID] = cloneExamSpace(space)
	return nil
}

func (r *fakeExamResourceSpaceRepo) ListByTenant(_ context.Context, tenantID uint64) ([]*types.ExamSpace, error) {
	out := []*types.ExamSpace{}
	for _, space := range r.spaces {
		if space.TenantID == tenantID && space.Status == types.ExamSpaceStatusActive {
			out = append(out, cloneExamSpace(space))
		}
	}
	return out, nil
}

func (r *fakeExamResourceSpaceRepo) GetByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamSpace, error) {
	space := r.spaces[id]
	if space == nil || space.TenantID != tenantID || space.Status != types.ExamSpaceStatusActive {
		return nil, repository.ErrExamSpaceNotFound
	}
	return cloneExamSpace(space), nil
}

func (r *fakeExamResourceSpaceRepo) GetPersonalByOwner(_ context.Context, tenantID uint64, userID string) (*types.ExamSpace, error) {
	for _, space := range r.spaces {
		if space.TenantID == tenantID && space.OwnerUserID != nil && *space.OwnerUserID == userID && space.SpaceType == types.ExamSpaceTypePersonal {
			return cloneExamSpace(space), nil
		}
	}
	return nil, repository.ErrExamSpaceNotFound
}

func cloneExamSpace(space *types.ExamSpace) *types.ExamSpace {
	if space == nil {
		return nil
	}
	cp := *space
	if space.OwnerUserID != nil {
		ownerID := *space.OwnerUserID
		cp.OwnerUserID = &ownerID
	}
	return &cp
}

func cloneExamSpaceResource(resource *types.ExamSpaceResource) *types.ExamSpaceResource {
	if resource == nil {
		return nil
	}
	cp := *resource
	if resource.SubjectID != nil {
		subjectID := *resource.SubjectID
		cp.SubjectID = &subjectID
	}
	return &cp
}
