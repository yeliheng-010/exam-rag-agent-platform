package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const examResourceTestDDL = `
CREATE TABLE exam_space_resources (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    space_id VARCHAR(36) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(36) NOT NULL,
    domain_id VARCHAR(36) NOT NULL,
    subject_id VARCHAR(36),
    material_type VARCHAR(64) NOT NULL DEFAULT 'learning_material',
    review_status VARCHAR(32) NOT NULL DEFAULT 'private',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by_user_id VARCHAR(36) NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE UNIQUE INDEX idx_exam_space_resources_unique_space_resource
    ON exam_space_resources(tenant_id, space_id, resource_type, resource_id);
`

func TestExamResourceRepositoryUpsertScopesSameResourcePerSpace(t *testing.T) {
	db := newExamResourceTestDB(t)
	repo := NewExamResourceRepository(db)
	ctx := context.Background()

	first := newExamResourceForTest("resource-binding-1", "space-a", "teacher-a")
	second := newExamResourceForTest("resource-binding-2", "space-b", "teacher-b")

	require.NoError(t, repo.Upsert(ctx, first))
	require.NoError(t, repo.Upsert(ctx, second))

	resources, err := repo.List(ctx, 10000, types.ListExamResourcesFilter{
		ResourceType: types.ExamResourceTypeKnowledgeBase,
	}, []string{"space-a", "space-b"})
	require.NoError(t, err)
	require.Len(t, resources, 2)
	require.ElementsMatch(t, []string{"space-a", "space-b"}, []string{resources[0].SpaceID, resources[1].SpaceID})
}

func newExamResourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(examResourceTestDDL).Error)
	return db
}

func newExamResourceForTest(id string, spaceID string, userID string) *types.ExamSpaceResource {
	now := time.Now()
	return &types.ExamSpaceResource{
		ID:              id,
		TenantID:        10000,
		SpaceID:         spaceID,
		ResourceType:    types.ExamResourceTypeKnowledgeBase,
		ResourceID:      "kb-1",
		DomainID:        "gaokao",
		MaterialType:    types.ExamMaterialTypeExamPaper,
		ReviewStatus:    types.ExamReviewStatusPrivate,
		Status:          types.ExamSpaceResourceStatusActive,
		CreatedByUserID: userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}
