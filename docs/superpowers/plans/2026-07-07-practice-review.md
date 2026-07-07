# 学生练习记录与错题本实现计划

> **面向 AI 代理的工作者：** 使用 superpowers:executing-plans 逐任务实现此计划。

**目标：** 为学生端新增个人练习记录与错题本第一阶段，让已沉淀的作答数据
可以用于复习。

**架构：** 不新增表，复用 `exam_practice_attempts` 与
`exam_practice_answers`。后端在 practice 服务中新增 attempt 列表、attempt
详情和错题列表能力；前端新增复盘页，并在学习中心提供入口。

**技术栈：** Go、Gin、GORM、Vue 3、TDesign、TypeScript、Node test。

---

## 文件结构

- 修改 `internal/types/exam_practice.go`：新增列表过滤和复盘 DTO。
- 修改 `internal/types/interfaces/exam_practice.go`：扩展 service/repository 接口。
- 修改 `internal/application/repository/exam_practice.go`：新增用户 attempt 列表查询。
- 修改 `internal/application/service/exam_practice.go`：新增练习记录、详情和错题逻辑。
- 修改 `internal/application/service/exam_practice_test.go`：覆盖本人记录和错题过滤。
- 修改 `internal/handler/exam_practice.go`：新增 HTTP handler。
- 修改 `internal/router/exam.go`：注册 viewer 路由。
- 修改 `internal/router/exam_rbac_routes_test.go`：断言新路由 guard。
- 修改 `frontend/src/types/exam.ts`：新增复盘 DTO 类型。
- 修改 `frontend/src/api/exam/practice.ts`：新增复盘 API。
- 修改 `frontend/src/router/index.ts`：新增复盘页路由。
- 修改 `frontend/src/views/learning/LearningHome.vue`：增加复盘入口。
- 新增 `frontend/src/views/practice/PracticeReviewHome.vue`：复盘页。
- 新增 `frontend/src/views/practice/practiceReviewSource.test.ts`：前端 wiring 测试。

## 任务 1：后端类型、仓储和服务

- [ ] 在 `exam_practice.go` 中新增：
  - `ListPracticeAttemptsFilter`
  - `PracticeAttemptSummary`
  - `PracticeAttemptDetail`
  - `WrongQuestionItem`
  - `ListWrongQuestionsFilter`
- [ ] 在 repository 中新增 `ListAttemptsByUser`，按 `tenant_id`、`user_id`、
  `space_id IN ?`、`limit` 查询。
- [ ] 在 service 中新增：
  - `ListAttempts`
  - `GetAttemptDetail`
  - `ListWrongQuestions`
- [ ] 错题列表从当前用户最近 attempt 的 answer 派生，只返回 `is_correct=false`。
- [ ] 运行 `go test ./internal/application/service -run Practice -count=1`。

## 任务 2：后端 HTTP 与 RBAC

- [ ] 在 handler 中新增：
  - `ListAttempts`
  - `GetAttempt`
  - `ListWrongQuestions`
- [ ] 在 router 中注册 viewer 路由。
- [ ] 更新 `exam_rbac_routes_test.go`。
- [ ] 运行 `go test ./internal/router -run ExamPractice -count=1`。

## 任务 3：前端 API、路由和复盘页

- [ ] 在 `frontend/src/api/exam/practice.ts` 增加记录和错题 API。
- [ ] 在 `frontend/src/router/index.ts` 注册 `/platform/practice/review`。
- [ ] 新增 `PracticeReviewHome.vue`，用 tabs 展示练习记录和错题。
- [ ] 在学习中心增加复盘入口按钮。
- [ ] 新增 source 测试，覆盖 API、路由和页面使用。
- [ ] 运行 `npm test -- src/views/practice/practiceReviewSource.test.ts`。

## 任务 4：最终验证与提交

- [ ] 运行后端组合验证：
  `go test ./internal/application/service ./internal/application/repository ./internal/router ./internal/handler -run "Exam|Practice" -count=1`
- [ ] 运行前端测试：
  `npm test`
- [ ] 运行前端构建：
  `npm run build-only`
- [ ] 检查 `git status --short`，确保不提交 `docker-compose.yml` 和题库详情页遗留改动。
- [ ] 提交：
  `git commit -m "feat(exam): 添加学生错题复盘"`
