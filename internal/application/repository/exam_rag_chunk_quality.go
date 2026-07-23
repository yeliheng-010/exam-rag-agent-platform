package repository

import (
	"context"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type examRAGChunkQualityRepository struct {
	db *gorm.DB
}

type chunkQualityKBRow struct {
	ID             string     `gorm:"column:id"`
	ChunkingConfig types.JSON `gorm:"column:chunking_config"`
}

type chunkQualityKnowledgeRow struct {
	ID              string `gorm:"column:id"`
	KnowledgeBaseID string `gorm:"column:knowledge_base_id"`
}

type chunkQualityChunkRow struct {
	KnowledgeBaseID string `gorm:"column:knowledge_base_id"`
	Content         string `gorm:"column:content"`
	ChunkType       string `gorm:"column:chunk_type"`
	ParentChunkID   string `gorm:"column:parent_chunk_id"`
}

type chunkQualitySpanRow struct {
	KnowledgeID string        `gorm:"column:knowledge_id"`
	Attempt     int           `gorm:"column:attempt"`
	Output      types.JSONMap `gorm:"column:output"`
}

type chunkQualitySpanDiagnostics struct {
	ParentTier      string         `json:"parent_tier"`
	ChildTierCounts map[string]int `json:"child_tier_counts"`
}

type chunkQualityAccumulator struct {
	snapshot     types.ExamRAGChunkingSnapshot
	lengths      []int
	parentLinked int
}

func NewExamRAGChunkQualityRepository(db *gorm.DB) interfaces.ExamRAGChunkQualityRepository {
	return &examRAGChunkQualityRepository{db: db}
}

func (r *examRAGChunkQualityRepository) Snapshot(
	ctx context.Context,
	tenantID uint64,
	knowledgeBaseIDs []string,
) ([]types.ExamRAGChunkingSnapshot, error) {
	requested := uniqueChunkQualityIDs(knowledgeBaseIDs)
	if tenantID == 0 || len(requested) == 0 {
		return []types.ExamRAGChunkingSnapshot{}, nil
	}
	kbs, err := r.loadChunkQualityKBs(ctx, tenantID, requested)
	if err != nil || len(kbs) == 0 {
		return nilIfError(err)
	}
	knowledge, err := r.loadChunkQualityKnowledge(ctx, tenantID, kbIDs(kbs))
	if err != nil {
		return nil, err
	}
	accumulators := newChunkQualityAccumulators(kbs, knowledge)
	if err := r.addChunkQualityStats(ctx, tenantID, kbIDs(kbs), accumulators); err != nil {
		return nil, err
	}
	if err := r.addChunkQualityTiers(ctx, knowledge, accumulators); err != nil {
		return nil, err
	}
	return finalizeChunkQualitySnapshots(requested, accumulators), nil
}

func (r *examRAGChunkQualityRepository) loadChunkQualityKBs(
	ctx context.Context,
	tenantID uint64,
	ids []string,
) ([]chunkQualityKBRow, error) {
	var rows []chunkQualityKBRow
	err := r.db.WithContext(ctx).Table("knowledge_bases").
		Select("id, chunking_config").
		Where("tenant_id = ? AND id IN ? AND deleted_at IS NULL", tenantID, ids).
		Find(&rows).Error
	return rows, err
}

func (r *examRAGChunkQualityRepository) loadChunkQualityKnowledge(
	ctx context.Context,
	tenantID uint64,
	kbIDs []string,
) ([]chunkQualityKnowledgeRow, error) {
	var rows []chunkQualityKnowledgeRow
	err := r.db.WithContext(ctx).Table("knowledges").
		Select("id, knowledge_base_id").
		Where("tenant_id = ? AND knowledge_base_id IN ? AND deleted_at IS NULL", tenantID, kbIDs).
		Find(&rows).Error
	return rows, err
}

func (r *examRAGChunkQualityRepository) addChunkQualityStats(
	ctx context.Context,
	tenantID uint64,
	kbIDs []string,
	accumulators map[string]*chunkQualityAccumulator,
) error {
	var rows []chunkQualityChunkRow
	err := r.db.WithContext(ctx).Table("chunks").
		Select("knowledge_base_id, content, chunk_type, parent_chunk_id").
		Where("tenant_id = ? AND knowledge_base_id IN ? AND chunk_type IN ? AND deleted_at IS NULL",
			tenantID, kbIDs, []string{types.ChunkTypeText, types.ChunkTypeParentText}).
		Find(&rows).Error
	if err != nil {
		return err
	}
	for _, row := range rows {
		acc := accumulators[row.KnowledgeBaseID]
		if acc == nil {
			continue
		}
		if row.ChunkType == types.ChunkTypeParentText {
			acc.snapshot.ParentChunkCount++
			continue
		}
		acc.lengths = append(acc.lengths, len([]rune(row.Content)))
		if strings.TrimSpace(row.ParentChunkID) != "" {
			acc.parentLinked++
		}
	}
	return nil
}

func (r *examRAGChunkQualityRepository) addChunkQualityTiers(
	ctx context.Context,
	knowledge []chunkQualityKnowledgeRow,
	accumulators map[string]*chunkQualityAccumulator,
) error {
	latest, err := r.loadLatestChunkQualitySpans(ctx, knowledgeIDs(knowledge))
	if err != nil {
		return err
	}
	for _, item := range knowledge {
		acc := accumulators[item.KnowledgeBaseID]
		diagnostics, ok := decodeChunkQualityDiagnostics(latest[item.ID])
		if acc == nil || !ok {
			if acc != nil {
				acc.snapshot.UnknownTierCount++
			}
			continue
		}
		acc.snapshot.ActualTierCounts[diagnostics.ParentTier]++
		for tier, count := range diagnostics.ChildTierCounts {
			acc.snapshot.ActualTierCounts[tier] += count
		}
	}
	return nil
}

func (r *examRAGChunkQualityRepository) loadLatestChunkQualitySpans(
	ctx context.Context,
	knowledgeIDs []string,
) (map[string]types.JSONMap, error) {
	latest := make(map[string]types.JSONMap)
	if len(knowledgeIDs) == 0 {
		return latest, nil
	}
	var rows []chunkQualitySpanRow
	err := r.db.WithContext(ctx).Table("knowledge_processing_spans").
		Select("knowledge_id, attempt, output").
		Where("knowledge_id IN ? AND name = ?", knowledgeIDs, types.StageChunking).
		Order("knowledge_id ASC, attempt DESC, updated_at DESC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, exists := latest[row.KnowledgeID]; !exists {
			latest[row.KnowledgeID] = row.Output
		}
	}
	return latest, nil
}
