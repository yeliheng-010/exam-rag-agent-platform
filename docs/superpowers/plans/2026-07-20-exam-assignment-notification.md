# 作业通知与催交实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为班级作业增加事务一致的发布/重发/撤回通知、24 小时限频催交、学生未读收件箱和统一全局铃铛。

**架构：** 使用 `exam_assignment_notifications` 保存作业域通知；现有 `ExamAssignmentService` 通过独立扩展文件编排自动通知、催交、收件箱和已读状态。生命周期 repository 将 assignment 状态写入和自动通知放在同一事务，手动催交在 assignment 行锁下重新校验状态、完成情况和限频。前端聚合通知未读与租户邀请角标，通过轻量轮询刷新。

**技术栈：** Go、Gin、GORM、PostgreSQL、SQLite repository tests、Vue 3、TypeScript、TDesign、Node test runner、Docker Compose。

---

## 文件结构

- 创建 `migrations/versioned/000083_exam_assignment_notifications.{up,down}.sql`：通知表和索引。
- 创建 `internal/types/exam_assignment_notification.go`：通知、列表和催交 DTO。
- 创建 `internal/types/interfaces/exam_assignment_notification.go`：通知 repository 方法。
- 创建 `internal/application/repository/exam_assignment_notification.go`：收件箱、已读、限频和事务写入。
- 创建 `internal/application/service/exam_assignment_notification.go`：文案、接收人、催交、通知列表和跳转上下文。
- 创建 `internal/handler/exam_assignment_notification.go`：催交与通知 HTTP handler。
- 修改现有 assignment interface/repository/service/progress：接入自动通知和催交状态。
- 创建 `frontend/src/api/exam/assignmentNotification.ts`：通知与催交 API。
- 创建 `frontend/src/views/classes/assignmentNotification.ts`：badge 和跳转纯函数。
- 创建 `frontend/src/components/GlobalNotificationBell.vue`、`ExamAssignmentNotificationDialog.vue`：统一铃铛和通知中心。
- 修改 `ClassDetail.vue`、`LearningHome.vue`、`platform/index.vue`：教师催交、学生定位和全局入口。
- 删除 `frontend/src/components/GlobalInvitationBell.vue`：统一铃铛接入后移除旧入口，保留 `MyInvitationsDialog.vue`。

## 任务 1：迁移、类型与接口契约

**文件：**
- 创建：`migrations/versioned/000083_exam_assignment_notifications.up.sql`
- 创建：`migrations/versioned/000083_exam_assignment_notifications.down.sql`
- 创建：`internal/types/exam_assignment_notification.go`
- 创建：`internal/types/exam_assignment_notification_test.go`
- 创建：`internal/types/interfaces/exam_assignment_notification.go`
- 修改：`internal/types/exam_assignment.go`
- 修改：`internal/types/interfaces/exam_assignment.go`

- [ ] **步骤 1：编写类型红灯测试**

```go
func TestExamAssignmentNotificationContract(t *testing.T) {
    require.Equal(t, "exam_assignment_notifications", (ExamAssignmentNotification{}).TableName())
    require.Equal(t, ExamAssignmentNotificationKind("reminder"), ExamAssignmentNotificationKindReminder)
    result := &SendExamAssignmentReminderResult{SentCount: 3, SentUserIDs: []string{"a", "b", "c"}}
    require.Equal(t, result.SentCount, len(result.SentUserIDs))
}
```

- [ ] **步骤 2：运行红灯**

运行：`go test ./internal/types -run TestExamAssignmentNotificationContract -count=1`
预期：FAIL，通知类型尚未定义。

- [ ] **步骤 3：实现类型和 SQL**

定义 `ExamAssignmentNotificationKind{Published,Republished,Withdrawn,Reminder}`、`ExamAssignmentNotification`、`ExamAssignmentNotificationItem`、`ExamAssignmentNotificationList`、`SendExamAssignmentReminderRequest/Result`。`ExamAssignmentMemberProgress` 增加 `LastRemindedAt *time.Time` 和 `CanRemind bool`。迁移建立规格中的三组索引和 assignment/class 外键，down 删除新表。

Repository 接口签名固定为：

```go
type ExamAssignmentNotificationRepository interface {
    ListForRecipient(ctx context.Context, tenantID uint64, userID string, limit int) ([]*types.ExamAssignmentNotification, int64, error)
    MarkRead(ctx context.Context, tenantID uint64, userID, notificationID string, readAt time.Time) error
    MarkAllRead(ctx context.Context, tenantID uint64, userID string, readAt time.Time) error
    ListLatestReminders(ctx context.Context, tenantID uint64, assignmentID string, userIDs []string) (map[string]time.Time, error)
    CreateRemindersIfEligible(ctx context.Context, tenantID uint64, assignmentID string, candidates []*types.ExamAssignmentNotification, now time.Time) (*types.SendExamAssignmentReminderResult, error)
}
```

- [ ] **步骤 4：运行类型测试并提交**

运行：`gofmt -w internal/types/exam_assignment_notification.go internal/types/exam_assignment_notification_test.go internal/types/exam_assignment.go internal/types/interfaces/exam_assignment_notification.go internal/types/interfaces/exam_assignment.go; go test ./internal/types -run TestExamAssignmentNotificationContract -count=1`
预期：PASS。

提交：`feat(exam): 增加作业通知持久化契约`

## 任务 2：通知 Repository 收件箱与已读状态

**文件：**
- 创建：`internal/application/repository/exam_assignment_notification.go`
- 创建：`internal/application/repository/exam_assignment_notification_test.go`
- 修改：`internal/container/container.go`

- [ ] **步骤 1：编写 repository 红灯测试**

测试同一 SQLite 数据库中的两个 tenant、两个 recipient，断言 `ListForRecipient` 只返回当前范围且按 `created_at DESC, id DESC` 排序；`MarkRead` 不能更新其他用户记录；`MarkAllRead` 只清空当前 tenant/user 未读；不存在记录返回 `ErrExamAssignmentNotificationNotFound`。

- [ ] **步骤 2：运行红灯**

运行：`go test ./internal/application/repository -run TestExamAssignmentNotificationRepository -count=1`
预期：FAIL，constructor 和方法尚不存在。

- [ ] **步骤 3：实现最小 repository**

```go
func NewExamAssignmentNotificationRepository(db *gorm.DB) interfaces.ExamAssignmentNotificationRepository
func (r *examAssignmentNotificationRepository) ListForRecipient(...) ([]*types.ExamAssignmentNotification, int64, error)
func (r *examAssignmentNotificationRepository) MarkRead(...) error
func (r *examAssignmentNotificationRepository) MarkAllRead(...) error
func (r *examAssignmentNotificationRepository) ListLatestReminders(...) (map[string]time.Time, error)
```

`MarkRead` 以 `RowsAffected == 1` 为成功条件；列表 limit 默认 50 且最大 50。向 container 注册 repository provider。

- [ ] **步骤 4：运行测试并提交**

运行：`gofmt -w internal/application/repository/exam_assignment_notification.go internal/application/repository/exam_assignment_notification_test.go internal/container/container.go; go test ./internal/application/repository -run TestExamAssignmentNotificationRepository -count=1`
预期：PASS。

提交：`feat(exam): 实现作业通知收件箱存储`

## 任务 3：生命周期与自动通知原子写入

**文件：**
- 修改：`internal/application/repository/exam_assignment.go`
- 修改：`internal/application/repository/exam_assignment_test.go`
- 修改：`internal/types/interfaces/exam_assignment.go`
- 创建：`internal/application/service/exam_assignment_notification.go`
- 修改：`internal/application/service/exam_assignment.go`
- 修改：`internal/application/service/exam_assignment_test.go`

- [ ] **步骤 1：编写事务与 service 红灯测试**

Repository 测试用重复 notification ID 强制批量插入失败，断言首次发布不留下 assignment，重发/撤回不改变原状态。Service 测试断言 published、republished、withdrawn 只为 active student 构造通知，pending 和 teacher 不接收，标题/content/group snapshot 正确。

- [ ] **步骤 2：运行红灯**

运行：`go test ./internal/application/repository ./internal/application/service -run "TestExamAssignment.*Notification|TestExamAssignment.*Rollback" -count=1`
预期：FAIL，新事务接口和通知构造逻辑不存在。

- [ ] **步骤 3：实现事务接口和通知构造**

```go
CreateAssignmentWithNotifications(ctx context.Context, assignment *types.ExamClassAssignment, notifications []*types.ExamAssignmentNotification) error
TransitionAssignmentStatusWithNotifications(ctx context.Context, tenantID uint64, classID, assignmentID string, expected, next types.ExamAssignmentStatus, updatedAt time.Time, notifications []*types.ExamAssignmentNotification) error
```

两者使用 GORM transaction；状态转换继续保留 deadline 条件更新并检查 `RowsAffected`，通知批量写入失败即回滚。`examAssignmentService` 新增 `notificationRepo` 依赖，并在独立文件实现 `assignmentNotificationsForStudents`、`assignmentNotificationContent`。首次发布和 `transitionAssignment` 改调事务接口。

- [ ] **步骤 4：运行相关测试并提交**

运行：`gofmt -w internal/application/repository/exam_assignment.go internal/application/repository/exam_assignment_test.go internal/application/service/exam_assignment.go internal/application/service/exam_assignment_notification.go internal/application/service/exam_assignment_test.go internal/types/interfaces/exam_assignment.go; go test ./internal/application/repository ./internal/application/service -run "TestExamAssignment" -count=1`
预期：PASS。

提交：`feat(exam): 原子生成作业生命周期通知`

## 任务 4：催交、进度限频与通知跳转上下文

**文件：**
- 修改：`internal/application/repository/exam_assignment_notification.go`
- 修改：`internal/application/repository/exam_assignment_notification_test.go`
- 修改：`internal/application/repository/exam_assignment.go`
- 修改：`internal/types/interfaces/exam_assignment.go`
- 修改：`internal/application/service/exam_assignment_notification.go`
- 修改：`internal/application/service/exam_assignment_test.go`

- [ ] **步骤 1：编写催交红灯测试**

覆盖：published 未截止可催交；withdrawn/expired 返回状态冲突且不插入；completed 被跳过；not_started/in_progress 被发送；24 小时内重复调用计入 cooldown；指定用户不能绕过 active membership；进度返回 `LastRemindedAt/CanRemind`；通知列表只返回当前用户并附最近 `last_attempt_id` 和可信 `can_start`。

- [ ] **步骤 2：运行红灯**

运行：`go test ./internal/application/repository ./internal/application/service -run "TestExamAssignment(Reminder|NotificationList|ProgressReminder)" -count=1`
预期：FAIL，催交事务与 service 方法尚未实现。

- [ ] **步骤 3：实现 repository 事务和 service 方法**

```go
func (s *examAssignmentService) SendAssignmentReminders(ctx context.Context, tenantID uint64, userID, classID, assignmentID string, req *types.SendExamAssignmentReminderRequest) (*types.SendExamAssignmentReminderResult, error)
func (s *examAssignmentService) ListAssignmentNotifications(ctx context.Context, tenantID uint64, userID string, limit int) (*types.ExamAssignmentNotificationList, error)
func (s *examAssignmentService) MarkAssignmentNotificationRead(ctx context.Context, tenantID uint64, userID, notificationID string) error
func (s *examAssignmentService) MarkAllAssignmentNotificationsRead(ctx context.Context, tenantID uint64, userID string) error
```

`CreateRemindersIfEligible` 事务锁定 assignment，校验 published/未截止，读取候选用户最新 attempt 和 `now-24h` reminder 后批量插入。新增 assignment 批量查询方法避免通知列表 N+1；列表用现有 `ListLatestAttemptsByAssignments` 生成 `LastAttemptID` 和 `CanStart`。进度接口合并 latest reminder map。

```go
ListAssignmentsByIDsAndTenant(ctx context.Context, tenantID uint64, assignmentIDs []string) (map[string]*types.ExamClassAssignment, error)
```

- [ ] **步骤 4：运行测试并提交**

运行：`gofmt -w internal/application/repository/exam_assignment_notification.go internal/application/repository/exam_assignment_notification_test.go internal/application/repository/exam_assignment.go internal/application/service/exam_assignment_notification.go internal/application/service/exam_assignment_test.go internal/types/interfaces/exam_assignment.go; go test ./internal/application/repository ./internal/application/service -run "TestExamAssignment" -count=1`
预期：PASS。

提交：`feat(exam): 实现作业催交与通知列表`

## 任务 5：HTTP 路由与依赖注入

**文件：**
- 创建：`internal/handler/exam_assignment_notification.go`
- 创建：`internal/handler/exam_assignment_notification_test.go`
- 修改：`internal/types/interfaces/exam_assignment.go`
- 修改：`internal/router/exam.go`
- 修改：`internal/router/exam_rbac_routes_test.go`

- [ ] **步骤 1：编写路由和 handler 红灯测试**

路由 source matrix 固定四条 Viewer 路由：reminders、notifications list、read-all、single read。Handler 测试固定 JSON 绑定、limit 上限和 `ErrExamStateConflict -> 409`；单条 read 使用 path notification ID，不能从 body 接收 recipient。

- [ ] **步骤 2：运行红灯**

运行：`go test ./internal/handler ./internal/router -run "TestExamAssignmentNotification|TestExamAssignmentRouteGuardSourceMatrix" -count=1`
预期：FAIL，新 handler 和路由不存在。

- [ ] **步骤 3：实现 handler 和路由**

```go
exam.POST("/classes/:class_id/assignments/:assignment_id/reminders", g.Viewer(), assignmentHandler.SendAssignmentReminders)
exam.GET("/notifications", g.Viewer(), assignmentHandler.ListAssignmentNotifications)
exam.POST("/notifications/read-all", g.Viewer(), assignmentHandler.MarkAllAssignmentNotificationsRead)
exam.POST("/notifications/:notification_id/read", g.Viewer(), assignmentHandler.MarkAssignmentNotificationRead)
```

成功响应统一为 `{"success":true,"data":...}`；已读接口返回 `204`。静态 `read-all` 路由注册在参数路由之前。

- [ ] **步骤 4：运行后端组合测试并提交**

运行：`gofmt -w internal/handler/exam_assignment_notification.go internal/handler/exam_assignment_notification_test.go internal/router/exam.go internal/router/exam_rbac_routes_test.go internal/types/interfaces/exam_assignment.go; go test ./internal/application/repository ./internal/application/service ./internal/handler ./internal/router -run "ExamAssignment|ExamError" -count=1`
预期：PASS。

提交：`feat(exam): 开放作业通知与催交接口`

## 任务 6：前端类型、API 与纯函数

**文件：**
- 修改：`frontend/src/types/exam.ts`
- 创建：`frontend/src/api/exam/assignmentNotification.ts`
- 创建：`frontend/src/views/classes/assignmentNotification.ts`
- 创建：`frontend/src/views/classes/assignmentNotification.test.ts`
- 创建：`frontend/src/views/classes/assignmentNotificationSource.test.ts`

- [ ] **步骤 1：编写前端红灯测试**

纯函数测试覆盖 badge 相加并封顶 99、已有 attempt 优先跳转、can_start 定位学习中心、不可开始返回 blocked。Source 测试固定四个 API 路径、POST body 和通知类型字段。

- [ ] **步骤 2：运行红灯**

运行：`node --test src/views/classes/assignmentNotification.test.ts src/views/classes/assignmentNotificationSource.test.ts`
预期：FAIL，API 和 helper 尚不存在。

- [ ] **步骤 3：实现契约**

```ts
export function notificationBadgeCount(unread: number, invitations: number) { return Math.min(99, Math.max(0, unread) + Math.max(0, invitations)) }
export function assignmentNotificationTarget(item: ExamAssignmentNotificationItem):
  | { kind: 'attempt'; groupId: string; attemptId: string }
  | { kind: 'assignment'; assignmentId: string }
  | { kind: 'blocked' }
```

API 导出 `listAssignmentNotifications`、`sendAssignmentReminders`、`markAssignmentNotificationRead`、`markAllAssignmentNotificationsRead`。

- [ ] **步骤 4：运行测试并提交**

运行：`node --test src/views/classes/assignmentNotification.test.ts src/views/classes/assignmentNotificationSource.test.ts`
预期：PASS。

提交：`feat(exam): 增加作业通知前端契约`

## 任务 7：统一铃铛与通知中心

**文件：**
- 创建：`frontend/src/components/GlobalNotificationBell.vue`
- 创建：`frontend/src/components/ExamAssignmentNotificationDialog.vue`
- 创建：`frontend/src/components/examAssignmentNotificationSource.test.ts`
- 修改：`frontend/src/views/platform/index.vue`
- 删除：`frontend/src/components/GlobalInvitationBell.vue`

- [ ] **步骤 1：编写 UI source 红灯测试**

断言铃铛始终渲染、badge 合并邀请和作业未读、30 秒 timer、`visibilitychange` 暂停/恢复、卸载清理、全部已读、单条已读、邀请入口和三类点击目标。CSS 断言 560px 桌面宽度、移动端 `calc(100vw - 24px)`、长文本换行和无嵌套卡片。

- [ ] **步骤 2：运行红灯**

运行：`node --test src/components/examAssignmentNotificationSource.test.ts`
预期：FAIL，新组件尚不存在。

- [ ] **步骤 3：实现组件**

铃铛管理轮询与总 badge；dialog 管理最近 50 条通知、已读和点击事件；`MyInvitationsDialog` 继续处理邀请。platform 只挂载一个 `GlobalNotificationBell`。点击 attempt 用 `/platform/practice/question-groups/${groupId}?attempt_id=${attemptId}`，assignment 用 `/platform/learning?assignment_id=${assignmentId}`，blocked 显示不可开始提示。

- [ ] **步骤 4：运行测试并提交**

运行：`node --test src/components/examAssignmentNotificationSource.test.ts src/views/classes/assignmentNotification.test.ts`
预期：PASS。

提交：`feat(exam): 建立统一作业通知中心`

## 任务 8：教师催交与学生任务定位

**文件：**
- 修改：`frontend/src/views/classes/ClassDetail.vue`
- 修改：`frontend/src/views/classes/classAssignmentSource.test.ts`
- 修改：`frontend/src/views/learning/LearningHome.vue`
- 修改：`frontend/src/views/learning/learningAssignmentSource.test.ts`

- [ ] **步骤 1：编写页面 source 红灯测试**

ClassDetail 断言批量催交、单人 send 图标、completed 隐藏、`can_remind` 禁用、成功后刷新进度。LearningHome 断言读取 `assignment_id`、列表后滚动定位、短暂 highlight 且不自动创建 attempt。移动 CSS 断言操作列可换行且无横向滚动。

- [ ] **步骤 2：运行红灯**

运行：`node --test src/views/classes/classAssignmentSource.test.ts src/views/learning/learningAssignmentSource.test.ts`
预期：FAIL，页面未接入催交和定位。

- [ ] **步骤 3：实现页面行为**

进度顶部按钮发送空 recipient 数组；单人按钮发送 `[row.user_id]`；请求期间使用独立 loading ID，结果 toast 展示发送、完成跳过和限频跳过数量，再 `await loadAssignmentProgress()`。LearningHome 使用稳定 DOM id 和 `scrollIntoView({block:'center'})`，highlight 2 秒后清除并在卸载时清理 timer。

- [ ] **步骤 4：运行定向与全量前端测试并提交**

运行：`node --test src/views/classes/assignmentNotification.test.ts src/views/classes/assignmentNotificationSource.test.ts src/components/examAssignmentNotificationSource.test.ts src/views/classes/classAssignmentSource.test.ts src/views/learning/learningAssignmentSource.test.ts; npm test`
预期：定向 PASS，前端全量 `fail 0`。

提交：`feat(exam): 完成教师催交与学生通知跳转`

## 任务 9：组合验证、迁移、真实运行与审查

**文件：**
- 检查本计划全部文件
- 不提交：`docker-compose.yml`

- [ ] **步骤 1：运行后端门禁**

```powershell
go test ./internal/types ./internal/application/repository ./internal/application/service ./internal/handler ./internal/router -count=1
go vet ./internal/types ./internal/application/repository ./internal/application/service ./internal/handler ./internal/router
```

预期：全部 exit `0`。

- [ ] **步骤 2：运行前端门禁**

运行：`Set-Location frontend; npm test; npm run build-only; Set-Location ..`
预期：`fail 0`，Vite build exit `0`。

- [ ] **步骤 3：重建服务并验证迁移**

```powershell
docker compose build app frontend
docker compose up -d --no-deps app frontend
docker compose ps app frontend
docker compose logs app --tail 160
```

预期：app healthy、frontend running、migration `83 / dirty: false`、`GET /health` 为 `200`。

- [ ] **步骤 4：浏览器业务验收**

使用有效 teacher/student 会话验证发布、重发、撤回通知；批量/单人催交；24 小时限频；铃铛未读、单条/全部已读；已有 attempt 直接继续；355px 无横向溢出。记录网络状态、console error、scrollWidth/clientWidth 和截图。登录会话失效时停止写操作并报告阻塞，不读取或猜测个人凭据。

- [ ] **步骤 5：代码审查和完成门禁**

使用 `requesting-code-review` 检查 tenant 隔离、成员权限、事务回滚、行锁、限频、attempt 不变性、timer 清理和移动布局。发现问题先写红灯测试再修复。最后运行：

```powershell
git diff --check
git status --short
git log -10 --oneline
```

预期：工作区只保留用户的 `docker-compose.yml`，不 push。
