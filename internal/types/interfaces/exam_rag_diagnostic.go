package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamRAGDiagnosticService interface {
	EvaluateQuestionBank(
		ctx context.Context,
		tenantID uint64,
		userID string,
		bankID string,
		req *types.RunExamRAGDiagnosticRequest,
	) (*types.ExamRAGDiagnosticResult, error)
}
