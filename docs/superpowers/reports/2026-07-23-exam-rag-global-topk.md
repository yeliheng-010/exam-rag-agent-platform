# 考试 RAG 全局 Top-K 与严格评测报告

## 结论

方案 A 已完成：跨知识库检索先按各库局部排名稳定融合，再截取全局 Top-K；最终 Top-K 与候选全集分别持久化和评测。Recall@K、MRR、`retrieval_passed` 仅使用最终 Top-K，候选全集只进入候选指标。

本轮未执行数据库迁移，未修改正式题库、知识库或 `question_chunk_refs`，未推广 `auto`，未 commit、未 push。

## 实现契约

- 排序键为 `LocalRank -> 知识库目标顺序 -> 主 Chunk ID`，不跨库比较不可比的原始分数。
- 主 Chunk 占一个全局排名；SubChunk、父块 ID 与父块内容作为该排名的别名和上下文附件。
- 去重身份只包含主 Chunk ID 与 SubChunk ID。共享同一父块的兄弟主块均会保留。
- `RetrievedItems` 只保存最终 Top-K，`CandidateItems` 保存去重后的候选全集。
- 旧 JSON 快照缺少候选字段时继续表达为“未记录”，不会被解释为 0。
- 前端显示总体、Top-K 命中率、Recall@K、MRR、候选命中率、候选 Recall、答案覆盖、结构化解析率和平均耗时共 9 项指标。

## 审查修复

代码审查发现两个缺口并已修复：

1. 父块曾参与候选身份去重，两个主 Chunk 共享父块时会丢弃第二个主 Chunk。新增 `TestSearchRankedItemsForQueryKeepsSiblingChildrenThatShareAParent`，改为独立维护主块/SubChunk 身份集合。
2. 前端排名格式化函数要求完整 `ExamRAGDiagnosticResultItem`，导致 3 条测试类型错误。参数已收窄为实际使用的 5 个检索排名字段。

收尾审查还将检索评测汇总和单案例映射拆到 `exam_context_retrieval_eval.go`。相关 Go 文件均低于 300 行，函数均低于 50 行，行为由完整 `internal/searchutil` 回归锁定。

## 真实评测

题库：`2358b54f-ce60-4965-b1ad-0d6ce23a497b`

| 运行 | Run ID | Recall@20 | 候选 Recall | Top-K 命中率 | 候选命中率 | MRR | 结构化解析率 | 答案覆盖 |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| Legacy | `23b4fb23-ec92-4e1f-ad43-11e200f73231` | 1 | 1 | 1 | 1 | 0.4871111111 | 1 | 1 |
| Auto | `d1c35142-55eb-40dd-9971-85fa4817fd55` | 1 | 1 | 1 | 1 | 0.5444444444 | 1 | 1 |
| Auto Repeat | `2fdf551c-3ad8-40a4-9f66-73499c4ea5fb` | 1 | 1 | 1 | 1 | 0.5444444444 | 1 | 1 |

Auto 两次逐案例核心输出 SHA-256 相同：

```text
bd7874d60c1b1eea9f9df193d9b1d0fe20e8594cb06af0525c91befd47503c32
```

这 25 条案例没有出现“候选命中但最终 Top-K 未命中”的样本，因此真实运行无法展示 K 外失败案例；该语义由 `TestEvaluateExamContextRetrievalSeparatesGlobalTopKFromCandidateRecall` 覆盖。

真实评测运行于核心实现镜像 `sha256:fdd8f5d78c099e20fa05ab7474231ec744c98acaebe7a09d682766019fcd9793`。完成行为保持的函数拆分后，已重建当前源码镜像并通过回归与运行时冒烟：

```text
sha256:0b3efca2445357e89b9037e54260ae7c1f90628ebb772f0d65b66d9606453838
running healthy
```

## 验证证据

### 后端

```text
go test ./internal/examrag ./internal/searchutil ./internal/application/service ./internal/handler ./internal/router ./internal/infrastructure/chunker -count=1
```

六个包均返回 `ok`。`gofmt -d` 为 0 行差异，`git diff --check` 退出码为 0。

### 前端

```text
node --test --experimental-strip-types \
  src/views/question-bank/ragEvaluationViewModel.test.ts \
  src/views/question-bank/questionBankRAGObservabilitySource.test.ts
```

结果为 19/19 通过。`npm run build-only` 成功，仅保留既有动态导入和大包体警告。

`npm run type-check` 仍以退出码 2 报出 101 条全局既有类型错误，但本轮 RAG 文件命中为 0。完整输出保存在：

```text
.artifacts/exam-rag-global-topk-typecheck-final.txt
```

### 数据库

最终只读核验结果：

```text
question_chunk_refs count = 240
question_chunk_refs digest = c182a2a0b9fbedc880dd5c85fe4c02de
```

### 浏览器

- `1440x1000`：9 项指标完整，Auto MRR 为 `54.4%`，逐案例显示 Top-K、候选和首个相关排名；旧快照候选指标显示“未记录”；控制台无 error。
- `390x844`：`clientWidth=390`、`scrollWidth=390`、可见越界元素 0、控制台 error 0。
- 重建后重新加载 Auto 运行详情，运行列表与详情 API 均返回 200，页面继续显示候选 Recall 与 MRR。

## 已知剩余项

- 前端全局 TypeScript 债务仍有 101 条，不属于本轮 RAG 文件。
- 当前 25 条真实案例没有 K 外失败样本；依赖单元测试锁定该分支语义。
- `auto` 仍仅用于评测和观测，不作为生产默认策略推广。
