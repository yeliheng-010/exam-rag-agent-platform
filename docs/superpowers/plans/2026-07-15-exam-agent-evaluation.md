# 题库级 Agent 行为评测实现计划

> **面向 AI 代理的工作者：** 使用 superpowers:executing-plans 按顺序实现，行为变更采用 TDD。

**目标：** 在现有题库级评测中心上增加真实 Agent 行为评测，持久化工具序列、参数、证据、引用和 groundedness 断言。

**架构：** 复用 `exam_rag_evaluation_runs` 生命周期，通过 `evaluation_kind` 隔离 RAG 与 Agent；新增 Agent service/worker，调用无会话持久化的生产 Agent 执行入口；前端新增独立 Agent 评测页面。

**技术栈：** Go、GORM、PostgreSQL migration、Asynq、Vue 3、TypeScript、TDesign、Node test。

---

### 任务 1：扩展运行模型和仓储隔离

- [x] 新增迁移 `000081_exam_agent_evaluation_runs`，增加 kind、agent_id 和复合索引。
- [x] 先写 repository 测试，要求 RAG/Agent 列表与详情按 kind 隔离。
- [x] 扩展 `ExamRAGEvaluationRun` 和 repository 接口并让测试通过。
- [x] 回归现有 RAG evaluation service/repository 测试。

### 任务 2：实现确定性 Agent 断言引擎

- [x] 定义 Agent 场景、期望工具调用、实际工具调用、场景结果和汇总类型。
- [x] 先写失败测试，覆盖工具顺序、递归参数子集、证据、引用、答案和 groundedness。
- [x] 实现纯函数评分器并让测试通过。
- [x] 覆盖空断言、重复工具、失败工具和 Unicode 短语。

### 任务 3：增加隔离的生产 Agent 执行入口

- [x] 先写 session service 测试，要求评测入口返回 `AgentState` 且不改变现有 AgentQA 错误事件行为。
- [x] 把 AgentQA 的公共构建/执行路径提取为内部函数。
- [x] 在 `SessionService` 增加不持久化会话的 `ExecuteAgentEvaluation`。
- [x] 回归 session/agent 相关测试。

### 任务 4：实现 Agent 运行 service 与 worker

- [x] 先写 service 测试：权限、Agent 模式、只读工具、快照和入队。
- [x] 先写 worker 测试：逐场景执行、进度、单场景失败、完成与空状态保护。
- [x] 实现 `ExamAgentEvaluationService`、安全 Agent 副本和异步 worker。
- [x] 列表裁剪完整场景详情，详情保留执行轨迹。

### 任务 5：接入 API、任务和 DI

- [x] 新增 handler 测试和三个 Contributor 路由。
- [x] 注册 `exam:agent_evaluation_run` 到 Redis Asynq 和 Lite worker。
- [x] 注册 repository/service/handler 到容器。
- [x] 运行 handler/router/container 定向测试和 `go vet`。

### 任务 6：实现 Agent 评测前端

- [x] 先写 source/view-model 测试，覆盖 API、路由、场景表单、轮询和结果断言。
- [x] 扩展考试类型和 API，新增 Agent 评测路由。
- [x] 实现场景编辑器、历史、指标和结果详情页面。
- [x] 在 RAG 评测中心与题库详情增加 Agent 评测入口。
- [x] 完成 390px 与 1440px 响应式布局。

### 任务 7：回归、部署和真实验收

- [x] 运行相关 Go 包测试、`go vet`、前端全量测试和 Vite 生产构建。
- [x] 运行 `git diff --check`，排除本机 `docker-compose.yml`。
- [x] 重建 app/frontend，确认五个核心容器健康且迁移为 81/clean。
- [x] 验证非 Contributor 三个 API 均为 403。
- [x] 使用现有考试 Agent 创建真实运行，核对工具序列、参数、证据和最终回答。
- [x] 浏览器检查桌面/移动页面无横向溢出。
