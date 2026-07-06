package service

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

type stubExamQuestionGroupDraftRepo struct {
	task          *types.ExamStructuringTask
	material      *types.ExamMaterial
	drafts        []*types.ExamQuestionGroupDraft
	createdDrafts []*types.ExamQuestionGroupDraft
	updatedStatus types.ExamStructuringTaskStatus
}

type stubQuestionGroupExtractor struct {
	candidates []*types.ExamQuestionGroupDraftCandidate
	rawOutput  string
	err        error
}

type stubQuestionGroupWriter struct{ created []*types.QuestionGroupDetail }

func newReadyQuestionGroupDraftRepo() *stubExamQuestionGroupDraftRepo {
	return &stubExamQuestionGroupDraftRepo{
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
			DomainID:        "gaokao",
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

func newTestExamQuestionGroupDraftService(
	repo *stubExamQuestionGroupDraftRepo,
	space *stubExamQuestionDraftSpace,
	chunks *stubExamQuestionDraftChunkReader,
	extractor *stubQuestionGroupExtractor,
	questionRepo *stubQuestionGroupWriter,
) *examQuestionGroupDraftService {
	if chunks == nil {
		chunks = &stubExamQuestionDraftChunkReader{}
	}
	if extractor == nil {
		extractor = &stubQuestionGroupExtractor{}
	}
	if questionRepo == nil {
		questionRepo = &stubQuestionGroupWriter{}
	}
	return NewExamQuestionGroupDraftService(repo, repo, questionRepo, space, chunks, extractor).(*examQuestionGroupDraftService)
}

func newReadingGroupCandidate() *types.ExamQuestionGroupDraftCandidate {
	return &types.ExamQuestionGroupDraftCandidate{
		GroupNo:        "Reading A",
		GroupType:      "reading_passage",
		Title:          "Reading A",
		MaterialText:   "Passage text",
		MaterialFormat: "plain_text",
		SourceChunkIDs: []string{"chunk-1"},
		Questions: []types.ExamQuestionGroupDraftQuestionCandidate{
			newReadingGroupQuestion("21", 1, "A"),
			newReadingGroupQuestion("22", 2, "B"),
			newReadingGroupQuestion("23", 3, "C"),
		},
		StrategyCode: "gaokao_english_reading_v1",
		Confidence:   0.9,
	}
}

func newReadingGroupQuestion(no string, order int, answer string) types.ExamQuestionGroupDraftQuestionCandidate {
	return types.ExamQuestionGroupDraftQuestionCandidate{
		QuestionNo:       no,
		QuestionTypeCode: "single_choice",
		Stem:             "Question " + no,
		Options: []types.ExamQuestionDraftOption{
			{Key: "A", Content: "Option A"},
			{Key: "B", Content: "Option B"},
			{Key: "C", Content: "Option C"},
		},
		Answer:         types.JSONMap{"value": answer},
		Explanation:    "Because of the passage.",
		Difficulty:     "unknown",
		Confidence:     0.9,
		OrderInGroup:   order,
		SourceChunkIDs: []string{"chunk-1"},
	}
}

func newPendingQuestionGroupDraft(id string) *types.ExamQuestionGroupDraft {
	candidate := newReadingGroupCandidate()
	return &types.ExamQuestionGroupDraft{
		ID:             id,
		TenantID:       10000,
		SpaceID:        "space-1",
		TaskID:         "task-1",
		MaterialID:     "material-1",
		QuestionBankID: "bank-1",
		DomainID:       "gaokao",
		GroupType:      candidate.GroupType,
		Title:          candidate.Title,
		MaterialText:   candidate.MaterialText,
		MaterialFormat: candidate.MaterialFormat,
		QuestionsJSON:  mustJSONForTest(candidate.Questions),
		AssetsJSON:     mustJSONForTest(candidate.Assets),
		SourceChunkIDs: mustJSONForTest(candidate.SourceChunkIDs),
		StrategyCode:   candidate.StrategyCode,
		Confidence:     candidate.Confidence,
		Status:         types.ExamQuestionGroupDraftStatusPendingReview,
		RawModelOutput: `{"question_groups":[{"group_no":"Reading A"}]}`,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func cloneExamQuestionGroupDraft(draft *types.ExamQuestionGroupDraft) *types.ExamQuestionGroupDraft {
	if draft == nil {
		return nil
	}
	cp := *draft
	return &cp
}

func (r *stubExamQuestionGroupDraftRepo) UpsertMaterial(context.Context, *types.ExamMaterial) error {
	return nil
}

func (r *stubExamQuestionGroupDraftRepo) GetMaterialByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamMaterial, error) {
	if r.material == nil || r.material.ID != id || r.material.TenantID != tenantID {
		return nil, repository.ErrExamMaterialNotFound
	}
	return cloneExamMaterial(r.material), nil
}

func (r *stubExamQuestionGroupDraftRepo) GetMaterialByKnowledge(context.Context, uint64, string) (*types.ExamMaterial, error) {
	return nil, repository.ErrExamMaterialNotFound
}

func (r *stubExamQuestionGroupDraftRepo) ListMaterials(context.Context, uint64, types.ListExamMaterialsFilter, []string) ([]*types.ExamMaterial, error) {
	return nil, nil
}

func (r *stubExamQuestionGroupDraftRepo) CreateStructuringTask(_ context.Context, task *types.ExamStructuringTask) error {
	r.task = cloneExamStructuringTask(task)
	return nil
}

func (r *stubExamQuestionGroupDraftRepo) GetStructuringTaskByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamStructuringTask, error) {
	if r.task == nil || r.task.ID != id || r.task.TenantID != tenantID {
		return nil, repository.ErrExamStructuringTaskNotFound
	}
	return cloneExamStructuringTask(r.task), nil
}

func (r *stubExamQuestionGroupDraftRepo) UpdateStructuringTask(_ context.Context, task *types.ExamStructuringTask) error {
	r.task = cloneExamStructuringTask(task)
	r.updatedStatus = task.Status
	return nil
}

func (r *stubExamQuestionGroupDraftRepo) ListStructuringTasks(context.Context, uint64, types.ListExamStructuringTasksFilter, []string) ([]*types.ExamStructuringTask, error) {
	return nil, nil
}

func (r *stubExamQuestionGroupDraftRepo) CreateDrafts(_ context.Context, drafts []*types.ExamQuestionGroupDraft) error {
	for _, draft := range drafts {
		cp := cloneExamQuestionGroupDraft(draft)
		r.createdDrafts = append(r.createdDrafts, cp)
		r.drafts = append(r.drafts, cp)
	}
	return nil
}

func (r *stubExamQuestionGroupDraftRepo) DeleteDraftsByTask(_ context.Context, tenantID uint64, taskID string) error {
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

func (r *stubExamQuestionGroupDraftRepo) GetDraftByIDAndTenant(_ context.Context, id string, tenantID uint64) (*types.ExamQuestionGroupDraft, error) {
	for _, draft := range r.drafts {
		if draft.ID == id && draft.TenantID == tenantID {
			return cloneExamQuestionGroupDraft(draft), nil
		}
	}
	return nil, repository.ErrQuestionGroupDraftNotFound
}

func (r *stubExamQuestionGroupDraftRepo) ListDraftsByTask(_ context.Context, tenantID uint64, taskID string) ([]*types.ExamQuestionGroupDraft, error) {
	out := []*types.ExamQuestionGroupDraft{}
	for _, draft := range r.drafts {
		if draft.TenantID == tenantID && draft.TaskID == taskID {
			out = append(out, cloneExamQuestionGroupDraft(draft))
		}
	}
	return out, nil
}

func (r *stubExamQuestionGroupDraftRepo) CountDraftsByTask(_ context.Context, tenantID uint64, taskID string) (types.ExamQuestionGroupDraftStats, error) {
	stats := types.ExamQuestionGroupDraftStats{}
	for _, draft := range r.drafts {
		if draft.TenantID != tenantID || draft.TaskID != taskID {
			continue
		}
		stats.Total++
		switch draft.Status {
		case types.ExamQuestionGroupDraftStatusPendingReview:
			stats.PendingReview++
		case types.ExamQuestionGroupDraftStatusApproved:
			stats.Approved++
		case types.ExamQuestionGroupDraftStatusRejected:
			stats.Rejected++
		}
	}
	return stats, nil
}

func (r *stubExamQuestionGroupDraftRepo) UpdateDraft(_ context.Context, draft *types.ExamQuestionGroupDraft) error {
	for i, existing := range r.drafts {
		if existing.ID == draft.ID {
			r.drafts[i] = cloneExamQuestionGroupDraft(draft)
			return nil
		}
	}
	r.drafts = append(r.drafts, cloneExamQuestionGroupDraft(draft))
	return nil
}

func (e *stubQuestionGroupExtractor) Extract(context.Context, *types.ExamMaterial, *types.ExamStructuringTask, []*types.Chunk) ([]*types.ExamQuestionGroupDraftCandidate, string, error) {
	return e.candidates, e.rawOutput, e.err
}

func (w *stubQuestionGroupWriter) CreateQuestionBank(context.Context, *types.QuestionBank) error {
	return nil
}

func (w *stubQuestionGroupWriter) ListQuestionBanks(context.Context, uint64, []string) ([]*types.QuestionBank, error) {
	return nil, nil
}

func (w *stubQuestionGroupWriter) GetQuestionBankByIDAndTenant(context.Context, string, uint64) (*types.QuestionBank, error) {
	return nil, repository.ErrQuestionBankNotFound
}

func (w *stubQuestionGroupWriter) CreateQuestionDetail(context.Context, *types.QuestionDetail) error {
	return nil
}

func (w *stubQuestionGroupWriter) ListQuestionDetailsByBank(context.Context, uint64, string) ([]*types.QuestionDetail, error) {
	return nil, nil
}

func (w *stubQuestionGroupWriter) GetQuestionDetailByIDAndTenant(context.Context, uint64, string) (*types.QuestionDetail, error) {
	return nil, repository.ErrQuestionNotFound
}

func (w *stubQuestionGroupWriter) CreateQuestionGroupDetail(_ context.Context, detail *types.QuestionGroupDetail) error {
	w.created = append(w.created, detail)
	return nil
}

func (w *stubQuestionGroupWriter) ListQuestionGroupDetailsByBank(context.Context, uint64, string) ([]*types.QuestionGroupDetail, error) {
	return w.created, nil
}

func (w *stubQuestionGroupWriter) GetQuestionGroupDetailByIDAndTenant(_ context.Context, _ uint64, groupID string) (*types.QuestionGroupDetail, error) {
	for _, detail := range w.created {
		if detail.Group != nil && detail.Group.ID == groupID {
			return detail, nil
		}
	}
	return nil, repository.ErrQuestionNotFound
}
