package repository

import (
	"context"
	"math"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExamRAGChunkQualityRepositoryBuildsTenantScopedSnapshot(t *testing.T) {
	t.Parallel()

	db := newExamRAGChunkQualityTestDB(t)
	seedExamRAGChunkQualityFixture(t, db)
	repo := NewExamRAGChunkQualityRepository(db)

	snapshots, err := repo.Snapshot(context.Background(), 10000, []string{"kb-1", "kb-other"})

	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	snapshot := snapshots[0]
	require.Equal(t, "kb-1", snapshot.KnowledgeBaseID)
	require.Equal(t, "auto", snapshot.Config.Strategy)
	require.Equal(t, 100, snapshot.Config.ChildChunkSize)
	require.Equal(t, 2, snapshot.KnowledgeCount)
	require.Equal(t, 3, snapshot.TextChunkCount)
	require.Equal(t, 1, snapshot.ParentChunkCount)
	require.Equal(t, 10, snapshot.MinChars)
	require.Equal(t, 50, snapshot.P50Chars)
	require.Equal(t, 120, snapshot.P90Chars)
	require.Equal(t, 120, snapshot.MaxChars)
	require.True(t, math.Abs(snapshot.TinyChunkRate-1.0/3.0) < 0.0001)
	require.True(t, math.Abs(snapshot.OversizeRate-1.0/3.0) < 0.0001)
	require.True(t, math.Abs(snapshot.ParentCoverage-2.0/3.0) < 0.0001)
	require.Equal(t, map[string]int{"heading": 1, "heuristic": 2}, snapshot.ActualTierCounts)
	require.Equal(t, 1, snapshot.UnknownTierCount)
}

func TestExamRAGChunkQualityRepositoryReturnsEmptyForUnownedKnowledgeBase(t *testing.T) {
	t.Parallel()

	db := newExamRAGChunkQualityTestDB(t)
	seedExamRAGChunkQualityFixture(t, db)
	repo := NewExamRAGChunkQualityRepository(db)

	snapshots, err := repo.Snapshot(context.Background(), 10000, []string{"kb-other"})

	require.NoError(t, err)
	require.Empty(t, snapshots)
}

func newExamRAGChunkQualityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, ddl := range []string{
		`CREATE TABLE knowledge_bases (id TEXT PRIMARY KEY, tenant_id INTEGER NOT NULL, chunking_config TEXT, deleted_at DATETIME)`,
		`CREATE TABLE knowledges (id TEXT PRIMARY KEY, tenant_id INTEGER NOT NULL, knowledge_base_id TEXT NOT NULL, deleted_at DATETIME)`,
		`CREATE TABLE chunks (id TEXT PRIMARY KEY, tenant_id INTEGER NOT NULL, knowledge_id TEXT NOT NULL, knowledge_base_id TEXT NOT NULL, content TEXT, chunk_type TEXT, parent_chunk_id TEXT, deleted_at DATETIME)`,
		`CREATE TABLE knowledge_processing_spans (id INTEGER PRIMARY KEY AUTOINCREMENT, knowledge_id TEXT NOT NULL, attempt INTEGER NOT NULL, span_id TEXT NOT NULL, name TEXT NOT NULL, kind TEXT, status TEXT, input TEXT, output TEXT, updated_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}
	return db
}

func seedExamRAGChunkQualityFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	config := `{"chunk_size":512,"chunk_overlap":80,"enable_parent_child":true,"parent_chunk_size":4096,"child_chunk_size":100,"strategy":"auto","languages":["en"]}`
	require.NoError(t, db.Exec(`INSERT INTO knowledge_bases(id, tenant_id, chunking_config) VALUES (?, ?, ?), (?, ?, ?)`,
		"kb-1", 10000, config, "kb-other", 20000, config).Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledges(id, tenant_id, knowledge_base_id) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)`,
		"doc-1", 10000, "kb-1", "doc-2", 10000, "kb-1", "doc-other", 20000, "kb-other").Error)
	require.NoError(t, db.Exec(`INSERT INTO chunks(id, tenant_id, knowledge_id, knowledge_base_id, content, chunk_type, parent_chunk_id) VALUES
		(?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?)`,
		"text-1", 10000, "doc-1", "kb-1", "1234567890", types.ChunkTypeText, "parent-1",
		"text-2", 10000, "doc-1", "kb-1", string(make([]byte, 50)), types.ChunkTypeText, "parent-1",
		"text-3", 10000, "doc-1", "kb-1", string(make([]byte, 120)), types.ChunkTypeText, "",
		"parent-1", 10000, "doc-1", "kb-1", string(make([]byte, 300)), types.ChunkTypeParentText, "",
		"other-1", 20000, "doc-other", "kb-other", string(make([]byte, 999)), types.ChunkTypeText, "").Error)
	require.NoError(t, db.Exec(`INSERT INTO knowledge_processing_spans(knowledge_id, attempt, span_id, name, kind, status, output, updated_at) VALUES
		(?, 1, 'old', ?, 'stage', 'done', ?, '2026-07-20T00:00:00Z'),
		(?, 2, 'latest', ?, 'stage', 'done', ?, '2026-07-21T00:00:00Z')`,
		"doc-1", types.StageChunking, `{"parent_tier":"legacy","child_tier_counts":{"legacy":1}}`,
		"doc-1", types.StageChunking, `{"parent_tier":"heading","child_tier_counts":{"heuristic":2}}`).Error)
}
