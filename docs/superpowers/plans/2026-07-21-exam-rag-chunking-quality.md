# 考试 RAG 切块与检索质量优化实施计划

> **面向 AI 代理的工作者：** 必需子技能：使用 `superpowers:executing-plans` 逐任务实现此计划。每项行为变更严格执行 red -> green -> refactor，并在完成前使用 `superpowers:verification-before-completion`。

**目标：** 修复父子切块绕过自适应策略的问题，建立可归因的切块诊断、稳定检索金标和不可变评测快照，并在评测中心完成对比展示与隔离 A/B 验证。

**架构：** 生产切块统一通过 service 层执行器调用 chunker，并把实际 tier、回退原因和大小分布写入现有 processing span JSON。评测运行创建时由独立、tenant-scoped 的质量仓储聚合知识库配置、chunk 统计和最近处理诊断，将快照固化到 request snapshot；检索指标同时支持 chunk ID 和内容短语金标。正式知识库不直接重切，真实验证只使用隔离副本。

**技术栈：** Go、GORM、PostgreSQL/SQLite tests、Vue 3、TypeScript、Node test runner、Docker Compose。

**授权边界：** 不新增数据库迁移，不修改正式知识库数据，不覆盖 `docker-compose.yml` 与学习中心现有修改，不执行 `commit`、`push` 或其他 Git 历史操作。

---

## 文件职责

- `internal/infrastructure/chunker/strategy.go`：提供保留兼容 API 的父子切块诊断入口。
- `internal/infrastructure/chunker/strategy_diagnostics_test.go`：定义 parent tier、child tier 分布和回退诊断语义。
- `internal/application/service/chunking_execution.go`：统一 flat/parent-child 切块执行、类型转换和大小统计。
- `internal/application/service/chunking_execution_test.go`：覆盖配置透传、token limit、统计阈值和执行结果。
- `internal/application/service/knowledge_process.go`：调用共享执行器，并把有效配置/诊断写入 chunking span。
- `internal/application/service/knowledge_create.go`：手工知识复用共享执行器。
- `internal/application/service/knowledge_process_config_test.go`：锁定父子配置继承策略、语言和子块 token limit。
- `internal/searchutil/exam_context_eval.go`：实现短语检索金标、正确 Recall@K/MRR 和空金标分母语义。
- `internal/searchutil/exam_context_retrieval_eval_test.go`：覆盖 ID/短语/混合金标与无金标案例。
- `internal/types/exam_rag_diagnostic.go`：扩展请求案例和结果中的短语命中信息。
- `internal/application/service/exam_rag_diagnostic.go`：规范化并映射新金标字段。
- `internal/application/service/exam_rag_diagnostic_test.go`：覆盖字段清洗和类型映射。
- `internal/types/exam_rag_chunking.go`：定义切块配置及质量快照的稳定 JSON contract。
- `internal/types/interfaces/exam_rag_chunk_quality.go`：定义 tenant-scoped 聚合仓储接口。
- `internal/application/repository/exam_rag_chunk_quality.go`：从 knowledge_bases、knowledge、chunks 和 processing spans 聚合快照。
- `internal/application/repository/exam_rag_chunk_quality_test.go`：覆盖租户隔离、分位数、parent coverage、旧 span 与 unknown tier。
- `internal/application/service/exam_rag_evaluation.go`：创建运行时固化切块快照并保持列表压缩兼容。
- `internal/application/service/exam_rag_evaluation_task.go`：解析带新字段或旧字段的 request snapshot。
- `internal/application/service/exam_rag_evaluation_test.go`：覆盖不可变快照和 compact list。
- `internal/application/service/exam_rag_evaluation_task_test.go`：覆盖旧 snapshot 的 worker 兼容。
- `internal/container/container.go`：注册新的质量仓储依赖。
- `frontend/src/types/exam.ts`：同步短语金标与切块快照类型。
- `frontend/src/views/question-bank/ragEvaluationViewModel.ts`：格式化无金标指标、快照和运行差异。
- `frontend/src/views/question-bank/ragEvaluationViewModel.test.ts`：覆盖展示/比较逻辑。
- `frontend/src/views/question-bank/RAGEvaluationRunDetail.vue`：展示切块快照、短语命中和无金标状态。
- `frontend/src/views/question-bank/QuestionBankRAGObservability.vue`：运行比较区域加入切块配置/质量差异。
- `frontend/src/views/question-bank/questionBankRAGObservabilitySource.test.ts`：锁定页面展示入口与响应式布局。

### 任务 1：父子切块配置透传与 chunker 诊断

**文件：**
- 修改：`internal/application/service/knowledge_process.go:228`
- 修改：`internal/infrastructure/chunker/strategy.go:92`
- 修改：`internal/infrastructure/chunker/strategy_diagnostics_test.go`
- 修改：`internal/application/service/knowledge_process_config_test.go`

- [x] **步骤 1：编写父子配置透传失败测试**

测试使用 `strategy=auto`、`languages=[en]`、`token_limit=256`，要求父级继承 strategy/languages 但不继承 token limit，子级继承三者：

```go
parent, child := buildParentChildConfigs(cc, buildSplitterConfigFromChunking(cc))
require.Equal(t, chunker.StrategyAuto, parent.Strategy)
require.Equal(t, []string{"en"}, parent.Languages)
require.Zero(t, parent.TokenLimit)
require.Equal(t, chunker.StrategyAuto, child.Strategy)
require.Equal(t, []string{"en"}, child.Languages)
require.Equal(t, 256, child.TokenLimit)
```

- [x] **步骤 2：运行红测**

运行：`go test ./internal/application/service -run TestBuildParentChildConfigs -count=1`

预期：FAIL，当前 `parent.Strategy` / `child.Strategy` 为空。

- [x] **步骤 3：最小修复配置透传**

```go
parent.Strategy, parent.Languages = base.Strategy, append([]string{}, base.Languages...)
child.Strategy, child.Languages = base.Strategy, append([]string{}, base.Languages...)
child.TokenLimit = base.TokenLimit
```

- [x] **步骤 4：编写父子诊断失败测试**

为 `SplitParentChildWithDiagnostics` 定义：

```go
type ParentChildDiagnostics struct {
    Parent       *Diagnostics          `json:"parent"`
    ChildTiers   map[StrategyTier]int  `json:"child_tiers"`
    Rejected     []TierRejection       `json:"rejected"`
}
```

测试要求 parent 记录唯一 selected tier，所有 child split 的 selected tier 计数之和等于实际参与分割的 parent 数，并保留 child 回退原因。

- [x] **步骤 5：运行 chunker 红测**

运行：`go test ./internal/infrastructure/chunker -run TestSplitParentChildWithDiagnostics -count=1`

预期：FAIL，诊断入口尚不存在。

- [x] **步骤 6：实现兼容诊断入口**

保留 `SplitParentChild(text, parentCfg, childCfg)`，内部委托：

```go
func SplitParentChild(text string, parentCfg, childCfg SplitterConfig) ParentChildResult {
    result, _ := SplitParentChildWithDiagnostics(text, parentCfg, childCfg)
    return result
}
```

诊断入口对 parent 和每次 child 调用 `SplitWithDiagnostics`，合并 child tier/rejections；空文本返回非 nil diagnostics 且默认 parent tier 为 legacy。

- [x] **步骤 7：运行定向测试**

运行：`go test ./internal/infrastructure/chunker ./internal/application/service -run 'TestSplitParentChildWithDiagnostics|TestBuildParentChildConfigs' -count=1`

预期：PASS。

### 任务 2：共享切块执行器与 processing span 输出

**文件：**
- 创建：`internal/application/service/chunking_execution.go`
- 创建：`internal/application/service/chunking_execution_test.go`
- 修改：`internal/application/service/knowledge_process.go:154`
- 修改：`internal/application/service/knowledge_process.go:3021`
- 修改：`internal/application/service/knowledge_create.go:1143`

- [x] **步骤 1：编写执行器结果与统计红测**

定义稳定 service 类型：

```go
type ChunkingExecutionResult struct {
    Chunks       []types.ParsedChunk
    ParentChunks []types.ParsedParentChunk
    Diagnostics  ChunkingExecutionDiagnostics
}

type ChunkingExecutionDiagnostics struct {
    RequestedStrategy string                       `json:"requested_strategy"`
    ParentTier        string                       `json:"parent_tier,omitempty"`
    ChildTierCounts   map[string]int               `json:"child_tier_counts,omitempty"`
    Rejections        []chunker.TierRejection      `json:"rejections,omitempty"`
    TextChunkCount    int                          `json:"text_chunk_count"`
    ParentChunkCount  int                          `json:"parent_chunk_count"`
    MinChars          int                          `json:"min_chars"`
    P50Chars          int                          `json:"p50_chars"`
    P90Chars          int                          `json:"p90_chars"`
    MaxChars          int                          `json:"max_chars"`
    TinyChunkCount    int                          `json:"tiny_chunk_count"`
    OversizeCount     int                          `json:"oversize_count"`
}
```

测试 flat 与 parent-child 两条路径，分位数采用排序后的 nearest-rank，tiny 为目标 child/flat size 的 25%，oversize 为大于目标 size，空白 chunk 不进入统计。

- [x] **步骤 2：运行执行器红测**

运行：`go test ./internal/application/service -run TestExecuteChunking -count=1`

预期：FAIL，执行器尚不存在。

- [x] **步骤 3：实现共享执行器**

```go
func executeChunking(markdown string, cc types.ChunkingConfig) ChunkingExecutionResult {
    base := buildSplitterConfigFromChunking(cc)
    if !cc.EnableParentChild {
        chunks, diag := chunker.SplitWithDiagnostics(markdown, base)
        return buildFlatExecutionResult(chunks, diag, base)
    }
    parentCfg, childCfg := buildParentChildConfigs(cc, base)
    split, diag := chunker.SplitParentChildWithDiagnostics(markdown, parentCfg, childCfg)
    return buildParentChildExecutionResult(split, diag, childCfg)
}
```

转换函数保持 `Content/ContextHeader/Seq/Start/End/ParentIndex` 不变；统计只看 embedding text chunks，父块单独计数。

- [x] **步骤 4：替换两处重复分支并传递 diagnostics**

给 `ProcessChunksOptions` 增加：

```go
ChunkingConfig      types.ChunkingConfig
ChunkingDiagnostics ChunkingExecutionDiagnostics
```

`knowledge_process.go` 和 `knowledge_create.go` 都调用 `executeChunking`，将 `ParentChunks` 和 diagnostics 传给 `processChunks`。

- [x] **步骤 5：把有效配置和诊断写入 chunking span**

`beginStage` input 保存有效配置摘要；`endStage` output 合并诊断和实际落库数量。禁止写原文/chunk 内容：

```go
input := chunkingSpanInput(options.ChunkingConfig)
input["chunks_planned"] = len(insertChunks)
s.beginStage(ctx, knowledge.ID, types.StageChunking, input)

output := chunkingSpanOutput(options.ChunkingDiagnostics)
output["chunks_written"] = len(insertChunks)
s.endStage(ctx, knowledge.ID, types.StageChunking, output)
```

- [x] **步骤 6：运行 service 回归测试**

运行：`go test ./internal/application/service -run 'TestExecuteChunking|TestBuildParentChildConfigs|Test.*Manual.*Chunk|Test.*Process.*Chunk' -count=1`

预期：PASS，且现有创建/处理测试不改变 chunk 数量或 parent 关系。

### 任务 3：稳定检索短语金标与真实 Recall@K/MRR

**文件：**
- 修改：`internal/searchutil/exam_context_eval.go`
- 修改：`internal/searchutil/exam_context_retrieval_eval_test.go`
- 修改：`internal/types/exam_rag_diagnostic.go`
- 修改：`internal/application/service/exam_rag_diagnostic.go`
- 修改：`internal/application/service/exam_rag_diagnostic_test.go`
- 修改：`frontend/src/types/exam.ts`

- [x] **步骤 1：编写无检索金标红测**

案例只有 `RequiredPhrases` 时，要求 `RankedCaseCount=0`、`RetrievalPassed=0`，但 `Passed` 只由 answer coverage 决定：

```go
require.Zero(t, summary.RankedCaseCount)
require.Zero(t, summary.RetrievalPassed)
require.True(t, summary.Results[0].AnswerPassed)
require.True(t, summary.Results[0].Passed)
```

- [x] **步骤 2：编写短语金标红测**

给 resolution 新增仅本次评测临时使用的 `RetrievedContents []string`，案例新增 `RequiredRetrievalPhrases`。要求内容匹配按 top-K 顺序计算覆盖率和首个命中的 reciprocal rank，持久化结果只保留命中/缺失短语及排名，不保留内容。

- [x] **步骤 3：运行 searchutil 红测**

运行：`go test ./internal/searchutil -run 'TestEvaluateExamContextRetrieval.*Gold|TestEvaluateExamContextRetrieval.*NoGold' -count=1`

预期：FAIL，当前空 `ExpectedChunkIDs` 自动通过且无短语字段。

- [x] **步骤 4：实现统一金标匹配**

```go
hasRetrievalGold := len(ids) > 0 || len(phrases) > 0
result.RetrievalPassed = hasRetrievalGold && len(missingIDs) == 0 && len(missingPhrases) == 0
result.Passed = result.AnswerPassed && (!hasRetrievalGold || result.RetrievalPassed)
```

Recall@K 分子/分母按所有 ID 与短语金标项累计；MRR 取 top-K 中首个命中任一 ID/短语的位置。`RetrievalHitRate` 的分母改为 `rankedCaseCount`。

- [x] **步骤 5：贯通 public types 与 diagnostic 映射**

新增字段：

```go
RequiredRetrievalPhrases []string `json:"required_retrieval_phrases"`
MatchedRetrievalPhrases  []string `json:"matched_retrieval_phrases"`
MissingRetrievalPhrases  []string `json:"missing_retrieval_phrases"`
FirstRelevantRank         int      `json:"first_relevant_rank"`
```

`buildExamRAGDiagnosticCases` 清洗新字段，`toExamRAGDiagnosticEvalCases` / `toExamRAGDiagnosticSummary` 完整映射。

- [x] **步骤 6：运行检索与诊断回归**

运行：`go test ./internal/searchutil ./internal/examrag ./internal/application/service -run 'TestEvaluateExamContextRetrieval|TestExamRAGDiagnostic|TestBuildExamRAGDiagnosticCases' -count=1`

预期：PASS。

### 任务 4：tenant-scoped 切块质量仓储

**文件：**
- 创建：`internal/types/exam_rag_chunking.go`
- 创建：`internal/types/interfaces/exam_rag_chunk_quality.go`
- 创建：`internal/application/repository/exam_rag_chunk_quality.go`
- 创建：`internal/application/repository/exam_rag_chunk_quality_test.go`
- 修改：`internal/container/container.go:173`

- [x] **步骤 1：定义快照 contract 与仓储红测**

```go
type ExamRAGChunkingSnapshot struct {
    KnowledgeBaseID  string                 `json:"knowledge_base_id"`
    Config           ChunkingConfigSnapshot `json:"config"`
    KnowledgeCount   int                    `json:"knowledge_count"`
    TextChunkCount   int                    `json:"text_chunk_count"`
    ParentChunkCount int                    `json:"parent_chunk_count"`
    MinChars         int                    `json:"min_chars"`
    P50Chars         int                    `json:"p50_chars"`
    P90Chars         int                    `json:"p90_chars"`
    MaxChars         int                    `json:"max_chars"`
    TinyChunkRate    float64                `json:"tiny_chunk_rate"`
    OversizeRate     float64                `json:"oversize_rate"`
    ParentCoverage   float64                `json:"parent_coverage"`
    ActualTierCounts map[string]int         `json:"actual_tier_counts"`
    UnknownTierCount int                    `json:"unknown_tier_count"`
}
```

SQLite fixture 插入两个 tenant 的同名 KB、text/parent/image chunks、带/不带 diagnostics 的最近 chunking span，要求只聚合传入 tenant + KB IDs。

- [x] **步骤 2：运行仓储红测**

运行：`go test ./internal/application/repository -run TestExamRAGChunkQualityRepository -count=1`

预期：FAIL，仓储尚不存在。

- [x] **步骤 3：实现最小 tenant-scoped 聚合**

接口仅暴露评测所需方法：

```go
type ExamRAGChunkQualityRepository interface {
    Snapshot(ctx context.Context, tenantID uint64, knowledgeBaseIDs []string) ([]types.ExamRAGChunkingSnapshot, error)
}
```

先校验 `knowledge_bases.tenant_id`，再查询其 knowledge/chunks；只统计 `text` 与 `parent_text`。实际 tier 读取每个 knowledge 最近 attempt 的 chunking span output；缺字段计入 unknown，不反推 tier。所有 SQL 使用参数绑定。

- [x] **步骤 4：注册依赖并运行仓储测试**

运行：`go test ./internal/application/repository -run TestExamRAGChunkQualityRepository -count=1`

预期：PASS，包括跨租户不可见、旧 span unknown 和 parent coverage。

### 任务 5：评测运行固化不可变切块快照

**文件：**
- 修改：`internal/application/service/exam_rag_evaluation.go`
- 修改：`internal/application/service/exam_rag_evaluation_task.go`
- 修改：`internal/application/service/exam_rag_evaluation_test.go`
- 修改：`internal/application/service/exam_rag_evaluation_task_test.go`
- 修改：`internal/types/interfaces/exam_rag_evaluation.go`

- [x] **步骤 1：编写创建运行快照红测**

注入质量仓储 stub，要求 `CreateRun` 只调用一次 `Snapshot(tenantID, normalized KB IDs)`，并把结果固化在 request snapshot：

```go
type examRAGEvaluationRequestSnapshot struct {
    types.RunExamRAGDiagnosticRequest
    UsedDefaultCases  bool                           `json:"used_default_cases"`
    ChunkingSnapshots []types.ExamRAGChunkingSnapshot `json:"chunking_snapshots,omitempty"`
}
```

后续修改 stub 返回值不应改变已创建 run 的 JSON。

- [x] **步骤 2：运行 service 红测**

运行：`go test ./internal/application/service -run 'TestExamRAGEvaluationService.*ChunkingSnapshot|TestParseRAGEvaluationRequest.*Legacy' -count=1`

预期：FAIL，service 尚未依赖质量仓储。

- [x] **步骤 3：扩展构造器并固化快照**

`NewExamRAGEvaluationService` 新增 `chunkQuality interfaces.ExamRAGChunkQualityRepository`。`CreateRun` 在准备完成、创建 DB row 之前获取快照；仓储错误阻止排队，避免生成缺乏归因信息的新运行。

- [x] **步骤 4：保持 worker/list 兼容**

`parseRAGEvaluationRequest` 继续只返回 diagnostic request 和 `UsedDefaultCases`，忽略新快照；旧 JSON 缺 `chunking_snapshots` 时正常解析为空。`compactRAGEvaluationRuns` 保留 request snapshot 中的快照，只压缩 result 的案例详情。

- [x] **步骤 5：运行评测 service/handler/router 回归**

运行：`go test ./internal/application/service ./internal/handler ./internal/router -run 'TestExamRAGEvaluation|TestExamEvaluationCenter' -count=1`

预期：PASS。

### 任务 6：评测中心切块快照展示与比较

**文件：**
- 修改：`frontend/src/types/exam.ts`
- 修改：`frontend/src/views/question-bank/ragEvaluationViewModel.ts`
- 修改：`frontend/src/views/question-bank/ragEvaluationViewModel.test.ts`
- 修改：`frontend/src/views/question-bank/RAGEvaluationRunDetail.vue`
- 修改：`frontend/src/views/question-bank/QuestionBankRAGObservability.vue`
- 修改：`frontend/src/views/question-bank/questionBankRAGObservabilitySource.test.ts`

- [x] **步骤 1：编写 view-model 红测**

覆盖：

```ts
assert.equal(formatRankedRate(0.8, 0), '无检索金标')
assert.equal(formatRankedRate(0.8, 3), '80%')
assert.equal(getChunkingSnapshotRows(run)[0].requestedStrategy, 'auto')
assert.equal(compareChunkingSnapshots(first, second).some(row => row.changed), true)
```

旧运行没有 snapshot 时显示 `未记录`，不得显示虚构的 legacy tier。

- [x] **步骤 2：运行前端红测**

运行：`node --test --experimental-strip-types frontend/src/views/question-bank/ragEvaluationViewModel.test.ts frontend/src/views/question-bank/questionBankRAGObservabilitySource.test.ts`

预期：FAIL，新 helper/文案/区域尚不存在。

- [x] **步骤 3：实现类型与纯函数**

新增 `ExamRAGChunkingSnapshot` / `ChunkingConfigSnapshot` / 扩展 request snapshot 类型。展示每个 KB 的 requested strategy、actual tier counts、chunk/parent 数量、P50/P90、tiny/oversize rate、parent coverage；比较只标变化，不推断因果。

- [x] **步骤 4：实现详情与比较 UI**

详情使用紧凑表格，移动端改为纵向键值列表；`ranked_case_count=0` 时召回率、Recall@K、MRR 均显示“无检索金标”。案例详情显示检索短语缺失项和 first relevant rank。

- [x] **步骤 5：运行前端测试与构建**

运行：`node --test --experimental-strip-types frontend/src/views/question-bank/*.test.ts frontend/src/views/evaluation/*.test.ts`

运行：`npm run build-only`

工作目录：`frontend`

预期：测试 PASS，构建 exit 0。`npm run type-check` 若仍受已有全局类型债影响，单独记录与本次变更无关的错误。

### 任务 7：全量验证、Docker 重建与真实隔离 A/B

**文件：**
- 更新：`docs/superpowers/specs/2026-07-21-exam-rag-chunking-quality-design.md`（仅记录实测结论与偏差）
- 创建：`docs/superpowers/reports/2026-07-21-exam-rag-chunking-ab.md`

- [x] **步骤 1：运行格式化和静态差异检查**

运行：`gofmt -w <本计划新增或修改的 Go 文件>`

运行：`git diff --check`

预期：无格式/空白错误，且 `docker-compose.yml` 与学习中心文件未被本任务改写。

- [x] **步骤 2：运行 Go 全量相关测试**

运行：`go test ./internal/infrastructure/chunker ./internal/searchutil ./internal/examrag ./internal/application/repository ./internal/application/service ./internal/handler ./internal/router -count=1`

预期：PASS。

- [x] **步骤 3：重建本地 app/frontend 并验证健康**

运行：`docker compose up -d --build app frontend`

运行：`docker compose ps`

预期：app healthy，frontend running；不修改 compose 配置。

- [x] **步骤 4：创建隔离 A/B 副本**

通过现有 UI/API 从四个正式 KB 的代表文档创建临时副本，命名包含 `rag-ab-20260721`。基线副本显式 `legacy`，候选副本使用修复后的 `auto`，其他参数保持 `512/80 + parent 4096/child 384`。不得对正式 KB 调用 reparse。

- [x] **步骤 5：建立至少 20 个真实检索短语金标案例**

英语、DOCX 数学、PDF 数学均覆盖题号、选项、答案/解析、公式/表格/图片邻接与跨页边界。金标短语从隔离文档原始内容核对；解析源缺字案例标记为 parser/data issue，不计为 chunker 回归。

- [x] **步骤 6：运行基线与候选评测并核对三层证据**

API：创建/读取两次 RAG evaluation run，确认 request snapshot 不变、`ranked_case_count >= 20`。

数据库：核对 processing span 中 requested strategy 与 actual parent/child tier、chunk 分布和 unknown 数量。

浏览器：评测中心打开两次运行，确认切块快照、无金标提示、配置/指标差异在桌面和移动宽度均无重叠。

- [x] **步骤 7：输出 A/B 报告并决定是否推广**

报告记录 Recall@K、MRR、answer coverage、structured resolution、chunk 数量与运行耗时。只有候选同时满足 `ranked cases >= 20`、`Recall@K >= 0.90`、`MRR >= 0.80`、`structured resolution >= 0.95`、`answer coverage >= 0.90` 且无边界破坏时，才提出正式 KB 配置变更建议；本计划不自动应用。

- [x] **步骤 8：最终完成门禁**

重新运行所有关键验证命令，读取最新输出，检查 `git status --short` 和 `git diff --stat`，逐项对照本计划与设计规格后再声明阶段完成。
