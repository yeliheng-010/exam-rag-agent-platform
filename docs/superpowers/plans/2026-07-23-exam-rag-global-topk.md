# 考试 RAG 全局 Top-K 与严格评测实现计划

> **面向 AI 代理的工作者：** 使用 `superpowers:executing-plans` 串行执行；检索契约、评测 DTO 与前端快照有顺序依赖，不派发并行写任务。本轮沿用用户此前要求，不 commit、不 push。

**目标：** 将考试 RAG 诊断修正为跨知识库全局 Top-K，并分别报告严格最终召回与候选全集召回。

**架构：** resolver 把每库结果转换为带主排名、ID 别名和父块上下文的结构化项，再按局部排名稳定融合。评测层只使用结构化项计算排名指标，兼容数组仅作为派生数据保留；候选指标以可选字段持久化，保证旧快照仍能表达“未记录”。

**技术栈：** Go、Vue 3、TypeScript、Node test、Docker Compose、现有考试 RAG 评测运行。

---

## 文件结构

- 修改 `internal/examrag/context_resolver_search.go`：建立每库排名项、父块附加和全局稳定融合。
- 修改 `internal/examrag/context_resolver_search_test.go`：覆盖跨库交错、稳定 tie-break、去重和父块不占 K。
- 修改 `internal/examrag/context_resolver.go`：在 resolve 结果中保存最终项和候选项，并派生兼容数组。
- 修改 `internal/examrag/context_resolver_evaluation.go`：把结构化项传入共享评测。
- 修改 `internal/searchutil/exam_context_eval.go`：定义排名项及严格/候选汇总字段。
- 修改 `internal/searchutil/exam_context_eval_helpers.go`：按排名项匹配 ID 与短语。
- 修改 `internal/searchutil/exam_context_retrieval_eval_test.go`：覆盖 K 外命中、父块同排名和候选指标。
- 修改 `internal/types/exam_rag_diagnostic.go`：增加候选指标及结构化排名项 DTO。
- 修改 `internal/application/service/exam_rag_diagnostic.go`：转换并持久化新字段。
- 修改 `internal/application/service/exam_rag_diagnostic_test.go`、`internal/application/service/exam_rag_evaluation_test.go`：覆盖 DTO 转换与旧快照兼容。
- 修改 `frontend/src/types/exam.ts`：增加可选候选指标和排名项类型。
- 修改 `frontend/src/views/question-bank/ragEvaluationViewModel.ts` 及测试：格式化和对比候选指标，缺失时显示“未记录”。
- 修改 `frontend/src/views/question-bank/RAGEvaluationRunDetail.vue`：逐案例区分严格前 K 与候选全集覆盖。
- 创建 `docs/superpowers/reports/2026-07-23-exam-rag-global-topk.md`：记录验证证据与修正后的真实基线。

### 任务 1：锁定全局排名契约

- [x] 在 `context_resolver_search_test.go` 新增测试：两个知识库各返回三个主结果且原始分数刻意冲突，`MatchCount=3` 时最终项应为 `kb-a-1,kb-b-1,kb-a-2`，候选项保留六条。
- [x] 新增测试：同一局部排名按目标顺序和 Chunk ID 稳定排序，重复别名只保留首次出现。
- [x] 新增测试：父块紧跟主块收进同一结构化项，父块不消耗 K，孤立父块被忽略。
- [x] 运行 `go test ./internal/examrag -run 'TestSearchRankedItemsForQuery' -count=1`，确认旧实现因缺少结构化全局 Top-K 而失败。
- [x] 在 `context_resolver_ranking.go` 实现最小排名项构建、去重和稳定融合。
- [x] 在 `context_resolver.go` 派生最终/候选兼容数组并更新现有 resolver 断言。
- [x] 重跑 `go test ./internal/examrag -count=1`，确认通过。

### 任务 2：锁定严格评测与候选指标

- [x] 在 `exam_context_retrieval_eval_test.go` 新增测试：金标只在候选第 K+1 条时，`retrieval_passed=false`、`recall_at_k=0`，但 `candidate_retrieval_passed=true`、`candidate_recall=1`。
- [x] 新增测试：父块内容命中短语时，`first_relevant_rank` 等于其主块排名，下一主块排名不被父块推后。
- [x] 新增混合 Chunk ID 与短语金标测试，确认分母、缺失项和 MRR 都只基于结构化主排名。
- [x] 运行 `go test ./internal/searchutil -run TestEvaluateExamContextRetrieval -count=1`，确认新增测试红灯且失败原因是旧全集口径。
- [x] 修改 `exam_context_eval.go` 和 `exam_context_eval_helpers.go`，分别对最终项、候选项应用同一金标匹配函数。
- [x] 保留没有检索金标的案例语义，重跑 `go test ./internal/searchutil -count=1`。

### 任务 3：持久化与历史快照兼容

- [x] 在 service 测试中先断言新运行会写入结构化排名项、`candidate_recall` 和 `candidate_hit_rate`。
- [x] 新增旧 JSON 快照测试：输入缺少候选字段的历史快照，compact 后字段仍缺失而不是变成 `0`。
- [x] 运行 service 定向测试，确认新字段缺失导致红灯。
- [x] 修改 Go DTO 与转换函数；候选汇总字段使用指针和 `omitempty`，新运行显式赋值。
- [x] 重跑 DTO、旧快照测试及三包联合回归。

### 任务 4：前端指标语义

- [x] 在 `ragEvaluationViewModel.test.ts` 新增测试：新快照显示严格 Recall@K 和候选 Recall；旧快照显示“未记录”；对比旧/新快照不把缺失当作 0。
- [x] 运行 `node --test --experimental-strip-types src/views/question-bank/ragEvaluationViewModel.test.ts`，确认新增测试红灯。
- [x] 更新前端类型和 view model，新增候选指标行与可选字段格式化。
- [x] 在详情页显示逐案例“前 K 召回”和“候选召回”，并保留移动端紧凑布局。
- [x] 重跑 view model 测试和页面 source 测试，19/19 通过。

### 任务 5：回归与真实运行

- [x] 运行 `gofmt` 处理本轮 Go 文件。
- [x] 运行 `go test ./internal/examrag ./internal/searchutil ./internal/application/service ./internal/handler ./internal/router ./internal/infrastructure/chunker -count=1`。
- [x] 在 `frontend` 运行定向 Node 测试与 `npm run build-only`；如 `type-check` 仍有既存债务，记录与本轮无关的具体输出。
- [x] 重建 `app` 容器并等待健康检查，记录实际镜像摘要。
- [x] 使用相同 25 条案例重新运行 legacy、auto、auto-repeat，记录严格 Recall@20、候选 Recall、MRR、关联率和答案覆盖率。
- [x] 只读核对 `question_chunk_refs` 仍为 240 行且摘要仍为 `c182a2a0b9fbedc880dd5c85fe4c02de`。
- [x] 在 `1440x1000` 和 `390x844` 检查新旧运行指标、K 外失败案例和逐案例候选信息，确认无横向溢出及控制台 error。
- [x] 完成代码审查并写入 `docs/superpowers/reports/2026-07-23-exam-rag-global-topk.md`。
