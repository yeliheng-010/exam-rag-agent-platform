package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamAnalyticsService interface {
	ListAnalyzableClasses(ctx context.Context, tenantID uint64, userID string) ([]*types.ExamClass, error)
	GetClassAnalytics(ctx context.Context, tenantID uint64, userID string, classID string) (*types.ExamClassAnalyticsSummary, error)
}
