# 试卷抽题草稿校对实现计划

> **审计状态（2026-07-23）：** 实现与验证已完成；下方未勾选框是未回填的历史执行记录，不代表当前功能缺失。完成证据见 [Phase 1 收尾审计报告](../reports/2026-07-23-phase1-closeout-audit.md)。

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 打通从结构化任务发起 LLM 抽题、生成草稿、老师校对、确认写入正式题库的第一版真实链路。

**架构：** 新增 `ExamQuestionDraftService` 负责抽题草稿、校对和正式题目落库；抽题器复用 WeKnora 现有 `ModelService` 和 `chat.Chat` 非流式接口；正式入库复用已有 `questions`、`question_options`、`question_answers`、`question_explanations` 和 `question_chunk_refs` 表。

**技术栈：** Go、Gin、GORM、PostgreSQL migration、WeKnora chat model abstraction、Vue 3、TDesign。

---

## 文件结构

- 创建 `internal/types/exam_question_draft.go`：草稿状态、草稿实体、抽题请求、更新请求、草稿统计和 LLM 输出 DTO。
- 修改 `internal/types/exam_material.go`：扩展 `ExamStructuringTaskStatus`，增加 `extracting`、`reviewing`。
- 修改 `internal/types/exam_question.go`：补正式题目详情类型，包括选项、答案、解析、chunk 引用和聚合详情。
- 创建 `internal/types/interfaces/exam_question_draft.go`：草稿服务、仓储和抽题器接口。
- 修改 `internal/types/interfaces/exam_material.go`：增加结构化任务读取、更新和统计依赖。
- 修改 `internal/types/interfaces/exam_question.go`：增加正式题目详情写入和列表读取接口。
- 创建 `internal/application/repository/exam_question_draft.go`：草稿 GORM 仓储。
- 修改 `internal/application/repository/exam_material.go`：增加结构化任务读取、状态更新和计数。
- 修改 `internal/application/repository/exam_question.go`：增加事务写入正式题目详情和题库题目列表。
- 创建 `internal/application/service/exam_question_draft.go`：抽题、草稿列表、更新、确认和驳回业务逻辑。
- 创建 `internal/application/service/exam_question_extractor.go`：LLM 抽题器、JSON 提取和校验。
- 创建 `internal/application/service/exam_question_draft_test.go`：覆盖权限、抽题、非法 JSON、确认入库和幂等确认。
- 创建 `internal/handler/exam_question_draft.go`：抽题草稿 HTTP handler。
- 修改 `internal/router/exam.go`：注册抽题、草稿、确认和驳回路由。
- 修改 `internal/container/container.go`：接入仓储、抽题器、服务和 handler。
- 创建 `migrations/versioned/000073_exam_question_drafts.up.sql` 和 `.down.sql`。
- 创建 `frontend/src/api/exam/question-draft.ts`：前端草稿 API。
- 修改 `frontend/src/types/exam.ts`：增加草稿、正式题目详情和任务状态类型。
- 修改 `frontend/src/router/index.ts`：增加校对工作台路由。
- 修改 `frontend/src/views/classes/ClassDetail.vue`：结构化任务表增加开始抽题和查看校对入口。
- 创建 `frontend/src/views/question-draft/QuestionDraftReview.vue`：老师校对工作台。
- 修改 `frontend/src/views/question-bank/QuestionBankDetail.vue`：展示正式题目列表和详情。

## 任务 1：后端服务测试先行

**文件：**
- 创建：`internal/application/service/exam_question_draft_test.go`
- 参考：`internal/application/service/exam_material_test.go`

- [ ] **步骤 1：编写学生不能抽题的失败测试**

在 `internal/application/service/exam_question_draft_test.go` 创建测试文件，先定义可复用 stub。

```go
package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

type stubExamQuestionDraftRepo struct {
	task     *types.ExamStructuringTask
	material *types.ExamMaterial
	drafts   []*types.ExamQuestionDraft
}

type stubExamQuestionDraftSpace struct {
	canRead  bool
	canWrite bool
}

func TestExamQuestionDraftService_ExtractRequiresWritableSpace(t *testing.T) {
	ctx := context.Background()
	repo := &stubExamQuestionDraftRepo{
		task: &types.ExamStructuringTask{
			ID:             "task-1",
			TenantID:       10000,
			MaterialID:     "material-1",
			SpaceID:        "space-1",
			QuestionBankID: "bank-1",
			Status:         types.ExamStructuringTaskStatusReadyForReview,
		},
		material: &types.ExamMaterial{
			ID:           "material-1",
			TenantID:     10000,
			SpaceID:      "space-1",
			KnowledgeID:  "knowledge-1",
			DomainID:     "domain-1",
			IngestStatus: types.ExamMaterialIngestStatusCompleted,
		},
	}
	svc := newTestExamQuestionDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: false}, nil, nil)

	_, err := svc.ExtractDrafts(ctx, 10000, "student-1", "task-1", &types.ExtractExamQuestionDraftsRequest{})

	require.ErrorIs(t, err, ErrExamPermissionDenied)
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
go test ./internal/application/service -run ExamQuestionDraft -count=1
```

预期：编译失败，报 `undefined: types.ExamQuestionDraft` 或 `undefined: newTestExamQuestionDraftService`。

- [ ] **步骤 3：补抽题写入草稿的失败测试**

继续在同一测试文件添加：

```go
func TestExamQuestionDraftService_ExtractWritesDraftsFromModelJSON(t *testing.T) {
	ctx := context.Background()
	repo := newReadyDraftRepo()
	space := &stubExamQuestionDraftSpace{canRead: true, canWrite: true}
	chunks := &stubExamQuestionDraftChunkReader{chunks: []*types.Chunk{
		{ID: "chunk-1", KnowledgeID: "knowledge-1", Content: "21. What does the man suggest?\nA. Stay home\nB. Go out\n答案 A"},
	}}
	extractor := &stubExamQuestionExtractor{candidates: []*types.ExamQuestionDraftCandidate{{
		QuestionNo:       "21",
		QuestionTypeCode: "single_choice",
		Stem:             "What does the man suggest?",
		Options:          []types.ExamQuestionDraftOption{{Key: "A", Content: "Stay home"}, {Key: "B", Content: "Go out"}},
		Answer:           types.JSONMap{"value": "A"},
		Explanation:      "The answer is stated in the dialogue.",
		Confidence:       0.9,
		SourceChunkIDs:   []string{"chunk-1"},
	}}}
	svc := newTestExamQuestionDraftService(repo, space, chunks, extractor)

	result, err := svc.ExtractDrafts(ctx, 10000, "teacher-1", "task-1", &types.ExtractExamQuestionDraftsRequest{})

	require.NoError(t, err)
	require.Len(t, result.Drafts, 1)
	require.Equal(t, types.ExamStructuringTaskStatusReviewing, repo.updatedStatus)
	require.Equal(t, "21", result.Drafts[0].QuestionNo)
}
```

- [ ] **步骤 4：补非法 JSON 不写草稿的失败测试**

```go
func TestExamQuestionDraftService_ExtractInvalidModelOutputFailsTask(t *testing.T) {
	ctx := context.Background()
	repo := newReadyDraftRepo()
	space := &stubExamQuestionDraftSpace{canRead: true, canWrite: true}
	chunks := &stubExamQuestionDraftChunkReader{chunks: []*types.Chunk{{ID: "chunk-1", KnowledgeID: "knowledge-1", Content: "bad"}}}
	extractor := &stubExamQuestionExtractor{err: errors.New("invalid model json")}
	svc := newTestExamQuestionDraftService(repo, space, chunks, extractor)

	_, err := svc.ExtractDrafts(ctx, 10000, "teacher-1", "task-1", &types.ExtractExamQuestionDraftsRequest{})

	require.Error(t, err)
	require.Equal(t, types.ExamStructuringTaskStatusFailed, repo.updatedStatus)
	require.Empty(t, repo.createdDrafts)
}
```

- [ ] **步骤 5：补确认草稿写正式题目的失败测试**

```go
func TestExamQuestionDraftService_ApproveCreatesOfficialQuestion(t *testing.T) {
	ctx := context.Background()
	repo := newReadyDraftRepo()
	repo.drafts = []*types.ExamQuestionDraft{newPendingDraft("draft-1")}
	space := &stubExamQuestionDraftSpace{canRead: true, canWrite: true}
	questionRepo := &stubExamQuestionWriter{}
	svc := newTestExamQuestionDraftServiceWithQuestionRepo(repo, space, nil, nil, questionRepo)

	result, err := svc.ApproveDraft(ctx, 10000, "teacher-1", "draft-1")

	require.NoError(t, err)
	require.Equal(t, types.ExamQuestionDraftStatusApproved, result.Draft.Status)
	require.Equal(t, "question-1", result.Draft.ApprovedQuestionID)
	require.Len(t, questionRepo.created, 1)
	require.Len(t, questionRepo.created[0].Options, 2)
	require.Len(t, questionRepo.created[0].Answers, 1)
}
```

- [ ] **步骤 6：运行测试验证红灯**

运行：

```powershell
go test ./internal/application/service -run ExamQuestionDraft -count=1
```

预期：当前项目可能先被 Windows cgo 工具链阻塞；若进入本模块编译，则因类型和服务尚未存在而失败。

## 任务 2：类型、接口和数据库迁移

**文件：**
- 创建：`internal/types/exam_question_draft.go`
- 修改：`internal/types/exam_material.go`
- 修改：`internal/types/exam_question.go`
- 创建：`internal/types/interfaces/exam_question_draft.go`
- 修改：`internal/types/interfaces/exam_material.go`
- 修改：`internal/types/interfaces/exam_question.go`
- 创建：`migrations/versioned/000073_exam_question_drafts.up.sql`
- 创建：`migrations/versioned/000073_exam_question_drafts.down.sql`

- [ ] **步骤 1：新增草稿类型**

创建 `internal/types/exam_question_draft.go`：

```go
package types

import "time"

type ExamQuestionDraftStatus string

const (
	ExamQuestionDraftStatusPendingReview ExamQuestionDraftStatus = "pending_review"
	ExamQuestionDraftStatusApproved      ExamQuestionDraftStatus = "approved"
	ExamQuestionDraftStatusRejected      ExamQuestionDraftStatus = "rejected"
)

type ExamQuestionDraftOption struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

type ExamQuestionDraft struct {
	ID                 string                  `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID           uint64                  `json:"tenant_id" gorm:"not null;index"`
	SpaceID            string                  `json:"space_id" gorm:"type:varchar(36);not null;index"`
	TaskID             string                  `json:"task_id" gorm:"type:varchar(36);not null;index"`
	MaterialID         string                  `json:"material_id" gorm:"type:varchar(36);not null;index"`
	QuestionBankID     string                  `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	DomainID           string                  `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID          *string                 `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	SourceChunkIDs     JSON                    `json:"source_chunk_ids" gorm:"type:jsonb;not null"`
	QuestionNo         string                  `json:"question_no" gorm:"type:varchar(64);not null;default:''"`
	QuestionTypeCode   string                  `json:"question_type_code" gorm:"type:varchar(64);not null;default:''"`
	Stem               string                  `json:"stem" gorm:"type:text;not null"`
	OptionsJSON        JSON                    `json:"options_json" gorm:"type:jsonb;not null"`
	AnswerJSON         JSONMap                 `json:"answer_json" gorm:"type:jsonb;not null"`
	Explanation        string                  `json:"explanation" gorm:"type:text;not null;default:''"`
	Difficulty         string                  `json:"difficulty" gorm:"type:varchar(32);not null;default:'unknown'"`
	Confidence         float64                 `json:"confidence" gorm:"type:numeric(5,4);not null;default:0"`
	Status             ExamQuestionDraftStatus `json:"status" gorm:"type:varchar(32);not null;default:'pending_review'"`
	RawModelOutput     string                  `json:"raw_model_output" gorm:"type:text;not null;default:''"`
	ErrorMessage       string                  `json:"error_message" gorm:"type:text;not null;default:''"`
	ApprovedQuestionID string                  `json:"approved_question_id" gorm:"type:varchar(36);not null;default:''"`
	ReviewedByUserID   string                  `json:"reviewed_by_user_id" gorm:"type:varchar(36);not null;default:''"`
	ReviewedAt         *time.Time              `json:"reviewed_at,omitempty"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}

func (ExamQuestionDraft) TableName() string {
	return "exam_question_drafts"
}
```

- [ ] **步骤 2：新增请求和响应类型**

继续在 `internal/types/exam_question_draft.go` 添加：

```go
type ExtractExamQuestionDraftsRequest struct {
	Force bool `json:"force"`
}

type ExamQuestionDraftCandidate struct {
	QuestionNo       string                    `json:"question_no"`
	QuestionTypeCode string                    `json:"question_type_code"`
	Stem             string                    `json:"stem"`
	Options          []ExamQuestionDraftOption `json:"options"`
	Answer           JSONMap                   `json:"answer"`
	Explanation      string                    `json:"explanation"`
	Difficulty       string                    `json:"difficulty"`
	Confidence       float64                   `json:"confidence"`
	SourceChunkIDs   []string                  `json:"source_chunk_ids"`
	RawModelOutput   string                    `json:"raw_model_output"`
}

type ExamQuestionDraftExtractionResult struct {
	Task   *ExamStructuringTask  `json:"task"`
	Drafts []*ExamQuestionDraft  `json:"drafts"`
	Stats  ExamQuestionDraftStats `json:"stats"`
}

type ExamQuestionDraftStats struct {
	Total         int `json:"total"`
	PendingReview int `json:"pending_review"`
	Approved      int `json:"approved"`
	Rejected      int `json:"rejected"`
}

type ListExamQuestionDraftsResult struct {
	Task   *ExamStructuringTask  `json:"task"`
	Drafts []*ExamQuestionDraft  `json:"drafts"`
	Stats  ExamQuestionDraftStats `json:"stats"`
}

type UpdateExamQuestionDraftRequest struct {
	QuestionNo       string                    `json:"question_no" binding:"omitempty,max=64"`
	QuestionTypeCode string                    `json:"question_type_code" binding:"omitempty,max=64"`
	Stem             string                    `json:"stem" binding:"required"`
	Options          []ExamQuestionDraftOption `json:"options"`
	Answer           JSONMap                   `json:"answer"`
	Explanation      string                    `json:"explanation"`
	Difficulty       string                    `json:"difficulty"`
	SourceChunkIDs   []string                  `json:"source_chunk_ids"`
}

type ApproveExamQuestionDraftResult struct {
	Draft    *ExamQuestionDraft `json:"draft"`
	Question *QuestionDetail    `json:"question"`
}
```

- [ ] **步骤 3：扩展结构化任务状态**

修改 `internal/types/exam_material.go`：

```go
const (
	ExamStructuringTaskStatusPending        ExamStructuringTaskStatus = "pending"
	ExamStructuringTaskStatusReadyForReview ExamStructuringTaskStatus = "ready_for_review"
	ExamStructuringTaskStatusBlocked        ExamStructuringTaskStatus = "blocked"
	ExamStructuringTaskStatusExtracting     ExamStructuringTaskStatus = "extracting"
	ExamStructuringTaskStatusReviewing      ExamStructuringTaskStatus = "reviewing"
	ExamStructuringTaskStatusCompleted      ExamStructuringTaskStatus = "completed"
	ExamStructuringTaskStatusFailed         ExamStructuringTaskStatus = "failed"
)
```

- [ ] **步骤 4：补正式题目详情类型**

修改 `internal/types/exam_question.go`，在 `Question` 后添加：

```go
type QuestionOption struct {
	ID        string `json:"id" gorm:"type:varchar(36);primaryKey"`
	QuestionID string `json:"question_id" gorm:"type:varchar(36);not null;index"`
	OptionKey string `json:"option_key" gorm:"type:varchar(16);not null"`
	Content   string `json:"content" gorm:"type:text;not null"`
	SortOrder int    `json:"sort_order" gorm:"not null;default:0"`
}

func (QuestionOption) TableName() string { return "question_options" }

type QuestionAnswer struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	QuestionID string    `json:"question_id" gorm:"type:varchar(36);not null;index"`
	AnswerText string    `json:"answer_text" gorm:"type:text;not null"`
	IsCorrect  bool      `json:"is_correct" gorm:"not null;default:true"`
	CreatedAt  time.Time `json:"created_at"`
}

func (QuestionAnswer) TableName() string { return "question_answers" }

type QuestionExplanation struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	QuestionID      string    `json:"question_id" gorm:"type:varchar(36);not null;index"`
	ExplanationText string    `json:"explanation_text" gorm:"type:text;not null"`
	SourceType      string    `json:"source_type" gorm:"type:varchar(64);not null;default:'manual'"`
	CreatedAt       time.Time `json:"created_at"`
}

func (QuestionExplanation) TableName() string { return "question_explanations" }

type QuestionChunkRef struct {
	QuestionID string    `json:"question_id" gorm:"type:varchar(36);primaryKey"`
	ChunkID    string    `json:"chunk_id" gorm:"type:varchar(36);primaryKey"`
	RefType    string    `json:"ref_type" gorm:"type:varchar(64);primaryKey"`
	Confidence float64   `json:"confidence" gorm:"type:numeric(5,4);not null;default:1"`
	CreatedAt  time.Time `json:"created_at"`
}

func (QuestionChunkRef) TableName() string { return "question_chunk_refs" }

type QuestionDetail struct {
	Question     *Question              `json:"question"`
	Options      []*QuestionOption      `json:"options"`
	Answers      []*QuestionAnswer      `json:"answers"`
	Explanations []*QuestionExplanation `json:"explanations"`
	ChunkRefs    []*QuestionChunkRef    `json:"chunk_refs"`
}
```

- [ ] **步骤 5：新增接口**

创建 `internal/types/interfaces/exam_question_draft.go`：

```go
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
	GetDraftByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamQuestionDraft, error)
	ListDraftsByTask(ctx context.Context, tenantID uint64, taskID string) ([]*types.ExamQuestionDraft, error)
	CountDraftsByTask(ctx context.Context, tenantID uint64, taskID string) (types.ExamQuestionDraftStats, error)
	UpdateDraft(ctx context.Context, draft *types.ExamQuestionDraft) error
	MarkDraftApproved(ctx context.Context, draftID string, tenantID uint64, questionID string, reviewerID string) error
	MarkDraftRejected(ctx context.Context, draftID string, tenantID uint64, reviewerID string) error
	DeleteDraftsByTask(ctx context.Context, tenantID uint64, taskID string) error
}

type ExamQuestionExtractor interface {
	Extract(ctx context.Context, tenantID uint64, material *types.ExamMaterial, chunks []*types.Chunk) ([]*types.ExamQuestionDraftCandidate, error)
}
```

- [ ] **步骤 6：扩展既有接口**

修改 `internal/types/interfaces/exam_material.go`：

```go
type ExamMaterialRepository interface {
	UpsertMaterial(ctx context.Context, material *types.ExamMaterial) error
	GetMaterialByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamMaterial, error)
	GetMaterialByKnowledge(ctx context.Context, tenantID uint64, knowledgeID string) (*types.ExamMaterial, error)
	ListMaterials(ctx context.Context, tenantID uint64, filter types.ListExamMaterialsFilter, spaceIDs []string) ([]*types.ExamMaterial, error)
	CreateStructuringTask(ctx context.Context, task *types.ExamStructuringTask) error
	GetStructuringTaskByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamStructuringTask, error)
	UpdateStructuringTask(ctx context.Context, task *types.ExamStructuringTask) error
	ListStructuringTasks(ctx context.Context, tenantID uint64, filter types.ListExamStructuringTasksFilter, spaceIDs []string) ([]*types.ExamStructuringTask, error)
}
```

修改 `internal/types/interfaces/exam_question.go`：

```go
type ExamQuestionRepository interface {
	CreateQuestionBank(ctx context.Context, bank *types.QuestionBank) error
	ListQuestionBanks(ctx context.Context, tenantID uint64, spaceIDs []string) ([]*types.QuestionBank, error)
	GetQuestionBankByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.QuestionBank, error)
	CreateQuestionDetail(ctx context.Context, detail *types.QuestionDetail) error
	GetQuestionDetail(ctx context.Context, tenantID uint64, questionID string) (*types.QuestionDetail, error)
	ListQuestionDetailsByBank(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionDetail, error)
}
```

- [ ] **步骤 7：新增迁移**

创建 `migrations/versioned/000073_exam_question_drafts.up.sql`：

```sql
-- Migration: 000073_exam_question_drafts
-- Description: Store LLM-extracted exam question drafts before teacher review.

CREATE TABLE IF NOT EXISTS exam_question_drafts (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    task_id VARCHAR(36) NOT NULL REFERENCES exam_structuring_tasks(id) ON DELETE CASCADE,
    material_id VARCHAR(36) NOT NULL REFERENCES exam_materials(id),
    question_bank_id VARCHAR(36) NOT NULL REFERENCES question_banks(id),
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    subject_id VARCHAR(36) REFERENCES exam_subjects(id),
    source_chunk_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    question_no VARCHAR(64) NOT NULL DEFAULT '',
    question_type_code VARCHAR(64) NOT NULL DEFAULT '',
    stem TEXT NOT NULL,
    options_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    answer_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    explanation TEXT NOT NULL DEFAULT '',
    difficulty VARCHAR(32) NOT NULL DEFAULT 'unknown',
    confidence NUMERIC(5, 4) NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_review',
    raw_model_output TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    approved_question_id VARCHAR(36) NOT NULL DEFAULT '',
    reviewed_by_user_id VARCHAR(36) NOT NULL DEFAULT '',
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_question_drafts_task
    ON exam_question_drafts(tenant_id, task_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_question_drafts_space
    ON exam_question_drafts(tenant_id, space_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_question_drafts_bank
    ON exam_question_drafts(question_bank_id, status);

CREATE UNIQUE INDEX IF NOT EXISTS idx_exam_question_drafts_unique_question_no
    ON exam_question_drafts(tenant_id, task_id, question_no)
    WHERE question_no <> '';
```

创建 `migrations/versioned/000073_exam_question_drafts.down.sql`：

```sql
DROP INDEX IF EXISTS idx_exam_question_drafts_unique_question_no;
DROP INDEX IF EXISTS idx_exam_question_drafts_bank;
DROP INDEX IF EXISTS idx_exam_question_drafts_space;
DROP INDEX IF EXISTS idx_exam_question_drafts_task;
DROP TABLE IF EXISTS exam_question_drafts;
```

- [ ] **步骤 8：运行静态检查**

运行：

```powershell
Get-ChildItem migrations/versioned | Sort-Object Name | Select-Object -Last 6
gofmt -w internal/types/exam_question_draft.go internal/types/exam_material.go internal/types/exam_question.go internal/types/interfaces/exam_question_draft.go internal/types/interfaces/exam_material.go internal/types/interfaces/exam_question.go
git diff --check -- internal/types internal/types/interfaces migrations/versioned
```

预期：`000073` 为最新迁移；`git diff --check` 无输出。

## 任务 3：仓储实现

**文件：**
- 创建：`internal/application/repository/exam_question_draft.go`
- 修改：`internal/application/repository/exam_material.go`
- 修改：`internal/application/repository/exam_question.go`

- [ ] **步骤 1：实现草稿仓储**

创建 `internal/application/repository/exam_question_draft.go`：

```go
package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrExamQuestionDraftNotFound = errors.New("exam question draft not found")

type examQuestionDraftRepository struct {
	db *gorm.DB
}

func NewExamQuestionDraftRepository(db *gorm.DB) interfaces.ExamQuestionDraftRepository {
	return &examQuestionDraftRepository{db: db}
}

func (r *examQuestionDraftRepository) CreateDrafts(ctx context.Context, drafts []*types.ExamQuestionDraft) error {
	if len(drafts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&drafts).Error
}
```

- [ ] **步骤 2：补草稿读取和更新方法**

继续在 `exam_question_draft.go` 添加：

```go
func (r *examQuestionDraftRepository) GetDraftByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamQuestionDraft, error) {
	var draft types.ExamQuestionDraft
	err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&draft).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamQuestionDraftNotFound
		}
		return nil, err
	}
	return &draft, nil
}

func (r *examQuestionDraftRepository) ListDraftsByTask(ctx context.Context, tenantID uint64, taskID string) ([]*types.ExamQuestionDraft, error) {
	var drafts []*types.ExamQuestionDraft
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND task_id = ?", tenantID, taskID).
		Order("question_no ASC, created_at ASC").
		Find(&drafts).Error
	return drafts, err
}

func (r *examQuestionDraftRepository) UpdateDraft(ctx context.Context, draft *types.ExamQuestionDraft) error {
	return r.db.WithContext(ctx).Save(draft).Error
}

func (r *examQuestionDraftRepository) DeleteDraftsByTask(ctx context.Context, tenantID uint64, taskID string) error {
	return r.db.WithContext(ctx).Where("tenant_id = ? AND task_id = ?", tenantID, taskID).Delete(&types.ExamQuestionDraft{}).Error
}
```

- [ ] **步骤 3：补草稿统计和状态更新**

继续添加：

```go
func (r *examQuestionDraftRepository) CountDraftsByTask(ctx context.Context, tenantID uint64, taskID string) (types.ExamQuestionDraftStats, error) {
	drafts, err := r.ListDraftsByTask(ctx, tenantID, taskID)
	if err != nil {
		return types.ExamQuestionDraftStats{}, err
	}
	stats := types.ExamQuestionDraftStats{Total: len(drafts)}
	for _, draft := range drafts {
		switch draft.Status {
		case types.ExamQuestionDraftStatusPendingReview:
			stats.PendingReview++
		case types.ExamQuestionDraftStatusApproved:
			stats.Approved++
		case types.ExamQuestionDraftStatusRejected:
			stats.Rejected++
		}
	}
	return stats, nil
}

func (r *examQuestionDraftRepository) MarkDraftApproved(ctx context.Context, draftID string, tenantID uint64, questionID string, reviewerID string) error {
	return r.db.WithContext(ctx).
		Model(&types.ExamQuestionDraft{}).
		Where("id = ? AND tenant_id = ?", draftID, tenantID).
		Updates(map[string]any{
			"status":               types.ExamQuestionDraftStatusApproved,
			"approved_question_id": questionID,
			"reviewed_by_user_id":  reviewerID,
			"reviewed_at":          gorm.Expr("NOW()"),
			"updated_at":           gorm.Expr("NOW()"),
		}).Error
}

func (r *examQuestionDraftRepository) MarkDraftRejected(ctx context.Context, draftID string, tenantID uint64, reviewerID string) error {
	return r.db.WithContext(ctx).
		Model(&types.ExamQuestionDraft{}).
		Where("id = ? AND tenant_id = ?", draftID, tenantID).
		Updates(map[string]any{
			"status":              types.ExamQuestionDraftStatusRejected,
			"reviewed_by_user_id": reviewerID,
			"reviewed_at":         gorm.Expr("NOW()"),
			"updated_at":          gorm.Expr("NOW()"),
		}).Error
}
```

- [ ] **步骤 4：扩展结构化任务仓储**

修改 `internal/application/repository/exam_material.go`，新增：

```go
var ErrExamStructuringTaskNotFound = errors.New("exam structuring task not found")

func (r *examMaterialRepository) GetStructuringTaskByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamStructuringTask, error) {
	var task types.ExamStructuringTask
	err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamStructuringTaskNotFound
		}
		return nil, err
	}
	return &task, nil
}

func (r *examMaterialRepository) UpdateStructuringTask(ctx context.Context, task *types.ExamStructuringTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}
```

- [ ] **步骤 5：实现正式题目详情写入**

修改 `internal/application/repository/exam_question.go`，新增事务方法：

```go
func (r *examQuestionRepository) CreateQuestionDetail(ctx context.Context, detail *types.QuestionDetail) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(detail.Question).Error; err != nil {
			return err
		}
		if len(detail.Options) > 0 {
			if err := tx.Create(&detail.Options).Error; err != nil {
				return err
			}
		}
		if len(detail.Answers) > 0 {
			if err := tx.Create(&detail.Answers).Error; err != nil {
				return err
			}
		}
		if len(detail.Explanations) > 0 {
			if err := tx.Create(&detail.Explanations).Error; err != nil {
				return err
			}
		}
		if len(detail.ChunkRefs) > 0 {
			if err := tx.Create(&detail.ChunkRefs).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
```

- [ ] **步骤 6：实现正式题目详情读取**

继续在 `exam_question.go` 添加：

```go
func (r *examQuestionRepository) GetQuestionDetail(ctx context.Context, tenantID uint64, questionID string) (*types.QuestionDetail, error) {
	var question types.Question
	if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", questionID, tenantID).First(&question).Error; err != nil {
		return nil, err
	}
	detail := &types.QuestionDetail{Question: &question}
	r.db.WithContext(ctx).Where("question_id = ?", questionID).Order("sort_order ASC").Find(&detail.Options)
	r.db.WithContext(ctx).Where("question_id = ?", questionID).Find(&detail.Answers)
	r.db.WithContext(ctx).Where("question_id = ?", questionID).Find(&detail.Explanations)
	r.db.WithContext(ctx).Where("question_id = ?", questionID).Find(&detail.ChunkRefs)
	return detail, nil
}

func (r *examQuestionRepository) ListQuestionDetailsByBank(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionDetail, error) {
	var questions []*types.Question
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND question_bank_id = ? AND status <> ?", tenantID, bankID, "deleted").
		Order("created_at ASC").
		Find(&questions).Error; err != nil {
		return nil, err
	}
	out := make([]*types.QuestionDetail, 0, len(questions))
	for _, question := range questions {
		detail, err := r.GetQuestionDetail(ctx, tenantID, question.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, detail)
	}
	return out, nil
}
```

- [ ] **步骤 7：运行定向编译**

运行：

```powershell
gofmt -w internal/application/repository/exam_question_draft.go internal/application/repository/exam_material.go internal/application/repository/exam_question.go
go test ./internal/application/repository -run ExamQuestionDraft -count=1
```

预期：若没有仓储测试包，`go test` 输出 `?` 或编译到既有 cgo 阻塞；无本次新增语法错误。

## 任务 4：LLM 抽题器与草稿服务

**文件：**
- 创建：`internal/application/service/exam_question_extractor.go`
- 创建：`internal/application/service/exam_question_draft.go`
- 修改：`internal/container/container.go`

- [ ] **步骤 1：实现 JSON 抽题器骨架**

创建 `internal/application/service/exam_question_extractor.go`：

```go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type examQuestionExtractor struct {
	modelService interfaces.ModelService
}

func NewExamQuestionExtractor(modelService interfaces.ModelService) interfaces.ExamQuestionExtractor {
	return &examQuestionExtractor{modelService: modelService}
}

type examQuestionExtractionPayload struct {
	Questions []*types.ExamQuestionDraftCandidate `json:"questions"`
}
```

- [ ] **步骤 2：实现抽取入口**

继续添加：

```go
func (e *examQuestionExtractor) Extract(ctx context.Context, tenantID uint64, material *types.ExamMaterial, chunks []*types.Chunk) ([]*types.ExamQuestionDraftCandidate, error) {
	if material == nil || len(chunks) == 0 {
		return nil, ErrExamInvalidRequest
	}
	model, err := e.resolveKnowledgeQAModel(ctx)
	if err != nil {
		return nil, err
	}
	windows := buildExamQuestionChunkWindows(chunks, 6000)
	var out []*types.ExamQuestionDraftCandidate
	for _, window := range windows {
		candidates, err := e.extractWindow(ctx, model, material, window)
		if err != nil {
			return nil, err
		}
		out = append(out, candidates...)
	}
	return normalizeExamQuestionCandidates(out), nil
}

func (e *examQuestionExtractor) resolveKnowledgeQAModel(ctx context.Context) (chat.Chat, error) {
	models, err := e.modelService.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	for _, model := range models {
		if model != nil && model.Type == types.ModelTypeKnowledgeQA {
			return e.modelService.GetChatModel(ctx, model.ID)
		}
	}
	return nil, fmt.Errorf("no KnowledgeQA model configured")
}
```

- [ ] **步骤 3：实现模型调用和 JSON 解析**

继续添加：

```go
func (e *examQuestionExtractor) extractWindow(ctx context.Context, model chat.Chat, material *types.ExamMaterial, chunks []*types.Chunk) ([]*types.ExamQuestionDraftCandidate, error) {
	prompt := buildExamQuestionExtractionPrompt(material, chunks)
	resp, err := model.Chat(ctx, []chat.Message{
		{Role: "system", Content: "你是考试试卷结构化助手。只返回合法 JSON，不要返回 Markdown。"},
		{Role: "user", Content: prompt},
	}, &chat.ChatOptions{Temperature: 0.1, TopP: 0.2, MaxTokens: 4096})
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(resp.Content)
	payload, err := parseExamQuestionExtractionPayload(raw)
	if err != nil {
		return nil, err
	}
	for _, q := range payload.Questions {
		q.RawModelOutput = raw
	}
	return payload.Questions, nil
}

func parseExamQuestionExtractionPayload(raw string) (*examQuestionExtractionPayload, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	var payload examQuestionExtractionPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	if len(payload.Questions) == 0 {
		return nil, fmt.Errorf("model returned no questions")
	}
	return &payload, nil
}
```

- [ ] **步骤 4：实现窗口构造和提示词**

继续添加：

```go
func buildExamQuestionChunkWindows(chunks []*types.Chunk, maxChars int) [][]*types.Chunk {
	var windows [][]*types.Chunk
	var current []*types.Chunk
	size := 0
	for _, chunk := range chunks {
		if chunk == nil || strings.TrimSpace(chunk.Content) == "" {
			continue
		}
		if len(current) > 0 && size+len(chunk.Content) > maxChars {
			windows = append(windows, current)
			current = nil
			size = 0
		}
		current = append(current, chunk)
		size += len(chunk.Content)
	}
	if len(current) > 0 {
		windows = append(windows, current)
	}
	return windows
}

func buildExamQuestionExtractionPrompt(material *types.ExamMaterial, chunks []*types.Chunk) string {
	var b strings.Builder
	b.WriteString("请从下面试卷片段中抽取题目，返回 JSON：{\"questions\":[...]}\n")
	b.WriteString("字段必须包含 question_no、question_type_code、stem、options、answer、explanation、difficulty、confidence、source_chunk_ids。\n")
	b.WriteString("单选题 question_type_code 使用 single_choice；写作题使用 writing；无法判断时使用 unknown。\n")
	for _, chunk := range chunks {
		b.WriteString("\n[chunk_id=")
		b.WriteString(chunk.ID)
		b.WriteString("]\n")
		b.WriteString(chunk.Content)
		b.WriteString("\n")
	}
	return b.String()
}
```

- [ ] **步骤 5：实现候选校验**

继续添加：

```go
func normalizeExamQuestionCandidates(input []*types.ExamQuestionDraftCandidate) []*types.ExamQuestionDraftCandidate {
	seen := map[string]bool{}
	out := make([]*types.ExamQuestionDraftCandidate, 0, len(input))
	for _, q := range input {
		if q == nil {
			continue
		}
		q.QuestionNo = strings.TrimSpace(q.QuestionNo)
		q.QuestionTypeCode = strings.TrimSpace(q.QuestionTypeCode)
		q.Stem = strings.TrimSpace(q.Stem)
		if q.QuestionNo == "" || q.Stem == "" {
			continue
		}
		key := q.QuestionNo + "|" + q.Stem
		if seen[key] {
			continue
		}
		seen[key] = true
		if q.QuestionTypeCode == "" {
			q.QuestionTypeCode = "unknown"
		}
		if q.Difficulty == "" {
			q.Difficulty = "unknown"
		}
		if q.Confidence < 0 {
			q.Confidence = 0
		}
		if q.Confidence > 1 {
			q.Confidence = 1
		}
		out = append(out, q)
	}
	return out
}
```

- [ ] **步骤 6：实现草稿服务构造和抽题主流程**

创建 `internal/application/service/exam_question_draft.go`：

```go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

type examQuestionDraftService struct {
	draftRepo    interfaces.ExamQuestionDraftRepository
	materialRepo interfaces.ExamMaterialRepository
	questionRepo interfaces.ExamQuestionRepository
	spaceService interfaces.ExamSpaceService
	chunkReader  interfaces.ExamMaterialChunkReader
	extractor    interfaces.ExamQuestionExtractor
}

func NewExamQuestionDraftService(
	draftRepo interfaces.ExamQuestionDraftRepository,
	materialRepo interfaces.ExamMaterialRepository,
	questionRepo interfaces.ExamQuestionRepository,
	spaceService interfaces.ExamSpaceService,
	chunkReader interfaces.ExamMaterialChunkReader,
	extractor interfaces.ExamQuestionExtractor,
) interfaces.ExamQuestionDraftService {
	return &examQuestionDraftService{
		draftRepo: draftRepo, materialRepo: materialRepo, questionRepo: questionRepo,
		spaceService: spaceService, chunkReader: chunkReader, extractor: extractor,
	}
}
```

- [ ] **步骤 7：实现 ExtractDrafts**

继续添加：

```go
func (s *examQuestionDraftService) ExtractDrafts(ctx context.Context, tenantID uint64, userID string, taskID string, req *types.ExtractExamQuestionDraftsRequest) (*types.ExamQuestionDraftExtractionResult, error) {
	task, material, err := s.loadWritableTaskMaterial(ctx, tenantID, userID, taskID)
	if err != nil {
		return nil, err
	}
	if task.Status == types.ExamStructuringTaskStatusExtracting {
		return s.buildExtractionResult(ctx, tenantID, task)
	}
	if task.Status != types.ExamStructuringTaskStatusReadyForReview && task.Status != types.ExamStructuringTaskStatusReviewing {
		return nil, ErrExamInvalidRequest
	}
	if material.IngestStatus != types.ExamMaterialIngestStatusCompleted {
		return nil, ErrExamInvalidRequest
	}
	if req == nil {
		req = &types.ExtractExamQuestionDraftsRequest{}
	}
	existing, err := s.draftRepo.ListDraftsByTask(ctx, tenantID, task.ID)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 && !req.Force {
		return s.buildExtractionResult(ctx, tenantID, task)
	}
	if req.Force {
		if err := s.draftRepo.DeleteDraftsByTask(ctx, tenantID, task.ID); err != nil {
			return nil, err
		}
	}
	chunks, err := s.chunkReader.ListChunksByKnowledgeID(ctx, material.KnowledgeID)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, ErrExamInvalidRequest
	}
	task.Status = types.ExamStructuringTaskStatusExtracting
	task.ErrorMessage = ""
	task.UpdatedAt = time.Now()
	_ = s.materialRepo.UpdateStructuringTask(ctx, task)

	candidates, err := s.extractor.Extract(ctx, tenantID, material, chunks)
	if err != nil {
		task.Status = types.ExamStructuringTaskStatusFailed
		task.ErrorMessage = err.Error()
		task.UpdatedAt = time.Now()
		_ = s.materialRepo.UpdateStructuringTask(ctx, task)
		return nil, err
	}
	drafts, err := s.candidatesToDrafts(task, material, candidates)
	if err != nil {
		return nil, err
	}
	if len(drafts) == 0 {
		task.Status = types.ExamStructuringTaskStatusFailed
		task.ErrorMessage = "没有抽取到可校对题目"
		task.UpdatedAt = time.Now()
		_ = s.materialRepo.UpdateStructuringTask(ctx, task)
		return nil, errors.New(task.ErrorMessage)
	}
	if err := s.draftRepo.CreateDrafts(ctx, drafts); err != nil {
		return nil, err
	}
	task.Status = types.ExamStructuringTaskStatusReviewing
	task.StructuredQuestionCount = len(drafts)
	task.UpdatedAt = time.Now()
	if err := s.materialRepo.UpdateStructuringTask(ctx, task); err != nil {
		return nil, err
	}
	return s.buildExtractionResult(ctx, tenantID, task)
}
```

- [ ] **步骤 8：实现辅助方法**

继续添加：

```go
func (s *examQuestionDraftService) loadWritableTaskMaterial(ctx context.Context, tenantID uint64, userID string, taskID string) (*types.ExamStructuringTask, *types.ExamMaterial, error) {
	task, err := s.materialRepo.GetStructuringTaskByIDAndTenant(ctx, strings.TrimSpace(taskID), tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamStructuringTaskNotFound) {
			return nil, nil, ErrExamNotFound
		}
		return nil, nil, err
	}
	canWrite, err := s.spaceService.CanWriteSpace(ctx, tenantID, userID, task.SpaceID)
	if err != nil {
		return nil, nil, err
	}
	if !canWrite {
		return nil, nil, ErrExamPermissionDenied
	}
	material, err := s.materialRepo.GetMaterialByIDAndTenant(ctx, task.MaterialID, tenantID)
	if err != nil {
		return nil, nil, err
	}
	return task, material, nil
}

func (s *examQuestionDraftService) candidatesToDrafts(task *types.ExamStructuringTask, material *types.ExamMaterial, candidates []*types.ExamQuestionDraftCandidate) ([]*types.ExamQuestionDraft, error) {
	now := time.Now()
	out := make([]*types.ExamQuestionDraft, 0, len(candidates))
	for _, c := range candidates {
		optionsRaw, err := json.Marshal(c.Options)
		if err != nil {
			return nil, err
		}
		chunkRaw, err := json.Marshal(c.SourceChunkIDs)
		if err != nil {
			return nil, err
		}
		out = append(out, &types.ExamQuestionDraft{
			ID: uuid.New().String(), TenantID: task.TenantID, SpaceID: task.SpaceID,
			TaskID: task.ID, MaterialID: material.ID, QuestionBankID: task.QuestionBankID,
			DomainID: material.DomainID, SubjectID: material.SubjectID,
			SourceChunkIDs: types.JSON(chunkRaw), QuestionNo: c.QuestionNo,
			QuestionTypeCode: c.QuestionTypeCode, Stem: c.Stem,
			OptionsJSON: types.JSON(optionsRaw), AnswerJSON: c.Answer,
			Explanation: c.Explanation, Difficulty: c.Difficulty, Confidence: c.Confidence,
			Status: types.ExamQuestionDraftStatusPendingReview, RawModelOutput: c.RawModelOutput,
			CreatedAt: now, UpdatedAt: now,
		})
	}
	return out, nil
}
```

- [ ] **步骤 9：实现列表、更新、确认和驳回**

继续添加：

```go
func (s *examQuestionDraftService) ListDrafts(ctx context.Context, tenantID uint64, userID string, taskID string) (*types.ListExamQuestionDraftsResult, error) {
	task, err := s.materialRepo.GetStructuringTaskByIDAndTenant(ctx, taskID, tenantID)
	if err != nil {
		return nil, err
	}
	canWrite, err := s.spaceService.CanWriteSpace(ctx, tenantID, userID, task.SpaceID)
	if err != nil {
		return nil, err
	}
	if !canWrite {
		return nil, ErrExamPermissionDenied
	}
	return s.buildDraftListResult(ctx, tenantID, task)
}

func (s *examQuestionDraftService) UpdateDraft(ctx context.Context, tenantID uint64, userID string, draftID string, req *types.UpdateExamQuestionDraftRequest) (*types.ExamQuestionDraft, error) {
	draft, err := s.loadWritableDraft(ctx, tenantID, userID, draftID)
	if err != nil {
		return nil, err
	}
	if draft.Status != types.ExamQuestionDraftStatusPendingReview {
		return nil, ErrExamInvalidRequest
	}
	optionsRaw, _ := json.Marshal(req.Options)
	chunkRaw, _ := json.Marshal(req.SourceChunkIDs)
	draft.QuestionNo = strings.TrimSpace(req.QuestionNo)
	draft.QuestionTypeCode = strings.TrimSpace(req.QuestionTypeCode)
	draft.Stem = strings.TrimSpace(req.Stem)
	draft.OptionsJSON = types.JSON(optionsRaw)
	draft.AnswerJSON = req.Answer
	draft.Explanation = strings.TrimSpace(req.Explanation)
	draft.Difficulty = strings.TrimSpace(req.Difficulty)
	draft.SourceChunkIDs = types.JSON(chunkRaw)
	draft.UpdatedAt = time.Now()
	if err := s.draftRepo.UpdateDraft(ctx, draft); err != nil {
		return nil, err
	}
	return draft, nil
}
```

确认和驳回在同一文件添加，确认方法调用 `questionRepo.CreateQuestionDetail`，成功后调用 `draftRepo.MarkDraftApproved`。

- [ ] **步骤 10：运行服务测试**

运行：

```powershell
gofmt -w internal/application/service/exam_question_extractor.go internal/application/service/exam_question_draft.go internal/application/service/exam_question_draft_test.go
go test ./internal/application/service -run ExamQuestionDraft -count=1
```

预期：目标测试通过，或被既有 Windows cgo 工具链问题阻塞；若阻塞，记录完整错误。

## 任务 5：HTTP API、路由和 DI

**文件：**
- 创建：`internal/handler/exam_question_draft.go`
- 修改：`internal/router/exam.go`
- 修改：`internal/router/exam_rbac_routes_test.go`
- 修改：`internal/container/container.go`

- [ ] **步骤 1：实现 handler**

创建 `internal/handler/exam_question_draft.go`：

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

type ExamQuestionDraftHandler struct {
	service interfaces.ExamQuestionDraftService
}

func NewExamQuestionDraftHandler(service interfaces.ExamQuestionDraftService) *ExamQuestionDraftHandler {
	return &ExamQuestionDraftHandler{service: service}
}

func (h *ExamQuestionDraftHandler) ExtractDrafts(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	var req types.ExtractExamQuestionDraftsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	result, err := h.service.ExtractDrafts(ctx, tenantID, userID, c.Param("task_id"), &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to extract exam question drafts: %v", err)
		writeExamError(c, err, "Failed to extract exam question drafts")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
```

- [ ] **步骤 2：补 handler 其余方法**

在同一文件添加 `ListDrafts`、`UpdateDraft`、`ApproveDraft`、`RejectDraft`，路径参数分别读取 `task_id` 和 `draft_id`，成功时返回 `{"success": true, "data": ...}`。

- [ ] **步骤 3：注册路由**

修改 `internal/router/exam.go` 的 `RegisterExamRoutes` 签名，增加 `questionDraftHandler *handler.ExamQuestionDraftHandler`。在 `exam` group 内增加：

```go
exam.POST("/structuring-tasks/:task_id/extract", g.Contributor(), questionDraftHandler.ExtractDrafts)
exam.GET("/structuring-tasks/:task_id/drafts", g.Contributor(), questionDraftHandler.ListDrafts)
exam.PATCH("/question-drafts/:draft_id", g.Contributor(), questionDraftHandler.UpdateDraft)
exam.POST("/question-drafts/:draft_id/approve", g.Contributor(), questionDraftHandler.ApproveDraft)
exam.POST("/question-drafts/:draft_id/reject", g.Contributor(), questionDraftHandler.RejectDraft)
```

- [ ] **步骤 4：更新路由权限测试**

修改 `internal/router/exam_rbac_routes_test.go`，补断言：

```go
assertRouteGuard(t, routes, "POST", "/api/v1/exam/structuring-tasks/:task_id/extract", "contributor")
assertRouteGuard(t, routes, "GET", "/api/v1/exam/structuring-tasks/:task_id/drafts", "contributor")
assertRouteGuard(t, routes, "PATCH", "/api/v1/exam/question-drafts/:draft_id", "contributor")
assertRouteGuard(t, routes, "POST", "/api/v1/exam/question-drafts/:draft_id/approve", "contributor")
assertRouteGuard(t, routes, "POST", "/api/v1/exam/question-drafts/:draft_id/reject", "contributor")
```

- [ ] **步骤 5：更新 DI 容器**

修改 `internal/container/container.go`，在考试模块附近增加：

```go
must(container.Provide(repository.NewExamQuestionDraftRepository))
must(container.Provide(service.NewExamQuestionExtractor))
must(container.Provide(service.NewExamQuestionDraftService))
must(container.Provide(handler.NewExamQuestionDraftHandler))
```

并更新 `router.RegisterExamRoutes` 调用参数。

- [ ] **步骤 6：运行路由测试**

运行：

```powershell
gofmt -w internal/handler/exam_question_draft.go internal/router/exam.go internal/router/exam_rbac_routes_test.go internal/container/container.go
go test ./internal/router -run Exam -count=1
```

预期：路由测试通过，或被既有 cgo 工具链问题阻塞；无本次新增语法错误。

## 任务 6：题库正式题目查询接口

**文件：**
- 修改：`internal/types/interfaces/exam_question.go`
- 修改：`internal/application/service/exam_question.go`
- 修改：`internal/handler/exam_question.go`
- 修改：`internal/router/exam.go`

- [ ] **步骤 1：扩展题库服务接口**

在 `interfaces.ExamQuestionService` 中增加：

```go
ListQuestionDetails(ctx context.Context, tenantID uint64, userID string, bankID string) ([]*types.QuestionDetail, error)
GetQuestionDetail(ctx context.Context, tenantID uint64, userID string, questionID string) (*types.QuestionDetail, error)
```

- [ ] **步骤 2：实现服务权限校验**

在 `internal/application/service/exam_question.go` 添加：

```go
func (s *examQuestionService) ListQuestionDetails(ctx context.Context, tenantID uint64, userID string, bankID string) ([]*types.QuestionDetail, error) {
	bank, err := s.GetQuestionBank(ctx, tenantID, userID, bankID)
	if err != nil {
		return nil, err
	}
	return s.questionRepo.ListQuestionDetailsByBank(ctx, tenantID, bank.ID)
}
```

- [ ] **步骤 3：新增 handler 方法**

在 `internal/handler/exam_question.go` 添加：

```go
func (h *ExamQuestionHandler) ListQuestionDetails(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	items, err := h.questionService.ListQuestionDetails(ctx, tenantID, userID, c.Param("bank_id"))
	if err != nil {
		writeExamError(c, err, "Failed to list questions")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}
```

- [ ] **步骤 4：注册题目列表路由**

在 `internal/router/exam.go` 增加：

```go
exam.GET("/question-banks/:bank_id/questions", g.Viewer(), questionHandler.ListQuestionDetails)
```

- [ ] **步骤 5：运行定向测试**

运行：

```powershell
gofmt -w internal/types/interfaces/exam_question.go internal/application/service/exam_question.go internal/handler/exam_question.go internal/router/exam.go
go test ./internal/application/service -run ExamQuestion -count=1
```

预期：目标包编译通过，或被既有 cgo 工具链问题阻塞。

## 任务 7：前端 API、类型和路由

**文件：**
- 创建：`frontend/src/api/exam/question-draft.ts`
- 修改：`frontend/src/api/exam/question-bank.ts`
- 修改：`frontend/src/types/exam.ts`
- 修改：`frontend/src/router/index.ts`

- [ ] **步骤 1：扩展前端类型**

修改 `frontend/src/types/exam.ts`：

```ts
export type ExamStructuringTaskStatus = 'pending' | 'ready_for_review' | 'blocked' | 'extracting' | 'reviewing' | 'completed' | 'failed'
export type ExamQuestionDraftStatus = 'pending_review' | 'approved' | 'rejected'

export interface ExamQuestionDraftOption {
  key: string
  content: string
}

export interface ExamQuestionDraft {
  id: string
  tenant_id: number
  space_id: string
  task_id: string
  material_id: string
  question_bank_id: string
  domain_id: string
  subject_id?: string
  source_chunk_ids: string[]
  question_no: string
  question_type_code: string
  stem: string
  options_json: ExamQuestionDraftOption[]
  answer_json: Record<string, any>
  explanation: string
  difficulty: string
  confidence: number
  status: ExamQuestionDraftStatus
  error_message: string
  approved_question_id: string
  created_at: string
  updated_at: string
}
```

- [ ] **步骤 2：补正式题目详情类型**

继续添加：

```ts
export interface QuestionDetail {
  question: {
    id: string
    question_bank_id: string
    stem: string
    difficulty: string
    status: string
  }
  options: Array<{ id: string; option_key: string; content: string; sort_order: number }>
  answers: Array<{ id: string; answer_text: string; is_correct: boolean }>
  explanations: Array<{ id: string; explanation_text: string; source_type: string }>
  chunk_refs: Array<{ question_id: string; chunk_id: string; ref_type: string; confidence: number }>
}
```

- [ ] **步骤 3：新增草稿 API**

创建 `frontend/src/api/exam/question-draft.ts`：

```ts
import { get, patch, post } from '@/utils/request'
import type { ApiResponse, ExamQuestionDraft, ExamStructuringTask, QuestionDetail } from '@/types/exam'

export interface ExamQuestionDraftStats {
  total: number
  pending_review: number
  approved: number
  rejected: number
}

export interface ListDraftsResult {
  task: ExamStructuringTask
  drafts: ExamQuestionDraft[]
  stats: ExamQuestionDraftStats
}

export function extractQuestionDrafts(taskId: string, force = false) {
  return post(`/api/v1/exam/structuring-tasks/${taskId}/extract`, { force }) as unknown as Promise<ApiResponse<ListDraftsResult>>
}

export function listQuestionDrafts(taskId: string) {
  return get(`/api/v1/exam/structuring-tasks/${taskId}/drafts`) as unknown as Promise<ApiResponse<ListDraftsResult>>
}

export function updateQuestionDraft(draftId: string, data: Partial<ExamQuestionDraft>) {
  return patch(`/api/v1/exam/question-drafts/${draftId}`, data) as unknown as Promise<ApiResponse<ExamQuestionDraft>>
}

export function approveQuestionDraft(draftId: string) {
  return post(`/api/v1/exam/question-drafts/${draftId}/approve`, {}) as unknown as Promise<ApiResponse<{ draft: ExamQuestionDraft; question: QuestionDetail }>>
}

export function rejectQuestionDraft(draftId: string) {
  return post(`/api/v1/exam/question-drafts/${draftId}/reject`, {}) as unknown as Promise<ApiResponse<ExamQuestionDraft>>
}
```

- [ ] **步骤 4：扩展题库 API**

修改 `frontend/src/api/exam/question-bank.ts`：

```ts
import type { ApiResponse, QuestionBank, QuestionDetail } from '@/types/exam'

export function listQuestionDetails(bankId: string) {
  return get(`/api/v1/exam/question-banks/${bankId}/questions`) as unknown as Promise<ApiResponse<QuestionDetail[]>>
}
```

- [ ] **步骤 5：新增前端路由**

修改 `frontend/src/router/index.ts`，在 `question-banks/:bankId` 后增加：

```ts
{
  path: "structuring-tasks/:taskId/review",
  name: "questionDraftReview",
  component: () => import("../views/question-draft/QuestionDraftReview.vue"),
  meta: { requiresInit: true, requiresAuth: true, minRole: 'contributor' as RoleKey }
},
```

- [ ] **步骤 6：运行前端类型检查构建**

运行：

```powershell
npm --prefix frontend run build
```

预期：若页面尚未创建，构建失败并提示缺少 `QuestionDraftReview.vue`；任务 8 创建页面后通过。

## 任务 8：班级入口和校对工作台

**文件：**
- 修改：`frontend/src/views/classes/ClassDetail.vue`
- 创建：`frontend/src/views/question-draft/QuestionDraftReview.vue`

- [ ] **步骤 1：班级任务表增加操作**

在 `ClassDetail.vue` 的结构化任务表 actions slot 中加入：

```vue
<template #actions="{ row }">
  <t-space size="small">
    <t-button
      v-if="canManageResources && row.status === 'ready_for_review'"
      size="small"
      theme="primary"
      :loading="extractingTaskId === row.id"
      @click="extractTask(row.id)"
    >
      开始抽题
    </t-button>
    <t-button
      v-if="canManageResources && ['reviewing', 'completed'].includes(row.status)"
      size="small"
      variant="outline"
      @click="router.push(`/platform/structuring-tasks/${row.id}/review`)"
    >
      查看校对
    </t-button>
  </t-space>
</template>
```

- [ ] **步骤 2：班级页接入抽题 API**

在 `<script setup>` 引入并实现：

```ts
import { extractQuestionDrafts } from '@/api/exam/question-draft'

const extractingTaskId = ref('')

const extractTask = async (taskId: string) => {
  extractingTaskId.value = taskId
  try {
    await extractQuestionDrafts(taskId)
    MessagePlugin.success('抽题完成，已生成待校对草稿')
    await loadResourceTab()
    router.push(`/platform/structuring-tasks/${taskId}/review`)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '抽题失败')
  } finally {
    extractingTaskId.value = ''
  }
}
```

- [ ] **步骤 3：创建校对工作台模板**

创建 `frontend/src/views/question-draft/QuestionDraftReview.vue`：

```vue
<template>
  <div class="exam-page">
    <div class="exam-header">
      <div>
        <t-button variant="text" size="small" @click="router.back()">
          <template #icon><t-icon name="chevron-left" /></template>
          返回
        </t-button>
        <h2>题目校对</h2>
        <p>校对 LLM 抽取的题目草稿，确认后写入正式题库。</p>
      </div>
      <t-tag v-if="data" variant="light">{{ data.stats.approved }} / {{ data.stats.total }} 已确认</t-tag>
    </div>

    <t-loading :loading="loading">
      <div class="review-layout">
        <aside class="draft-list">
          <button
            v-for="draft in drafts"
            :key="draft.id"
            class="draft-item"
            :class="{ active: draft.id === selectedDraft?.id }"
            @click="selectDraft(draft)"
          >
            <strong>{{ draft.question_no || '未编号' }}</strong>
            <span>{{ draft.question_type_code || 'unknown' }}</span>
          </button>
        </aside>
        <section v-if="selectedDraft" class="editor-panel">
          <t-form :data="form" label-align="top">
            <t-form-item label="题号"><t-input v-model="form.question_no" /></t-form-item>
            <t-form-item label="题型"><t-input v-model="form.question_type_code" /></t-form-item>
            <t-form-item label="题干"><t-textarea v-model="form.stem" :autosize="{ minRows: 5 }" /></t-form-item>
            <t-form-item label="答案 JSON"><t-textarea v-model="answerText" :autosize="{ minRows: 3 }" /></t-form-item>
            <t-form-item label="解析"><t-textarea v-model="form.explanation" :autosize="{ minRows: 4 }" /></t-form-item>
          </t-form>
          <t-space>
            <t-button theme="primary" :loading="saving" @click="saveDraft">保存</t-button>
            <t-button theme="success" :loading="approving" @click="approveDraft">确认入库</t-button>
            <t-button theme="danger" variant="outline" :loading="rejecting" @click="rejectDraft">驳回</t-button>
          </t-space>
        </section>
      </div>
    </t-loading>
  </div>
</template>
```

- [ ] **步骤 4：实现校对页脚本**

在同文件 `<script setup lang="ts">` 中实现加载、选择、保存、确认和驳回。保存时将 `answerText` 解析为 JSON，解析失败时用 `MessagePlugin.error('答案 JSON 格式不正确')`。

- [ ] **步骤 5：实现样式**

使用现有 `exam-page` 风格，增加：

```less
.review-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 16px;
}

.draft-list,
.editor-panel {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}
```

- [ ] **步骤 6：运行前端构建**

运行：

```powershell
npm --prefix frontend run build
```

预期：构建成功；若 TypeScript 报字段类型不匹配，按 `frontend/src/types/exam.ts` 中定义的类型修正。

## 任务 9：题库详情页展示正式题目

**文件：**
- 修改：`frontend/src/views/question-bank/QuestionBankDetail.vue`
- 修改：`frontend/src/api/exam/question-bank.ts`

- [ ] **步骤 1：加载题目详情**

在 `QuestionBankDetail.vue` 中引入 `listQuestionDetails`，新增：

```ts
const questions = ref<QuestionDetail[]>([])

const loadQuestions = async (bankId: string) => {
  const res = await listQuestionDetails(bankId)
  questions.value = res.data || []
}
```

并在 `loadData` 成功获取题库后调用 `await loadQuestions(bankId)`。

- [ ] **步骤 2：替换空题目列表**

将现有 `t-empty` 替换为 `t-table`：

```vue
<t-table
  row-key="question.id"
  :data="questions"
  :columns="questionColumns"
  :pagination="{ pageSize: 10, total: questions.length }"
  size="small"
>
  <template #stem="{ row }">
    <div class="question-stem-cell">{{ row.question.stem }}</div>
  </template>
  <template #answer="{ row }">
    {{ row.answers?.map((item) => item.answer_text).join('，') || '-' }}
  </template>
</t-table>
<t-empty v-if="!questions.length && !loading" description="暂无结构化题目" />
```

- [ ] **步骤 3：增加列定义**

在脚本中添加：

```ts
const questionColumns = [
  { colKey: 'stem', title: '题干', minWidth: 320 },
  { colKey: 'answer', title: '答案', width: 160 },
  { colKey: 'question.difficulty', title: '难度', width: 100 },
  { colKey: 'question.status', title: '状态', width: 100 },
]
```

- [ ] **步骤 4：运行构建**

运行：

```powershell
npm --prefix frontend run build
```

预期：构建成功。

## 任务 10：最终验证、提交和推送

**文件：**
- 所有本次新增和修改文件。

- [ ] **步骤 1：运行后端目标测试**

运行：

```powershell
go test ./internal/application/service -run ExamQuestionDraft -count=1
go test ./internal/router -run Exam -count=1
```

预期：若 Windows cgo 工具链仍阻塞，记录完整错误；若工具链可用，目标测试通过。

- [ ] **步骤 2：运行前端构建**

运行：

```powershell
npm --prefix frontend run build
```

预期：构建成功。

- [ ] **步骤 3：运行 diff 检查**

运行：

```powershell
git diff --check
git status --short
```

预期：`git diff --check` 无输出；`git status --short` 中 `docker-compose.yml` 仍为本地未提交改动，本次提交不包含它。

- [ ] **步骤 4：分批提交**

建议提交顺序：

```powershell
git add internal/types internal/types/interfaces migrations/versioned
git commit -m "feat(exam): 添加试卷抽题草稿模型"

git add internal/application/repository internal/application/service internal/handler internal/router internal/container
git commit -m "feat(exam): 接入试卷抽题校对接口"

git add frontend/src/api/exam frontend/src/types/exam.ts frontend/src/router/index.ts frontend/src/views/classes/ClassDetail.vue frontend/src/views/question-draft frontend/src/views/question-bank/QuestionBankDetail.vue
git commit -m "feat(exam): 添加老师题目校对工作台"
```

- [ ] **步骤 5：推送当前分支**

运行：

```powershell
git push origin codex/exam-platform-phase1
```

预期：推送成功。若远端有新提交，先 `git pull --rebase` 并处理冲突，不使用破坏性命令。

## 计划自检

- 规格中的数据模型由任务 2 覆盖。
- 规格中的抽题流程由任务 4 覆盖。
- 规格中的后端 API 由任务 5 和任务 6 覆盖。
- 规格中的前端入口、校对工作台和题库详情由任务 7、任务 8、任务 9 覆盖。
- 规格中的权限模型由任务 1、任务 4、任务 5、任务 6 覆盖。
- 规格中的错误处理由任务 1、任务 4 覆盖。
- 计划没有保留悬空项、未定义类型或跨任务命名不一致。
