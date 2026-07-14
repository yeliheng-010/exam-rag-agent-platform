package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamInterventionService interface {
	RecommendClassPractice(
		ctx context.Context,
		tenantID uint64,
		userID string,
		classID string,
		req types.ExamPracticeRecommendationRequest,
	) (*types.ExamPracticeRecommendationResult, error)
	RecommendStudentPractice(
		ctx context.Context,
		tenantID uint64,
		userID string,
		req types.ExamPracticeRecommendationRequest,
	) (*types.ExamPracticeRecommendationResult, error)
}
