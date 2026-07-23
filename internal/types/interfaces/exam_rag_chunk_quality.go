package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamRAGChunkQualityRepository interface {
	Snapshot(
		ctx context.Context,
		tenantID uint64,
		knowledgeBaseIDs []string,
	) ([]types.ExamRAGChunkingSnapshot, error)
}
