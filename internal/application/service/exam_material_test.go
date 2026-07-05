package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

type fakeExamMaterialRepo struct {
	materials map[string]*types.ExamMaterial
	tasks     map[string]*types.ExamStructuringTask
}

func newFakeExamMaterialRepo() *fakeExamMaterialRepo {
	return &fakeExamMaterialRepo{
		materials: map[string]*types.ExamMaterial{},
		tasks:     map[string]*types.ExamStructuringTask{},
	}
}

func cloneExamMaterial(m *types.ExamMaterial) *types.ExamMaterial {
	if m == nil {
		return nil
	}
	cp := *m
	return &cp
}

func cloneExamStructuringTask(task *types.ExamStructuringTask) *types.ExamStructuringTask {
	if task == nil {
		return nil
	}
	cp := *task
	return &cp
}

func (r *fakeExamMaterialRepo) UpsertMaterial(_ context.Context, material *types.ExamMaterial) error {
	r.materials[material.ID] = cloneExamMaterial(material)
	return nil
}

func (r *fakeExamMaterialRepo) GetMaterialByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamMaterial, error) {
	material := r.materials[id]
	if material == nil || material.TenantID != tenantID || material.Status != "active" {
		return nil, repository.ErrExamMaterialNotFound
	}
	return cloneExamMaterial(material), nil
}

func (r *fakeExamMaterialRepo) GetMaterialByKnowledge(_ context.Context, tenantID uint64, knowledgeID string) (*types.ExamMaterial, error) {
	for _, material := range r.materials {
		if material.TenantID == tenantID && material.KnowledgeID == knowledgeID && material.Status == "active" {
			return cloneExamMaterial(material), nil
		}
	}
	return nil, repository.ErrExamMaterialNotFound
}

func (r *fakeExamMaterialRepo) ListMaterials(_ context.Context, tenantID uint64, filter types.ListExamMaterialsFilter, spaceIDs []string) ([]*types.ExamMaterial, error) {
	allowedSpaces := map[string]bool{}
	for _, id := range spaceIDs {
		allowedSpaces[id] = true
	}
	out := []*types.ExamMaterial{}
	for _, material := range r.materials {
		if material.TenantID != tenantID || material.Status != "active" || !allowedSpaces[material.SpaceID] {
			continue
		}
		if filter.MaterialType != "" && material.MaterialType != filter.MaterialType {
			continue
		}
		out = append(out, cloneExamMaterial(material))
	}
	return out, nil
}

func (r *fakeExamMaterialRepo) CreateStructuringTask(_ context.Context, task *types.ExamStructuringTask) error {
	r.tasks[task.ID] = cloneExamStructuringTask(task)
	return nil
}

func (r *fakeExamMaterialRepo) GetStructuringTaskByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamStructuringTask, error) {
	task := r.tasks[id]
	if task == nil || task.TenantID != tenantID {
		return nil, repository.ErrExamStructuringTaskNotFound
	}
	return cloneExamStructuringTask(task), nil
}

func (r *fakeExamMaterialRepo) UpdateStructuringTask(_ context.Context, task *types.ExamStructuringTask) error {
	r.tasks[task.ID] = cloneExamStructuringTask(task)
	return nil
}

func (r *fakeExamMaterialRepo) ListStructuringTasks(_ context.Context, tenantID uint64, filter types.ListExamStructuringTasksFilter, spaceIDs []string) ([]*types.ExamStructuringTask, error) {
	allowedSpaces := map[string]bool{}
	for _, id := range spaceIDs {
		allowedSpaces[id] = true
	}
	out := []*types.ExamStructuringTask{}
	for _, task := range r.tasks {
		if task.TenantID != tenantID || !allowedSpaces[task.SpaceID] {
			continue
		}
		if filter.MaterialID != "" && task.MaterialID != filter.MaterialID {
			continue
		}
		out = append(out, cloneExamStructuringTask(task))
	}
	return out, nil
}

type fakeExamMaterialSpaceService struct {
	canRead  map[string]bool
	canWrite map[string]bool
	spaces   []*types.ExamSpace
}

func newFakeExamMaterialSpaceService() *fakeExamMaterialSpaceService {
	return &fakeExamMaterialSpaceService{
		canRead:  map[string]bool{},
		canWrite: map[string]bool{},
		spaces:   []*types.ExamSpace{},
	}
}

func (s *fakeExamMaterialSpaceService) ListSpaces(context.Context, uint64, string) ([]*types.ExamSpace, error) {
	return s.spaces, nil
}

func (s *fakeExamMaterialSpaceService) EnsurePersonalSpace(context.Context, uint64, string) (*types.ExamSpace, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeExamMaterialSpaceService) CanReadSpace(_ context.Context, _ uint64, _ string, spaceID string) (bool, error) {
	return s.canRead[spaceID], nil
}

func (s *fakeExamMaterialSpaceService) CanWriteSpace(_ context.Context, _ uint64, _ string, spaceID string) (bool, error) {
	return s.canWrite[spaceID], nil
}

func (s *fakeExamMaterialSpaceService) GetSpace(_ context.Context, tenantID uint64, _ string, spaceID string) (*types.ExamSpace, error) {
	for _, space := range s.spaces {
		if space.ID == spaceID && space.TenantID == tenantID {
			return space, nil
		}
	}
	return nil, ErrExamNotFound
}

type fakeExamMaterialDomainRepo struct {
	subjectDomain map[string]string
}

func (r *fakeExamMaterialDomainRepo) ListDomains(context.Context) ([]*types.ExamDomain, error) {
	return nil, nil
}

func (r *fakeExamMaterialDomainRepo) GetDomainByID(_ context.Context, id string) (*types.ExamDomain, error) {
	if id == "missing-domain" {
		return nil, repository.ErrExamDomainNotFound
	}
	return &types.ExamDomain{ID: id, Status: types.ExamDomainStatusActive}, nil
}

func (r *fakeExamMaterialDomainRepo) ListSubjects(context.Context, string) ([]*types.ExamSubject, error) {
	return nil, nil
}

func (r *fakeExamMaterialDomainRepo) GetSubjectByID(_ context.Context, id string) (*types.ExamSubject, error) {
	domainID := r.subjectDomain[id]
	if domainID == "" {
		return nil, repository.ErrExamSubjectNotFound
	}
	return &types.ExamSubject{ID: id, DomainID: domainID, Status: types.ExamDomainStatusActive}, nil
}

type fakeExamMaterialKBService struct {
	kbs map[string]*types.KnowledgeBase
}

func (s *fakeExamMaterialKBService) GetKnowledgeBaseByID(_ context.Context, id string) (*types.KnowledgeBase, error) {
	kb := s.kbs[id]
	if kb == nil {
		return nil, repository.ErrKnowledgeBaseNotFound
	}
	cp := *kb
	return &cp, nil
}

type fakeExamMaterialKnowledgeService struct {
	knowledges map[string]*types.Knowledge
}

func (s *fakeExamMaterialKnowledgeService) GetKnowledgeByID(_ context.Context, id string) (*types.Knowledge, error) {
	knowledge := s.knowledges[id]
	if knowledge == nil {
		return nil, repository.ErrKnowledgeNotFound
	}
	cp := *knowledge
	return &cp, nil
}

type fakeExamMaterialChunkService struct {
	countByKnowledge map[string]int
}

func (s *fakeExamMaterialChunkService) ListChunksByKnowledgeID(_ context.Context, knowledgeID string) ([]*types.Chunk, error) {
	count := s.countByKnowledge[knowledgeID]
	chunks := make([]*types.Chunk, 0, count)
	for i := 0; i < count; i++ {
		chunks = append(chunks, &types.Chunk{ID: "chunk"})
	}
	return chunks, nil
}

type fakeExamMaterialQuestionService struct {
	banks map[string]*types.QuestionBank
}

func (s *fakeExamMaterialQuestionService) CreateQuestionBank(_ context.Context, tenantID uint64, userID string, req *types.CreateQuestionBankRequest) (*types.QuestionBank, error) {
	bank := &types.QuestionBank{
		ID:              "bank-created",
		TenantID:        tenantID,
		SpaceID:         req.SpaceID,
		DomainID:        req.DomainID,
		SubjectID:       req.SubjectID,
		Name:            req.Name,
		Description:     req.Description,
		SourceType:      "exam_material",
		ReviewStatus:    types.ExamReviewStatusPrivate,
		Status:          "active",
		CreatedByUserID: userID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	s.banks[bank.ID] = bank
	return bank, nil
}

func (s *fakeExamMaterialQuestionService) ListQuestionBanks(context.Context, uint64, string, string) ([]*types.QuestionBank, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeExamMaterialQuestionService) GetQuestionBank(_ context.Context, tenantID uint64, _ string, bankID string) (*types.QuestionBank, error) {
	bank := s.banks[bankID]
	if bank == nil || bank.TenantID != tenantID {
		return nil, ErrExamNotFound
	}
	cp := *bank
	return &cp, nil
}

func (s *fakeExamMaterialQuestionService) ListQuestionDetails(context.Context, uint64, string, string) ([]*types.QuestionDetail, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeExamMaterialQuestionService) GetQuestionDetail(context.Context, uint64, string, string) (*types.QuestionDetail, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeExamMaterialQuestionService) ListQuestionGroupDetails(context.Context, uint64, string, string) ([]*types.QuestionGroupDetail, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeExamMaterialQuestionService) GetQuestionGroupDetail(context.Context, uint64, string, string) (*types.QuestionGroupDetail, error) {
	return nil, errors.New("not implemented")
}

type fakeExamMaterialResourceService struct {
	bindCalls []types.BindKnowledgeBaseResourceRequest
}

func (s *fakeExamMaterialResourceService) BindKnowledgeBase(_ context.Context, _ uint64, _ string, _ string, req *types.BindKnowledgeBaseResourceRequest) (*types.ExamSpaceResource, error) {
	if req == nil {
		return nil, ErrExamInvalidRequest
	}
	s.bindCalls = append(s.bindCalls, *req)
	return &types.ExamSpaceResource{
		ID:           "resource-1",
		SpaceID:      req.SpaceID,
		DomainID:     req.DomainID,
		SubjectID:    req.SubjectID,
		MaterialType: req.MaterialType,
		Status:       types.ExamSpaceResourceStatusActive,
	}, nil
}

func (s *fakeExamMaterialResourceService) GetKnowledgeBaseBinding(context.Context, uint64, string, string) (*types.ExamSpaceResource, error) {
	return nil, errors.New("not implemented")
}

func (s *fakeExamMaterialResourceService) CanReadKnowledgeBase(context.Context, uint64, string, string) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *fakeExamMaterialResourceService) ListResources(context.Context, uint64, string, types.ListExamResourcesFilter) ([]*types.ExamSpaceResource, error) {
	return nil, errors.New("not implemented")
}

type examMaterialServiceTestHarness struct {
	svc      *examMaterialService
	resource *fakeExamMaterialResourceService
}

func newExamMaterialServiceForTest(space *fakeExamMaterialSpaceService, repo *fakeExamMaterialRepo, chunks int) *examMaterialService {
	return newExamMaterialServiceHarness(space, repo, chunks).svc
}

func newExamMaterialServiceHarness(space *fakeExamMaterialSpaceService, repo *fakeExamMaterialRepo, chunks int) *examMaterialServiceTestHarness {
	resource := &fakeExamMaterialResourceService{}
	svc := NewExamMaterialService(
		repo,
		space,
		&fakeExamMaterialDomainRepo{subjectDomain: map[string]string{"subject-1": "domain-1"}},
		&fakeExamMaterialKBService{kbs: map[string]*types.KnowledgeBase{
			"kb-1": {ID: "kb-1", TenantID: 1, CreatorID: "teacher-1", Name: "IELTS KB"},
		}},
		&fakeExamMaterialKnowledgeService{knowledges: map[string]*types.Knowledge{
			"knowledge-1": {
				ID:              "knowledge-1",
				TenantID:        1,
				KnowledgeBaseID: "kb-1",
				Title:           "2025 IELTS Reading",
				ParseStatus:     types.ParseStatusCompleted,
			},
		}},
		&fakeExamMaterialChunkService{countByKnowledge: map[string]int{"knowledge-1": chunks}},
		&fakeExamMaterialQuestionService{banks: map[string]*types.QuestionBank{
			"bank-1": {
				ID:              "bank-1",
				TenantID:        1,
				SpaceID:         "space-1",
				DomainID:        "domain-1",
				Name:            "IELTS Bank",
				Status:          "active",
				CreatedByUserID: "teacher-1",
			},
		}},
		resource,
	).(*examMaterialService)
	return &examMaterialServiceTestHarness{svc: svc, resource: resource}
}

func TestExamMaterialRegisterRequiresWritableSpace(t *testing.T) {
	space := newFakeExamMaterialSpaceService()
	space.spaces = []*types.ExamSpace{{ID: "space-1", TenantID: 1, SpaceType: types.ExamSpaceTypeClass}}
	space.canRead["space-1"] = true
	repo := newFakeExamMaterialRepo()
	svc := newExamMaterialServiceForTest(space, repo, 3)

	_, err := svc.RegisterMaterial(context.Background(), 1, "student-1", &types.RegisterExamMaterialRequest{
		SpaceID:         "space-1",
		KnowledgeBaseID: "kb-1",
		KnowledgeID:     "knowledge-1",
		DomainID:        "domain-1",
		MaterialType:    types.ExamMaterialTypeExamPaper,
	})
	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("RegisterMaterial error = %v, want ErrExamPermissionDenied", err)
	}
	if len(repo.materials) != 0 {
		t.Fatalf("denied register created %d materials", len(repo.materials))
	}
}

func TestExamMaterialRegisterCreatesReadyStructuringTaskForParsedPaper(t *testing.T) {
	space := newFakeExamMaterialSpaceService()
	space.spaces = []*types.ExamSpace{{ID: "space-1", TenantID: 1, SpaceType: types.ExamSpaceTypeClass}}
	space.canRead["space-1"] = true
	space.canWrite["space-1"] = true
	repo := newFakeExamMaterialRepo()
	harness := newExamMaterialServiceHarness(space, repo, 4)
	svc := harness.svc
	subjectID := "subject-1"

	result, err := svc.RegisterMaterial(context.Background(), 1, "teacher-1", &types.RegisterExamMaterialRequest{
		SpaceID:         "space-1",
		KnowledgeBaseID: "kb-1",
		KnowledgeID:     "knowledge-1",
		DomainID:        "domain-1",
		SubjectID:       &subjectID,
		MaterialType:    types.ExamMaterialTypeExamPaper,
		QuestionBankID:  "bank-1",
		CreateTask:      true,
		SourceYear:      intPtr(2025),
		SourceRegion:    "Cambridge",
		PaperType:       "Reading",
	})
	if err != nil {
		t.Fatalf("RegisterMaterial returned error: %v", err)
	}
	if result.Material == nil || result.Material.Title != "2025 IELTS Reading" {
		t.Fatalf("material title = %#v, want knowledge title fallback", result.Material)
	}
	if result.Material.IngestStatus != types.ExamMaterialIngestStatusCompleted {
		t.Fatalf("ingest status = %s, want completed", result.Material.IngestStatus)
	}
	if result.StructuringTask == nil {
		t.Fatalf("expected structuring task")
	}
	if result.StructuringTask.Status != types.ExamStructuringTaskStatusReadyForReview {
		t.Fatalf("task status = %s, want ready_for_review", result.StructuringTask.Status)
	}
	if result.StructuringTask.SourceChunkCount != 4 {
		t.Fatalf("task chunk count = %d, want 4", result.StructuringTask.SourceChunkCount)
	}
	if len(harness.resource.bindCalls) != 1 {
		t.Fatalf("resource bind calls = %d, want 1", len(harness.resource.bindCalls))
	}
	if harness.resource.bindCalls[0].SpaceID != "space-1" || harness.resource.bindCalls[0].MaterialType != types.ExamMaterialTypeExamPaper {
		t.Fatalf("resource bind call = %#v, want class paper binding", harness.resource.bindCalls[0])
	}
}

func TestExamMaterialCreateStructuringTaskBlocksWhenNoChunks(t *testing.T) {
	space := newFakeExamMaterialSpaceService()
	space.spaces = []*types.ExamSpace{{ID: "space-1", TenantID: 1, SpaceType: types.ExamSpaceTypeClass}}
	space.canRead["space-1"] = true
	space.canWrite["space-1"] = true
	repo := newFakeExamMaterialRepo()
	svc := newExamMaterialServiceForTest(space, repo, 0)
	result, err := svc.RegisterMaterial(context.Background(), 1, "teacher-1", &types.RegisterExamMaterialRequest{
		SpaceID:         "space-1",
		KnowledgeBaseID: "kb-1",
		KnowledgeID:     "knowledge-1",
		DomainID:        "domain-1",
		MaterialType:    types.ExamMaterialTypeExamPaper,
	})
	if err != nil {
		t.Fatalf("RegisterMaterial returned error: %v", err)
	}

	task, err := svc.CreateStructuringTask(context.Background(), 1, "teacher-1", result.Material.ID, &types.CreateExamStructuringTaskRequest{
		QuestionBankID: "bank-1",
	})
	if err != nil {
		t.Fatalf("CreateStructuringTask returned error: %v", err)
	}
	if task.Status != types.ExamStructuringTaskStatusBlocked {
		t.Fatalf("task status = %s, want blocked", task.Status)
	}
	if task.ErrorMessage == "" {
		t.Fatalf("blocked task should include error message")
	}
}

func intPtr(v int) *int {
	return &v
}
