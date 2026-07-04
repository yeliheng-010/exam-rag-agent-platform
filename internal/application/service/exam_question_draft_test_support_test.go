package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

type stubExamQuestionDraftRepo struct {
	task          *types.ExamStructuringTask
	material      *types.ExamMaterial
	drafts        []*types.ExamQuestionDraft
	createdDrafts []*types.ExamQuestionDraft
	updatedStatus types.ExamStructuringTaskStatus
}

type stubExamQuestionDraftSpace struct {
	canRead  bool
	canWrite bool
}

type stubExamQuestionDraftChunkReader struct {
	chunks []*types.Chunk
	err    error
}

type stubExamQuestionExtractor struct {
	candidates []*types.ExamQuestionDraftCandidate
	rawOutput  string
	err        error
}

type stubExamQuestionWriter struct {
	created []*types.QuestionDetail
}

func newReadyDraftRepo() *stubExamQuestionDraftRepo {
	return &stubExamQuestionDraftRepo{
		task: &types.ExamStructuringTask{
			ID:             "task-1",
			TenantID:       10000,
			MaterialID:     "material-1",
			SpaceID:        "space-1",
			QuestionBankID: "bank-1",
			Status:         types.ExamStructuringTaskStatusReadyForReview,
		},
		material: &types.ExamMaterial{
			ID:              "material-1",
			TenantID:        10000,
			SpaceID:         "space-1",
			KnowledgeID:     "knowledge-1",
			DomainID:        "domain-1",
			IngestStatus:    types.ExamMaterialIngestStatusCompleted,
			MaterialType:    types.ExamMaterialTypeExamPaper,
			ReviewStatus:    types.ExamReviewStatusPrivate,
			Status:          "active",
			CreatedByUserID: "teacher-1",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}
}

func newPendingDraft(id string) *types.ExamQuestionDraft {
	return &types.ExamQuestionDraft{
		ID:               id,
		TenantID:         10000,
		SpaceID:          "space-1",
		TaskID:           "task-1",
		MaterialID:       "material-1",
		QuestionBankID:   "bank-1",
		DomainID:         "domain-1",
		SourceChunkIDs:   mustJSONForTest([]string{"chunk-1"}),
		QuestionNo:       "21",
		QuestionTypeCode: "single_choice",
		Stem:             "What does the man suggest?",
		OptionsJSON: mustJSONForTest([]types.ExamQuestionDraftOption{
			{Key: "A", Content: "Stay home"},
			{Key: "B", Content: "Go out"},
		}),
		AnswerJSON:     types.JSONMap{"value": "A"},
		Explanation:    "The answer is stated in the dialogue.",
		Difficulty:     "unknown",
		Confidence:     0.9,
		Status:         types.ExamQuestionDraftStatusPendingReview,
		RawModelOutput: `{"questions":[{"question_no":"21"}]}`,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func cloneExamQuestionDraft(draft *types.ExamQuestionDraft) *types.ExamQuestionDraft {
	if draft == nil {
		return nil
	}
	cp := *draft
	return &cp
}

func mustJSONForTest(value any) types.JSON {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return types.JSON(data)
}

func newTestExamQuestionDraftService(
	repo *stubExamQuestionDraftRepo,
	space *stubExamQuestionDraftSpace,
	chunks *stubExamQuestionDraftChunkReader,
	extractor *stubExamQuestionExtractor,
) *examQuestionDraftService {
	return newTestExamQuestionDraftServiceWithQuestionRepo(repo, space, chunks, extractor, &stubExamQuestionWriter{})
}

func newTestExamQuestionDraftServiceWithQuestionRepo(
	repo *stubExamQuestionDraftRepo,
	space *stubExamQuestionDraftSpace,
	chunks *stubExamQuestionDraftChunkReader,
	extractor *stubExamQuestionExtractor,
	questionRepo *stubExamQuestionWriter,
) *examQuestionDraftService {
	if chunks == nil {
		chunks = &stubExamQuestionDraftChunkReader{}
	}
	if extractor == nil {
		extractor = &stubExamQuestionExtractor{}
	}
	return NewExamQuestionDraftService(repo, repo, questionRepo, space, chunks, extractor).(*examQuestionDraftService)
}

func (r *stubExamQuestionDraftRepo) UpsertMaterial(context.Context, *types.ExamMaterial) error {
	return nil
}

func (r *stubExamQuestionDraftRepo) GetMaterialByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamMaterial, error) {
	if r.material == nil || r.material.ID != id || r.material.TenantID != tenantID {
		return nil, repository.ErrExamMaterialNotFound
	}
	cp := *r.material
	return &cp, nil
}

func (r *stubExamQuestionDraftRepo) GetMaterialByKnowledge(context.Context, uint64, string) (*types.ExamMaterial, error) {
	return nil, repository.ErrExamMaterialNotFound
}

func (r *stubExamQuestionDraftRepo) ListMaterials(context.Context, uint64, types.ListExamMaterialsFilter, []string) ([]*types.ExamMaterial, error) {
	return nil, nil
}

func (r *stubExamQuestionDraftRepo) CreateStructuringTask(_ context.Context, task *types.ExamStructuringTask) error {
	r.task = cloneExamStructuringTask(task)
	return nil
}

func (r *stubExamQuestionDraftRepo) GetStructuringTaskByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamStructuringTask, error) {
	if r.task == nil || r.task.ID != id || r.task.TenantID != tenantID {
		return nil, repository.ErrExamStructuringTaskNotFound
	}
	return cloneExamStructuringTask(r.task), nil
}

func (r *stubExamQuestionDraftRepo) UpdateStructuringTask(_ context.Context, task *types.ExamStructuringTask) error {
	r.task = cloneExamStructuringTask(task)
	r.updatedStatus = task.Status
	return nil
}

func (r *stubExamQuestionDraftRepo) ListStructuringTasks(context.Context, uint64, types.ListExamStructuringTasksFilter, []string) ([]*types.ExamStructuringTask, error) {
	return nil, nil
}

func (r *stubExamQuestionDraftRepo) CreateDrafts(_ context.Context, drafts []*types.ExamQuestionDraft) error {
	for _, draft := range drafts {
		cp := cloneExamQuestionDraft(draft)
		r.createdDrafts = append(r.createdDrafts, cp)
		r.drafts = append(r.drafts, cp)
	}
	return nil
}

func (r *stubExamQuestionDraftRepo) DeleteDraftsByTask(_ context.Context, tenantID uint64, taskID string) error {
	kept := r.drafts[:0]
	for _, draft := range r.drafts {
		if draft.TenantID == tenantID && draft.TaskID == taskID {
			continue
		}
		kept = append(kept, draft)
	}
	r.drafts = kept
	return nil
}

func (r *stubExamQuestionDraftRepo) GetDraftByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamQuestionDraft, error) {
	for _, draft := range r.drafts {
		if draft.ID == id && draft.TenantID == tenantID {
			return cloneExamQuestionDraft(draft), nil
		}
	}
	return nil, repository.ErrExamQuestionDraftNotFound
}

func (r *stubExamQuestionDraftRepo) ListDraftsByTask(_ context.Context, tenantID uint64, taskID string) ([]*types.ExamQuestionDraft, error) {
	out := []*types.ExamQuestionDraft{}
	for _, draft := range r.drafts {
		if draft.TenantID == tenantID && draft.TaskID == taskID {
			out = append(out, cloneExamQuestionDraft(draft))
		}
	}
	return out, nil
}

func (r *stubExamQuestionDraftRepo) CountDraftsByTask(_ context.Context, tenantID uint64, taskID string) (types.ExamQuestionDraftStats, error) {
	stats := types.ExamQuestionDraftStats{}
	for _, draft := range r.drafts {
		if draft.TenantID != tenantID || draft.TaskID != taskID {
			continue
		}
		stats.Total++
		switch draft.Status {
		case types.ExamQuestionDraftStatusPendingReview:
			stats.PendingReview++
		case types.ExamQuestionDraftStatusApproved:
			stats.Approved++
		case types.ExamQuestionDraftStatusRejected:
			stats.Rejected++
		}
	}
	return stats, nil
}

func (r *stubExamQuestionDraftRepo) UpdateDraft(_ context.Context, draft *types.ExamQuestionDraft) error {
	for i, existing := range r.drafts {
		if existing.ID == draft.ID {
			r.drafts[i] = cloneExamQuestionDraft(draft)
			return nil
		}
	}
	r.drafts = append(r.drafts, cloneExamQuestionDraft(draft))
	return nil
}

func (s *stubExamQuestionDraftSpace) ListSpaces(context.Context, uint64, string) ([]*types.ExamSpace, error) {
	return []*types.ExamSpace{{ID: "space-1", TenantID: 10000}}, nil
}

func (s *stubExamQuestionDraftSpace) EnsurePersonalSpace(context.Context, uint64, string) (*types.ExamSpace, error) {
	return nil, errors.New("not implemented")
}

func (s *stubExamQuestionDraftSpace) CanReadSpace(context.Context, uint64, string, string) (bool, error) {
	return s.canRead, nil
}

func (s *stubExamQuestionDraftSpace) CanWriteSpace(context.Context, uint64, string, string) (bool, error) {
	return s.canWrite, nil
}

func (s *stubExamQuestionDraftSpace) GetSpace(context.Context, uint64, string, string) (*types.ExamSpace, error) {
	return &types.ExamSpace{ID: "space-1", TenantID: 10000}, nil
}

func (r *stubExamQuestionDraftChunkReader) ListChunksByKnowledgeID(context.Context, string) ([]*types.Chunk, error) {
	return r.chunks, r.err
}

func (e *stubExamQuestionExtractor) Extract(context.Context, *types.ExamMaterial, *types.ExamStructuringTask, []*types.Chunk) ([]*types.ExamQuestionDraftCandidate, string, error) {
	return e.candidates, e.rawOutput, e.err
}
