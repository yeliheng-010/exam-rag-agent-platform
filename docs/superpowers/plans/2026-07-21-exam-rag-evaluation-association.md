# 评测副本题组动态关联实现计划

> **面向 AI 代理的工作者：** 在当前已批准方案和现有功能分支内使用 `superpowers:executing-plans` 执行；每一步按 TDD 红-绿循环推进。本轮用户明确要求不 commit、不 push。

**目标：** 为评测副本建立只读、可解释的题组动态关联，并补齐 25 条答案证据金标。

**架构：** repository 依据评测知识文件哈希收敛正式题库候选；`examrag` 使用稳定文本锚点做唯一匹配；现有评测结果契约携带来源与置信度，前端负责可读展示。普通解析路径不启用 fallback。

**技术栈：** Go、GORM、PostgreSQL/SQLite 测试、Vue 3、TypeScript、Node test、Docker Compose。

---

## 文件结构

- 创建 `internal/application/repository/exam_question_evaluation.go`：按评测知识文件哈希加载题组候选。
- 创建 `internal/application/repository/exam_question_evaluation_test.go`：验证租户、文件哈希和相关题库收敛。
- 创建 `internal/examrag/evaluation_anchor_match.go`：纯文本归一化、锚点计分、唯一性和置信度。
- 创建 `internal/examrag/evaluation_anchor_match_test.go`：唯一、歧义、弱证据和中英文格式差异。
- 修改 `internal/examrag/context_resolver.go`：只在 `EvalResolver` 精确关联失败后调用 fallback。
- 修改 `internal/examrag/context_resolver_test.go`：验证 evaluation-only 行为。
- 修改 `internal/types/exam_rag_diagnostic.go`、`internal/searchutil/exam_context_eval.go`、`internal/application/service/exam_rag_diagnostic.go`：传递关联来源和置信度并统计结构化解析率。
- 修改 `docs/superpowers/reports/2026-07-21-exam-rag-chunking-cases.json`：补齐 25 条答案证据金标。
- 修改 `frontend/src/types/exam.ts`、`frontend/src/views/question-bank/ragEvaluationViewModel.ts`、`frontend/src/views/question-bank/RAGEvaluationRunDetail.vue`：展示关联类型和置信度。

### 任务 1：候选题组只读收敛

- [x] 在 repository 测试中创建正式知识、评测副本知识、来源 Chunk、题组和引用，调用：

```go
candidates, err := repo.ListEvaluationQuestionGroupCandidates(ctx, 10000, []string{"eval-kb"})
require.NoError(t, err)
require.Equal(t, []string{"group-source"}, candidateGroupIDs(candidates))
```

- [x] 运行 `go test ./internal/application/repository -run TestExamQuestionRepository_ListEvaluationQuestionGroupCandidates -count=1`，确认因方法不存在而失败。
- [x] 实现按 `tenant_id + file_hash` 查相关题库并加载题组详情；空 hash 或无正式引用返回空数组。
- [x] 重跑定向测试，确认通过。

### 任务 2：稳定锚点唯一匹配

- [x] 写失败测试，覆盖唯一英语段落、中文 Markdown/空白差异、并列候选、弱证据。
- [x] 运行 `go test ./internal/examrag -run TestMatchEvaluationQuestionGroup -count=1`，确认缺少实现导致失败。
- [x] 实现以下内部入口及最少辅助函数：

```go
func matchEvaluationQuestionGroup(
    query string,
    retrievedContents []string,
    candidates []*types.QuestionGroupDetail,
) (*types.QuestionGroupDetail, float64)
```

- [x] 重跑定向测试，确认唯一候选通过、歧义和弱证据拒绝。

### 任务 3：接入 EvalResolver 和指标契约

- [x] 先写 resolver 失败测试：普通 `Resolve` 不 fallback，`EvalResolver` 在精确 ID 失败后返回 `evaluation_anchor_match` 和置信度。
- [x] 写 searchutil 失败测试：动态关联计入结构化解析率，并原样传递置信度。
- [x] 运行 `go test ./internal/examrag ./internal/searchutil -run "EvaluationAnchor|DynamicAssociation" -count=1`，确认预期失败。
- [x] 增加 `ExamRAGContextSourceEvaluationAnchorMatch`、`AssociationConfidence` 字段，并在服务 DTO 映射中传递。
- [x] 在 `EvalResolver` 中调用可选 repository 能力，复用现有 bundle 构建逻辑，不修改 `Resolve` 的正式行为。
- [x] 重跑 `go test ./internal/examrag ./internal/searchutil ./internal/application/service -count=1`。

### 任务 4：补齐金标并展示关联来源

- [x] 从正式 `question_answers` / `question_explanations` 只读核对 25 条案例，更新每条 `required_phrases`。
- [x] 写前端失败测试，期望 `evaluation_anchor_match` 显示“评测动态关联”，并格式化置信度。
- [x] 运行 `node --test --experimental-strip-types frontend/src/views/question-bank/ragEvaluationViewModel.test.ts`，确认失败。
- [x] 实现 `formatContextSource`、`formatAssociationConfidence`，更新类型和详情页展示。
- [x] 重跑前端定向测试及 `npm run build-only`。

### 任务 5：端到端验收

- [x] 记录正式 `question_chunk_refs` 的行数与稳定摘要。
- [x] 运行 Go 定向测试、相关包回归测试和前端构建。
- [x] 重建 `app` 与 `frontend` 容器，等待健康检查。
- [x] 使用 25 条案例分别运行 legacy 与 auto 联合评测，再重复 auto 一次。
- [x] 通过 API 和数据库核对结构化解析率、答案覆盖率、来源分布、置信度和重复稳定性。
- [x] 在浏览器检查运行详情能区分精确关联、动态关联、未关联且无文本重叠。
- [x] 再次读取 `question_chunk_refs` 行数与摘要，确认正式引用未变化。

## 实际结果

- 真实运行：legacy `ac2810ef-a5d9-43f2-8a29-f8b7eeadf97a`，auto `aa1162e3-5c48-448f-80cb-8f8e9720ed44`，auto-repeat `7a1f25d8-f942-4d59-a718-84a398557c06`。
- resolver 文件拆分后的最终镜像为 `sha256:16076126b51b2e479d80d17f837c1958c2ce8ff386982c9391222dee71870611`；该镜像上的 auto smoke `c002b052-9ed2-40ed-898b-99d914755388` 仍为 `0.96 / 0.84`。
- 三次运行 Recall@20 均为 `0.96`，结构化解析率均为 `0.96`；auto 两次答案覆盖率均为 `0.84`。
- auto 两次核心逐案例结果签名均为 `2be08db0fde2ad64e4e6a2dc5bd32633`。同分检索 Chunk 的返回顺序仍可能变化，但关联题组、置信度和评测指标一致。
- 正式 `question_chunk_refs` 前后均为 240 行；按 `question_id|chunk_id|ref_type|confidence` 排序计算的摘要前后均为 `c182a2a0b9fbedc880dd5c85fe4c02de`。
- 浏览器在 `1440x1000` 与 `390x844` 验收通过，无横向溢出和控制台错误；Q9、Q12 展开后显示正确的动态关联题组。
