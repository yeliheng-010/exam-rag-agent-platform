# 考试资料入库与试卷结构化实现计划

> **审计状态（2026-07-23）：** 实现与验证已完成；下方未勾选项是历史提交/推送记录，不代表当前功能缺失。完成证据见 [Phase 1 收尾审计报告](../reports/2026-07-23-phase1-closeout-audit.md)。

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 打通老师登记考试资料、创建试卷结构化任务、关联题库的第一阶段真实链路。

**架构：** 后端新增 `ExamMaterialService`，复用知识库、知识文档、chunk、考试空间和题库服务。数据库新增 `exam_materials` 与 `exam_structuring_tasks`，前端在班级详情资料页增加登记资料与结构化任务入口。

**技术栈：** Go、Gin、GORM、PostgreSQL migration、Vue 3、TDesign。

---

## 文件结构

- 新增 `internal/types/exam_material.go`：考试资料和结构化任务类型、请求、过滤器。
- 新增 `internal/types/interfaces/exam_material.go`：资料服务与仓储接口。
- 新增 `internal/application/repository/exam_material.go`：GORM 仓储。
- 新增 `internal/application/service/exam_material.go`：权限校验、资料登记、任务创建。
- 新增 `internal/application/service/exam_material_test.go`：TDD 覆盖核心行为。
- 新增 `internal/handler/exam_material.go`：HTTP handler。
- 修改 `internal/router/exam.go`、`internal/router/router.go`、`internal/container/container.go`：接入路由和 DI。
- 修改 `internal/handler/exam_error.go`：新增 not found 映射。
- 新增 `migrations/versioned/000072_exam_material_structuring.up.sql` 和 `.down.sql`。
- 新增 `frontend/src/api/exam/material.ts`，修改 `frontend/src/types/exam.ts`、`frontend/src/views/classes/ClassDetail.vue`。

## 任务 1：后端服务 TDD

- [x] 编写失败测试：学生或无写权限用户不能登记资料。
- [x] 编写失败测试：老师登记试卷资料时会校验知识库、文档归属并自动绑定知识库资源。
- [x] 编写失败测试：试卷资料创建结构化任务时，已完成解析且有 chunk 的文档进入 `ready_for_review`。
- [x] 运行定向 Go 测试，确认红灯；当前 Windows 工具链先被 `internal/utils/inject.go` 中 `pg_query.Parse/Deparse` 编译问题阻塞。

## 任务 2：类型、仓储和迁移

- [x] 新增考试资料、结构化任务类型和接口。
- [x] 新增 GORM 仓储与 not found sentinel。
- [x] 新增 PostgreSQL migration，包含索引和唯一约束。
- [x] 接入 DI 容器。

## 任务 3：服务与 HTTP API

- [x] 实现 `RegisterMaterial`、`ListMaterials`、`CreateStructuringTask`、`ListStructuringTasks`。
- [x] 接入知识库和知识文档校验。
- [x] 接入考试空间读写权限。
- [x] 接入题库创建 / 选择。
- [x] 接入路由 guard：读接口 Viewer，写接口 Contributor。

## 任务 4：前端班级资料页

- [x] 新增资料 API。
- [x] 扩展考试类型定义。
- [x] 在班级详情资料 tab 中增加“登记资料”按钮、资料表格和结构化任务表格。
- [x] 复用现有知识库、知识文档、考试域、题库 API。
- [x] 保持 WeKnora / TDesign 风格，不新增营销式页面。

## 任务 5：验证与提交

- [x] 运行后端定向测试；Windows 当前报 `undefined: pg_query.Parse` / `undefined: pg_query.Deparse`，未进入本模块测试执行。
- [x] 运行 `npm --prefix frontend run build`。
- [x] 运行 `git diff --check`。
- [x] 确认不提交本地 `docker-compose.yml` 端口映射改动。
- [ ] 提交并推送，commit message：`feat(exam): 添加考试资料入库结构化基础`。
