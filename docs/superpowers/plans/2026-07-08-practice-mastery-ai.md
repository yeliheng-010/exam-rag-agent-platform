# 错题掌握状态与 AI 讲解实现计划

> **面向 AI 代理的工作流：** 使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框语法跟踪进度。

**目标：** 让学生错题本支持掌握状态、复盘备注和带上下文的 AI 讲解入口。

**架构：** 复用 `exam_practice_answers` 存储每次作答的复盘状态，不新增错题表。后端新增 answer review PATCH 接口并沿用 attempt 归属校验；前端在 `PracticeReviewHome.vue` 错题卡上增加状态操作和 AI prompt 预填。

**技术栈：** Go、Gin、GORM、PostgreSQL migration、Vue 3、TDesign、TypeScript、Node test。

---

## 文件结构

- 修改 `internal/types/exam_practice.go`：新增复盘状态类型、answer 字段和请求 DTO。
- 修改 `internal/types/interfaces/exam_practice.go`：扩展 practice service/repository 接口。
- 修改 `internal/application/repository/exam_practice.go`：新增 answer 按 ID 读取和保存。
- 修改 `internal/application/service/exam_practice.go`：新增状态校验、归属校验和复盘更新逻辑。
- 修改 `internal/application/service/exam_practice_test.go`：覆盖复盘状态保存、归属校验和非法状态。
- 修改 `internal/handler/exam_practice.go`：新增 PATCH handler。
- 修改 `internal/router/exam.go` 与 `internal/router/exam_rbac_routes_test.go`：注册 viewer 路由并守护。
- 新增 `migrations/versioned/000076_exam_practice_answer_review.*.sql`：扩展 answer 表。
- 修改 `frontend/src/types/exam.ts`：新增前端复盘状态类型和字段。
- 修改 `frontend/src/api/exam/practice.ts`：新增复盘更新 API。
- 修改 `frontend/src/views/practice/PracticeReviewHome.vue`：新增状态按钮、备注输入和 AI 讲解入口。
- 修改 `frontend/src/views/practice/practiceReviewSource.test.ts`：覆盖前端 wiring。

## 任务 1：后端复盘状态

- [x] 编写 service 红灯测试：保存状态、拒绝他人 answer、拒绝非法状态。
- [x] 新增 `PracticeAnswerReviewStatus` 与 `UpdatePracticeAnswerReviewRequest`。
- [x] 新增 repository answer 读取/保存方法。
- [x] 实现 `UpdateAnswerReview`，通过 answer 所属 attempt 校验用户归属。
- [x] 新增 PATCH handler 和 viewer 路由。
- [ ] 运行 `go test ./internal/application/service -run Practice -count=1`。

## 任务 2：数据库迁移

- [x] 新增 `review_status`、`review_note`、`reviewed_at` 字段。
- [x] 新增按 `tenant_id, review_status, reviewed_at` 的索引。
- [x] 编写 down migration 删除新增索引和字段。

## 任务 3：前端复盘页

- [x] 编写 source 红灯测试：API、状态字段、AI 入口。
- [x] 新增 `updatePracticeAnswerReview` API。
- [x] 错题卡新增掌握状态按钮组和备注输入。
- [x] 使用 `useStartChat()` 将错题上下文预填到 AI 对话。
- [x] 运行 `node --test src/views/practice/practiceReviewSource.test.ts`。

## 任务 4：最终验证与提交

- [ ] 运行后端组合验证：
  `go test ./internal/application/service ./internal/application/repository ./internal/router ./internal/handler -run "Exam|Practice" -count=1`
- [ ] 运行前端完整测试：`npm test`
- [ ] 运行前端构建：`npm run build-only`
- [ ] 检查 `git status --short`，确保不提交 `docker-compose.yml` 和题库详情页遗留改动。
- [ ] 提交：`feat(exam): 添加错题掌握复盘`
