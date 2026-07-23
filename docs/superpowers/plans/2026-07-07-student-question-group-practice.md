# 学生端题组练习与 AI 讲解实现计划

> **审计状态（2026-07-23）：** 实现与验证已完成；下方未勾选框是未回填的历史执行记录，不代表当前功能缺失。完成证据见 [Phase 1 收尾审计报告](../reports/2026-07-23-phase1-closeout-audit.md)。

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为学生端新增正式题组练习闭环，让学生可以看到授权题组、提交客观题答案、查看解析，并把当前题上下文带入基础 RAG 对话。

**架构：** 新增 `exam_practice_attempts` 和 `exam_practice_answers`，后端提供 viewer 级 `exam/practice` API，服务层复用 `ExamSpaceService` 和正式题组仓储做空间权限校验。前端在学习中心新增题组练习入口，并新增题组练习页，AI 讲解通过现有 `menuStore.setPrefillQuery()` 和 `/platform/creatChat` 串联。

**技术栈：** Go、Gin、GORM、PostgreSQL migration、Vue 3、TDesign、TypeScript、Pinia。

---

## 文件结构

- 创建 `migrations/versioned/000075_exam_practice_attempts.up.sql`：新增练习 attempt 和答案表。
- 创建 `migrations/versioned/000075_exam_practice_attempts.down.sql`：回滚练习表。
- 创建 `internal/types/exam_practice.go`：练习实体、请求、结果和题组摘要 DTO。
- 创建 `internal/types/interfaces/exam_practice.go`：练习服务和仓储接口。
- 修改 `internal/types/interfaces/exam_question.go`：增加按可读空间列出题组摘要的方法。
- 修改 `internal/application/repository/exam_question.go`：实现练习题组摘要列表。
- 创建 `internal/application/repository/exam_practice.go`：GORM 练习仓储。
- 创建 `internal/application/service/exam_practice.go`：练习业务逻辑和判分。
- 创建 `internal/application/service/exam_practice_test.go`：覆盖权限、判分和 attempt 归属。
- 创建 `internal/handler/exam_practice.go`：练习 HTTP handler。
- 修改 `internal/router/exam.go`：注册 viewer 级练习接口。
- 修改 `internal/router/router.go`：注入练习 handler 参数。
- 修改 `internal/router/exam_rbac_routes_test.go`：补充练习路由 guard 断言。
- 修改 `internal/container/container.go`：注册练习仓储、服务和 handler。
- 修改 `frontend/src/types/exam.ts`：新增练习类型。
- 创建 `frontend/src/api/exam/practice.ts`：练习 API。
- 修改 `frontend/src/router/index.ts`：注册学生练习页路由。
- 修改 `frontend/src/views/learning/LearningHome.vue`：新增题组练习入口和数据加载。
- 创建 `frontend/src/views/practice/QuestionGroupPractice.vue`：题组练习页。
- 创建 `frontend/src/views/practice/questionGroupPracticeSource.test.ts`：前端 wiring 测试。

## 任务 1：数据库迁移、类型和接口

**文件：**
- 创建：`migrations/versioned/000075_exam_practice_attempts.up.sql`
- 创建：`migrations/versioned/000075_exam_practice_attempts.down.sql`
- 创建：`internal/types/exam_practice.go`
- 创建：`internal/types/interfaces/exam_practice.go`
- 修改：`internal/types/interfaces/exam_question.go`

- [ ] **步骤 1：新增迁移**

`000075_exam_practice_attempts.up.sql` 创建 `exam_practice_attempts` 与 `exam_practice_answers`，包含租户、用户、题组、统计字段和唯一约束 `(attempt_id, question_id)`。

- [ ] **步骤 2：新增 Go 类型**

`internal/types/exam_practice.go` 定义：

- `ExamPracticeAttemptStatus`
- `ExamPracticeAttempt`
- `ExamPracticeAnswer`
- `QuestionGroupPracticeSummary`
- `ListPracticeQuestionGroupsFilter`
- `CreatePracticeAttemptResult`
- `SubmitPracticeAnswerRequest`
- `PracticeAnswerResult`

- [ ] **步骤 3：新增接口**

`internal/types/interfaces/exam_practice.go` 定义 `ExamPracticeService` 和 `ExamPracticeRepository`。

`internal/types/interfaces/exam_question.go` 增加：

```go
ListQuestionGroupPracticeSummaries(ctx context.Context, tenantID uint64, spaceIDs []string, filter types.ListPracticeQuestionGroupsFilter) ([]*types.QuestionGroupPracticeSummary, error)
```

- [ ] **步骤 4：运行类型编译**

```powershell
go test ./internal/types/... -run TestNoSuchTest -count=1
```

预期：exit 0 或 `[no test files]`，没有类型错误。

## 任务 2：后端仓储和服务

**文件：**
- 修改：`internal/application/repository/exam_question.go`
- 创建：`internal/application/repository/exam_practice.go`
- 创建：`internal/application/service/exam_practice.go`
- 创建：`internal/application/service/exam_practice_test.go`

- [ ] **步骤 1：先写服务测试**

覆盖：

- 不可读题组不能创建 attempt。
- 正确答案 `B` 和学生答案 `b` 判为正确。
- 重复提交同一题更新答案和统计。
- 非本人 attempt 提交返回权限错误。

- [ ] **步骤 2：实现题组摘要仓储**

按 `space_id IN ?`、租户、状态过滤 `question_groups`，关联 `question_banks` 名称，并统计题组下正式题目数。

- [ ] **步骤 3：实现练习仓储**

提供创建 attempt、读取 attempt、upsert answer、列出 attempt answers、更新 attempt 统计、读取用户最近 attempt。

- [ ] **步骤 4：实现练习服务**

服务流程：

1. 解析用户可读空间。
2. 列出正式题组摘要并合并最近 attempt。
3. 创建 attempt 时读取题组详情并校验可读。
4. 提交答案时校验 attempt 属于当前用户。
5. 使用正式题目答案进行客观题判分。
6. 保存题目、答案和解析快照。

- [ ] **步骤 5：运行服务测试**

```powershell
go test ./internal/application/service -run Practice -count=1
```

预期：PASS。

## 任务 3：HTTP 接口、路由和容器

**文件：**
- 创建：`internal/handler/exam_practice.go`
- 修改：`internal/router/exam.go`
- 修改：`internal/router/router.go`
- 修改：`internal/router/exam_rbac_routes_test.go`
- 修改：`internal/container/container.go`

- [ ] **步骤 1：实现 handler**

实现：

- `ListQuestionGroups`
- `GetQuestionGroup`
- `CreateAttempt`
- `SubmitAnswer`
- `CompleteAttempt`

- [ ] **步骤 2：注册路由**

在 `RegisterExamRoutes` 增加 `practiceHandler` 参数，并注册：

```go
exam.GET("/practice/question-groups", g.Viewer(), practiceHandler.ListQuestionGroups)
exam.GET("/practice/question-groups/:group_id", g.Viewer(), practiceHandler.GetQuestionGroup)
exam.POST("/practice/question-groups/:group_id/attempts", g.Viewer(), practiceHandler.CreateAttempt)
exam.POST("/practice/attempts/:attempt_id/answers", g.Viewer(), practiceHandler.SubmitAnswer)
exam.POST("/practice/attempts/:attempt_id/complete", g.Viewer(), practiceHandler.CompleteAttempt)
```

- [ ] **步骤 3：注册容器依赖**

注册 `repository.NewExamPracticeRepository`、`service.NewExamPracticeService`、`handler.NewExamPracticeHandler`，并在 `RouterParams` 传入路由。

- [ ] **步骤 4：补路由 RBAC 测试**

`internal/router/exam_rbac_routes_test.go` 断言 practice 路由为 `g.Viewer()`。

- [ ] **步骤 5：运行路由测试**

```powershell
go test ./internal/router -run Exam -count=1
```

预期：PASS。

## 任务 4：前端 API、路由和学习中心入口

**文件：**
- 修改：`frontend/src/types/exam.ts`
- 创建：`frontend/src/api/exam/practice.ts`
- 修改：`frontend/src/router/index.ts`
- 修改：`frontend/src/views/learning/LearningHome.vue`
- 创建：`frontend/src/views/practice/questionGroupPracticeSource.test.ts`

- [ ] **步骤 1：新增前端类型和 API**

添加 `ExamPracticeAttempt`、`ExamPracticeAnswer`、`QuestionGroupPracticeSummary`、`PracticeAnswerResult`，并封装 practice API。

- [ ] **步骤 2：注册练习页路由**

新增：

```ts
{
  path: "practice/question-groups/:groupId",
  name: "questionGroupPractice",
  component: () => import("../views/practice/QuestionGroupPractice.vue"),
  meta: { requiresInit: true, requiresAuth: true }
}
```

- [ ] **步骤 3：学习中心新增题组练习区域**

调用 `listPracticeQuestionGroups()`，展示最近题组卡片，并允许所有 viewer 进入练习页。

- [ ] **步骤 4：新增 wiring 测试**

测试学习中心引用 practice API、路由注册练习页、学生入口不依赖 `canUseQuestionBanks`。

- [ ] **步骤 5：运行前端测试**

```powershell
cd frontend
npm test -- questionGroupPracticeSource
```

预期：PASS。

## 任务 5：题组练习页和 AI 讲解入口

**文件：**
- 创建：`frontend/src/views/practice/QuestionGroupPractice.vue`
- 修改：`frontend/src/views/practice/questionGroupPracticeSource.test.ts`

- [ ] **步骤 1：实现练习页数据加载**

加载题组详情，创建 attempt，并初始化当前题索引、答案状态和材料折叠状态。

- [ ] **步骤 2：实现答题交互**

客观题选项使用按钮式选择；提交后调用 `submitPracticeAnswer()` 并展示判分、答案、解析和证据。

- [ ] **步骤 3：实现翻页和完成练习**

提供上一题、下一题、题号 tabs 和完成练习按钮，完成后展示统计。

- [ ] **步骤 4：实现 AI 讲解**

使用 `menuStore.setPrefillQuery()` 写入当前题组和题目的讲解请求，选择可用班级知识库后跳转 `/platform/creatChat`。

- [ ] **步骤 5：补充前端 wiring 测试**

断言练习页使用 `submitPracticeAnswer`、`completePracticeAttempt`、`setPrefillQuery` 和 `/platform/creatChat`。

- [ ] **步骤 6：运行前端构建**

```powershell
cd frontend
npm run build-only
```

预期：exit 0。

## 任务 6：最终验证与收尾

**文件：**
- 修改：验证失败指向的具体实现文件。
- 不提交：`docker-compose.yml` 的本地环境改动。

- [ ] **步骤 1：运行后端定向测试**

```powershell
go test ./internal/application/service ./internal/application/repository ./internal/router ./internal/handler -run "Exam|Practice" -count=1
```

预期：PASS。若 Windows cgo 工具链阻塞，记录完整错误并运行可替代的前端和静态测试。

- [ ] **步骤 2：运行前端测试和构建**

```powershell
cd frontend
npm test
npm run build-only
```

预期：测试 PASS，构建 exit 0。

- [ ] **步骤 3：检查工作区**

```powershell
git status --short
```

确认实现文件齐全，`docker-compose.yml` 不进入本次提交。

- [ ] **步骤 4：提交**

```powershell
git add migrations/versioned/000075_exam_practice_attempts.up.sql migrations/versioned/000075_exam_practice_attempts.down.sql internal/types/exam_practice.go internal/types/interfaces/exam_practice.go internal/types/interfaces/exam_question.go internal/application/repository/exam_question.go internal/application/repository/exam_practice.go internal/application/service/exam_practice.go internal/application/service/exam_practice_test.go internal/handler/exam_practice.go internal/router/exam.go internal/router/router.go internal/router/exam_rbac_routes_test.go internal/container/container.go frontend/src/types/exam.ts frontend/src/api/exam/practice.ts frontend/src/router/index.ts frontend/src/views/learning/LearningHome.vue frontend/src/views/practice/QuestionGroupPractice.vue frontend/src/views/practice/questionGroupPracticeSource.test.ts docs/superpowers/specs/2026-07-07-student-question-group-practice-design.md docs/superpowers/plans/2026-07-07-student-question-group-practice.md
git commit -m "feat(exam): 实现学生题组练习"
```

## 执行检查点

1. 任务 1 完成后，数据库和类型边界明确。
2. 任务 2 完成后，核心练习行为可由后端服务测试验证。
3. 任务 3 完成后，学生可通过 viewer 路由访问练习 API。
4. 任务 4 完成后，学习中心出现学生练习入口。
5. 任务 5 完成后，学生可以完成一次题组练习并进入 AI 讲解。
6. 任务 6 是最终验收，不通过就回到对应任务修复。
