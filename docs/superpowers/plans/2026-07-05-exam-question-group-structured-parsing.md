# 考试题组结构化实现计划

> **审计状态（2026-07-23）：** 实现与验证已完成；下方未勾选框是未回填的历史执行记录，不代表当前功能缺失。完成证据见 [Phase 1 收尾审计报告](../reports/2026-07-23-phase1-closeout-audit.md)。

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将考试抽题链路从扁平题目升级为 `QuestionGroup + Question`，让高考英语阅读和高考数学图文题可以以真实题组形态进入题库、校对页和学生可用的对话上下文。

**架构：** 在保留现有 `exam_question_drafts` 链路的基础上新增题组草稿、题组正式表和题组读取接口。抽取层改为策略注册表，第一阶段实现高考英语阅读和高考数学基础策略；前端题库详情页优先消费题组接口，并把历史孤立题包装为 `single_question` 题组。

**技术栈：** Go、Gin、GORM、PostgreSQL migration、WeKnora chat model abstraction、Vue 3、TDesign、TypeScript。

---

## 文件结构

- 创建 `migrations/versioned/000074_exam_question_groups.up.sql`：新增 `question_groups`、`question_group_assets`、`exam_question_group_drafts`，扩展 `questions`。
- 创建 `migrations/versioned/000074_exam_question_groups.down.sql`：回滚题组表和 `questions` 扩展字段。
- 修改 `internal/types/exam_question.go`：增加 `QuestionGroup`、`QuestionGroupAsset`、`QuestionGroupDetail`、题组状态与 `Question` 扩展字段。
- 创建 `internal/types/exam_question_group_draft.go`：题组草稿、题组候选 DTO、题组校对请求与统计结构。
- 修改 `internal/types/interfaces/exam_question.go`：增加题组正式仓储和服务方法。
- 创建 `internal/types/interfaces/exam_question_group_draft.go`：题组草稿服务、仓储、抽取器和策略接口。
- 修改 `internal/application/repository/exam_question.go`：写入正式题组、读取题组详情、包装历史孤立题。
- 创建 `internal/application/repository/exam_question_group_draft.go`：题组草稿 GORM 仓储。
- 创建 `internal/application/service/exam_question_group_strategy.go`：策略注册表、通用 schema 校验和策略选择。
- 创建 `internal/application/service/exam_question_group_extractor.go`：题组 LLM 抽取器、JSON 解析和候选规范化。
- 创建 `internal/application/service/exam_question_group_draft.go`：题组抽取、列表、更新、确认、驳回业务逻辑。
- 创建 `internal/application/service/exam_question_group_build.go`：题组草稿到正式题组的构建函数。
- 创建 `internal/application/service/exam_question_group_extractor_test.go`：覆盖策略选择、题组 JSON 解析和分科 prompt。
- 创建 `internal/application/service/exam_question_group_draft_test.go`：覆盖权限、题组草稿抽取和确认入库。
- 创建 `internal/handler/exam_question_group_draft.go`：题组草稿 HTTP handler。
- 修改 `internal/handler/exam_question.go`：增加题组详情列表 handler。
- 修改 `internal/router/exam.go`：注册题组抽取、题组草稿、题组列表路由。
- 修改 `internal/container/container.go`：注册题组仓储、策略、抽取器、服务和 handler。
- 修改 `frontend/src/types/exam.ts`：增加题组、题组草稿、资产、小题扩展类型。
- 创建 `frontend/src/api/exam/question-group.ts`：正式题组 API。
- 创建 `frontend/src/api/exam/question-group-draft.ts`：题组草稿 API。
- 修改 `frontend/src/views/classes/ClassDetail.vue`：结构化任务入口优先进入题组校对。
- 创建 `frontend/src/views/question-draft/QuestionGroupDraftReview.vue`：题组校对工作台。
- 修改 `frontend/src/views/question-bank/QuestionBankDetail.vue`：从表格改成题组卡片展示。
- 修改 `frontend/src/router/index.ts`：注册题组校对页面路由。

## 任务 1：数据库迁移与 Go 类型

**文件：**
- 创建：`migrations/versioned/000074_exam_question_groups.up.sql`
- 创建：`migrations/versioned/000074_exam_question_groups.down.sql`
- 修改：`internal/types/exam_question.go`
- 创建：`internal/types/exam_question_group_draft.go`

- [ ] **步骤 1：编写迁移文件**

创建 `migrations/versioned/000074_exam_question_groups.up.sql`：

```sql
CREATE TABLE IF NOT EXISTS question_groups (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    space_id VARCHAR(36) NOT NULL,
    question_bank_id VARCHAR(36) NOT NULL,
    domain_id VARCHAR(36) NOT NULL,
    subject_id VARCHAR(36),
    group_type VARCHAR(64) NOT NULL DEFAULT 'single_question',
    title VARCHAR(255) NOT NULL DEFAULT '',
    material_text TEXT NOT NULL DEFAULT '',
    material_format VARCHAR(32) NOT NULL DEFAULT 'plain_text',
    asset_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_chunk_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_year INT,
    source_region VARCHAR(128) NOT NULL DEFAULT '',
    paper_type VARCHAR(128) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    review_status VARCHAR(32) NOT NULL DEFAULT 'private',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by_user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_question_groups_tenant_bank
    ON question_groups (tenant_id, question_bank_id, status);
CREATE INDEX IF NOT EXISTS idx_question_groups_space
    ON question_groups (tenant_id, space_id);

CREATE TABLE IF NOT EXISTS question_group_assets (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    group_id VARCHAR(36) NOT NULL,
    asset_type VARCHAR(64) NOT NULL,
    storage_uri TEXT NOT NULL DEFAULT '',
    alt_text TEXT NOT NULL DEFAULT '',
    source_chunk_id VARCHAR(36) NOT NULL DEFAULT '',
    bbox JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_question_group_assets_group
    ON question_group_assets (tenant_id, group_id, sort_order);

CREATE TABLE IF NOT EXISTS exam_question_group_drafts (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    space_id VARCHAR(36) NOT NULL,
    task_id VARCHAR(36) NOT NULL,
    material_id VARCHAR(36) NOT NULL,
    question_bank_id VARCHAR(36) NOT NULL,
    domain_id VARCHAR(36) NOT NULL,
    subject_id VARCHAR(36),
    group_type VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    material_text TEXT NOT NULL DEFAULT '',
    material_format VARCHAR(32) NOT NULL DEFAULT 'plain_text',
    questions_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    assets_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_chunk_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    strategy_code VARCHAR(64) NOT NULL DEFAULT '',
    confidence NUMERIC(5,4) NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_review',
    raw_model_output TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    approved_group_id VARCHAR(36) NOT NULL DEFAULT '',
    reviewed_by_user_id VARCHAR(36) NOT NULL DEFAULT '',
    reviewed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_exam_question_group_drafts_task
    ON exam_question_group_drafts (tenant_id, task_id, status);

ALTER TABLE questions
    ADD COLUMN IF NOT EXISTS group_id VARCHAR(36),
    ADD COLUMN IF NOT EXISTS question_no VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS order_in_group INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS question_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_questions_group_order
    ON questions (tenant_id, group_id, order_in_group);
```

创建 `migrations/versioned/000074_exam_question_groups.down.sql`：

```sql
DROP INDEX IF EXISTS idx_questions_group_order;
ALTER TABLE questions
    DROP COLUMN IF EXISTS question_metadata,
    DROP COLUMN IF EXISTS order_in_group,
    DROP COLUMN IF EXISTS question_no,
    DROP COLUMN IF EXISTS group_id;

DROP INDEX IF EXISTS idx_exam_question_group_drafts_task;
DROP TABLE IF EXISTS exam_question_group_drafts;

DROP INDEX IF EXISTS idx_question_group_assets_group;
DROP TABLE IF EXISTS question_group_assets;

DROP INDEX IF EXISTS idx_question_groups_space;
DROP INDEX IF EXISTS idx_question_groups_tenant_bank;
DROP TABLE IF EXISTS question_groups;
```

- [ ] **步骤 2：扩展正式题型类型**

在 `internal/types/exam_question.go` 增加：

```go
type QuestionGroup struct {
	ID              string           `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64           `json:"tenant_id" gorm:"not null;index"`
	SpaceID         string           `json:"space_id" gorm:"type:varchar(36);not null;index"`
	QuestionBankID  string           `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	DomainID        string           `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID       *string          `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	GroupType       string           `json:"group_type" gorm:"type:varchar(64);not null;default:'single_question'"`
	Title           string           `json:"title" gorm:"type:varchar(255);not null;default:''"`
	MaterialText    string           `json:"material_text" gorm:"type:text;not null;default:''"`
	MaterialFormat  string           `json:"material_format" gorm:"type:varchar(32);not null;default:'plain_text'"`
	AssetRefs       JSON             `json:"asset_refs" gorm:"type:jsonb;not null"`
	SourceChunkIDs  JSON             `json:"source_chunk_ids" gorm:"type:jsonb;not null"`
	SourceYear      *int             `json:"source_year,omitempty"`
	SourceRegion    string           `json:"source_region" gorm:"type:varchar(128);not null;default:''"`
	PaperType       string           `json:"paper_type" gorm:"type:varchar(128);not null;default:''"`
	SortOrder       int              `json:"sort_order" gorm:"not null;default:0"`
	ReviewStatus    ExamReviewStatus `json:"review_status" gorm:"type:varchar(32);not null;default:'private'"`
	Status          string           `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedByUserID string           `json:"created_by_user_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

func (QuestionGroup) TableName() string {
	return "question_groups"
}
```

继续增加：

```go
type QuestionGroupAsset struct {
	ID            string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID      uint64    `json:"tenant_id" gorm:"not null;index"`
	GroupID       string    `json:"group_id" gorm:"type:varchar(36);not null;index"`
	AssetType     string    `json:"asset_type" gorm:"type:varchar(64);not null"`
	StorageURI    string    `json:"storage_uri" gorm:"type:text;not null;default:''"`
	AltText       string    `json:"alt_text" gorm:"type:text;not null;default:''"`
	SourceChunkID string    `json:"source_chunk_id" gorm:"type:varchar(36);not null;default:''"`
	BBox          JSONMap   `json:"bbox" gorm:"type:jsonb;not null"`
	Metadata      JSONMap   `json:"metadata" gorm:"type:jsonb;not null"`
	SortOrder     int       `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt     time.Time `json:"created_at"`
}

func (QuestionGroupAsset) TableName() string {
	return "question_group_assets"
}

type QuestionGroupDetail struct {
	Group     *QuestionGroup       `json:"group"`
	Assets    []*QuestionGroupAsset `json:"assets"`
	Questions []*QuestionDetail     `json:"questions"`
}
```

扩展 `Question`：

```go
	GroupID          *string `json:"group_id,omitempty" gorm:"type:varchar(36);index"`
	QuestionNo       string  `json:"question_no" gorm:"type:varchar(64);not null;default:''"`
	OrderInGroup     int     `json:"order_in_group" gorm:"not null;default:0"`
	QuestionMetadata JSONMap `json:"question_metadata" gorm:"type:jsonb;not null"`
```

- [ ] **步骤 3：新增题组草稿类型**

创建 `internal/types/exam_question_group_draft.go`：

```go
package types

import "time"

type ExamQuestionGroupDraftStatus string

const (
	ExamQuestionGroupDraftStatusPendingReview ExamQuestionGroupDraftStatus = "pending_review"
	ExamQuestionGroupDraftStatusApproved      ExamQuestionGroupDraftStatus = "approved"
	ExamQuestionGroupDraftStatusRejected      ExamQuestionGroupDraftStatus = "rejected"
)

type ExamQuestionGroupDraft struct {
	ID                 string                       `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID           uint64                       `json:"tenant_id" gorm:"not null;index"`
	SpaceID            string                       `json:"space_id" gorm:"type:varchar(36);not null;index"`
	TaskID             string                       `json:"task_id" gorm:"type:varchar(36);not null;index"`
	MaterialID         string                       `json:"material_id" gorm:"type:varchar(36);not null;index"`
	QuestionBankID     string                       `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	DomainID           string                       `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID          *string                      `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	GroupType          string                       `json:"group_type" gorm:"type:varchar(64);not null"`
	Title              string                       `json:"title" gorm:"type:varchar(255);not null;default:''"`
	MaterialText       string                       `json:"material_text" gorm:"type:text;not null;default:''"`
	MaterialFormat     string                       `json:"material_format" gorm:"type:varchar(32);not null;default:'plain_text'"`
	QuestionsJSON      JSON                         `json:"questions_json" gorm:"type:jsonb;not null"`
	AssetsJSON         JSON                         `json:"assets_json" gorm:"type:jsonb;not null"`
	SourceChunkIDs     JSON                         `json:"source_chunk_ids" gorm:"type:jsonb;not null"`
	StrategyCode       string                       `json:"strategy_code" gorm:"type:varchar(64);not null;default:''"`
	Confidence         float64                      `json:"confidence" gorm:"type:numeric(5,4);not null;default:0"`
	Status             ExamQuestionGroupDraftStatus `json:"status" gorm:"type:varchar(32);not null;default:'pending_review'"`
	RawModelOutput     string                       `json:"raw_model_output" gorm:"type:text;not null;default:''"`
	ErrorMessage       string                       `json:"error_message" gorm:"type:text;not null;default:''"`
	ApprovedGroupID    string                       `json:"approved_group_id" gorm:"type:varchar(36);not null;default:''"`
	ReviewedByUserID   string                       `json:"reviewed_by_user_id" gorm:"type:varchar(36);not null;default:''"`
	ReviewedAt         *time.Time                   `json:"reviewed_at,omitempty"`
	CreatedAt          time.Time                    `json:"created_at"`
	UpdatedAt          time.Time                    `json:"updated_at"`
}

func (ExamQuestionGroupDraft) TableName() string {
	return "exam_question_group_drafts"
}
```

继续在同一文件增加候选 DTO：

```go
type ExamQuestionGroupDraftQuestionCandidate struct {
	QuestionNo       string                    `json:"question_no"`
	QuestionTypeCode string                    `json:"question_type_code"`
	Stem             string                    `json:"stem"`
	Options          []ExamQuestionDraftOption `json:"options"`
	Answer           JSONMap                   `json:"answer"`
	Explanation      string                    `json:"explanation"`
	Evidence         []JSONMap                 `json:"evidence"`
	Metadata         JSONMap                   `json:"metadata"`
	Difficulty       string                    `json:"difficulty"`
	Confidence       float64                   `json:"confidence"`
	OrderInGroup     int                       `json:"order_in_group"`
	SourceChunkIDs   []string                  `json:"source_chunk_ids"`
}

type ExamQuestionGroupDraftAssetCandidate struct {
	AssetType     string  `json:"asset_type"`
	StorageURI    string  `json:"storage_uri"`
	AltText       string  `json:"alt_text"`
	SourceChunkID string  `json:"source_chunk_id"`
	BBox          JSONMap `json:"bbox"`
	Metadata      JSONMap `json:"metadata"`
	SortOrder     int     `json:"sort_order"`
}

type ExamQuestionGroupDraftCandidate struct {
	GroupNo        string                                      `json:"group_no"`
	GroupType      string                                      `json:"group_type"`
	Title          string                                      `json:"title"`
	MaterialText   string                                      `json:"material_text"`
	MaterialFormat string                                      `json:"material_format"`
	SourceChunkIDs []string                                    `json:"source_chunk_ids"`
	Assets         []ExamQuestionGroupDraftAssetCandidate      `json:"assets"`
	Questions      []ExamQuestionGroupDraftQuestionCandidate   `json:"questions"`
	StrategyCode   string                                      `json:"strategy_code"`
	Confidence     float64                                     `json:"confidence"`
	RawModelOutput string                                      `json:"raw_model_output"`
}
```

增加请求和结果类型：

```go
type ExtractExamQuestionGroupDraftsRequest struct {
	Force bool `json:"force"`
}

type ExamQuestionGroupDraftStats struct {
	Total         int `json:"total"`
	PendingReview int `json:"pending_review"`
	Approved      int `json:"approved"`
	Rejected      int `json:"rejected"`
}

type ListExamQuestionGroupDraftsResult struct {
	Task   *ExamStructuringTask        `json:"task"`
	Drafts []*ExamQuestionGroupDraft   `json:"drafts"`
	Stats  ExamQuestionGroupDraftStats `json:"stats"`
}

type ExamQuestionGroupDraftExtractionResult = ListExamQuestionGroupDraftsResult

type UpdateExamQuestionGroupDraftRequest struct {
	GroupType      string                                    `json:"group_type" binding:"required"`
	Title          string                                    `json:"title" binding:"omitempty,max=255"`
	MaterialText   string                                    `json:"material_text"`
	MaterialFormat string                                    `json:"material_format"`
	Questions      []ExamQuestionGroupDraftQuestionCandidate `json:"questions" binding:"required"`
	Assets         []ExamQuestionGroupDraftAssetCandidate    `json:"assets"`
	SourceChunkIDs []string                                  `json:"source_chunk_ids"`
}

type ApproveExamQuestionGroupDraftResult struct {
	Draft *ExamQuestionGroupDraft `json:"draft"`
	Group *QuestionGroupDetail    `json:"group"`
}
```

- [ ] **步骤 4：运行类型编译验证**

运行：

```powershell
go test ./internal/types/... -run TestNoSuchTest -count=1
```

预期：exit 0 或提示 `[no test files]`，不能出现类型编译错误。

- [ ] **步骤 5：提交**

```powershell
git add migrations/versioned/000074_exam_question_groups.up.sql migrations/versioned/000074_exam_question_groups.down.sql internal/types/exam_question.go internal/types/exam_question_group_draft.go
git commit -m "feat(exam): 添加题组数据模型"
```

## 任务 2：题组仓储与历史孤立题兼容

**文件：**
- 修改：`internal/types/interfaces/exam_question.go`
- 修改：`internal/application/repository/exam_question.go`
- 创建：`internal/application/repository/exam_question_group_draft.go`
- 创建：`internal/application/repository/exam_question_group_test.go`

- [ ] **步骤 1：先写题组读取仓储测试**

创建 `internal/application/repository/exam_question_group_test.go`：

```go
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

func TestExamQuestionRepository_ListQuestionGroupDetailsByBankWrapsLegacyQuestions(t *testing.T) {
	db := newExamQuestionGroupTestDB(t)
	repo := NewExamQuestionRepository(db)
	now := time.Now()

	question := &types.Question{
		ID:              "question-1",
		TenantID:        10000,
		QuestionBankID:  "bank-1",
		DomainID:        "gaokao",
		Stem:            "What is the answer?",
		Difficulty:      "unknown",
		ReviewStatus:    types.ExamReviewStatusPrivate,
		Status:          "active",
		CreatedByUserID: "teacher-1",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(t, db.Create(question).Error)

	groups, err := repo.ListQuestionGroupDetailsByBank(context.Background(), 10000, "bank-1")

	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "single_question", groups[0].Group.GroupType)
	require.Equal(t, question.Stem, groups[0].Group.MaterialText)
	require.Len(t, groups[0].Questions, 1)
	require.Equal(t, "question-1", groups[0].Questions[0].Question.ID)
}

func newExamQuestionGroupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.QuestionBank{},
		&types.QuestionGroup{},
		&types.QuestionGroupAsset{},
		&types.Question{},
		&types.QuestionOption{},
		&types.QuestionAnswer{},
		&types.QuestionExplanation{},
		&types.QuestionChunkRef{},
	))
	return db
}
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
go test ./internal/application/repository -run QuestionGroup -count=1
```

预期：编译失败，报 `ListQuestionGroupDetailsByBank undefined`。

- [ ] **步骤 3：扩展仓储接口**

在 `internal/types/interfaces/exam_question.go` 的 `ExamQuestionRepository` 增加：

```go
	CreateQuestionGroupDetail(ctx context.Context, detail *types.QuestionGroupDetail) error
	ListQuestionGroupDetailsByBank(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionGroupDetail, error)
	GetQuestionGroupDetailByIDAndTenant(ctx context.Context, tenantID uint64, groupID string) (*types.QuestionGroupDetail, error)
```

在 `ExamQuestionService` 增加：

```go
	ListQuestionGroupDetails(ctx context.Context, tenantID uint64, userID string, bankID string) ([]*types.QuestionGroupDetail, error)
	GetQuestionGroupDetail(ctx context.Context, tenantID uint64, userID string, groupID string) (*types.QuestionGroupDetail, error)
```

- [ ] **步骤 4：实现正式题组写入**

在 `internal/application/repository/exam_question.go` 增加：

```go
func (r *examQuestionRepository) CreateQuestionGroupDetail(ctx context.Context, detail *types.QuestionGroupDetail) error {
	if detail == nil || detail.Group == nil {
		return errors.New("question group detail is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(detail.Group).Error; err != nil {
			return err
		}
		if len(detail.Assets) > 0 {
			if err := tx.Create(&detail.Assets).Error; err != nil {
				return err
			}
		}
		for _, question := range detail.Questions {
			if question == nil || question.Question == nil {
				return errors.New("question detail is required")
			}
			if err := tx.Create(question.Question).Error; err != nil {
				return err
			}
			if len(question.Options) > 0 {
				if err := tx.Create(&question.Options).Error; err != nil {
					return err
				}
			}
			if len(question.Answers) > 0 {
				if err := tx.Create(&question.Answers).Error; err != nil {
					return err
				}
			}
			if len(question.Explanations) > 0 {
				if err := tx.Create(&question.Explanations).Error; err != nil {
					return err
				}
			}
			if len(question.ChunkRefs) > 0 {
				if err := tx.Create(&question.ChunkRefs).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
```

- [ ] **步骤 5：实现题组列表和历史包装**

在 `internal/application/repository/exam_question.go` 增加：

```go
func (r *examQuestionRepository) ListQuestionGroupDetailsByBank(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionGroupDetail, error) {
	groups, err := r.listStoredQuestionGroups(ctx, tenantID, bankID)
	if err != nil {
		return nil, err
	}
	details := make([]*types.QuestionGroupDetail, 0, len(groups))
	for _, group := range groups {
		details = append(details, &types.QuestionGroupDetail{Group: group})
	}
	if err := r.loadQuestionGroupsChildren(ctx, details); err != nil {
		return nil, err
	}
	legacy, err := r.listLegacyQuestionGroups(ctx, tenantID, bankID)
	if err != nil {
		return nil, err
	}
	return append(details, legacy...), nil
}

func (r *examQuestionRepository) listStoredQuestionGroups(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionGroup, error) {
	var groups []*types.QuestionGroup
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND question_bank_id = ? AND status <> ?", tenantID, bankID, "deleted").
		Order("sort_order ASC, created_at DESC").
		Find(&groups).Error
	return groups, err
}
```

继续增加历史包装函数：

```go
func (r *examQuestionRepository) listLegacyQuestionGroups(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionGroupDetail, error) {
	questions, err := r.listQuestionsByGroupState(ctx, tenantID, bankID, false)
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return []*types.QuestionGroupDetail{}, nil
	}
	ids := make([]string, 0, len(questions))
	byID := make(map[string]*types.QuestionDetail, len(questions))
	out := make([]*types.QuestionGroupDetail, 0, len(questions))
	for _, question := range questions {
		ids = append(ids, question.ID)
		detail := &types.QuestionDetail{Question: question}
		byID[question.ID] = detail
		out = append(out, &types.QuestionGroupDetail{
			Group: &types.QuestionGroup{
				ID:             "legacy-" + question.ID,
				TenantID:       question.TenantID,
				QuestionBankID: question.QuestionBankID,
				DomainID:       question.DomainID,
				SubjectID:      question.SubjectID,
				GroupType:      "single_question",
				Title:          question.QuestionNo,
				MaterialText:   question.Stem,
				MaterialFormat: "plain_text",
				ReviewStatus:   question.ReviewStatus,
				Status:         question.Status,
				CreatedAt:      question.CreatedAt,
				UpdatedAt:      question.UpdatedAt,
			},
			Questions: []*types.QuestionDetail{detail},
		})
	}
	if err := r.loadQuestionChildren(ctx, ids, byID); err != nil {
		return nil, err
	}
	return out, nil
}
```

- [ ] **步骤 6：实现题组子对象加载和单组读取**

在同一文件增加：

```go
func (r *examQuestionRepository) loadQuestionGroupsChildren(ctx context.Context, details []*types.QuestionGroupDetail) error {
	if len(details) == 0 {
		return nil
	}
	groupIDs := make([]string, 0, len(details))
	byGroupID := make(map[string]*types.QuestionGroupDetail, len(details))
	for _, detail := range details {
		groupIDs = append(groupIDs, detail.Group.ID)
		byGroupID[detail.Group.ID] = detail
	}
	var assets []*types.QuestionGroupAsset
	if err := r.db.WithContext(ctx).Where("group_id IN ?", groupIDs).Order("sort_order ASC").Find(&assets).Error; err != nil {
		return err
	}
	for _, asset := range assets {
		if detail := byGroupID[asset.GroupID]; detail != nil {
			detail.Assets = append(detail.Assets, asset)
		}
	}
	var questions []*types.Question
	if err := r.db.WithContext(ctx).Where("group_id IN ?", groupIDs).Order("order_in_group ASC, created_at ASC").Find(&questions).Error; err != nil {
		return err
	}
	questionIDs := make([]string, 0, len(questions))
	byQuestionID := make(map[string]*types.QuestionDetail, len(questions))
	for _, question := range questions {
		questionIDs = append(questionIDs, question.ID)
		qd := &types.QuestionDetail{Question: question}
		byQuestionID[question.ID] = qd
		if question.GroupID != nil {
			if detail := byGroupID[*question.GroupID]; detail != nil {
				detail.Questions = append(detail.Questions, qd)
			}
		}
	}
	if len(questionIDs) > 0 {
		return r.loadQuestionChildren(ctx, questionIDs, byQuestionID)
	}
	return nil
}
```

继续增加历史查询和单组读取：

```go
func (r *examQuestionRepository) listQuestionsByGroupState(ctx context.Context, tenantID uint64, bankID string, grouped bool) ([]*types.Question, error) {
	var questions []*types.Question
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND question_bank_id = ? AND status <> ?", tenantID, bankID, "deleted")
	if grouped {
		query = query.Where("group_id IS NOT NULL AND group_id <> ''")
	} else {
		query = query.Where("group_id IS NULL OR group_id = ''")
	}
	err := query.Order("created_at DESC").Find(&questions).Error
	return questions, err
}

func (r *examQuestionRepository) GetQuestionGroupDetailByIDAndTenant(ctx context.Context, tenantID uint64, groupID string) (*types.QuestionGroupDetail, error) {
	var group types.QuestionGroup
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status <> ?", groupID, tenantID, "deleted").
		First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	detail := &types.QuestionGroupDetail{Group: &group}
	if err := r.loadQuestionGroupsChildren(ctx, []*types.QuestionGroupDetail{detail}); err != nil {
		return nil, err
	}
	return detail, nil
}
```

- [ ] **步骤 7：创建题组草稿仓储**

创建 `internal/application/repository/exam_question_group_draft.go`：

```go
package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrQuestionGroupDraftNotFound = errors.New("question group draft not found")

type examQuestionGroupDraftRepository struct {
	db *gorm.DB
}

func NewExamQuestionGroupDraftRepository(db *gorm.DB) interfaces.ExamQuestionGroupDraftRepository {
	return &examQuestionGroupDraftRepository{db: db}
}

func (r *examQuestionGroupDraftRepository) CreateDrafts(ctx context.Context, drafts []*types.ExamQuestionGroupDraft) error {
	if len(drafts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&drafts).Error
}

func (r *examQuestionGroupDraftRepository) DeleteDraftsByTask(ctx context.Context, tenantID uint64, taskID string) error {
	return r.db.WithContext(ctx).Where("tenant_id = ? AND task_id = ?", tenantID, taskID).Delete(&types.ExamQuestionGroupDraft{}).Error
}
```

继续补全读取、统计和更新：

```go
func (r *examQuestionGroupDraftRepository) GetDraftByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamQuestionGroupDraft, error) {
	var draft types.ExamQuestionGroupDraft
	err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&draft).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionGroupDraftNotFound
		}
		return nil, err
	}
	return &draft, nil
}

func (r *examQuestionGroupDraftRepository) ListDraftsByTask(ctx context.Context, tenantID uint64, taskID string) ([]*types.ExamQuestionGroupDraft, error) {
	var drafts []*types.ExamQuestionGroupDraft
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND task_id = ?", tenantID, taskID).
		Order("created_at ASC").
		Find(&drafts).Error
	return drafts, err
}

func (r *examQuestionGroupDraftRepository) CountDraftsByTask(ctx context.Context, tenantID uint64, taskID string) (types.ExamQuestionGroupDraftStats, error) {
	drafts, err := r.ListDraftsByTask(ctx, tenantID, taskID)
	if err != nil {
		return types.ExamQuestionGroupDraftStats{}, err
	}
	stats := types.ExamQuestionGroupDraftStats{Total: len(drafts)}
	for _, draft := range drafts {
		switch draft.Status {
		case types.ExamQuestionGroupDraftStatusPendingReview:
			stats.PendingReview++
		case types.ExamQuestionGroupDraftStatusApproved:
			stats.Approved++
		case types.ExamQuestionGroupDraftStatusRejected:
			stats.Rejected++
		}
	}
	return stats, nil
}

func (r *examQuestionGroupDraftRepository) UpdateDraft(ctx context.Context, draft *types.ExamQuestionGroupDraft) error {
	return r.db.WithContext(ctx).Save(draft).Error
}
```

- [ ] **步骤 8：运行仓储测试**

```powershell
go test ./internal/application/repository -run QuestionGroup -count=1
```

预期：PASS。

- [ ] **步骤 9：提交**

```powershell
git add internal/types/interfaces/exam_question.go internal/application/repository/exam_question.go internal/application/repository/exam_question_group_draft.go internal/application/repository/exam_question_group_test.go
git commit -m "feat(exam): 支持题组仓储与历史题兼容"
```

## 任务 3：题组策略、抽取器和 JSON 校验

**文件：**
- 创建：`internal/types/interfaces/exam_question_group_draft.go`
- 创建：`internal/application/service/exam_question_group_strategy.go`
- 创建：`internal/application/service/exam_question_group_extractor.go`
- 创建：`internal/application/service/exam_question_group_extractor_test.go`

- [ ] **步骤 1：编写策略选择和 JSON 解析测试**

创建 `internal/application/service/exam_question_group_extractor_test.go`：

```go
package service

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestQuestionGroupStrategyRegistrySelectsGaokaoEnglishReading(t *testing.T) {
	subjectID := "english"
	registry := NewQuestionGroupStrategyRegistry()
	strategy := registry.Match(&types.ExamMaterial{DomainID: "gaokao", SubjectID: &subjectID}, &types.ExamStructuringTask{})

	require.NotNil(t, strategy)
	require.Equal(t, "gaokao_english_reading_v1", strategy.Code())
}

func TestParseQuestionGroupCandidatesKeepsReadingMaterialAndQuestions(t *testing.T) {
	raw := `{
	  "question_groups": [{
	    "group_no": "阅读理解A",
	    "group_type": "reading_passage",
	    "title": "阅读理解 A",
	    "material_text": "Passage text",
	    "source_chunk_ids": ["chunk-1"],
	    "questions": [{
	      "question_no": "21",
	      "question_type_code": "single_choice",
	      "stem": "What is true?",
	      "options": [{"key":"A","content":"One"},{"key":"B","content":"Two"}],
	      "answer": {"value":"B"},
	      "explanation": "Because of sentence one.",
	      "confidence": 0.8,
	      "order_in_group": 1
	    }],
	    "confidence": 0.9
	  }]
	}`

	groups, err := parseQuestionGroupDraftCandidates(raw)

	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "reading_passage", groups[0].GroupType)
	require.Equal(t, "Passage text", groups[0].MaterialText)
	require.Len(t, groups[0].Questions, 1)
	require.Equal(t, "21", groups[0].Questions[0].QuestionNo)
}

func TestBuildGaokaoMathPromptMentionsFormulaAndAssets(t *testing.T) {
	subjectID := "math"
	registry := NewQuestionGroupStrategyRegistry()
	strategy := registry.Match(&types.ExamMaterial{DomainID: "gaokao", SubjectID: &subjectID}, &types.ExamStructuringTask{})
	prompt, _, err := strategy.BuildPrompt(QuestionGroupStrategyInput{
		Material: &types.ExamMaterial{Title: "2026 高考数学", DomainID: "gaokao", SubjectID: &subjectID},
		Task:     &types.ExamStructuringTask{QuestionBankID: "bank-1"},
		Chunks:   []*types.Chunk{{ID: "chunk-1", ChunkIndex: 1, Content: "17. 已知函数 f(x)=x^2"}},
	})

	require.NoError(t, err)
	require.True(t, strings.Contains(prompt, "LaTeX"))
	require.True(t, strings.Contains(prompt, "assets"))
	require.True(t, strings.Contains(prompt, "math_problem"))
}
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
go test ./internal/application/service -run QuestionGroup -count=1
```

预期：编译失败，报 `NewQuestionGroupStrategyRegistry undefined`。

- [ ] **步骤 3：定义接口**

创建 `internal/types/interfaces/exam_question_group_draft.go`：

```go
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
```

- [ ] **步骤 4：实现策略注册表**

创建 `internal/application/service/exam_question_group_strategy.go`：

```go
package service

import (
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

type QuestionGroupStrategyInput struct {
	Material *types.ExamMaterial
	Task     *types.ExamStructuringTask
	Chunks   []*types.Chunk
}

type QuestionGroupStrategyOptions struct {
	Temperature float64
	TopP        float64
	MaxTokens   int
}

type ExamQuestionGroupExtractionStrategy interface {
	Code() string
	Match(material *types.ExamMaterial, task *types.ExamStructuringTask) bool
	BuildPrompt(input QuestionGroupStrategyInput) (string, QuestionGroupStrategyOptions, error)
	Validate(candidate *types.ExamQuestionGroupDraftCandidate) error
}
```

继续增加注册表：

```go
type QuestionGroupStrategyRegistry struct {
	strategies []ExamQuestionGroupExtractionStrategy
}

func NewQuestionGroupStrategyRegistry() *QuestionGroupStrategyRegistry {
	return &QuestionGroupStrategyRegistry{
		strategies: []ExamQuestionGroupExtractionStrategy{
			gaokaoEnglishReadingStrategy{},
			gaokaoMathBasicStrategy{},
			genericQuestionGroupStrategy{},
		},
	}
}

func (r *QuestionGroupStrategyRegistry) Match(material *types.ExamMaterial, task *types.ExamStructuringTask) ExamQuestionGroupExtractionStrategy {
	for _, strategy := range r.strategies {
		if strategy.Match(material, task) {
			return strategy
		}
	}
	return genericQuestionGroupStrategy{}
}
```

- [ ] **步骤 5：实现英语和数学策略**

在同一文件增加：

```go
type gaokaoEnglishReadingStrategy struct{}

func (gaokaoEnglishReadingStrategy) Code() string { return "gaokao_english_reading_v1" }
func (gaokaoEnglishReadingStrategy) Match(material *types.ExamMaterial, task *types.ExamStructuringTask) bool {
	return material != nil && material.DomainID == "gaokao" && promptSubjectID(material) == "english"
}
func (gaokaoEnglishReadingStrategy) BuildPrompt(input QuestionGroupStrategyInput) (string, QuestionGroupStrategyOptions, error) {
	return buildQuestionGroupPrompt(input, "请优先抽取第一篇完整高考英语阅读理解，返回 reading_passage 题组。material_text 必须包含完整阅读原文，每道小题必须包含选项、答案、解析、证据句和来源 chunk。"), defaultQuestionGroupStrategyOptions(), nil
}
func (gaokaoEnglishReadingStrategy) Validate(candidate *types.ExamQuestionGroupDraftCandidate) error {
	return validateQuestionGroupCandidate(candidate, true)
}

type gaokaoMathBasicStrategy struct{}

func (gaokaoMathBasicStrategy) Code() string { return "gaokao_math_basic_v1" }
func (gaokaoMathBasicStrategy) Match(material *types.ExamMaterial, task *types.ExamStructuringTask) bool {
	return material != nil && material.DomainID == "gaokao" && promptSubjectID(material) == "math"
}
func (gaokaoMathBasicStrategy) BuildPrompt(input QuestionGroupStrategyInput) (string, QuestionGroupStrategyOptions, error) {
	return buildQuestionGroupPrompt(input, "请抽取高考数学题组，返回 math_problem 或 single_question。保留公式原文并尽量生成 LaTeX；图片、图形、表格用 assets 表达，不要塞进纯文本题干。"), defaultQuestionGroupStrategyOptions(), nil
}
func (gaokaoMathBasicStrategy) Validate(candidate *types.ExamQuestionGroupDraftCandidate) error {
	return validateQuestionGroupCandidate(candidate, false)
}
```

增加通用策略和 prompt：

```go
type genericQuestionGroupStrategy struct{}

func (genericQuestionGroupStrategy) Code() string { return "generic_question_group_v1" }
func (genericQuestionGroupStrategy) Match(material *types.ExamMaterial, task *types.ExamStructuringTask) bool {
	return true
}
func (genericQuestionGroupStrategy) BuildPrompt(input QuestionGroupStrategyInput) (string, QuestionGroupStrategyOptions, error) {
	return buildQuestionGroupPrompt(input, "请抽取一个完整题组。无法判断材料型题组时，使用 single_question。"), defaultQuestionGroupStrategyOptions(), nil
}
func (genericQuestionGroupStrategy) Validate(candidate *types.ExamQuestionGroupDraftCandidate) error {
	return validateQuestionGroupCandidate(candidate, false)
}

func defaultQuestionGroupStrategyOptions() QuestionGroupStrategyOptions {
	return QuestionGroupStrategyOptions{Temperature: 0.1, TopP: 0.2, MaxTokens: 4096}
}
```

- [ ] **步骤 6：实现题组 prompt 和校验**

在同一文件增加：

```go
func buildQuestionGroupPrompt(input QuestionGroupStrategyInput, instruction string) string {
	var builder strings.Builder
	builder.WriteString(instruction)
	builder.WriteString("\n只返回 JSON，不要输出 Markdown、解释或代码块。根对象必须包含 question_groups 数组。")
	if input.Material != nil {
		builder.WriteString(fmt.Sprintf("\n资料标题：%s\n考试域：%s\n科目：%s\n", input.Material.Title, input.Material.DomainID, promptSubjectID(input.Material)))
	}
	if input.Task != nil {
		builder.WriteString(fmt.Sprintf("目标题库：%s\n", input.Task.QuestionBankID))
	}
	builder.WriteString("\nJSON 字段要求：group_type、title、material_text、source_chunk_ids、assets、questions、confidence。")
	builder.WriteString("\n每个 question 必须包含 question_no、question_type_code、stem、options、answer、explanation、evidence、difficulty、confidence、order_in_group。")
	for _, chunk := range chooseQuestionExtractionChunks(input.Chunks) {
		builder.WriteString(fmt.Sprintf("\n--- chunk_id: %s | chunk_index: %d ---\n%s\n", chunk.ID, chunk.ChunkIndex, chunk.Content))
	}
	return builder.String()
}

func validateQuestionGroupCandidate(candidate *types.ExamQuestionGroupDraftCandidate, requireMaterial bool) error {
	if candidate == nil {
		return fmt.Errorf("question group candidate is required")
	}
	if strings.TrimSpace(candidate.GroupType) == "" {
		return fmt.Errorf("question group type is required")
	}
	if requireMaterial && strings.TrimSpace(candidate.MaterialText) == "" {
		return fmt.Errorf("reading question group material_text is required")
	}
	if len(candidate.Questions) == 0 {
		return fmt.Errorf("question group must contain questions")
	}
	for _, question := range candidate.Questions {
		if strings.TrimSpace(question.Stem) == "" {
			return fmt.Errorf("question stem is required")
		}
		if strings.Contains(question.QuestionTypeCode, "choice") && len(question.Options) < 2 {
			return fmt.Errorf("choice question requires at least two options")
		}
	}
	return nil
}
```

- [ ] **步骤 7：实现抽取器和 JSON 解析**

创建 `internal/application/service/exam_question_group_extractor.go`：

```go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const examQuestionGroupExtractionSystemPrompt = `你是考试试卷题组结构化助手。请从用户提供的试卷 chunk 中抽取完整题组。
只返回 JSON，不要输出 Markdown、解释或代码块。`

type examQuestionGroupExtractor struct {
	modelService interfaces.ModelService
	registry     *QuestionGroupStrategyRegistry
}

func NewExamQuestionGroupExtractor(modelService interfaces.ModelService) interfaces.ExamQuestionGroupExtractor {
	return &examQuestionGroupExtractor{modelService: modelService, registry: NewQuestionGroupStrategyRegistry()}
}
```

继续实现 `Extract`：

```go
func (e *examQuestionGroupExtractor) Extract(ctx context.Context, material *types.ExamMaterial, task *types.ExamStructuringTask, chunks []*types.Chunk) ([]*types.ExamQuestionGroupDraftCandidate, string, error) {
	chatModel, err := loadExamQuestionChatModel(ctx, e.modelService)
	if err != nil {
		return nil, "", err
	}
	strategy := e.registry.Match(material, task)
	prompt, opts, err := strategy.BuildPrompt(QuestionGroupStrategyInput{Material: material, Task: task, Chunks: chunks})
	if err != nil {
		return nil, "", err
	}
	raw, err := callQuestionGroupExtractionModel(ctx, chatModel, prompt, opts)
	if err != nil {
		return nil, raw, err
	}
	candidates, err := parseQuestionGroupDraftCandidates(raw)
	if err != nil {
		return nil, raw, err
	}
	for _, candidate := range candidates {
		candidate.StrategyCode = strategy.Code()
		if err := strategy.Validate(candidate); err != nil {
			return nil, raw, err
		}
	}
	return candidates, raw, nil
}
```

增加模型调用和 JSON 解析：

```go
func callQuestionGroupExtractionModel(ctx context.Context, chatModel chat.Chat, prompt string, opts QuestionGroupStrategyOptions) (string, error) {
	temperature := opts.Temperature
	topP := opts.TopP
	response, err := chatModel.Chat(ctx, []*chat.Message{
		{Role: "system", Content: examQuestionGroupExtractionSystemPrompt},
		{Role: "user", Content: prompt},
	}, &chat.ChatOptions{Temperature: &temperature, TopP: &topP})
	if err != nil {
		return "", fmt.Errorf("extract exam question groups: %w", err)
	}
	if response == nil || strings.TrimSpace(response.Content) == "" {
		return "", errors.New("empty model response")
	}
	return response.Content, nil
}

func parseQuestionGroupDraftCandidates(raw string) ([]*types.ExamQuestionGroupDraftCandidate, error) {
	payload := extractJSONPayload(raw)
	var envelope struct {
		QuestionGroups []*types.ExamQuestionGroupDraftCandidate `json:"question_groups"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		return nil, fmt.Errorf("invalid model json: %w", err)
	}
	if len(envelope.QuestionGroups) == 0 {
		return nil, errors.New("model returned no question groups")
	}
	for _, group := range envelope.QuestionGroups {
		normalizeQuestionGroupCandidate(group, raw)
	}
	return envelope.QuestionGroups, nil
}
```

增加规范化：

```go
func normalizeQuestionGroupCandidate(candidate *types.ExamQuestionGroupDraftCandidate, raw string) {
	candidate.GroupType = strings.TrimSpace(candidate.GroupType)
	if candidate.GroupType == "" {
		candidate.GroupType = "single_question"
	}
	if candidate.MaterialFormat == "" {
		candidate.MaterialFormat = "plain_text"
	}
	candidate.Confidence = normalizeConfidence(candidate.Confidence)
	candidate.RawModelOutput = raw
	for i := range candidate.Questions {
		if candidate.Questions[i].OrderInGroup == 0 {
			candidate.Questions[i].OrderInGroup = i + 1
		}
		candidate.Questions[i].Confidence = normalizeConfidence(candidate.Questions[i].Confidence)
		if candidate.Questions[i].Difficulty == "" {
			candidate.Questions[i].Difficulty = "unknown"
		}
		if candidate.Questions[i].Answer == nil {
			candidate.Questions[i].Answer = types.JSONMap{}
		}
		if candidate.Questions[i].Metadata == nil {
			candidate.Questions[i].Metadata = types.JSONMap{}
		}
	}
}
```

- [ ] **步骤 8：运行抽取器测试**

```powershell
go test ./internal/application/service -run QuestionGroup -count=1
```

预期：PASS。

- [ ] **步骤 9：提交**

```powershell
git add internal/types/interfaces/exam_question_group_draft.go internal/application/service/exam_question_group_strategy.go internal/application/service/exam_question_group_extractor.go internal/application/service/exam_question_group_extractor_test.go
git commit -m "feat(exam): 添加题组抽取策略"
```

## 任务 4：题组草稿服务与确认入库

**文件：**
- 创建：`internal/application/service/exam_question_group_draft.go`
- 创建：`internal/application/service/exam_question_group_build.go`
- 创建：`internal/application/service/exam_question_group_draft_test.go`
- 修改：`internal/types/interfaces/exam_question_group_draft.go`

- [ ] **步骤 1：编写权限和入库测试**

创建 `internal/application/service/exam_question_group_draft_test.go`：

```go
package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestExamQuestionGroupDraftService_ExtractRequiresWritableSpace(t *testing.T) {
	ctx := context.Background()
	repo := newReadyQuestionGroupDraftRepo()
	svc := newTestExamQuestionGroupDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: false}, nil, nil, nil)

	_, err := svc.ExtractDrafts(ctx, 10000, "student-1", "task-1", &types.ExtractExamQuestionGroupDraftsRequest{})

	require.ErrorIs(t, err, ErrExamPermissionDenied)
}

func TestExamQuestionGroupDraftService_ExtractWritesGroupDrafts(t *testing.T) {
	ctx := context.Background()
	repo := newReadyQuestionGroupDraftRepo()
	extractor := &stubQuestionGroupExtractor{candidates: []*types.ExamQuestionGroupDraftCandidate{newReadingGroupCandidate()}}
	svc := newTestExamQuestionGroupDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: true}, &stubExamQuestionDraftChunkReader{
		chunks: []*types.Chunk{{ID: "chunk-1", KnowledgeID: "knowledge-1", Content: "Passage"}},
	}, extractor, nil)

	result, err := svc.ExtractDrafts(ctx, 10000, "teacher-1", "task-1", &types.ExtractExamQuestionGroupDraftsRequest{})

	require.NoError(t, err)
	require.Len(t, result.Drafts, 1)
	require.Equal(t, "reading_passage", result.Drafts[0].GroupType)
	require.Equal(t, types.ExamStructuringTaskStatusReviewing, repo.updatedStatus)
}
```

继续添加确认入库测试：

```go
func TestExamQuestionGroupDraftService_ApproveCreatesOfficialGroup(t *testing.T) {
	ctx := context.Background()
	repo := newReadyQuestionGroupDraftRepo()
	repo.drafts = []*types.ExamQuestionGroupDraft{newPendingQuestionGroupDraft("draft-1")}
	questionRepo := &stubQuestionGroupWriter{}
	svc := newTestExamQuestionGroupDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: true}, nil, nil, questionRepo)

	result, err := svc.ApproveDraft(ctx, 10000, "teacher-1", "draft-1")

	require.NoError(t, err)
	require.Equal(t, types.ExamQuestionGroupDraftStatusApproved, result.Draft.Status)
	require.Equal(t, "group-1", result.Draft.ApprovedGroupID)
	require.Len(t, questionRepo.created, 1)
	require.Equal(t, "reading_passage", questionRepo.created[0].Group.GroupType)
	require.Len(t, questionRepo.created[0].Questions, 3)
}
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
go test ./internal/application/service -run ExamQuestionGroupDraft -count=1
```

预期：编译失败，报 `NewExamQuestionGroupDraftService undefined`。

- [ ] **步骤 3：实现服务骨架**

创建 `internal/application/service/exam_question_group_draft.go`：

```go
package service

import (
	"context"
	"errors"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type examQuestionGroupDraftService struct {
	draftRepo    interfaces.ExamQuestionGroupDraftRepository
	materialRepo interfaces.ExamMaterialRepository
	questionRepo interfaces.ExamQuestionRepository
	spaceService interfaces.ExamSpaceService
	chunkReader  interfaces.ExamMaterialChunkReader
	extractor    interfaces.ExamQuestionGroupExtractor
}

func NewExamQuestionGroupDraftService(
	draftRepo interfaces.ExamQuestionGroupDraftRepository,
	materialRepo interfaces.ExamMaterialRepository,
	questionRepo interfaces.ExamQuestionRepository,
	spaceService interfaces.ExamSpaceService,
	chunkReader interfaces.ExamMaterialChunkReader,
	extractor interfaces.ExamQuestionGroupExtractor,
) interfaces.ExamQuestionGroupDraftService {
	return &examQuestionGroupDraftService{
		draftRepo: draftRepo, materialRepo: materialRepo, questionRepo: questionRepo,
		spaceService: spaceService, chunkReader: chunkReader, extractor: extractor,
	}
}
```

- [ ] **步骤 4：实现抽取和列表**

在同一文件增加：

```go
func (s *examQuestionGroupDraftService) ExtractDrafts(ctx context.Context, tenantID uint64, userID string, taskID string, req *types.ExtractExamQuestionGroupDraftsRequest) (*types.ExamQuestionGroupDraftExtractionResult, error) {
	task, material, err := s.prepareExtraction(ctx, tenantID, userID, taskID)
	if err != nil {
		return nil, err
	}
	if req != nil && req.Force {
		if err := s.draftRepo.DeleteDraftsByTask(ctx, tenantID, task.ID); err != nil {
			return nil, err
		}
	}
	if _, err := s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusExtracting, -1, ""); err != nil {
		return nil, err
	}
	chunks, err := s.chunkReader.ListChunksByKnowledgeID(ctx, material.KnowledgeID)
	if err != nil {
		_, _ = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusFailed, 0, err.Error())
		return nil, err
	}
	if len(chunks) == 0 {
		err = errors.New("exam material has no available chunks")
		_, _ = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusFailed, 0, err.Error())
		return nil, err
	}
	candidates, rawOutput, err := s.extractor.Extract(context.WithValue(ctx, types.TenantIDContextKey, tenantID), material, task, chunks)
	if err != nil {
		_, _ = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusFailed, 0, err.Error())
		return nil, err
	}
	drafts, err := buildQuestionGroupDrafts(task, material, candidates, rawOutput)
	if err != nil {
		_, _ = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusFailed, 0, err.Error())
		return nil, err
	}
	if err := s.draftRepo.CreateDrafts(ctx, drafts); err != nil {
		return nil, err
	}
	task, err = s.updateTaskStatus(ctx, task, types.ExamStructuringTaskStatusReviewing, len(drafts), "")
	if err != nil {
		return nil, err
	}
	stats, err := s.draftRepo.CountDraftsByTask(ctx, tenantID, task.ID)
	if err != nil {
		return nil, err
	}
	return &types.ExamQuestionGroupDraftExtractionResult{Task: task, Drafts: drafts, Stats: stats}, nil
}
```

复用现有草稿服务的权限逻辑：把 `prepareExtraction`、`getTaskForRead`、`updateTaskStatus` 中与仓储无关的逻辑抽到共享 helper，或在题组服务中保留同名私有方法。计划执行时优先选择小范围复制，避免影响现有扁平草稿路径。

- [ ] **步骤 5：实现草稿构建**

创建 `internal/application/service/exam_question_group_build.go`：

```go
package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
)

func buildQuestionGroupDrafts(task *types.ExamStructuringTask, material *types.ExamMaterial, candidates []*types.ExamQuestionGroupDraftCandidate, rawOutput string) ([]*types.ExamQuestionGroupDraft, error) {
	if len(candidates) == 0 {
		return nil, errors.New("model returned no question group drafts")
	}
	now := time.Now()
	out := make([]*types.ExamQuestionGroupDraft, 0, len(candidates))
	for _, candidate := range candidates {
		draft, err := buildQuestionGroupDraftFromCandidate(task, material, candidate, rawOutput, now)
		if err != nil {
			return nil, err
		}
		out = append(out, draft)
	}
	return out, nil
}
```

继续增加单个草稿构建：

```go
func buildQuestionGroupDraftFromCandidate(task *types.ExamStructuringTask, material *types.ExamMaterial, candidate *types.ExamQuestionGroupDraftCandidate, rawOutput string, now time.Time) (*types.ExamQuestionGroupDraft, error) {
	if candidate == nil || len(candidate.Questions) == 0 {
		return nil, errors.New("question group draft questions are required")
	}
	questionsJSON, err := json.Marshal(candidate.Questions)
	if err != nil {
		return nil, err
	}
	assetsJSON, err := json.Marshal(candidate.Assets)
	if err != nil {
		return nil, err
	}
	chunkJSON, err := json.Marshal(candidate.SourceChunkIDs)
	if err != nil {
		return nil, err
	}
	if candidate.RawModelOutput != "" {
		rawOutput = candidate.RawModelOutput
	}
	return &types.ExamQuestionGroupDraft{
		ID:             uuid.New().String(),
		TenantID:       task.TenantID,
		SpaceID:        task.SpaceID,
		TaskID:         task.ID,
		MaterialID:     material.ID,
		QuestionBankID: task.QuestionBankID,
		DomainID:       material.DomainID,
		SubjectID:      material.SubjectID,
		GroupType:      strings.TrimSpace(candidate.GroupType),
		Title:          strings.TrimSpace(candidate.Title),
		MaterialText:   strings.TrimSpace(candidate.MaterialText),
		MaterialFormat: defaultString(candidate.MaterialFormat, "plain_text"),
		QuestionsJSON:  types.JSON(questionsJSON),
		AssetsJSON:     types.JSON(assetsJSON),
		SourceChunkIDs: types.JSON(chunkJSON),
		StrategyCode:   strings.TrimSpace(candidate.StrategyCode),
		Confidence:     normalizeConfidence(candidate.Confidence),
		Status:         types.ExamQuestionGroupDraftStatusPendingReview,
		RawModelOutput: rawOutput,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}
```

同一文件增加通用字符串 helper：

```go
func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
```

- [ ] **步骤 6：实现确认入正式题组**

在 `exam_question_group_build.go` 增加：

```go
func buildQuestionGroupDetailFromDraft(draft *types.ExamQuestionGroupDraft, userID string) (*types.QuestionGroupDetail, error) {
	questions, err := draftGroupQuestions(draft)
	if err != nil {
		return nil, err
	}
	assets, err := draftGroupAssets(draft)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	groupID := uuid.New().String()
	detail := &types.QuestionGroupDetail{
		Group:  buildQuestionGroupFromDraft(draft, groupID, userID, now),
		Assets: buildQuestionGroupAssets(groupID, draft.TenantID, assets, now),
	}
	for _, candidate := range questions {
		qd, err := buildQuestionDetailFromGroupCandidate(draft, groupID, candidate, userID, now)
		if err != nil {
			return nil, err
		}
		detail.Questions = append(detail.Questions, qd)
	}
	return detail, nil
}
```

实现小题转换：

```go
func buildQuestionDetailFromGroupCandidate(draft *types.ExamQuestionGroupDraft, groupID string, candidate types.ExamQuestionGroupDraftQuestionCandidate, userID string, now time.Time) (*types.QuestionDetail, error) {
	answer := draftAnswerText(candidate.Answer)
	if answer == "" {
		return nil, errors.New("question answer is required")
	}
	questionID := uuid.New().String()
	metadata := candidate.Metadata
	if metadata == nil {
		metadata = types.JSONMap{}
	}
	return &types.QuestionDetail{
		Question: &types.Question{
			ID: questionID, TenantID: draft.TenantID, QuestionBankID: draft.QuestionBankID,
			DomainID: draft.DomainID, SubjectID: draft.SubjectID, GroupID: &groupID,
			QuestionNo: strings.TrimSpace(candidate.QuestionNo), OrderInGroup: candidate.OrderInGroup,
			QuestionMetadata: metadata, Stem: strings.TrimSpace(candidate.Stem),
			Difficulty: defaultString(candidate.Difficulty, "unknown"),
			ReviewStatus: types.ExamReviewStatusPrivate, Status: "active",
			CreatedByUserID: userID, CreatedAt: now, UpdatedAt: now,
		},
		Options:      buildQuestionOptions(questionID, candidate.Options),
		Answers:      buildQuestionAnswers(questionID, answer, now),
		Explanations: buildQuestionExplanations(questionID, candidate.Explanation, now),
		ChunkRefs:    buildQuestionChunkRefs(questionID, candidate.SourceChunkIDs, candidate.Confidence, now),
	}, nil
}
```

补齐题组、资产和 JSON 读取 helper：

```go
func buildQuestionGroupFromDraft(draft *types.ExamQuestionGroupDraft, groupID string, userID string, now time.Time) *types.QuestionGroup {
	return &types.QuestionGroup{
		ID: groupID, TenantID: draft.TenantID, SpaceID: draft.SpaceID,
		QuestionBankID: draft.QuestionBankID, DomainID: draft.DomainID, SubjectID: draft.SubjectID,
		GroupType: draft.GroupType, Title: draft.Title, MaterialText: draft.MaterialText,
		MaterialFormat: defaultString(draft.MaterialFormat, "plain_text"),
		AssetRefs: draft.AssetsJSON, SourceChunkIDs: draft.SourceChunkIDs,
		ReviewStatus: types.ExamReviewStatusPrivate, Status: "active",
		CreatedByUserID: userID, CreatedAt: now, UpdatedAt: now,
	}
}

func buildQuestionGroupAssets(groupID string, tenantID uint64, assets []types.ExamQuestionGroupDraftAssetCandidate, now time.Time) []*types.QuestionGroupAsset {
	out := make([]*types.QuestionGroupAsset, 0, len(assets))
	for i, asset := range assets {
		out = append(out, &types.QuestionGroupAsset{
			ID: uuid.New().String(), TenantID: tenantID, GroupID: groupID,
			AssetType: strings.TrimSpace(asset.AssetType), StorageURI: strings.TrimSpace(asset.StorageURI),
			AltText: strings.TrimSpace(asset.AltText), SourceChunkID: strings.TrimSpace(asset.SourceChunkID),
			BBox: asset.BBox, Metadata: asset.Metadata, SortOrder: i + 1, CreatedAt: now,
		})
	}
	return out
}
```

继续增加 JSON 解析：

```go
func draftGroupQuestions(draft *types.ExamQuestionGroupDraft) ([]types.ExamQuestionGroupDraftQuestionCandidate, error) {
	var questions []types.ExamQuestionGroupDraftQuestionCandidate
	if len(draft.QuestionsJSON) == 0 {
		return questions, nil
	}
	if err := json.Unmarshal(draft.QuestionsJSON, &questions); err != nil {
		return nil, err
	}
	return questions, nil
}

func draftGroupAssets(draft *types.ExamQuestionGroupDraft) ([]types.ExamQuestionGroupDraftAssetCandidate, error) {
	var assets []types.ExamQuestionGroupDraftAssetCandidate
	if len(draft.AssetsJSON) == 0 {
		return assets, nil
	}
	if err := json.Unmarshal(draft.AssetsJSON, &assets); err != nil {
		return nil, err
	}
	return assets, nil
}
```

- [ ] **步骤 7：实现 `ApproveDraft`**

在 `exam_question_group_draft.go` 增加：

```go
func (s *examQuestionGroupDraftService) ApproveDraft(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ApproveExamQuestionGroupDraftResult, error) {
	draft, err := s.getDraftForWrite(ctx, tenantID, userID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status == types.ExamQuestionGroupDraftStatusApproved && draft.ApprovedGroupID != "" {
		group, _ := s.questionRepo.GetQuestionGroupDetailByIDAndTenant(ctx, tenantID, draft.ApprovedGroupID)
		return &types.ApproveExamQuestionGroupDraftResult{Draft: draft, Group: group}, nil
	}
	if draft.Status != types.ExamQuestionGroupDraftStatusPendingReview {
		return nil, ErrExamInvalidRequest
	}
	group, err := buildQuestionGroupDetailFromDraft(draft, userID)
	if err != nil {
		return nil, err
	}
	if err := s.questionRepo.CreateQuestionGroupDetail(ctx, group); err != nil {
		return nil, err
	}
	now := time.Now()
	draft.Status = types.ExamQuestionGroupDraftStatusApproved
	draft.ApprovedGroupID = group.Group.ID
	draft.ReviewedByUserID = userID
	draft.ReviewedAt = &now
	draft.UpdatedAt = now
	if err := s.draftRepo.UpdateDraft(ctx, draft); err != nil {
		return nil, err
	}
	return &types.ApproveExamQuestionGroupDraftResult{Draft: draft, Group: group}, nil
}
```

- [ ] **步骤 8：实现列表、更新和驳回**

在 `exam_question_group_draft.go` 增加 `ListDrafts`、`UpdateDraft`、`RejectDraft`，行为与现有 `ExamQuestionDraftService` 对齐。更新时重新 marshal `Questions`、`Assets`、`SourceChunkIDs`，并保持 `pending_review` 才能编辑。

```go
func (s *examQuestionGroupDraftService) RejectDraft(ctx context.Context, tenantID uint64, userID string, draftID string) (*types.ExamQuestionGroupDraft, error) {
	draft, err := s.getDraftForWrite(ctx, tenantID, userID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status != types.ExamQuestionGroupDraftStatusPendingReview {
		return nil, ErrExamInvalidRequest
	}
	now := time.Now()
	draft.Status = types.ExamQuestionGroupDraftStatusRejected
	draft.ReviewedByUserID = userID
	draft.ReviewedAt = &now
	draft.UpdatedAt = now
	if err := s.draftRepo.UpdateDraft(ctx, draft); err != nil {
		return nil, err
	}
	return draft, nil
}
```

- [ ] **步骤 9：运行服务测试**

```powershell
go test ./internal/application/service -run ExamQuestionGroupDraft -count=1
```

预期：PASS。

- [ ] **步骤 10：提交**

```powershell
git add internal/application/service/exam_question_group_draft.go internal/application/service/exam_question_group_build.go internal/application/service/exam_question_group_draft_test.go internal/types/interfaces/exam_question_group_draft.go
git commit -m "feat(exam): 实现题组草稿校对服务"
```

## 任务 5：HTTP 路由、容器注册和后端集成

**文件：**
- 创建：`internal/handler/exam_question_group_draft.go`
- 修改：`internal/handler/exam_question.go`
- 修改：`internal/router/exam.go`
- 修改：`internal/container/container.go`
- 修改：`internal/router/exam_rbac_routes_test.go`

- [ ] **步骤 1：编写路由守卫测试**

在 `internal/router/exam_rbac_routes_test.go` 增加：

```go
func TestExamRoutes_QuestionGroupDraftRoutesRequireContributor(t *testing.T) {
	routes := collectExamRoutePermissions(t)

	require.Equal(t, "contributor", routes["POST /exam/structuring-tasks/:task_id/group-extract"])
	require.Equal(t, "contributor", routes["GET /exam/structuring-tasks/:task_id/group-drafts"])
	require.Equal(t, "contributor", routes["PATCH /exam/question-group-drafts/:draft_id"])
	require.Equal(t, "contributor", routes["POST /exam/question-group-drafts/:draft_id/approve"])
	require.Equal(t, "contributor", routes["POST /exam/question-group-drafts/:draft_id/reject"])
}

func TestExamRoutes_QuestionGroupListAllowsViewer(t *testing.T) {
	routes := collectExamRoutePermissions(t)

	require.Equal(t, "viewer", routes["GET /exam/question-banks/:bank_id/question-groups"])
}
```

- [ ] **步骤 2：运行测试验证失败**

```powershell
go test ./internal/router -run ExamRoutes -count=1
```

预期：失败，提示缺少新路由。

- [ ] **步骤 3：创建题组草稿 handler**

创建 `internal/handler/exam_question_group_draft.go`：

```go
package handler

import (
	"net/http"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamQuestionGroupDraftHandler struct {
	service interfaces.ExamQuestionGroupDraftService
}

func NewExamQuestionGroupDraftHandler(service interfaces.ExamQuestionGroupDraftService) *ExamQuestionGroupDraftHandler {
	return &ExamQuestionGroupDraftHandler{service: service}
}
```

继续增加方法：

```go
func (h *ExamQuestionGroupDraftHandler) ExtractDrafts(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	var req types.ExtractExamQuestionGroupDraftsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	result, err := h.service.ExtractDrafts(ctx, tenantID, userID, c.Param("task_id"), &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to extract exam question group drafts: %v", err)
		writeExamError(c, err, "Failed to extract exam question group drafts")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
```

按同一模式实现 `ListDrafts`、`UpdateDraft`、`ApproveDraft`、`RejectDraft`。

- [ ] **步骤 4：增加正式题组 handler**

在 `internal/handler/exam_question.go` 增加：

```go
func (h *ExamQuestionHandler) ListQuestionGroupDetails(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	groups, err := h.service.ListQuestionGroupDetails(ctx, tenantID, userID, c.Param("bank_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam question groups: %v", err)
		writeExamError(c, err, "Failed to list exam question groups")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": groups})
}
```

在 `internal/application/service/exam_question.go` 增加服务方法，权限逻辑复用题库读取：

```go
func (s *examQuestionService) ListQuestionGroupDetails(ctx context.Context, tenantID uint64, userID string, bankID string) ([]*types.QuestionGroupDetail, error) {
	bank, err := s.questionRepo.GetQuestionBankByIDAndTenant(ctx, bankID, tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureReadableSpace(ctx, tenantID, userID, bank.SpaceID); err != nil {
		return nil, err
	}
	return s.questionRepo.ListQuestionGroupDetailsByBank(ctx, tenantID, bankID)
}
```

- [ ] **步骤 5：注册路由**

修改 `internal/router/exam.go` 的 `RegisterExamRoutes` 签名，增加：

```go
	questionGroupDraftHandler *handler.ExamQuestionGroupDraftHandler,
```

注册路由：

```go
exam.GET("/question-banks/:bank_id/question-groups", g.Viewer(), questionHandler.ListQuestionGroupDetails)
exam.POST("/structuring-tasks/:task_id/group-extract", g.Contributor(), questionGroupDraftHandler.ExtractDrafts)
exam.GET("/structuring-tasks/:task_id/group-drafts", g.Contributor(), questionGroupDraftHandler.ListDrafts)
exam.PATCH("/question-group-drafts/:draft_id", g.Contributor(), questionGroupDraftHandler.UpdateDraft)
exam.POST("/question-group-drafts/:draft_id/approve", g.Contributor(), questionGroupDraftHandler.ApproveDraft)
exam.POST("/question-group-drafts/:draft_id/reject", g.Contributor(), questionGroupDraftHandler.RejectDraft)
```

- [ ] **步骤 6：注册容器依赖**

在 `internal/container/container.go` 仓储区域增加：

```go
must(container.Provide(repository.NewExamQuestionGroupDraftRepository))
```

在服务区域增加：

```go
must(container.Provide(service.NewExamQuestionGroupExtractor))
must(container.Provide(service.NewExamQuestionGroupDraftService))
```

在 handler 注册区域增加：

```go
must(container.Provide(handler.NewExamQuestionGroupDraftHandler))
```

同步修改调用 `router.RegisterExamRoutes` 的参数列表，把新 handler 传入。

- [ ] **步骤 7：运行后端集成验证**

```powershell
go test ./internal/router ./internal/handler ./internal/container -run Exam -count=1
```

预期：PASS。

- [ ] **步骤 8：提交**

```powershell
git add internal/handler/exam_question_group_draft.go internal/handler/exam_question.go internal/router/exam.go internal/container/container.go internal/router/exam_rbac_routes_test.go internal/application/service/exam_question.go
git commit -m "feat(exam): 接入题组接口路由"
```

## 任务 6：前端类型、API 和题组校对入口

**文件：**
- 修改：`frontend/src/types/exam.ts`
- 创建：`frontend/src/api/exam/question-group.ts`
- 创建：`frontend/src/api/exam/question-group-draft.ts`
- 修改：`frontend/src/router/index.ts`
- 修改：`frontend/src/views/classes/ClassDetail.vue`
- 创建：`frontend/src/views/question-draft/QuestionGroupDraftReview.vue`

- [ ] **步骤 1：扩展前端类型**

在 `frontend/src/types/exam.ts` 增加：

```ts
export type ExamQuestionGroupDraftStatus = 'pending_review' | 'approved' | 'rejected'
export type QuestionGroupType = 'reading_passage' | 'math_problem' | 'single_question' | string

export interface QuestionGroupAsset {
  id: string
  tenant_id: number
  group_id: string
  asset_type: 'image' | 'audio' | 'table' | 'formula' | string
  storage_uri: string
  alt_text: string
  source_chunk_id: string
  bbox: Record<string, any>
  metadata: Record<string, any>
  sort_order: number
}

export interface QuestionGroup {
  id: string
  tenant_id: number
  space_id: string
  question_bank_id: string
  domain_id: string
  subject_id?: string
  group_type: QuestionGroupType
  title: string
  material_text: string
  material_format: string
  source_chunk_ids: string[]
  review_status: ReviewStatus
  status: string
  created_at: string
  updated_at: string
}

export interface QuestionGroupDetail {
  group: QuestionGroup
  assets: QuestionGroupAsset[]
  questions: QuestionDetail[]
}
```

增加题组草稿类型：

```ts
export interface ExamQuestionGroupDraftQuestion {
  question_no: string
  question_type_code: string
  stem: string
  options: ExamQuestionDraftOption[]
  answer: Record<string, any>
  explanation: string
  evidence?: Array<Record<string, any>>
  metadata?: Record<string, any>
  difficulty: string
  confidence: number
  order_in_group: number
  source_chunk_ids?: string[]
}

export interface ExamQuestionGroupDraft {
  id: string
  tenant_id: number
  space_id: string
  task_id: string
  material_id: string
  question_bank_id: string
  domain_id: string
  subject_id?: string
  group_type: QuestionGroupType
  title: string
  material_text: string
  material_format: string
  questions_json: ExamQuestionGroupDraftQuestion[]
  assets_json: Array<Record<string, any>>
  source_chunk_ids: string[]
  strategy_code: string
  confidence: number
  status: ExamQuestionGroupDraftStatus
  error_message: string
  approved_group_id: string
  created_at: string
  updated_at: string
}
```

- [ ] **步骤 2：创建正式题组 API**

创建 `frontend/src/api/exam/question-group.ts`：

```ts
import { get } from '@/utils/request'
import type { ApiResponse, QuestionGroupDetail } from '@/types/exam'

export function listQuestionGroupDetails(bankId: string) {
  return get(`/api/v1/exam/question-banks/${bankId}/question-groups`) as unknown as Promise<ApiResponse<QuestionGroupDetail[]>>
}
```

- [ ] **步骤 3：创建题组草稿 API**

创建 `frontend/src/api/exam/question-group-draft.ts`：

```ts
import { get, patch, post } from '@/utils/request'
import type { ApiResponse, ExamQuestionGroupDraft, ExamQuestionGroupDraftQuestion, ExamStructuringTask, QuestionGroupDetail } from '@/types/exam'

export interface ListQuestionGroupDraftsResult {
  task: ExamStructuringTask
  drafts: ExamQuestionGroupDraft[]
  stats: {
    total: number
    pending_review: number
    approved: number
    rejected: number
  }
}

const GROUP_EXTRACTION_TIMEOUT_MS = 10 * 60 * 1000

export function extractQuestionGroupDrafts(taskId: string, force = false) {
  return post(`/api/v1/exam/structuring-tasks/${taskId}/group-extract`, { force }, {
    timeout: GROUP_EXTRACTION_TIMEOUT_MS,
  }) as unknown as Promise<ApiResponse<ListQuestionGroupDraftsResult>>
}
```

继续增加：

```ts
export function listQuestionGroupDrafts(taskId: string) {
  return get(`/api/v1/exam/structuring-tasks/${taskId}/group-drafts`) as unknown as Promise<ApiResponse<ListQuestionGroupDraftsResult>>
}

export interface UpdateQuestionGroupDraftPayload {
  group_type: string
  title: string
  material_text: string
  material_format: string
  questions: ExamQuestionGroupDraftQuestion[]
  assets: Array<Record<string, any>>
  source_chunk_ids: string[]
}

export function updateQuestionGroupDraft(draftId: string, data: UpdateQuestionGroupDraftPayload) {
  return patch(`/api/v1/exam/question-group-drafts/${draftId}`, data) as unknown as Promise<ApiResponse<ExamQuestionGroupDraft>>
}

export function approveQuestionGroupDraft(draftId: string) {
  return post(`/api/v1/exam/question-group-drafts/${draftId}/approve`, {}) as unknown as Promise<ApiResponse<{ draft: ExamQuestionGroupDraft; group: QuestionGroupDetail }>>
}

export function rejectQuestionGroupDraft(draftId: string) {
  return post(`/api/v1/exam/question-group-drafts/${draftId}/reject`, {}) as unknown as Promise<ApiResponse<ExamQuestionGroupDraft>>
}
```

- [ ] **步骤 4：注册校对页面路由**

在 `frontend/src/router/index.ts` 增加：

```ts
{
  path: '/exam/question-group-drafts/:taskId',
  name: 'QuestionGroupDraftReview',
  component: () => import('@/views/question-draft/QuestionGroupDraftReview.vue'),
  meta: { requiresAuth: true },
}
```

- [ ] **步骤 5：班级详情页入口切到题组校对**

修改 `frontend/src/views/classes/ClassDetail.vue` 中结构化任务操作区：

```ts
const goQuestionGroupReview = (task: ExamStructuringTask) => {
  router.push({ name: 'QuestionGroupDraftReview', params: { taskId: task.id } })
}
```

按钮文案使用「题组校对」，保留旧「题目校对」入口作为调试入口或隐藏入口。新入口调用 `goQuestionGroupReview(task)`。

- [ ] **步骤 6：创建题组校对页骨架**

创建 `frontend/src/views/question-draft/QuestionGroupDraftReview.vue`：

```vue
<template>
  <main class="question-group-review">
    <header class="review-header">
      <div>
        <h1>题组校对</h1>
        <p>校对阅读篇章、数学题干、小题、答案和来源证据。</p>
      </div>
      <t-space>
        <t-button theme="primary" :loading="extracting" @click="handleExtract(false)">开始题组抽取</t-button>
        <t-button variant="outline" :loading="extracting" @click="handleExtract(true)">重新抽取</t-button>
      </t-space>
    </header>

    <section v-if="drafts.length" class="draft-list">
      <article v-for="draft in drafts" :key="draft.id" class="draft-panel">
        <header class="draft-panel__head">
          <div>
            <h2>{{ draft.title || draft.group_type }}</h2>
            <p>{{ draft.group_type }} · {{ draft.strategy_code || 'default' }}</p>
          </div>
          <t-tag>{{ draft.status }}</t-tag>
        </header>
        <t-textarea v-model="draft.material_text" autosize />
        <div v-for="question in draft.questions_json" :key="question.question_no" class="question-editor">
          <t-input v-model="question.question_no" placeholder="题号" />
          <t-textarea v-model="question.stem" autosize />
        </div>
        <t-space>
          <t-button theme="primary" @click="handleSave(draft)">保存</t-button>
          <t-button theme="success" @click="handleApprove(draft)">确认入库</t-button>
          <t-button theme="danger" variant="outline" @click="handleReject(draft)">驳回</t-button>
        </t-space>
      </article>
    </section>
    <t-empty v-else description="暂无题组草稿" />
  </main>
</template>
```

脚本区使用 `listQuestionGroupDrafts`、`extractQuestionGroupDrafts`、`updateQuestionGroupDraft`、`approveQuestionGroupDraft`、`rejectQuestionGroupDraft`，并在 `onMounted` 时加载。

- [ ] **步骤 7：运行前端类型检查**

```powershell
cd frontend
npm run type-check
```

预期：PASS。若项目没有 `type-check` 脚本，运行：

```powershell
cd frontend
npm run build
```

预期：exit 0。

- [ ] **步骤 8：提交**

```powershell
git add frontend/src/types/exam.ts frontend/src/api/exam/question-group.ts frontend/src/api/exam/question-group-draft.ts frontend/src/router/index.ts frontend/src/views/classes/ClassDetail.vue frontend/src/views/question-draft/QuestionGroupDraftReview.vue
git commit -m "feat(exam): 添加题组校对前端入口"
```

## 任务 7：题库详情页题组卡片展示

**文件：**
- 修改：`frontend/src/views/question-bank/QuestionBankDetail.vue`
- 修改：`frontend/src/api/exam/question-bank.ts`
- 测试：前端构建和浏览器 smoke

- [ ] **步骤 1：API 改为读取题组**

在 `frontend/src/api/exam/question-bank.ts` 中保留 `listQuestionDetails`，新增导出：

```ts
export { listQuestionGroupDetails } from './question-group'
```

- [ ] **步骤 2：题库详情页引入题组 API**

在 `QuestionBankDetail.vue` 中把：

```ts
import { getQuestionBank, listQuestionDetails } from '@/api/exam/question-bank'
```

改为：

```ts
import { getQuestionBank, listQuestionGroupDetails } from '@/api/exam/question-bank'
import type { QuestionGroupDetail } from '@/types/exam'
```

把状态：

```ts
const questions = ref<QuestionDetail[]>([])
```

改为：

```ts
const questionGroups = ref<QuestionGroupDetail[]>([])
```

- [ ] **步骤 3：改造加载逻辑**

把题目列表加载改成：

```ts
const groupRes = await listQuestionGroupDetails(bankId)
questionGroups.value = groupRes.data || []
```

- [ ] **步骤 4：替换表格为题组卡片**

替换原来的 `t-table` 区块：

```vue
<section class="question-section">
  <div class="section-title">
    <h2>题组</h2>
    <span>{{ questionGroups.length }} 组</span>
  </div>
  <div v-if="questionGroups.length" class="question-group-list">
    <article v-for="group in questionGroups" :key="group.group.id" class="question-group-card">
      <header class="question-group-card__head">
        <div>
          <h3>{{ group.group.title || group.group.group_type }}</h3>
          <p>{{ group.group.group_type }} · {{ group.questions.length }} 题</p>
        </div>
        <t-tag>{{ group.group.status }}</t-tag>
      </header>
      <p v-if="group.group.material_text" class="question-group-card__material">
        {{ group.group.material_text }}
      </p>
      <div v-if="group.assets?.length" class="question-group-assets">
        <div v-for="asset in group.assets" :key="asset.id" class="question-group-asset">
          {{ asset.asset_type }} · {{ asset.alt_text || asset.storage_uri }}
        </div>
      </div>
      <div class="question-items">
        <div v-for="item in group.questions" :key="item.question.id" class="question-item">
          <strong>{{ item.question.question_no || item.question.order_in_group }}</strong>
          <span>{{ item.question.stem }}</span>
          <small>答案：{{ item.answers?.map(answer => answer.answer_text).join('，') || '-' }}</small>
        </div>
      </div>
    </article>
  </div>
  <t-empty v-else description="暂无结构化题组" />
</section>
```

- [ ] **步骤 5：添加样式**

在 `<style scoped>` 增加：

```css
.question-group-list {
  display: grid;
  gap: 16px;
}

.question-group-card {
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  padding: 16px;
  background: var(--td-bg-color-container);
}

.question-group-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.question-group-card__head h3 {
  margin: 0;
  font-size: 16px;
}

.question-group-card__head p,
.question-group-card__material,
.question-item small {
  color: var(--td-text-color-secondary);
}

.question-group-card__material {
  display: -webkit-box;
  margin: 12px 0;
  overflow: hidden;
  -webkit-line-clamp: 8;
  -webkit-box-orient: vertical;
  white-space: pre-wrap;
}

.question-items {
  display: grid;
  gap: 10px;
}

.question-item {
  display: grid;
  grid-template-columns: 56px 1fr;
  gap: 8px 12px;
  padding: 10px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.question-item small {
  grid-column: 2;
}
```

- [ ] **步骤 6：前端构建验证**

```powershell
cd frontend
npm run build
```

预期：exit 0。

- [ ] **步骤 7：提交**

```powershell
git add frontend/src/views/question-bank/QuestionBankDetail.vue frontend/src/api/exam/question-bank.ts
git commit -m "feat(exam): 题库详情展示题组"
```

## 任务 8：端到端验证与修复

**文件：**
- 修改：实现过程中失败验证指向的具体文件
- 不提交：`docker-compose.yml` 的本地环境改动

- [ ] **步骤 1：运行后端考试模块测试**

```powershell
go test ./internal/application/service ./internal/application/repository ./internal/router ./internal/handler -run Exam -count=1
```

预期：exit 0。

- [ ] **步骤 2：运行前端构建**

```powershell
cd frontend
npm run build
```

预期：exit 0。

- [ ] **步骤 3：检查暂存和未暂存文件**

```powershell
git status --short
```

预期：实现相关文件可以出现；`docker-compose.yml` 仍保持未暂存，不进入提交。

- [ ] **步骤 4：启动服务 smoke**

使用项目既有启动方式启动后端和前端。若本地已经有服务运行，先确认端口和进程来源，不终止非当前任务启动的进程。

后端 smoke：

```powershell
Invoke-WebRequest -UseBasicParsing http://localhost:8080/api/v1/health
```

预期：HTTP 200，或项目当前健康检查路径返回可识别的成功响应。

前端 smoke：

```powershell
Invoke-WebRequest -UseBasicParsing http://localhost:5173
```

预期：HTTP 200，返回 HTML。

- [ ] **步骤 5：手动业务验证**

使用老师账号进入班级空间：

1. 选择已经解析完成的高考英语试卷资料。
2. 打开结构化任务并点击「开始题组抽取」。
3. 确认生成 `reading_passage` 题组草稿。
4. 检查 `material_text` 是否包含阅读原文。
5. 检查题组下是否有多道小题、选项、答案和解析。
6. 确认入库后进入题库详情页。
7. 确认题库详情页展示一张阅读题组卡片和多道小题。

- [ ] **步骤 6：提交收尾修复**

如果步骤 1 到 5 产生必要修复，只 stage 本计划列出的实现文件。不要 stage `docker-compose.yml`。

```powershell
git add migrations/versioned/000074_exam_question_groups.up.sql migrations/versioned/000074_exam_question_groups.down.sql internal/types/exam_question.go internal/types/exam_question_group_draft.go internal/types/interfaces/exam_question.go internal/types/interfaces/exam_question_group_draft.go internal/application/repository/exam_question.go internal/application/repository/exam_question_group_draft.go internal/application/repository/exam_question_group_test.go internal/application/service/exam_question_group_strategy.go internal/application/service/exam_question_group_extractor.go internal/application/service/exam_question_group_extractor_test.go internal/application/service/exam_question_group_draft.go internal/application/service/exam_question_group_build.go internal/application/service/exam_question_group_draft_test.go internal/application/service/exam_question.go internal/handler/exam_question_group_draft.go internal/handler/exam_question.go internal/router/exam.go internal/router/exam_rbac_routes_test.go internal/container/container.go frontend/src/types/exam.ts frontend/src/api/exam/question-group.ts frontend/src/api/exam/question-group-draft.ts frontend/src/api/exam/question-bank.ts frontend/src/router/index.ts frontend/src/views/classes/ClassDetail.vue frontend/src/views/question-draft/QuestionGroupDraftReview.vue frontend/src/views/question-bank/QuestionBankDetail.vue
git commit -m "fix(exam): 修复题组结构化集成问题"
```

- [ ] **步骤 7：推送分支**

```powershell
git push origin codex/exam-platform-phase1
```

预期：远端 `codex/exam-platform-phase1` 更新到本地最新提交。

## 执行顺序与检查点

1. 任务 1 和任务 2 完成后，数据库和正式题组读取能力可独立验证。
2. 任务 3 和任务 4 完成后，后端可以生成题组草稿并确认入库。
3. 任务 5 完成后，API 可以被前端调用。
4. 任务 6 和任务 7 完成后，老师校对和题库展示形成闭环。
5. 任务 8 是最终验收，不通过则回到对应任务修复。

每个任务都应独立提交。执行期间不要 stage 或提交本地 `docker-compose.yml` 改动。
