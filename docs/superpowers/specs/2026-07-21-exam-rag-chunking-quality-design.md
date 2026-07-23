# 考试 RAG 切块与检索质量优化设计

## 背景

考试平台已经具备试卷解析、题组结构化、混合检索、RAG 诊断和评测中心，但当前评测仍不能回答“某次结果为什么变好或变差”：

- 四个真实考试知识库都保存了 `auto + 512/80 + 父子块 4096/384`，却没有持久化本次处理实际选中的切块 tier。
- `buildParentChildConfigs` 没有把知识库的 `strategy` 和 `languages` 传给父、子 splitter；由于空 strategy 为兼容历史数据会解析为 `legacy`，父子块开启时可能绕过数据库声明的 `auto`。
- RAG 评测只保存知识库 ID、检索阈值和案例，没有保存切块配置、块大小分布或处理版本。
- 没有 `expected_chunk_ids` 的案例仍被统计为“召回通过”，导致现有 `retrieval_hit_rate=1` 不能证明真实召回质量。
- chunk ID 在重新切块后会变化，单纯用 ID 作为金标无法稳定比较不同切块配置。

因此本阶段先建设可重复、可归因的切块质量评测闭环，再根据真实英语和数学试卷的 A/B 结果修改参数或算法。不能先拍脑袋调整全局默认值。

## 目标

1. 修复父子切块丢失自适应 strategy 的生产链路缺陷。
2. 记录每次文档处理实际使用的父级 tier、子级 tier 分布、回退原因和块大小统计。
3. 在 RAG 评测运行中保存不可变的知识库切块配置与质量快照。
4. 支持不依赖 chunk ID 的“检索内容短语”金标，真实计算 Recall@K 和 MRR。
5. 在评测中心展示并比较两次运行的切块配置、分布和检索质量。
6. 用现有高考英语、DOCX 数学和 PDF 数学资料建立基线并完成隔离 A/B。
7. 只有 A/B 证明有效时，才修改考试知识库参数或通用切块算法。

## 非目标

- 不继续建设题集、作业、通知、班级设置等业务功能。
- 不在没有金标和对照组的情况下修改所有知识库默认参数。
- 不直接重切当前四个正式知识库；先在隔离副本或只读预览中评测。
- 不把完整原文写入评测快照，只保存配置、统计、匹配结果和必要的短语。
- 不把解析器缺陷伪装成切块缺陷；OCR、公式媒体和结构化材料问题单独归因。

## 方案比较

### 方案 A：生产诊断 + 稳定金标 + 隔离 A/B（采用）

让生产切块链路输出实际 tier 和统计，RAG 评测保存配置快照，并增加检索内容短语金标。先修复 strategy 透传，再用隔离知识库比较参数和算法。

优点：结果可归因、可回放，可区分解析、切块、检索和结构化材料问题；后续每次优化都能复用。缺点：需要同时修改 chunker、处理管线、评测服务和前端展示。

### 方案 B：只写一次性离线脚本

脚本读取文件并比较多个切块参数，不改生产评测系统。

优点：启动快。缺点：与生产解析、父子块、向量索引和 RRF 链路容易漂移，结果不能在评测中心复用，拒绝作为主方案。可作为 A/B 数据准备的辅助工具。

### 方案 C：直接增加考试专用切块规则

立即识别题号、选项、答案和解析边界。

优点：短期可见。缺点：没有基线时无法证明收益，也容易把 DOCX/PDF 解析问题掩盖为规则问题。只在方案 A 证明通用策略仍无法满足金标后采用。

## 总体架构

```text
DocReader Markdown
  -> shared chunking executor
       -> flat chunks / parent-child chunks
       -> actual tier diagnostics
       -> size and boundary statistics
  -> chunks + processing span output
  -> embedding / hybrid search / RRF
  -> exam RAG evaluation
       -> immutable chunking snapshots
       -> ID gold + retrieval phrase gold
       -> Recall@K / MRR / answer coverage
  -> evaluation center comparison UI
```

## 共享切块执行器

新增 service 层共享辅助模块，替换 `knowledge_create.go` 和 `knowledge_process.go` 中重复的 flat/parent-child 分支。输入为 Markdown 与有效 `ChunkingConfig`，输出：

```go
type ChunkingExecutionResult struct {
    Chunks       []types.ParsedChunk
    ParentChunks []types.ParsedParentChunk
    Diagnostics  ChunkingExecutionDiagnostics
}

type ChunkingExecutionDiagnostics struct {
    RequestedStrategy string
    ParentTier        string
    ChildTierCounts   map[string]int
    Rejections        []ChunkingTierRejection
    TextChunkCount    int
    ParentChunkCount  int
    MinChars          int
    P50Chars          int
    P90Chars          int
    MaxChars          int
    TinyChunkCount    int
    OversizeCount     int
}
```

要求：

- flat 模式使用 `SplitWithDiagnostics`。
- parent-child 模式新增带 diagnostics 的公共入口；父文档记录一个实际 tier，每个父块重新切子块时累计子 tier。
- `buildParentChildConfigs` 必须向父、子配置透传 `Strategy` 和 `Languages`。
- `TokenLimit` 只约束参与 embedding 的子块；父块不参与 embedding，不应被 embedding token 上限截断。
- 保留现有 `Split` / `SplitParentChild` API，内部复用同一实现，避免调用者漂移。
- tiny 阈值为目标 child/flat size 的 25%；oversize 阈值为目标 size。受保护的公式、表格或媒体跨度允许超限，但必须进入统计，不能静默消失。

## 生产可观测性

现有 `knowledge_processing_spans` 已有 JSON `input/output`，不新增数据库迁移。切块阶段：

- input 保存有效配置摘要：strategy、chunk size、overlap、parent-child、parent/child size、token limit、languages。
- output 保存 `ChunkingExecutionDiagnostics`。
- 不保存原始文档全文或 chunk 内容。
- 旧处理记录没有 diagnostics 时按“未知”展示，不反推虚假 tier。

这样能够确认“数据库配置是 auto”与“本次处理实际选中了哪个 tier”是否一致。

## 稳定检索金标

扩展案例：

```go
type ExamRAGDiagnosticCase struct {
    Name                     string
    Query                    string
    RequiredPhrases          []string
    ExpectedChunkIDs         []string
    RequiredRetrievalPhrases []string
}
```

- `RequiredPhrases`：最终结构化上下文必须包含的答案证据，继续计算 answer coverage。
- `ExpectedChunkIDs`：适合不重新切块的固定索引回归。
- `RequiredRetrievalPhrases`：在原始 top-K 检索结果内容中匹配，重新切块后仍稳定。
- 评测过程只临时读取检索结果内容；持久化结果仅保存命中/缺失短语和对应排名，不保存完整 chunk 内容。

指标语义调整：

- 只有配置了 `ExpectedChunkIDs` 或 `RequiredRetrievalPhrases` 的案例才计入 `ranked_case_count`。
- 没有检索金标的案例不再自动算“召回通过”，也不进入 retrieval hit rate 分母。
- Recall@K 为所有检索金标项在 top-K 中的平均覆盖率。
- MRR 使用首个命中 ID 或短语的排名。
- overall pass 对无检索金标案例只要求答案覆盖；对有检索金标案例同时要求检索和答案通过。

## 切块质量快照

为每个评测使用的知识库生成：

```go
type ExamRAGChunkingSnapshot struct {
    KnowledgeBaseID   string
    Config            ChunkingConfigSnapshot
    KnowledgeCount    int
    TextChunkCount    int
    ParentChunkCount  int
    MinChars          int
    P50Chars          int
    P90Chars          int
    MaxChars          int
    TinyChunkRate     float64
    OversizeRate      float64
    ParentCoverage    float64
    ActualTierCounts  map[string]int
    UnknownTierCount  int
}
```

实现采用独立的 `ExamRAGChunkQualityRepository`，只暴露 tenant-scoped 聚合所需的数据，不扩大通用 `ChunkRepository` 接口。快照在创建运行时写入 request snapshot；运行完成后不回读当前配置覆盖历史。

## 前端评测中心

在现有 RAG 可观测页面增加“切块快照”区：

- 展示请求 strategy 与实际 tier。
- 展示 chunk/parent 数量、P50/P90、tiny/oversize rate 和 parent coverage。
- 两次运行比较时标出配置变化和指标变化，但不自动声称因果。
- `ranked_case_count=0` 时 Recall@K、MRR 和召回覆盖显示“无检索金标”，不能显示 0% 或 100% 误导用户。
- 移动端使用纵向键值列表，桌面端使用紧凑表格；不新增图表依赖。

## 真实试卷金标与 A/B

第一批至少 20 个案例：

- 高考英语：阅读标题、材料首尾、题号、选项、答案与解析跨块边界。
- DOCX 数学：题号、行内/块级公式、选项、分问和答案解析。
- PDF 数学：页边界、公式图片、题干与图形关联、跨页材料。

对照顺序：

1. 记录当前实际行为，包含父子模式因 strategy 丢失而走 legacy 的基线。
2. 仅修复 strategy 透传，保持其他参数不变。
3. 比较 auto、显式 heuristic/heading/legacy。
4. 只对表现较好的 tier 比较 chunk size、overlap 和 parent/child size。
5. 通用策略仍失败的金标案例，才进入考试专用边界规则设计。

A/B 使用隔离副本，禁止直接覆盖正式知识库。候选方案至少满足以下条件才可推广：

- ranked cases 不少于 20。
- Recall@K 不低于 0.90，MRR 不低于 0.80。
- structured resolution rate 不低于 0.95。
- answer coverage 不低于 0.90；解析源本身缺字的案例单独标记，不归责于 chunker。
- 公式、图片、题干、选项边界金标无阻断性破坏。
- 相比基线没有显著增加 embedding chunk 数和检索延迟；若增加，必须有对应质量收益。

## 错误处理与兼容性

- 旧评测快照缺少新字段时保持可读，前端显示“未记录”。
- 旧案例没有检索金标时只参与答案覆盖，不伪造召回成功。
- diagnostics 采集失败不能阻断文档处理；记录 unknown tier 和错误摘要。
- tenant 隔离在仓储查询中完成，不接受前端传入 tenant ID。
- 不改变普通 HybridSearch 返回契约，也不把完整检索内容写入日志。

## 测试策略

严格执行 red -> green -> refactor：

1. service 测试先证明 parent-child config 当前丢失 strategy/languages。
2. chunker 测试先定义 parent/child diagnostics 的 tier 统计。
3. searchutil 测试先定义空金标不计召回、短语金标 Recall@K/MRR。
4. repository 测试覆盖 tenant 隔离、块分布和旧 span 无 tier。
5. evaluation service 测试覆盖不可变 snapshot、旧 snapshot 兼容和 compact list。
6. 前端 source/view-model 测试覆盖快照展示、比较和“无检索金标”。
7. 运行 chunker、repository、service、handler、router、前端全量测试和生产构建。
8. 重建 app/frontend，完成真实 API、数据库和浏览器 A/B 验收。

## 回滚边界

- production splitter API 保持兼容；诊断入口可以回退到现有 `Split`。
- 新评测字段全部为 JSON 可选字段，不需要回滚迁移。
- 正式知识库不在本阶段直接重切；隔离 A/B 失败时删除或保留测试副本由用户决定。
- 若 strategy 透传修复导致质量下降，可在单个知识库配置中显式选择 `legacy`，不回滚整个 chunker。

## 完成标准

- 数据库声明 `auto` 且开启父子块时，处理 span 能证明实际 parent/child tier，不再静默走 legacy。
- 无检索金标的旧评测不再显示虚假的 100% 召回。
- 新评测运行包含不可变切块快照，并能在浏览器比较。
- 至少 20 个真实英语/数学案例完成隔离 A/B，结果可重复。
- 只有达到质量门槛的配置或算法改动才应用到正式考试知识库。
+
## 实测结论与计划偏差（2026-07-21）

- 25 条真实检索短语金标完成了 legacy/auto 隔离 A/B；联合 Recall@20 均为 0.96。
- auto 没有提升总体 Recall，文本块从 284 增至 297，联合平均耗时从 986.92ms 增至 1059.76ms；2026 数学 DOCX 单库 MRR 从 0.7222 降至 0.5833，因此不推广到正式知识库。
- 联合 MRR 受多知识库 resolver 拼接顺序影响，只作为观察指标；切块判断以单库公平对照、Recall、块分布和耗时为主。
- 隔离副本没有重建正式题组到新 chunk ID 的引用，导致 structured resolution 为 0；这属于关联重建限制，不归因于 chunker。
- 本批金标聚焦 retrieval phrase，没有配置答案 required phrase，answer coverage 不参与本轮策略优劣判断。
- 浏览器验收新增一个计划内修复：390px 下为固定通知按钮预留 40px 右侧安全区，避免遮挡案例状态。
- 完整结果见 [A/B 报告](../reports/2026-07-21-exam-rag-chunking-ab.md)。
