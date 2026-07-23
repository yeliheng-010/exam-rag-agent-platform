# 考试 RAG 全局 Top-K 与严格评测设计

**日期：** 2026-07-23
**状态：** 已确认
**范围：** 考试题库 RAG 诊断的跨知识库检索、排名口径、评测快照与前端展示

## 问题

当前 resolver 对每个知识库分别请求 `MatchCount=K`，再按知识库顺序拼接全部结果。四个知识库会产生约 `4K` 个主候选及父块 enrichment，但评测仍把完整列表称为 Recall@K。父块和 `SubChunkID` 还会让 `RetrievedChunkIDs` 与 `RetrievedContents` 的数组坐标失去一致性，导致首个相关排名并不代表真正的主结果排名。

因此旧运行中显示的 `Recall@20 = 1.00` 实际是候选全集覆盖率，不是最终前 20 条的严格召回率。

## 目标

- 跨知识库只保留全局前 K 个主结果。
- 不直接比较不同知识库的原始 `Score`，以每库局部排名做稳定融合。
- 一个主结果只占一个排名；父块紧跟对应主结果作为附加上下文，但不占排名。
- ID、正文和父块上下文以结构化排名项传递，评测不再依赖平行数组坐标。
- `retrieval_passed`、Recall@K 和 MRR 严格基于最终前 K 主结果。
- 候选全集单独计算 `candidate_recall` 与 `candidate_hit_rate`。
- 历史快照缺少候选指标时保持缺失语义，前端显示“未记录”。

## 非目标

- 不修改数据库 schema、正式题库、题组引用或评测金标。
- 不调整向量/关键词阈值、单库召回算法或 RRF 参数。
- 不在本轮推广 `auto` 切块策略或重切正式知识库。
- 不把不同检索模式的原始分数用于跨库排序。

## 排名项契约

每个主结果转换为一个 `ExamContextRankedItem`：

```go
type ExamContextRankedItem struct {
	Rank            int
	KnowledgeBaseID string
	LocalRank       int
	ChunkIDs        []string
	Contents        []string
}
```

- `Rank` 是全局主结果排名，从 1 开始。
- `LocalRank` 是该主结果在单个知识库中的主结果排名，父块不参与计数。
- `ChunkIDs` 包含主 Chunk ID、有效的 `SubChunkID` 和对应父块 ID，去重后共享同一主排名。
- `Contents` 第一项为主 Chunk 内容，随后为对应父块内容；短语命中任一内容时，排名仍记为主结果的 `Rank`。
- 孤立父块没有对应主结果时不进入排名，也不单独进入最终上下文。

旧的 `RetrievedChunkIDs`、`RetrievedContents` 和 `CandidateChunkIDs` 继续由结构化项派生，供已有题组关联和 JSON 客户端兼容；所有排名计算改用结构化项。

## 跨知识库融合

每个知识库先按服务返回顺序提取主结果，并记录 `LocalRank`。全局候选按以下键排序：

1. `LocalRank` 升序；
2. 搜索目标在请求中的顺序升序；
3. 主 Chunk ID 字典序升序。

这等价于按每库排名做稳定的轮次融合，不依赖 hybrid、vector-only 或 keyword-only 下不可比较的原始分数。重复主结果按全部 Chunk ID 别名去重，首次出现位置获胜。排序后的完整去重列表是候选全集，前 `K` 个主结果是最终结果。

## 评测口径

对于包含检索金标的案例：

- 最终命中：只在 `RetrievedItems[0:K]` 中匹配 Chunk ID 或检索短语。
- `retrieval_score` / Recall@K：最终前 K 命中的金标数除以金标总数。
- `retrieval_passed`：最终前 K 命中全部检索金标。
- `first_relevant_rank` / MRR：首个命中金标的最终主结果排名。
- `candidate_retrieval_score`：候选全集命中的金标比例。
- `candidate_retrieval_passed`：候选全集命中全部检索金标。
- `candidate_recall`：所有带检索金标案例的候选分数平均值。
- `candidate_hit_rate`：候选全集完全命中的案例比例。

没有检索金标的案例继续排除在上述排名指标之外。

## 快照兼容

新候选指标在持久化 DTO 中使用可选字段。新运行写入明确数值，包括合法的 `0`；旧快照反序列化后保持 `nil` 并通过 `omitempty` 保持字段缺失。前端类型使用可选属性：

- 缺失：显示“未记录”；
- 存在且 `ranked_case_count == 0`：显示“无检索金标”；
- 存在且有检索金标：按百分比显示。

详情页同时显示“严格前 K”和“候选全集”两组召回信息，避免把候选覆盖误读为线上最终召回。

## 验收

- 多知识库测试证明排名按 `KB1#1, KB2#1, KB1#2, KB2#2` 交错，稳定 tie-break 不受原始分数影响。
- 父块内容能满足短语金标，但其排名等于对应主 Chunk 且不占 K。
- 位于候选全集但在 K 之外的金标使严格召回失败，同时候选召回通过。
- 历史快照没有候选字段时，列表、详情和对比均显示“未记录”。
- 相关 Go 包、前端定向测试、生产构建、真实容器运行和桌面/移动浏览器检查完成。
