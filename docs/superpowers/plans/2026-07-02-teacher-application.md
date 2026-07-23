# 班主任申请审核实现计划

> **审计状态（2026-07-23）：** 实现与验证已完成；下方未勾选项是历史提交/推送记录，不代表当前功能缺失。完成证据见 [Phase 1 收尾审计报告](../reports/2026-07-23-phase1-closeout-audit.md)。

> 面向 AI 代理的工作者：按顺序执行。每个任务完成后运行对应验证，最后统一运行后端测试和前端构建。

**目标：** 增加普通用户申请成为班主任、管理员审核、获批后才能创建班级的完整闭环。

**架构：** 在考试业务域新增独立的班主任申请模型和服务，班级创建服务依赖该业务服务做权限判断。审批通过时只把用户提升到 `contributor`，不授予租户管理权限。

**技术栈：** Go、Gin、GORM、PostgreSQL migrations、Vue 3、TDesign。

---

## 文件结构

- `internal/types/exam_teacher_application.go`：新增申请模型、状态、请求体和响应类型。
- `internal/types/interfaces/exam_teacher_application.go`：新增 service/repository 接口。
- `internal/application/repository/exam_teacher_application.go`：持久化申请记录。
- `internal/application/service/exam_teacher_application.go`：申请、查询、审核、班主任资格判断。
- `internal/application/service/exam_teacher_application_test.go`：TDD 覆盖身份审批行为。
- `internal/application/service/exam_class.go`：创建班级前校验班主任资格。
- `internal/handler/exam_teacher_application.go`：新增 HTTP handler。
- `internal/router/exam.go`、`internal/router/router.go`、`internal/container/container.go`：注册依赖和路由。
- `migrations/versioned/000068_exam_teacher_applications.*.sql`：新增表和索引。
- `frontend/src/api/exam/teacher.ts`、`frontend/src/types/exam.ts`：前端 API 和类型。
- `frontend/src/views/classes/ClassList.vue`：申请入口、状态、创建按钮权限。
- `frontend/src/views/admin/AdminHome.vue`、`frontend/src/views/admin/TeacherApplications.vue`：管理员审核页签。

## 任务 1：后端类型与测试

- [x] 新增申请类型和接口。
- [x] 写服务测试：未获批创建班级失败。
- [x] 写服务测试：提交申请、管理员通过后创建班级成功。
- [x] 写服务测试：非管理员审核失败。
- [x] 运行目标测试，确认红灯。

## 任务 2：后端实现

- [x] 新增 repository。
- [x] 新增 service。
- [x] 修改 `NewExamClassService` 注入班主任申请服务。
- [x] 修改创建班级逻辑，调用 `IsApprovedTeacher`。
- [x] 新增 handler 和路由。
- [x] 新增 migration。
- [x] 注册 container 和 router 参数。
- [x] 运行目标测试，确认绿灯。

## 任务 3：前端实现

- [x] 新增前端 API 和类型。
- [x] 班级中心加载我的班主任状态。
- [x] 普通用户展示申请入口，获批后展示创建班级。
- [x] 管理端新增班主任申请页签。
- [x] 运行前端构建。

## 任务 4：验证与交付

- [x] 运行 Go 目标测试。
- [x] 运行前端构建。
- [x] Docker 重建并重启后端/前端。
- [x] 用 API smoke test 验证申请、审核、创建班级。
- [ ] 提交并推送 GitHub。
