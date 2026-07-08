package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamAnalyticsService interface {
	GetClassAnalytics(ctx context.Context, tenantID uint64, userID string, classID string) (*types.ExamClassAnalyticsSummary, error)
}
