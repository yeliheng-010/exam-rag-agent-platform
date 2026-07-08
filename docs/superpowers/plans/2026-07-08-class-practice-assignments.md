# 班级练习任务发布实现计划

> **面向 AI 代理的工作流：** 使用 superpowers:executing-plans 在当前分支内逐任务实现此计划。步骤使用复选框语法跟踪进度。

**目标：** 让老师可以把正式题组发布为班级练习任务，学生可以在学习中心看到任务并从任务入口创建练习 attempt。

**架构：** 新增 exam assignment 模块，负责班级任务的持久化、权限校验和 assignment attempt 创建；复用现有 question group 与 practice answer 流程，给 `exam_practice_attempts` 增加可选 `assignment_id`。

**技术栈：** Go、Gin、GORM、PostgreSQL migration、Vue 3、TDesign、TypeScript、Node test。

---

## 文件结构

- 新增 `internal/types/exam_assignment.go`：任务模型、请求、列表过滤和 summary。
- 新增 `internal/types/interfaces/exam_assignment.go`：任务 service/repository 接口。
- 新增 `internal/application/repository/exam_assignment.go`：任务创建、班级列表、个人列表。
- 新增 `internal/application/service/exam_assignment.go`：班级权限、题组归属校验、任务 attempt 创建。
- 新增 `internal/application/service/exam_assignment_test.go`：覆盖发布、权限、任务 attempt。
- 新增 `internal/handler/exam_assignment.go`：任务 API handler。
- 修改 `internal/router/exam.go`、`internal/router/router.go`、`internal/container/container.go`：注册依赖和路由。
- 修改 `internal/types/exam_practice.go`、`internal/types/interfaces/exam_practice.go`、`internal/application/repository/exam_practice.go`：attempt 支持 `assignment_id` 和按 assignment 读取最近 attempt。
- 新增 `migrations/versioned/000077_exam_class_assignments.*.sql`。
- 修改 `frontend/src/types/exam.ts`：新增 assignment 类型，attempt 加 `assignment_id`。
- 新增 `frontend/src/api/exam/assignment.ts`。
- 修改 `frontend/src/views/classes/ClassDetail.vue`：新增练习任务页签和发布弹窗。
- 修改 `frontend/src/views/learning/LearningHome.vue`：新增班级任务区域。
- 新增 `frontend/src/views/classes/classAssignmentSource.test.ts` 与 `frontend/src/views/learning/learningAssignmentSource.test.ts`。

## 任务 1：后端模型与红灯测试

- [ ] 编写 service 红灯测试：学生发布被拒绝、跨 space 题组被拒绝、任务 attempt 写入 assignment。
- [ ] 定义 `ExamClassAssignment`、`CreateExamAssignmentRequest`、`ExamAssignmentSummary`。
- [ ] 扩展 `ExamPracticeAttempt.AssignmentID`。
- [ ] 运行 `go test ./internal/application/service -run Assignment -count=1`，确认因缺少实现失败。

## 任务 2：后端实现与迁移

- [ ] 实现 assignment repository 和 practice repository 的 assignment attempt 查询。
- [ ] 实现 assignment service：创建任务、按班级列出、按当前用户列出、从任务创建 attempt。
- [ ] 新增 handler、DI 和路由。
- [ ] 新增 `000077_exam_class_assignments` migration。
- [ ] 更新路由 guard source test。
- [ ] 运行 `go test ./internal/application/service ./internal/application/repository ./internal/router ./internal/handler -run "Exam|Assignment|Practice" -count=1`。

## 任务 3：前端任务发布与学习入口

- [ ] 编写 source 红灯测试：assignment API、班级页任务 tab、学习中心任务卡。
- [ ] 新增 `frontend/src/api/exam/assignment.ts`。
- [ ] 班级详情页加载任务和可练题组，老师可发布任务。
- [ ] 学习中心加载当前用户任务，点击创建 assignment attempt 后进入练习页。
- [ ] 运行 `node --test src/views/classes/classAssignmentSource.test.ts src/views/learning/learningAssignmentSource.test.ts`。

## 任务 4：最终验证与提交

- [ ] 运行后端组合验证：
  `go test ./internal/application/service ./internal/application/repository ./internal/router ./internal/handler -run "Exam|Assignment|Practice" -count=1`
- [ ] 运行前端完整测试：`npm test`
- [ ] 运行前端构建：`npm run build-only`
- [ ] 检查 `git status --short`，确保不提交 `docker-compose.yml` 和题库详情页遗留改动。
- [ ] 提交并推送：`feat(exam): 添加班级练习任务`
