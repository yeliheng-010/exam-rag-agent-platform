# 题库级 RAG 可观测与评测中心设计

## 目标

把现有“运行一次并显示通过率”的题库 RAG 诊断升级为可追踪、可持久化、可比较的工程评测能力。教师和平台维护者应能回答：

- 查询分别从向量检索和关键词检索召回了什么。
- RRF 融合后为什么保留某个 chunk。
- 检索结果是否成功回链到正式结构化题组。
- 最终上下文来自结构化题组还是没有形成可回答上下文。
- 修改 embedding、阈值或 TopK 后，召回率和答案覆盖是否退化。

本阶段聚焦题库级 RAG。Agent 工具选择、参数正确率和多轮轨迹评测作为下一阶段建立在本模块的 trace 数据之上，不在本次同时实现。

## 当前基础

- ExamRAGDiagnosticService 已能按题库解析知识库范围并运行固定或自定义评测案例。
- ExamQuestionContextResolver 已复用生产路径执行 HybridSearch -> chunk IDs -> QuestionGroupDetail -> StructuredBundle。
- 当前结果已有检索覆盖、答案覆盖、命中/缺失短语和 chunk ID，但没有分阶段候选、耗时、上下文来源枚举和历史记录。
- 当前前端只在题库详情右侧窄栏显示本次结果，无法查看运行历史或比较两次配置。
- 普通 Chat/Agent 路径必须保持现有 KnowledgeBaseService.HybridSearch 行为和返回契约。

## 方案比较

### 方案 A：平台级独立评测中心

提供跨题库、跨 Agent 的统一一级菜单和聚合看板。长期价值高，但当前缺少统一评测集、跨题库权限模型和 Agent 场景数据，首版会产生大量空页面和共享契约。

### 方案 B：题库级持久化评测中心（采用）

在题库详情中提供二级入口，评测运行、历史、对比和 trace 都绑定 question_bank_id。复用现有权限与默认案例，先把单题库的检索链路做深，再向平台级聚合。

### 方案 C：只扩展当前前端诊断卡片

可以快速展示现有 matched_chunk_ids 和 source_chunk_ids，但无法观察向量/关键词/RRF 阶段，也无法保留历史，不能支撑参数回归和工程学习。

采用方案 B。它能提供真实工程闭环，同时避免提前设计跨题库和跨 Agent 的大而空平台。

## 总体架构

    题库观测页
      -> 创建评测运行（202）
      -> exam:rag_evaluation_run 后台任务
      -> ExamRAGEvaluationService
           -> ExamQuestionContextResolver
                -> HybridSearchWithTrace（可选扩展接口）
                     -> vector candidates
                     -> keyword candidates
                     -> RRF/deduplicate candidates
                     -> final chunks
                -> QuestionGroupDetail
                -> StructuredBundle
           -> deterministic metrics
      -> 保存运行快照
      -> 查询历史、运行详情、双运行对比

普通 HybridSearch、Chat pipeline 和 Agent tools 不改变签名。可观测调用通过单独的可选接口获取 trace；未实现 trace 的 mock 或替代服务仍可运行现有诊断，只是不返回分阶段候选。

## 检索 Trace

新增独立、只读的检索 trace 类型：

- query：原始查询。
- knowledge_base_id：本次检索目标。
- parameters：match_count、向量阈值、关键词阈值和启停状态。
- embedding_model_id：实际使用的 embedding 模型。
- embedding_dimensions：查询向量维度，不保存向量内容。
- vector_candidates：chunk ID、原始分数、向量排名。
- keyword_candidates：chunk ID、原始分数、关键词排名。
- fusion_method：rrf、vector_only 或 keyword_only。
- fusion_candidates：chunk ID、融合分数、向量排名、关键词排名、融合排名。
- final_chunks：上下文扩展和 MatchCount 截断后的 chunk ID、分数和排名。
- duration_ms：检索总耗时。

不保存 chunk 全文和 embedding 数组，避免评测表重复存储资料内容或扩大敏感数据面。

KnowledgeBaseSearchTraceService 作为可选接口新增 HybridSearchWithTrace。具体 service 使用 context 内部 recorder 记录现有 HybridSearch 各阶段，普通调用没有 recorder 时不产生额外快照。

## 结构化上下文 Trace

ExamQuestionContextResolveResult 增加：

- ContextSource：structured_question_group 或 none。
- GroupID：成功回链的正式题组。
- CandidateChunkIDs：用于题组回链的候选。
- SourceChunkIDs：结构化题组声明的证据 chunk。
- SearchTraces：每个知识库的检索 trace。
- DurationMs：从检索到结构化上下文完成的总耗时。

题库评测不伪造 raw_chunk_fallback。当前 resolver 的真实生产行为是“成功形成结构化题组上下文”或“没有结构化上下文”；原始 chunk 仍可由通用 Chat 流程消费，但不应在题库结构化评测中被误报成已形成答案上下文。

## 持久化模型

新增迁移 000080_exam_rag_evaluation_runs，创建 exam_rag_evaluation_runs：

- id
- tenant_id
- question_bank_id
- created_by
- status：queued、running、completed、failed
- progress：已完成案例数与总案例数
- request_snapshot：知识库范围、案例和搜索参数
- result_snapshot：汇总、逐案例结果和 trace
- error_message
- started_at
- completed_at
- created_at
- updated_at

索引使用 (tenant_id, question_bank_id, created_at desc)。运行记录是不可变评测快照；重跑会创建新记录，不覆盖旧记录。

保留现有 POST /question-banks/:bank_id/rag-diagnostics，避免破坏已有调用。新观测页面使用：

- POST /question-banks/:bank_id/rag-evaluation-runs
- GET /question-banks/:bank_id/rag-evaluation-runs
- GET /question-banks/:bank_id/rag-evaluation-runs/:run_id

创建返回 HTTP 202。Redis/Asynq 与 Lite 模式注册同一任务处理器。

## 评测配置与指标

运行请求允许设置：

- knowledge_base_ids
- cases
- match_count：1-50，默认 8
- vector_threshold：0-1，默认 0.5
- keyword_threshold：0-1，默认 0.3

每次运行保存实际生效配置。RRF K 和权重读取租户检索配置并写入 trace，本阶段不允许单次运行绕过租户配置。

汇总指标：

- retrieval_hit_rate
- answer_hit_rate
- overall_hit_rate
- recall_at_k：仅对配置了 expected chunk 的案例计算。
- mean_reciprocal_rank：仅对配置了 expected chunk 的案例计算。
- structured_resolution_rate
- average_duration_ms
- failed_case_count

没有 expected chunk 的案例不参与 Recall@K 和 MRR 分母，避免把答案短语案例误算为满分召回。

本阶段不调用 LLM-as-judge。答案覆盖继续使用确定性 required phrases，保证回归可重复、无额外模型费用。

## 权限与隔离

- 路由继续要求 Contributor。
- service 必须通过现有题库读取权限和知识库 CanReadKnowledgeBase 校验。
- repository 的创建、列表和详情查询必须同时带 tenant_id 与 question_bank_id。
- run_id 不得绕过题库作用域直接读取。
- trace 不保存 chunk 正文、API Key、模型凭证或 query embedding。
- 后台任务从持久化记录读取 tenant/user/bank，不接受任务 payload 覆盖身份。

## 前端信息架构

新增题库内二级路由：

    /platform/question-banks/:bankId/rag-observability

题库详情右侧保留紧凑摘要和“进入评测中心”按钮，不新增一级菜单。

评测中心包含：

1. 顶部题库标题、返回按钮和“运行评测”命令。
2. 指标带：总体、召回、答案覆盖、结构化解析率、平均耗时。
3. 运行历史：时间、状态、配置、通过率；最多选择两次运行进行对比。
4. 案例列表：查询、状态、Recall、答案覆盖、上下文来源和耗时。
5. 案例详情：向量、关键词、RRF、最终 chunk、题组回链和缺失证据。

页面使用全宽无嵌套卡片布局。桌面端案例列表使用紧凑表格；移动端切换为平铺条目，长 chunk ID 可换行并提供复制按钮。运行详情、空状态、失败状态和轮询状态必须完整。

## 错误处理

- 无绑定知识库：创建运行返回业务错误，页面显示绑定入口说明。
- 某案例检索失败：记录该案例错误并继续其他案例。
- 整体配置或权限错误：运行进入 failed，保存错误信息。
- worker 重试：只允许 queued/running 记录执行；completed/failed 幂等跳过。
- 页面刷新：通过运行 ID 恢复轮询，不依赖内存状态。
- 两次运行配置不同：对比页显式展示差异，不把指标变化直接归因于模型。

## 测试策略

### 后端

- trace recorder 测试向量、关键词、RRF 和 final 阶段排名。
- resolver 测试结构化题组成功与 none 两种来源。
- 指标测试 Recall@K、MRR、结构化解析率和无 expected chunk 分母。
- repository 测试 tenant/bank 双重隔离。
- service 测试 202 创建、权限拒绝、案例级失败继续和运行状态。
- worker 测试幂等、成功、失败与 Lite/Redis 注册。
- handler/router 测试 Contributor 路由和 run ID 作用域。

### 前端

- API 与路由 source test。
- 运行状态、指标格式和双运行比较 adapter 测试。
- 观测页面 loading、empty、failed、completed 状态测试。
- 案例阶段 trace 展示和移动端 source/style test。

### 真实验收

- 对正式 2026 高考英语题库运行默认案例。
- 对正式 2025 高考数学题库运行默认案例。
- 页面可展开至少一个案例，看到向量、关键词、RRF、最终 chunk 和结构化题组。
- 修改 MatchCount 后重跑并比较两次配置和指标。
- 非 Contributor 不能创建或读取运行。
- 刷新页面后历史和运行详情仍存在。

## 非目标

- 本阶段不做跨题库平台聚合看板。
- 本阶段不做 Agent 自动场景生成、工具选择评分和多轮回放。
- 本阶段不做 LLM-as-judge。
- 本阶段不允许单次评测覆盖租户 RRF 参数。
- 本阶段不修改普通 Chat/Agent 的检索返回契约。

## 后续阶段

在本模块稳定后，Agent 评测复用相同运行模型，新增场景输入、期望工具序列、工具参数断言、证据引用检查和最终回答 groundedness；随后再建设跨题库平台看板。
