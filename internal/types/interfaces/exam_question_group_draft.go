package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamQuestionGroupDraftService interface {
	ExtractDrafts(ctx context.Context, tenantID uint64, userID string, taskID string, req *types.ExtractExamQuestionGroupDraftsRequest) (*types.ExamQuestionGroupDraftExtractionResult, error)
	ListDrafts(ctx context.Context, tenantID uint64, userID string, taskID string) (*types.ListExamQuestionGroupDraftsResult, error)
	UpdateDraft(ctx context.Context, tenantID uint64, userID string, draftID string, req *types.UpdateExamQuestionGroupDraftRequest) (*types.ExamQuestionGroupDraft, error)
	ApproveDraft(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ApproveExamQuestionGroupDraftResult, error)
	RejectDraft(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ExamQuestionGroupDraft, error)
}

type ExamQuestionGroupDraftRepository interface {
	CreateDrafts(ctx context.Context, drafts []*types.ExamQuestionGroupDraft) error
	DeleteDraftsByTask(ctx context.Context, tenantID uint64, taskID string) error
	GetDraftByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamQuestionGroupDraft, error)
	ListDraftsByTask(ctx context.Context, tenantID uint64, taskID string) ([]*types.ExamQuestionGroupDraft, error)
	CountDraftsByTask(ctx context.Context, tenantID uint64, taskID string) (types.ExamQuestionGroupDraftStats, error)
	UpdateDraft(ctx context.Context, draft *types.ExamQuestionGroupDraft) error
}

type ExamQuestionGroupExtractor interface {
	Extract(ctx context.Context, material *types.ExamMaterial, task *types.ExamStructuringTask, chunks []*types.Chunk) ([]*types.ExamQuestionGroupDraftCandidate, string, error)
}
