package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamQuestionDraftService interface {
	ExtractDrafts(ctx context.Context, tenantID uint64, userID string, taskID string, req *types.ExtractExamQuestionDraftsRequest) (*types.ExamQuestionDraftExtractionResult, error)
	ListDrafts(ctx context.Context, tenantID uint64, userID string, taskID string) (*types.ListExamQuestionDraftsResult, error)
	UpdateDraft(ctx context.Context, tenantID uint64, userID string, draftID string, req *types.UpdateExamQuestionDraftRequest) (*types.ExamQuestionDraft, error)
	ApproveDraft(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ApproveExamQuestionDraftResult, error)
	RejectDraft(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ExamQuestionDraft, error)
}

type ExamQuestionDraftRepository interface {
	CreateDrafts(ctx context.Context, drafts []*types.ExamQuestionDraft) error
	DeleteDraftsByTask(ctx context.Context, tenantID uint64, taskID string) error
	GetDraftByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamQuestionDraft, error)
	ListDraftsByTask(ctx context.Context, tenantID uint64, taskID string) ([]*types.ExamQuestionDraft, error)
	CountDraftsByTask(ctx context.Context, tenantID uint64, taskID string) (types.ExamQuestionDraftStats, error)
	UpdateDraft(ctx context.Context, draft *types.ExamQuestionDraft) error
}

type ExamQuestionExtractor interface {
	Extract(ctx context.Context, material *types.ExamMaterial, task *types.ExamStructuringTask, chunks []*types.Chunk) ([]*types.ExamQuestionDraftCandidate, string, error)
}
