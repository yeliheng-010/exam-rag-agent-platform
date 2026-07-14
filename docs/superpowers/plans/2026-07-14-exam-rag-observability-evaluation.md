# 题库级 RAG 可观测与评测中心实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框跟踪。

**目标：** 为题库提供可持久化的异步 RAG 评测运行、分阶段检索 trace、确定性指标、历史对比和题库内观测页面。

**架构：** 保留现有同步 rag-diagnostics 接口。检索层增加可选 trace recorder，不改变普通 HybridSearch 契约；题库评测运行通过 Asynq/Lite 共用 worker 执行并保存不可变 JSON 快照；前端通过题库二级路由查看历史、案例详情与双运行对比。

**技术栈：** Go、Gin、GORM、PostgreSQL JSONB、Asynq、Vue 3、TypeScript、TDesign、Node test。

---

## 文件结构

- 创建 internal/types/search_trace.go：检索参数、候选、阶段和总 trace。
- 创建 internal/application/service/knowledgebase_search_trace.go：context recorder 与 trace 快照辅助函数。
- 修改 internal/application/service/knowledgebase_search.go：记录查询、向量、关键词、融合和最终结果。
- 修改 internal/application/service/knowledgebase_search_fusion.go：生成带双路排名的融合 trace。
- 修改 internal/examrag/context_resolver.go：返回上下文来源、题组、耗时和检索 trace。
- 修改 internal/searchutil/exam_context_eval.go：补充 MRR、Recall@K、结构化解析率和耗时指标。
- 创建 internal/types/exam_rag_evaluation.go：运行、请求、进度、状态和快照类型。
- 创建 internal/types/interfaces/exam_rag_evaluation.go：repository/service 契约。
- 创建 internal/application/repository/exam_rag_evaluation.go：tenant/bank scoped 持久化。
- 创建 internal/application/service/exam_rag_evaluation.go：创建、列表、详情和执行编排。
- 创建 internal/application/service/exam_rag_evaluation_task.go：worker payload 执行与幂等。
- 创建 internal/handler/exam_rag_evaluation.go：HTTP 202、列表和详情 handler。
- 修改 internal/container/container.go、internal/router/exam.go、internal/router/task.go、internal/router/sync_task.go、internal/types/task.go：注册依赖、路由和 worker。
- 创建 migrations/versioned/000080_exam_rag_evaluation_runs.up.sql 与 down.sql。
- 创建 frontend/src/api/exam/rag-evaluation.ts：运行 API。
- 修改 frontend/src/types/exam.ts：评测运行前端类型。
- 修改 frontend/src/router/index.ts：题库二级路由。
- 修改 frontend/src/views/question-bank/QuestionBankDetail.vue：观测入口。
- 创建 frontend/src/views/question-bank/QuestionBankRAGObservability.vue 与 less：观测页面。
- 创建 frontend/src/views/question-bank/ragEvaluationViewModel.ts：指标、比较和 trace 展示模型。
- 创建对应 Go/Node 测试。

## 任务 1：检索 Trace

- [x] 编写失败测试，构造向量与关键词候选，断言 trace 保留原始分数、各自排名、RRF 排名和最终 chunk。
- [x] 运行 knowledgebase search 定向测试，确认因 trace 类型和 recorder 缺失失败。
- [x] 新增 SearchTrace、SearchTraceCandidate、SearchTraceParameters 和 fusion method 类型。
- [x] 新增 context recorder；无 recorder 时所有记录函数为 no-op。
- [x] 在 HybridSearch 中记录 embedding 模型、向量维度、向量候选、关键词候选、融合候选、最终 chunk 和耗时。
- [x] 增加 HybridSearchWithTrace 可选方法，内部仍调用原 HybridSearch。
- [x] 运行定向测试确认普通 HybridSearch 与 trace 路径都通过。

验证命令：

    go test ./internal/application/service -run "SearchTrace|Fusion" -count=1

## 任务 2：结构化解析 Trace 与指标

- [x] 编写 resolver 失败测试：成功回链返回 structured_question_group、group ID、trace 和耗时；无题组返回 none。
- [x] 编写指标失败测试：expected chunks 计算 Recall@K/MRR；无 expected chunks 不进入分母；统计结构化解析率和平均耗时。
- [x] 运行 internal/examrag 与 internal/searchutil 定向测试确认失败。
- [x] 扩展 ExamQuestionContextResolveResult 和 EvalResolver，传递 resolver metadata。
- [x] 扩展评测结果与汇总类型，保持现有字段兼容。
- [x] 实现确定性指标并补零案例、错误案例边界。
- [x] 运行全部 examrag/searchutil 测试确认通过。

验证命令：

    go test ./internal/examrag ./internal/searchutil -count=1

## 任务 3：评测运行持久化

- [x] 编写 repository 失败测试：创建运行、按 tenant/bank 列表、按 tenant/bank/run 读取、跨租户和跨题库不可见。
- [x] 编写迁移 000080，创建 JSONB 快照表和 tenant/bank/time 索引。
- [x] 新增运行状态、进度、请求快照、结果快照和列表 DTO。
- [x] 新增 repository 接口和 GORM 实现；所有查询显式包含 tenant_id 与 question_bank_id。
- [x] 运行 repository 测试确认通过。

验证命令：

    go test ./internal/application/repository -run "ExamRAGEvaluation" -count=1

## 任务 4：异步 Service、Worker 与 API

- [x] 编写 service 失败测试：Contributor 创建返回 queued；权限和 KB 范围复用现有诊断校验；列表与详情保持 bank scope。
- [x] 编写 worker 失败测试：queued 进入 running/completed；案例级失败继续；整体失败保存错误；终态重入幂等跳过。
- [x] 编写 handler/router 失败测试：POST、GET list、GET detail 使用 Contributor；POST 返回 HTTP 202。
- [x] 新增 ExamRAGEvaluationService，复用 ExamRAGDiagnosticService 的题库、KB 和默认案例解析能力。
- [x] 新增 exam:rag_evaluation_run payload 和 question queue worker，并注册 Redis/Lite 两种执行器。
- [x] 新增 handler、路由和 container 注入。
- [x] 运行 service、handler、router 和任务注册测试。

验证命令：

    go test ./internal/application/service ./internal/handler ./internal/router -run "ExamRAGEvaluation|RAGEvaluationTask" -count=1

## 任务 5：题库内观测页面

- [x] 编写前端失败测试：API、二级路由、题库入口、运行轮询、指标带、历史选择和 trace 阶段存在。
- [x] 编写 view model 失败测试：状态归一化、指标格式、两次运行配置差异、候选排名展示。
- [x] 运行 Node 定向测试确认失败。
- [x] 新增前端类型、API 和 ragEvaluationViewModel。
- [x] 新增 QuestionBankRAGObservability 页面，覆盖 loading、empty、queued/running、failed、completed。
- [x] 桌面使用紧凑表格，移动端使用平铺条目；长 chunk ID 可换行和复制。
- [x] 在 QuestionBankDetail 增加摘要入口并注册二级路由，不新增一级菜单。
- [x] 运行定向测试、前端全量测试和 build-only。

验证命令：

    node --test src/views/question-bank/ragEvaluationViewModel.test.ts src/views/question-bank/questionBankRAGObservabilitySource.test.ts
    npm run test
    npm run build-only

## 任务 6：集成审查与真实验收

- [x] 运行 gofmt 和迁移静态检查。
- [x] 运行 Go 相关包测试与 go vet。
- [x] 重建 app/frontend，确认迁移到 000080 且五个容器健康。
- [x] 对正式英语题库运行默认评测并在页面展开 vector、keyword、RRF、final 和结构化题组。
- [x] 修改 MatchCount 后重跑，验证历史和双运行配置对比。
- [x] 对数学题库运行默认评测。
- [x] 用非 Contributor 验证创建和读取均被拒绝。
- [x] 刷新页面确认运行历史和详情可恢复。
- [x] 请求独立代码审查；上游审查通道三次中断，随后完成本地逐文件审查并修复空结果崩溃、轮询竞态和历史快照过大问题。
- [x] 运行 git diff --check 和最终工作区检查。

真实验收记录：

- 英语题库默认运行 4/4 完成，结构化解析率 100%，TopK 8/12 历史可对比；存量材料仍缺少 `Upcoming Football Events`。
- 数学题库默认运行 4/4 完成，结构化解析率 100%，3 个知识库均保存 vector、keyword、RRF 和 final trace。
- 列表 API 仅返回指标级快照，4 条英语历史约 7 KB；详情 API 保留完整案例和四阶段 trace。
- 390px 与 1440px 视口均无横向溢出；移动端长案例名与状态标签不重叠。

最终验证命令：

    go test ./internal/application/repository ./internal/application/service ./internal/examrag ./internal/searchutil ./internal/handler ./internal/router -count=1
    go vet ./internal/application/repository ./internal/application/service ./internal/examrag ./internal/searchutil ./internal/handler ./internal/router
    npm --prefix frontend run test
    npm --prefix frontend run build-only
