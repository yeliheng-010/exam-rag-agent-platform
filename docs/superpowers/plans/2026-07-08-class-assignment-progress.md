# 班级练习结果查看实现计划

> **面向 AI 代理的工作者：** 使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框语法跟踪进度。

**目标：** 让班主任和助教在班级练习任务发布后，可以查看每个学生的开始、完成、答题数、正确数和正确率，形成发布练习后的教学闭环。

**架构：** 不新增数据库表。复用 `exam_class_members`、`exam_class_assignments`、`exam_practice_attempts` 中已有数据，后端在 assignment service 聚合学生最新 assignment attempt，前端在班级详情的练习任务卡片中提供“查看结果”弹窗。

**技术栈：** Go、Gin、GORM、Vue 3、TDesign、TypeScript、Node test、Docker Go test。

---

## 文件结构

- 修改 `internal/types/exam_assignment.go`：新增任务结果汇总 DTO。
- 修改 `internal/types/interfaces/exam_assignment.go`：扩展 assignment service。
- 修改 `internal/types/interfaces/exam_practice.go`：扩展 practice repository 的批量 assignment attempt 查询。
- 修改 `internal/application/repository/exam_practice.go`：实现按 assignment 和学生批量读取最近 attempt。
- 修改 `internal/application/service/exam_assignment.go`：新增班级写权限校验、学生成员过滤和结果汇总。
- 修改 `internal/application/service/exam_assignment_test.go`：覆盖老师查看结果、学生被拒绝、未开始学生展示。
- 修改 `internal/handler/exam_assignment.go`：新增结果查看 handler。
- 修改 `internal/router/exam.go` 与 `internal/router/exam_rbac_routes_test.go`：注册 viewer 路由，实际写权限由 service 控制。
- 修改 `frontend/src/types/exam.ts`：新增结果汇总类型。
- 修改 `frontend/src/api/exam/assignment.ts`：新增结果 API。
- 修改 `frontend/src/views/classes/ClassDetail.vue`：任务卡增加“查看结果”入口和结果弹窗。
- 修改 `frontend/src/views/classes/classAssignmentSource.test.ts`：覆盖前端 wiring。

## 任务 1：后端结果汇总

- [x] 编写 service 红灯测试：老师能看到两个学生结果，其中一个未开始、一个已完成；学生查看全班结果被拒绝。
- [x] 新增 `ExamAssignmentProgressStatus`、`ExamAssignmentMemberProgress`、`ExamAssignmentProgressSummary`。
- [x] 新增 repository 方法 `ListLatestAttemptsByAssignmentUsers`。
- [x] 实现 `GetAssignmentProgress`，仅班级老师或助教可访问，pending/removed 成员不纳入统计。
- [x] 运行：
  `docker run --rm -v "D:\rag\WeKnora:/app" -w /app golang:1.26 go test ./internal/application/service -run "ExamAssignment.*Progress" -count=1`

## 任务 2：HTTP 路由

- [x] 新增 `GetAssignmentProgress` handler。
- [x] 注册 `GET /api/v1/exam/classes/:class_id/assignments/:assignment_id/progress`。
- [x] 更新路由 guard source test。
- [x] 运行：
  `docker run --rm -v "D:\rag\WeKnora:/app" -w /app golang:1.26 go test ./internal/router ./internal/handler -run "ExamAssignment|ExamPractice" -count=1`

## 任务 3：前端结果弹窗

- [x] 新增前端类型和 `getClassAssignmentProgress` API。
- [x] 练习任务卡在老师/助教视角展示“查看结果”。
- [x] 弹窗展示完成概览、平均正确率和学生结果表。
- [x] 更新 source test，覆盖 API、按钮和弹窗关键文案。
- [x] 运行：
  `node --test src/views/classes/classAssignmentSource.test.ts`

## 任务 4：最终验证与提交

- [x] 运行后端组合验证：
  `docker run --rm -v "D:\rag\WeKnora:/app" -w /app golang:1.26 go test ./internal/application/service ./internal/application/repository ./internal/router ./internal/handler -run "Exam|Assignment|Practice" -count=1`
- [x] 运行前端构建：
  `npm run build-only`
- [x] 检查 `git status --short`，不提交 `docker-compose.yml` 和题库详情页遗留改动。
- [x] 提交并推送：`feat(exam): 添加班级练习结果查看`
