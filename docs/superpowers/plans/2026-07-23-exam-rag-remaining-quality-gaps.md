# 考试 RAG 剩余质量缺口实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 `superpowers:executing-plans` 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法跟踪进度。本轮沿用用户此前要求，不 commit、不 push。

**目标：** 补齐 E01 英语赛事表格上下文召回，以及 M25-05、M25-06、M25-08 的结构化答案文本，使 25 条联合评测不再存在已知质量缺口。

**架构：** 保留子 Chunk 的排序与检索阈值，只修复检索结果组装阶段错误过滤 `parent_text` 的问题，让已命中的问题 Chunk 能携带父块表格上下文。答案缺口不引入运行时硬编码，直接依据正式源文档中的答案图片修复对应正式题目文本，并保留原始图片 URI 到 explanation 作为审计证据。

**技术栈：** Go、PostgreSQL、Docker Compose、现有考试 RAG 联合评测。

---

## 文件结构

- 创建 `internal/application/service/knowledgebase_search_results_test.go`：覆盖父子切块上下文组装回归。
- 修改 `internal/application/service/knowledgebase_search_results.go`：区分直接检索 Chunk 与 enrichment Chunk 的可返回类型。
- 创建 `docs/superpowers/reports/2026-07-23-exam-rag-remaining-quality-gaps.md`：记录图片核验、数据修复、评测运行与只读性摘要。

### 任务 1：父块上下文组装

- [x] 新增失败测试：输入命中的 `text` 子块及其 `parent_text` 父块，期望 `assembleSearchResults` 同时返回两者，且父块标记为 `MatchTypeParentChunk`。
- [x] 运行 `go test ./internal/application/service -run TestAssembleSearchResultsIncludesParentContext -count=1`，确认当前因父块被过滤而失败。
- [x] 增加 enrichment 专用类型判断：只允许 `MatchTypeParentChunk` 对应的 `parent_text` 绕过直接检索类型过滤；直接检索仍不接受 `parent_text`。
- [x] 重跑定向测试，确认通过；再运行 `go test ./internal/application/service -count=1`。

### 任务 2：答案图片人工结构化

- [x] 在事务前记录 5 条目标 `question_answers` / `question_explanations` 原值，并计算 `question_chunk_refs` 行数与摘要。
- [x] 依据正式源图片写入以下结构化答案：

```text
Question 13 / 13      -> 2
Question 14 / 14      -> $\frac{61}{25}$ (2.44)
Question 18 / 18(1)   -> $\frac{x^2}{9}+y^2=1$
Question 18 / 18(2)(i)-> $(\frac{3m}{m^2+(n+1)^2},\frac{n+2-m^2-n^2}{m^2+(n+1)^2})$
Question 18 / 18(2)(ii)-> $3(\sqrt{3}+\sqrt{2})$
```

- [x] explanation 写入简洁的“图片识别答案”，并附原始 `local://` URI；不改题干、选项、Chunk、题组关联或评测金标。
- [x] 重新读取 5 条记录，确认内容与题号一一对应。

### 任务 3：端到端验证

- [x] 重建 `app` 镜像并等待健康检查，确认运行镜像包含父块修复。
- [x] 使用相同 25 条案例运行 legacy、auto、auto-repeat。
- [x] 核对 E01 检索短语、4 个历史失败案例、Recall@20、结构化解析率、答案覆盖率和重复运行签名。
- [x] 重新读取 `question_chunk_refs` 行数与摘要，确认正式引用未变化。
- [x] 运行相关 Go 包回归、前端测试和 `npm run build-only`。
- [x] 在 `1440x1000` 与 `390x844` 检查详情页指标、E01 和三道图片答案案例，并确认无控制台错误或横向溢出。

## 实际结果

- 父块 enrichment 已恢复，且直接检索或关系 enrichment 的 `parent_text` 仍会被拒绝；考试 resolver 将父块稳定排在命中子块之后。
- 正式答案图片人工核验并结构化 5 条答案；每条 explanation 保留对应 `local://` 原始图片 URI。
- 最终运行镜像为 `sha256:f765a572430607958e394191a75640945501fb76bbf3ec8b86e963fb4183166f`，`app` 健康检查通过。
- 三次 25 案例评测均达到 Recall@20、结构化解析率、答案覆盖率和总体通过率 `1.00`。
- E01 在两次 auto 运行中的首个相关排名均为 `2`，动态关联题组均为 `6adc1f07-6eb8-48e7-aa77-9733328c1cfc`。
- 正式 `question_chunk_refs` 保持 240 行，稳定摘要保持 `c182a2a0b9fbedc880dd5c85fe4c02de`。
- `1440x1000` 与 `390x844` 浏览器验收均无页面横向溢出，控制台无 error。
