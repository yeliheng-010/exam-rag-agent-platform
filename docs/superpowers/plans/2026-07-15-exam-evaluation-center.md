# 平台级 RAG/Agent 评测与回归中心实现计划

> **面向 AI 代理的工作者：** 使用 `superpowers:executing-plans` 在当前工作区逐项实现；所有行为变更遵循红灯、绿灯、重构循环。

**目标：** 建成跨题库的评测集、版本、基线、确定性回归比较和 trace 下钻平台，并完成权限、Docker、数据库、API 和响应式页面验收。

**架构：** 复用 `exam_rag_evaluation_runs` 的不可变快照和现有 RAG/Agent 执行服务；新增评测集/版本表及平台聚合 Service。授权在 Service 层解析 tenant role 和可见题库范围，Repository 只执行显式 tenant/bank 作用域查询。

**技术栈：** Go、Gin、GORM、PostgreSQL、SQLite repository tests、Vue 3、TypeScript、TDesign、Node test、Docker Compose。

---

## 文件结构

- 新增 `migrations/versioned/000082_exam_evaluation_center.*.sql`：基线、评测集和版本 schema。
- 修改 `internal/types/exam_rag_evaluation.go`：运行来源和基线字段。
- 新增 `internal/types/exam_evaluation_center.go`：平台筛选、DTO、评测集和版本类型。
- 修改 `internal/types/interfaces/exam_rag_evaluation.go`：平台 repository 能力。
- 新增 `internal/types/interfaces/exam_evaluation_center.go`：平台 service 契约。
- 修改 `internal/application/repository/exam_rag_evaluation.go`：聚合查询、基线事务、评测集版本事务。
- 新增 `internal/application/service/exam_evaluation_center.go`：权限、快照指标、回归和评测集编排。
- 新增 `internal/application/service/exam_agent_evaluation_snapshot.go`：校验并按不可变 Agent 定义快照重跑。
- 新增 `internal/handler/exam_evaluation_center.go`：HTTP 参数与响应。
- 修改 `internal/container/container.go`、`internal/router/exam.go`：依赖和 Contributor 路由。
- 新增/修改对应 Go tests：repository、service、handler、router。
- 新增 `frontend/src/api/exam/evaluation-center.ts`：平台 API。
- 修改 `frontend/src/types/exam.ts`：平台 DTO。
- 新增 `frontend/src/views/evaluation/EvaluationCenter.vue`、`.less`、`evaluationCenterViewModel.ts`：平台页面与纯函数。
- 修改 `frontend/src/router/index.ts`、`frontend/src/stores/menu.ts`、`frontend/src/components/menu.vue`：一级入口和角色控制。
- 新增对应 Node source/view-model tests。

## 任务 1：持久化与 Repository

- [x] 先写 repository 测试，覆盖可见题库过滤、Admin tenant 全量、唯一基线替换、跨 tenant 拒绝、评测集版本递增和不兼容来源拒绝。
- [x] 运行 `go test ./internal/application/repository -run 'EvaluationCenter|RAGEvaluationRepository' -count=1`，确认因方法/schema 缺失失败。
- [x] 添加迁移、类型和 repository 接口/实现。
- [x] 重跑定向测试并保持现有题库级 repository 测试通过。

## 任务 2：Service、Handler 与路由

- [x] 先写 service 测试，覆盖 Admin/Contributor 可见范围、指标 delta、回归阈值、基线、创建集、追加版本和按版本调用真实 RAG/Agent service。
- [x] 先写 handler/router 测试，覆盖查询解析、HTTP 201/202/200 和 Contributor 路由。
- [x] 运行定向测试确认红灯原因是平台能力缺失。
- [x] 实现平台 service、handler、依赖注入和路由。
- [x] 重跑 `go test ./internal/application/service ./internal/handler ./internal/router -run 'EvaluationCenter|ExamRAGEvaluation|ExamAgentEvaluation' -count=1`。

## 任务 3：前端 API、视图模型与导航

- [x] 先写 Node 测试，覆盖平台 API 路径、路由/菜单角色、回归状态、指标格式、兼容运行筛选和下钻路径。
- [x] 运行定向 `node --test`，确认页面/API/纯函数缺失导致红灯。
- [x] 实现 TypeScript DTO、API、纯视图模型、一级路由和菜单入口。
- [x] 重跑定向 Node 测试。

## 任务 4：完整响应式页面

- [x] 实现回归运行和评测集两个视图、筛选、指标带、状态、空态、错误态、轮询和操作反馈。
- [x] 实现设置基线、从运行保存评测集、从兼容运行追加版本、按版本重跑和题库详情下钻。
- [x] 实现 1440px 稠密表格与 390px 堆叠条目，不嵌套卡片、不出现横向溢出。
- [x] 运行 `npm test` 和 `npm run build-only`；检查 `type-check`，本次评测中心文件无新增错误，全仓保留既有类型债务。

## 任务 5：真实部署与权限验收

- [x] 重建并启动 app/frontend，确认迁移 000082 成功。
- [x] 用 Admin/Owner、Contributor、Viewer 三类真实身份请求 API，核对 200/403 和题库隔离。
- [x] 从真实 RAG 和 Agent 已完成运行创建评测集、追加版本、重跑并设置基线。
- [x] 核对 PostgreSQL 中唯一基线、版本递增和运行来源字段。
- [x] 通过浏览器在 1440px 和 390px 验证筛选、操作、下钻、长 ID、空态和无横向滚动。

## 任务 6：统一质量门禁

- [x] 运行相关 Go tests、相关包 `go vet`、前端全量 tests、build-only 和 `git diff --check`；全仓测试/type-check 的既有无关失败单独记录。
- [x] 使用 `requesting-code-review` 发起独立审查；审查代理超时后由当前会话按同一清单完成，修复 Agent 版本未复用不可变配置快照的问题。
- [x] 使用 `verification-before-completion` 复核最新命令和真实运行证据。
- [x] 同步计划勾选状态；不自动提交、不暂存 `docker-compose.yml`。
