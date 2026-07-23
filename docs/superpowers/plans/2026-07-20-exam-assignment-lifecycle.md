# 班级作业运营闭环实现计划

> **审计状态（2026-07-23）：** 实现与验证已完成；下方未勾选框是未回填的历史执行记录，不代表当前功能缺失。完成证据见 [Phase 1 收尾审计报告](../reports/2026-07-23-phase1-closeout-audit.md)。

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development 或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法跟踪进度。

**目标：** 为班级练习任务补齐编辑、软撤回、重新发布、截止控制和继续已有 attempt 的运营闭环。

**架构：** 在现有 assignment 模块内增加 `withdrawn` 状态和原子状态转换，Repository 显式接收状态集合，Service 统一权限与截止规则。前端复用班级详情和学习中心入口，并用纯函数共享任务状态、按钮能力和继续 attempt 规则。

**技术栈：** Go、Gin、GORM、PostgreSQL、Vue 3、TDesign Vue Next、TypeScript、Node test、Docker Compose。

**设计规格：** `docs/superpowers/specs/2026-07-20-exam-assignment-lifecycle-design.md`

---

## 文件结构

- 修改 `internal/types/exam_assignment.go`：增加 `withdrawn` 状态和完整元数据更新请求。
- 修改 `internal/types/interfaces/exam_assignment.go`：增加更新、撤回、重发方法及显式状态过滤 Repository 契约。
- 修改 `internal/application/repository/exam_assignment.go`：实现按状态列表、条件元数据更新和原子状态转换。
- 修改 `internal/application/repository/exam_assignment_test.go`：覆盖状态过滤和条件更新冲突。
- 修改 `internal/application/service/exam_space.go`：增加统一的 `ErrExamStateConflict`。
- 修改 `internal/application/service/exam_assignment.go`：实现权限、截止、状态机和 attempt 创建约束。
- 修改 `internal/application/service/exam_assignment_test.go`：覆盖完整生命周期与权限矩阵。
- 修改 `internal/application/service/exam_analytics.go`、`internal/application/service/exam_analytics_test.go`：显式只统计 `published`。
- 修改 `internal/application/service/exam_intervention_test.go`：同步 Repository 状态过滤签名，保持推荐模块测试替身可编译。
- 修改 `internal/handler/exam_assignment.go`、`internal/handler/exam_error.go`：增加三个接口和 `409` 映射。
- 创建 `internal/handler/exam_error_test.go`：验证作业状态冲突映射为 HTTP 409 AppError。
- 修改 `internal/router/exam.go`、`internal/router/exam_rbac_routes_test.go`：注册更新、撤回、重发路由。
- 修改 `frontend/src/types/exam.ts`：增加 `withdrawn`。
- 修改 `frontend/src/api/exam/assignment.ts`：增加更新、撤回、重发 API。
- 创建 `frontend/src/views/classes/assignmentLifecycle.ts`：共享状态、能力和继续 attempt 规则。
- 创建 `frontend/src/views/classes/assignmentLifecycle.test.ts`：纯函数单元测试。
- 修改 `frontend/src/views/classes/ClassDetail.vue`：教师任务管理与学生截止状态。
- 修改 `frontend/src/views/classes/classAssignmentSource.test.ts`：教师管理 UI source 测试。
- 修改 `frontend/src/views/learning/LearningHome.vue`：截止状态和继续已有 attempt。
- 修改 `frontend/src/views/learning/learningAssignmentSource.test.ts`：学生入口 source 测试。

## 任务 1：持久化契约与原子状态转换

**文件：**

- 修改：`internal/types/exam_assignment.go`
- 修改：`internal/types/interfaces/exam_assignment.go`
- 修改：`internal/application/repository/exam_assignment.go`
- 测试：`internal/application/repository/exam_assignment_test.go`

- [ ] **步骤 1：编写 Repository 红灯测试**

在 `exam_assignment_test.go` 增加 SQLite 测试，固定验证状态过滤、元数据条件更新和旧状态冲突：

```go
func TestExamAssignmentRepositoryFiltersLifecycleStatuses(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:exam-assignment-lifecycle?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ExamClassAssignment{}))
	repo := &examAssignmentRepository{db: db}
	ctx := context.Background()
	now := time.Now().UTC()
	require.NoError(t, db.Create(testExamAssignment("published-1", "group-1", types.ExamAssignmentStatusPublished, now)).Error)
	require.NoError(t, db.Create(testExamAssignment("withdrawn-1", "group-2", types.ExamAssignmentStatusWithdrawn, now.Add(time.Minute))).Error)
	require.NoError(t, db.Create(testExamAssignment("archived-1", "group-3", types.ExamAssignmentStatusArchived, now.Add(2*time.Minute))).Error)

	items, err := repo.ListAssignmentsByClass(ctx, 10000, "class-1", []types.ExamAssignmentStatus{
		types.ExamAssignmentStatusPublished,
		types.ExamAssignmentStatusWithdrawn,
	}, 50)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "withdrawn-1", items[0].ID)
	require.Equal(t, "published-1", items[1].ID)
}

func TestExamAssignmentRepositoryTransitionRequiresExpectedStatus(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:exam-assignment-transition?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ExamClassAssignment{}))
	repo := &examAssignmentRepository{db: db}
	ctx := context.Background()
	require.NoError(t, db.Create(testExamAssignment("assignment-1", "group-1", types.ExamAssignmentStatusPublished, time.Now().UTC())).Error)

	err = repo.TransitionAssignmentStatus(ctx, 10000, "class-1", "assignment-1",
		types.ExamAssignmentStatusWithdrawn, types.ExamAssignmentStatusPublished, time.Now().UTC())
	require.ErrorIs(t, err, ErrExamClassAssignmentStateConflict)
}
```

- [ ] **步骤 2：运行红灯测试**

运行：

```powershell
go test ./internal/application/repository -run "TestExamAssignmentRepository(FiltersLifecycleStatuses|TransitionRequiresExpectedStatus|UpdatesMetadata)" -count=1
```

预期：FAIL，缺少 `ExamAssignmentStatusWithdrawn`、新 Repository 方法或新签名。

- [ ] **步骤 3：定义状态、请求与 Repository 契约**

在 `internal/types/exam_assignment.go` 增加：

```go
const ExamAssignmentStatusWithdrawn ExamAssignmentStatus = "withdrawn"

type UpdateExamAssignmentRequest struct {
	Title        string     `json:"title" binding:"required,max=255"`
	Instructions string     `json:"instructions" binding:"omitempty,max=2000"`
	DueAt        *time.Time `json:"due_at"`
}
```

将 Repository 接口调整为：

```go
ListAssignmentsByClass(ctx context.Context, tenantID uint64, classID string, statuses []types.ExamAssignmentStatus, limit int) ([]*types.ExamClassAssignment, error)
UpdateAssignmentMetadata(ctx context.Context, tenantID uint64, classID, assignmentID string, allowed []types.ExamAssignmentStatus, title, instructions string, dueAt *time.Time, updatedAt time.Time) error
TransitionAssignmentStatus(ctx context.Context, tenantID uint64, classID, assignmentID string, expected, next types.ExamAssignmentStatus, updatedAt time.Time) error
```

- [ ] **步骤 4：实现条件更新**

在 Repository 中使用 `WHERE tenant_id = ? AND class_id = ? AND id = ? AND status IN ?` 更新元数据，状态转换额外使用 `status = expected`。两个更新都检查 `RowsAffected == 1`，否则返回：

```go
var ErrExamClassAssignmentStateConflict = errors.New("exam class assignment state conflict")
```

列表方法在状态集合为空时直接返回空列表；非空时使用 `status IN ?`，保留 `created_at DESC` 和现有限制规则。

- [ ] **步骤 5：运行 Repository 测试与格式化**

```powershell
gofmt -w internal/types/exam_assignment.go internal/types/interfaces/exam_assignment.go internal/application/repository/exam_assignment.go internal/application/repository/exam_assignment_test.go
go test ./internal/application/repository -run TestExamAssignmentRepository -count=1
```

预期：相关测试全部 PASS。

- [ ] **步骤 6：提交持久化层**

```powershell
git add -- internal/types/exam_assignment.go internal/types/interfaces/exam_assignment.go internal/application/repository/exam_assignment.go internal/application/repository/exam_assignment_test.go
git commit -m "feat(exam): 增加作业生命周期持久化"
```

## 任务 2：Service 状态机、权限与截止规则

**文件：**

- 修改：`internal/application/service/exam_space.go`
- 修改：`internal/application/service/exam_assignment.go`
- 修改：`internal/application/service/exam_assignment_test.go`
- 修改：`internal/application/service/exam_analytics.go`
- 测试：`internal/application/service/exam_analytics_test.go`
- 测试：`internal/application/service/exam_intervention_test.go`

- [ ] **步骤 1：编写生命周期红灯测试**

在 service 测试中加入固定时钟，并覆盖编辑、撤回、重发、截止、角色列表和 attempt 数量：

```go
type assignmentLifecycleFixture struct {
	svc            *examAssignmentService
	assignmentRepo *stubExamAssignmentRepo
	practiceRepo   *stubPracticeRepo
}

func newAssignmentLifecycleService(t *testing.T, now time.Time) *assignmentLifecycleFixture {
	t.Helper()
	classRepo := newFakeExamClassRepo()
	assignmentRepo := newStubExamAssignmentRepo(classRepo)
	class := seedExamClass(classRepo, "class-1", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, class.ID, 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	group := newPracticeGroupDetail()
	group.Group.SpaceID = class.SpaceID
	questionRepo := &stubQuestionGroupWriter{created: []*types.QuestionGroupDetail{group}}
	practiceRepo := newStubPracticeRepo()
	svc := NewExamAssignmentService(assignmentRepo, classRepo, questionRepo, practiceRepo, newStubExamAssignmentSpaceService()).(*examAssignmentService)
	svc.now = func() time.Time { return now }
	due := now.Add(24 * time.Hour)
	assignmentRepo.assignments = append(assignmentRepo.assignments, &types.ExamClassAssignment{
		ID: "assignment-1", TenantID: 10000, ClassID: class.ID, SpaceID: class.SpaceID,
		QuestionBankID: group.Group.QuestionBankID, GroupID: group.Group.ID, Title: "练习",
		Status: types.ExamAssignmentStatusPublished, DueAt: &due, CreatedByUserID: "teacher-1",
		CreatedAt: now, UpdatedAt: now,
	})
	return &assignmentLifecycleFixture{svc: svc, assignmentRepo: assignmentRepo, practiceRepo: practiceRepo}
}

func TestExamAssignmentLifecycleWithdrawEditRepublish(t *testing.T) {
	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	fixture := newAssignmentLifecycleService(t, now)
	svc := fixture.svc
	ctx := context.Background()

	withdrawn, err := svc.WithdrawAssignment(ctx, 10000, "teacher-1", "class-1", "assignment-1")
	require.NoError(t, err)
	require.Equal(t, types.ExamAssignmentStatusWithdrawn, withdrawn.Assignment.Status)

	due := now.Add(24 * time.Hour)
	updated, err := svc.UpdateAssignment(ctx, 10000, "teacher-1", "class-1", "assignment-1", &types.UpdateExamAssignmentRequest{
		Title: "延期练习", Instructions: "按时完成", DueAt: &due,
	})
	require.NoError(t, err)
	require.Equal(t, "延期练习", updated.Assignment.Title)

	republished, err := svc.RepublishAssignment(ctx, 10000, "teacher-1", "class-1", "assignment-1")
	require.NoError(t, err)
	require.Equal(t, types.ExamAssignmentStatusPublished, republished.Assignment.Status)
}

func TestExamAssignmentAttemptRejectsExpiredWithoutCreatingRecord(t *testing.T) {
	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	fixture := newAssignmentLifecycleService(t, now)
	due := now.Add(-time.Minute)
	fixture.assignmentRepo.assignments[0].DueAt = &due

	_, err := fixture.svc.CreateAssignmentAttempt(context.Background(), 10000, "student-1", "assignment-1")
	require.ErrorIs(t, err, ErrExamStateConflict)
	require.Len(t, fixture.practiceRepo.attempts, 0)
}
```

再增加学生写操作返回 `ErrExamPermissionDenied`、pending 成员无读取权限、重复撤回/重发返回冲突、withdrawn 可读进度、教师列表包含 withdrawn、学生列表不含 withdrawn 的表驱动用例。

- [ ] **步骤 2：运行 Service 红灯测试**

```powershell
go test ./internal/application/service -run "TestExamAssignment(Lifecycle|Update|Withdraw|Republish|AttemptRejects|List|Progress)" -count=1
```

预期：FAIL，缺少 lifecycle service 方法和冲突错误。

- [ ] **步骤 3：增加冲突错误和可测试时钟**

在 `exam_space.go` 的统一错误变量中增加：

```go
ErrExamStateConflict = errors.New("exam state conflict")
```

在 `examAssignmentService` 增加 `now func() time.Time`，构造函数设置为 `time.Now`。测试辅助函数覆盖为固定 UTC 时间，创建、更新、转换和截止校验都调用 `s.now()`。

- [ ] **步骤 4：实现权限和状态规则**

在 Service interface 和实现中增加：

```go
UpdateAssignment(ctx context.Context, tenantID uint64, userID, classID, assignmentID string, req *types.UpdateExamAssignmentRequest) (*types.ExamAssignmentSummary, error)
WithdrawAssignment(ctx context.Context, tenantID uint64, userID, classID, assignmentID string) (*types.ExamAssignmentSummary, error)
RepublishAssignment(ctx context.Context, tenantID uint64, userID, classID, assignmentID string) (*types.ExamAssignmentSummary, error)
```

实现统一 helper：

```go
func assignmentExpired(assignment *types.ExamClassAssignment, now time.Time) bool {
	return assignment != nil && assignment.DueAt != nil && !assignment.DueAt.After(now)
}
```

逐项落实：active teacher/assistant 才能写；未截止 published 和 withdrawn 可编辑；published 可撤回；withdrawn 仅在截止为空或未来时可重发；repository 状态冲突统一映射 `ErrExamStateConflict`；`ensureActiveClassMember` 明确检查 `member.Status == active`。

- [ ] **步骤 5：调整列表、进度、分析与 attempt**

班级列表中 teacher/assistant 传入 `[published, withdrawn]`，student 传入 `[published]`；个人列表保持 published。进度允许 published/withdrawn。创建 attempt 在写入前执行 published 与截止校验。

将 analytics 调用改为显式传入：

```go
[]types.ExamAssignmentStatus{types.ExamAssignmentStatusPublished}
```

并增加 withdrawn 不进入 `AssignmentCount` 的回归测试。

同步调整 `exam_assignment_test.go` 和 `exam_intervention_test.go` 内的 Repository stub：`ListAssignmentsByClass` 增加 `statuses []types.ExamAssignmentStatus` 参数，assignment stub 还要实现元数据更新与状态转换，确保构造函数继续接收完整接口。

- [ ] **步骤 6：运行 Service 与 analytics 测试**

```powershell
gofmt -w internal/application/service/exam_space.go internal/application/service/exam_assignment.go internal/application/service/exam_assignment_test.go internal/application/service/exam_analytics.go internal/application/service/exam_analytics_test.go internal/application/service/exam_intervention_test.go
go test ./internal/application/service -run "TestExamAssignment|TestExamClassAnalytics|TestExamIntervention" -count=1
```

预期：全部 PASS，且过期/撤回路径没有新增 attempt。

- [ ] **步骤 7：提交业务层**

```powershell
git add -- internal/application/service/exam_space.go internal/application/service/exam_assignment.go internal/application/service/exam_assignment_test.go internal/application/service/exam_analytics.go internal/application/service/exam_analytics_test.go internal/application/service/exam_intervention_test.go internal/types/interfaces/exam_assignment.go
git commit -m "feat(exam): 实现作业软撤回状态机"
```

## 任务 3：HTTP 接口与错误契约

**文件：**

- 修改：`internal/handler/exam_assignment.go`
- 修改：`internal/handler/exam_error.go`
- 创建：`internal/handler/exam_error_test.go`
- 修改：`internal/router/exam.go`
- 测试：`internal/router/exam_rbac_routes_test.go`

- [ ] **步骤 1：扩充路由红灯测试**

在 assignment route matrix 中加入精确源码断言：

```go
`exam.PUT("/classes/:class_id/assignments/:assignment_id", g.Viewer(), assignmentHandler.UpdateAssignment)`,
`exam.POST("/classes/:class_id/assignments/:assignment_id/withdraw", g.Viewer(), assignmentHandler.WithdrawAssignment)`,
`exam.POST("/classes/:class_id/assignments/:assignment_id/republish", g.Viewer(), assignmentHandler.RepublishAssignment)`,
```

同时创建 `exam_error_test.go`：

```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestWriteExamErrorMapsStateConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	writeExamError(ctx, service.ErrExamStateConflict, "fallback")

	require.Len(t, ctx.Errors, 1)
	appErr, ok := ctx.Errors[0].Err.(*apperrors.AppError)
	require.True(t, ok)
	require.Equal(t, apperrors.ErrConflict, appErr.Code)
	require.Equal(t, http.StatusConflict, appErr.HTTPCode)
}
```

- [ ] **步骤 2：运行路由红灯测试**

```powershell
go test ./internal/router -run TestExamAssignmentRouteGuardSourceMatrix -count=1
go test ./internal/handler -run TestWriteExamErrorMapsStateConflict -count=1
```

预期：第一条 FAIL，因为三条新路由尚不存在；第二条 FAIL，因为冲突仍被映射为内部错误。

- [ ] **步骤 3：实现 Handler 和冲突映射**

`UpdateAssignment` 使用 `ShouldBindJSON` 读取 `UpdateExamAssignmentRequest`；三个 handler 都从 context 读取 tenant/user，并传递两个 path ID。成功返回：

```go
c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
```

在 `writeExamError` 中将 `ErrExamStateConflict` 映射为：

```go
c.Error(apperrors.NewConflictError("Assignment state conflict").WithDetails(err.Error()))
```

- [ ] **步骤 4：注册路由并运行 HTTP 层测试**

```powershell
gofmt -w internal/handler/exam_assignment.go internal/handler/exam_error.go internal/handler/exam_error_test.go internal/router/exam.go internal/router/exam_rbac_routes_test.go
go test ./internal/handler ./internal/router -run "ExamAssignment|ExamError" -count=1
```

预期：相关测试 PASS。

- [ ] **步骤 5：提交 HTTP 层**

```powershell
git add -- internal/handler/exam_assignment.go internal/handler/exam_error.go internal/handler/exam_error_test.go internal/router/exam.go internal/router/exam_rbac_routes_test.go
git commit -m "feat(exam): 开放作业生命周期接口"
```

## 任务 4：前端生命周期纯函数与 API

**文件：**

- 修改：`frontend/src/types/exam.ts`
- 修改：`frontend/src/api/exam/assignment.ts`
- 创建：`frontend/src/views/classes/assignmentLifecycle.ts`
- 创建：`frontend/src/views/classes/assignmentLifecycle.test.ts`

- [ ] **步骤 1：编写前端红灯测试**

测试固定当前时间并覆盖 active、expired、withdrawn、继续 attempt：

```ts
import assert from 'node:assert/strict'
import test from 'node:test'
import type { ExamAssignmentSummary, ExamClassAssignment, ExamPracticeAttempt } from '../../types/exam'
import { assignmentLifecycleState, canCreateAssignmentAttempt, existingAssignmentAttemptID } from './assignmentLifecycle'

const makeAssignment = (overrides: Partial<ExamClassAssignment> = {}): ExamClassAssignment => ({
  id: 'assignment-1', tenant_id: 10000, class_id: 'class-1', space_id: 'space-1',
  question_bank_id: 'bank-1', group_id: 'group-1', title: '练习', instructions: '',
  status: 'published', created_by_user_id: 'teacher-1', created_at: '2026-07-19T08:00:00Z',
  updated_at: '2026-07-19T08:00:00Z', ...overrides,
})

const makeSummary = (overrides: Partial<ExamAssignmentSummary> = {}): ExamAssignmentSummary => ({
  assignment: makeAssignment(), bank_name: '题库', question_count: 1, ...overrides,
})

test('derives assignment lifecycle without persisting expired status', () => {
  const now = Date.parse('2026-07-20T08:00:00Z')
  assert.equal(assignmentLifecycleState(makeAssignment({ due_at: '2026-07-20T09:00:00Z' }), now), 'active')
  assert.equal(assignmentLifecycleState(makeAssignment({ due_at: '2026-07-20T08:00:00Z' }), now), 'expired')
  assert.equal(assignmentLifecycleState(makeAssignment({ status: 'withdrawn' }), now), 'withdrawn')
})

test('continues the latest attempt without creating another one', () => {
  const item = makeSummary({ last_attempt: { id: 'attempt-1', status: 'in_progress' } as ExamPracticeAttempt })
  assert.equal(existingAssignmentAttemptID(item), 'attempt-1')
  assert.equal(canCreateAssignmentAttempt(item, Date.now()), false)
})
```

API source 测试断言 `put`、`withdraw` 和 `republish` 的路径及方法。

- [ ] **步骤 2：运行前端红灯测试**

```powershell
node --test src/views/classes/assignmentLifecycle.test.ts src/views/classes/classAssignmentSource.test.ts
```

预期：FAIL，生命周期 helper 和 API 尚未定义。

- [ ] **步骤 3：实现类型、API 和纯函数**

前端状态类型改为：

```ts
export type ExamAssignmentStatus = 'published' | 'withdrawn' | 'archived'
```

API 增加 `UpdateClassAssignmentPayload` 和：

```ts
export interface UpdateClassAssignmentPayload {
  title: string
  instructions: string
  due_at: string | null
}

export function updateClassAssignment(classId: string, assignmentId: string, data: UpdateClassAssignmentPayload) {
  return put(`/api/v1/exam/classes/${classId}/assignments/${assignmentId}`, data)
}

export function withdrawClassAssignment(classId: string, assignmentId: string) {
  return post(`/api/v1/exam/classes/${classId}/assignments/${assignmentId}/withdraw`, {})
}

export function republishClassAssignment(classId: string, assignmentId: string) {
  return post(`/api/v1/exam/classes/${classId}/assignments/${assignmentId}/republish`, {})
}
```

纯函数导出 `assignmentLifecycleState`、`assignmentStatusLabel`、`canEditAssignment`、`canWithdrawAssignment`、`canRepublishAssignment`、`existingAssignmentAttemptID` 和 `canCreateAssignmentAttempt`。所有截止比较接收可注入的 `nowMs`。

核心实现固定为：

```ts
import type { ExamAssignmentSummary, ExamClassAssignment } from '@/types/exam'

export type AssignmentLifecycleState = 'active' | 'expired' | 'withdrawn' | 'archived'

export function assignmentLifecycleState(assignment: ExamClassAssignment, nowMs = Date.now()): AssignmentLifecycleState {
  if (assignment.status === 'withdrawn') return 'withdrawn'
  if (assignment.status === 'archived') return 'archived'
  if (assignment.due_at && Date.parse(assignment.due_at) <= nowMs) return 'expired'
  return 'active'
}

export function assignmentStatusLabel(assignment: ExamClassAssignment, nowMs = Date.now()) {
  const state = assignmentLifecycleState(assignment, nowMs)
  return ({ active: '进行中', expired: '已截止', withdrawn: '已撤回', archived: '已归档' } as const)[state]
}

export function canEditAssignment(assignment: ExamClassAssignment, nowMs = Date.now()) {
  const state = assignmentLifecycleState(assignment, nowMs)
  return state === 'active' || state === 'withdrawn'
}

export function canWithdrawAssignment(assignment: ExamClassAssignment) {
  return assignment.status === 'published'
}

export function canRepublishAssignment(assignment: ExamClassAssignment, nowMs = Date.now()) {
  return assignment.status === 'withdrawn' && (!assignment.due_at || Date.parse(assignment.due_at) > nowMs)
}

export function existingAssignmentAttemptID(item: ExamAssignmentSummary) {
  return item.last_attempt?.id || ''
}

export function canCreateAssignmentAttempt(item: ExamAssignmentSummary, nowMs = Date.now()) {
  return !existingAssignmentAttemptID(item) && assignmentLifecycleState(item.assignment, nowMs) === 'active'
}
```

- [ ] **步骤 4：运行前端单元测试**

```powershell
node --test src/views/classes/assignmentLifecycle.test.ts src/views/classes/classAssignmentSource.test.ts
```

预期：全部 PASS。

- [ ] **步骤 5：提交前端契约**

```powershell
git add -- frontend/src/types/exam.ts frontend/src/api/exam/assignment.ts frontend/src/views/classes/assignmentLifecycle.ts frontend/src/views/classes/assignmentLifecycle.test.ts frontend/src/views/classes/classAssignmentSource.test.ts
git commit -m "feat(exam): 增加作业生命周期前端契约"
```

## 任务 5：教师管理与学生任务入口

**文件：**

- 修改：`frontend/src/views/classes/ClassDetail.vue`
- 修改：`frontend/src/views/classes/classAssignmentSource.test.ts`
- 修改：`frontend/src/views/learning/LearningHome.vue`
- 修改：`frontend/src/views/learning/learningAssignmentSource.test.ts`

- [ ] **步骤 1：编写 UI source 红灯测试**

班级详情断言存在编辑、撤回、重新发布调用和确认文案；学习中心断言已有 attempt 分支先于创建调用：

```ts
assert.match(classDetail, /updateClassAssignment/)
assert.match(classDetail, /withdrawClassAssignment/)
assert.match(classDetail, /republishClassAssignment/)
assert.match(classDetail, /学生入口将隐藏，历史答题不会删除/)
assert.match(learningHome, /existingAssignmentAttemptID/)
assert.match(learningHome, /canCreateAssignmentAttempt/)
assert.match(learningHome, /已截止/)
```

- [ ] **步骤 2：运行 UI 红灯测试**

```powershell
node --test src/views/classes/classAssignmentSource.test.ts src/views/learning/learningAssignmentSource.test.ts
```

预期：FAIL，页面尚未接入生命周期行为。

- [ ] **步骤 3：实现教师任务卡操作**

在 `ClassDetail.vue`：

- 使用状态 helper 显示 `进行中/已截止/已撤回`。
- create/edit 共用表单，edit 模式题组控件禁用并提交完整 title/instructions/due_at。
- 未截止 published 显示编辑、查看结果、撤回。
- 已截止 published 显示查看结果、撤回。
- withdrawn 显示编辑、查看结果、重新发布。
- 使用 `DialogPlugin.confirm`，danger confirm 文案为“确认撤回”。
- 所有 lifecycle 请求成功后关闭弹窗并 `await loadAssignments()`；`409` 时先显示错误再刷新。

- [ ] **步骤 4：实现学生继续与截止按钮**

在 `ClassDetail.vue` 和 `LearningHome.vue` 使用同一 helper：

```ts
const existingAttemptId = existingAssignmentAttemptID(item)
if (existingAttemptId) {
  router.push(`/platform/practice/question-groups/${item.assignment.group_id}?attempt_id=${existingAttemptId}`)
  return
}
if (!canCreateAssignmentAttempt(item)) return
```

已截止且无 attempt 时按钮内容为“已截止”且 `disabled`；已有 attempt 时保持“继续任务”。撤回任务不需要前端过滤，学生 API 不返回该状态。

- [ ] **步骤 5：补移动端约束并运行定向测试**

任务卡 action 区在 `max-width: 720px` 下允许换行，按钮不使用固定文字宽度，卡片保持 `min-width: 0` 和 `overflow-wrap: anywhere`。

```powershell
node --test src/views/classes/assignmentLifecycle.test.ts src/views/classes/classAssignmentSource.test.ts src/views/learning/learningAssignmentSource.test.ts
```

预期：全部 PASS。

- [ ] **步骤 6：提交页面行为**

```powershell
git add -- frontend/src/views/classes/ClassDetail.vue frontend/src/views/classes/classAssignmentSource.test.ts frontend/src/views/learning/LearningHome.vue frontend/src/views/learning/learningAssignmentSource.test.ts
git commit -m "feat(exam): 完成作业编辑撤回与重发"
```

## 任务 6：组合验证、真实运行验收与收尾

**文件：**

- 检查：本计划列出的全部修改文件
- 不提交：`docker-compose.yml`

- [ ] **步骤 1：运行后端组合测试**

```powershell
go test ./internal/application/repository ./internal/application/service ./internal/handler ./internal/router -run "ExamAssignment|ExamClassAnalytics|ExamError" -count=1
go test ./internal/application/repository ./internal/application/service ./internal/handler ./internal/router -count=1
go vet ./internal/application/repository ./internal/application/service ./internal/handler ./internal/router
```

预期：命令 exit `0`，无 FAIL 和 vet error。

- [ ] **步骤 2：运行前端全量测试与生产构建**

```powershell
Set-Location frontend
npm test
npm run build-only
Set-Location ..
```

预期：测试 `fail 0`，Vite 输出 `built in` 并 exit `0`。

- [ ] **步骤 3：重建真实服务**

```powershell
docker compose build app frontend
docker compose up -d --no-deps app frontend
docker compose ps
docker compose logs app --tail 120
```

预期：app 为 healthy、frontend 为 running，迁移版本保持 `82` 且 `dirty: false`；本功能没有新增 migration。

- [ ] **步骤 4：执行 API 与浏览器 smoke**

使用本机已登录的教师浏览器会话，按顺序验证：

1. 打开班级详情“练习任务”，编辑未截止任务，网络请求 `PUT .../assignments/:id` 返回 `200`。
2. 撤回同一任务，`POST .../withdraw` 返回 `200`，页面显示“已撤回”，历史进度仍可打开。
3. 编辑撤回任务的截止时间并重发，`POST .../republish` 返回 `200`，assignment ID 保持不变。
4. 调用学生任务列表确认撤回期间任务不出现；重发后重新出现。
5. 在已有 attempt 的任务点击“继续”，确认没有新的 `POST .../attempts`，URL 使用原 attempt ID。
6. 在 355px 移动视口确认任务卡无横向滚动，编辑、撤回、重发操作可触达。

记录 API status、`consoleErrors`、页面 `scrollWidth/clientWidth` 和桌面/移动截图；不得在日志中输出 token 或密码。

- [ ] **步骤 5：运行完成门禁**

```powershell
git diff --check
git status --short
git log -7 --oneline
```

预期：`git diff --check` 无输出；`git status --short` 只保留用户的 `docker-compose.yml` 本机改动，不存在临时 smoke 脚本或截图。

- [ ] **步骤 6：执行代码审查并修复发现**

使用 `superpowers-zh:requesting-code-review` 对照设计规格检查状态机、权限、截止边界、attempt 不变性和移动端布局。对必须修复项先补红灯测试，再修复并重复步骤 1、2、4、5。

- [ ] **步骤 7：提交整合性修复**

仅在步骤 6 产生修复时执行：

```powershell
git add -- internal/types/exam_assignment.go internal/types/interfaces/exam_assignment.go internal/application/repository/exam_assignment.go internal/application/repository/exam_assignment_test.go internal/application/service/exam_space.go internal/application/service/exam_assignment.go internal/application/service/exam_assignment_test.go internal/application/service/exam_analytics.go internal/application/service/exam_analytics_test.go internal/application/service/exam_intervention_test.go internal/handler/exam_assignment.go internal/handler/exam_error.go internal/handler/exam_error_test.go internal/router/exam.go internal/router/exam_rbac_routes_test.go frontend/src/types/exam.ts frontend/src/api/exam/assignment.ts frontend/src/views/classes/assignmentLifecycle.ts frontend/src/views/classes/assignmentLifecycle.test.ts frontend/src/views/classes/ClassDetail.vue frontend/src/views/classes/classAssignmentSource.test.ts frontend/src/views/learning/LearningHome.vue frontend/src/views/learning/learningAssignmentSource.test.ts
git diff --cached --check
git commit -m "fix(exam): 修复作业生命周期回归"
```

暂存前必须以 `git status --short` 展开实际文件，并确认不包含 `docker-compose.yml`。不执行 push。
