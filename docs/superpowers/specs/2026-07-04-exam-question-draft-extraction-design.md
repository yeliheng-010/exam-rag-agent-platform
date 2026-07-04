# 试卷抽题草稿与老师校对工作台设计

## 背景

考试平台已经具备班级空间、资料登记、结构化任务和题库基础。老师可以把 WeKnora 知识库中的试卷文档登记为考试资料，并创建 `exam_structuring_tasks`。现在需要把结构化任务从占位状态推进到真实可用：系统根据已解析的试卷 chunk 调用 LLM 抽取题目草稿，老师校对后再写入正式题库。

第一版以高考英语试卷为主线做深做通。雅思 Reading、Listening、Writing、Speaking 暂不实现专门解析规则，但数据结构保留考试域、科目、题型和来源信息，后续可扩展。

## 设计目标

- 老师可以对已完成解析和向量化的试卷资料发起抽题。
- 系统读取文档 chunk，调用当前租户的 KnowledgeQA 模型生成结构化题目草稿。
- 草稿进入人工校对状态，不直接污染正式题库。
- 老师可以修改题干、题型、选项、答案、解析和来源信息。
- 老师确认后，草稿写入正式题库相关表。
- 学生不能创建、修改或查看未审核草稿，只能使用正式题库内容。

## 非目标

- 不在第一版实现完全无人值守的自动入库。
- 不在第一版做自动批改、错题本、作业发布和学情分析。
- 不重写 WeKnora 现有文档解析、chunk、embedding 和检索链路。
- 不为雅思各模块编写专门抽题规则，只保留扩展点。

## 核心方案

采用「LLM 抽题草稿 + 老师人工校对 + 确认入正式题库」。

系统对每个结构化任务执行抽题时，会按 chunk 窗口组装上下文，向模型要求输出严格 JSON。后端只接受通过 schema 校验的题目，并写入 `exam_question_drafts`。老师确认后，系统再将草稿转换为正式题目，写入以下已有正式表：

- `questions`
- `question_options`
- `question_answers`
- `question_explanations`
- `question_chunk_refs`

## 数据模型

### `exam_question_drafts`

表示 LLM 抽取出的待校对题目。

关键字段：

- `id`：草稿 ID。
- `tenant_id`：租户隔离。
- `space_id`：班级、个人或共享空间隔离。
- `task_id`：来源结构化任务。
- `material_id`：来源考试资料。
- `question_bank_id`：确认后写入的目标题库。
- `domain_id` / `subject_id`：考试域和科目。
- `source_chunk_ids`：来源 chunk ID 列表，使用 JSONB。
- `question_no`：题号，例如 `21`、`31`、`66`。
- `question_type_code`：题型编码，例如 `single_choice`、`cloze`、`writing`。
- `stem`：题干。
- `options_json`：选项数组，适配单选、完形填空、七选五等题型。
- `answer_json`：答案结构，支持单答案、多答案和自由文本答案。
- `explanation`：解析。
- `difficulty`：难度，默认 `unknown`。
- `confidence`：模型置信度，范围 0 到 1。
- `status`：`pending_review`、`approved`、`rejected`。
- `raw_model_output`：模型原始输出，用于排查和二次修复。
- `error_message`：单题抽取或确认失败原因。
- `approved_question_id`：确认后对应的正式题目 ID。
- `reviewed_by_user_id` / `reviewed_at`：校对人和校对时间。

### `exam_structuring_tasks` 状态扩展

现有状态保留，并新增抽题执行状态：

- `pending`：任务已创建，但还不能执行。
- `ready_for_review`：文档解析完成且有 chunk，可以开始抽题。
- `blocked`：文档未完成解析或没有可用 chunk。
- `extracting`：正在调用模型抽题。
- `reviewing`：已生成草稿，等待老师校对。
- `completed`：草稿已确认并写入正式题库。
- `failed`：抽题任务失败。

### 正式题库写入

确认草稿时，写入顺序如下：

1. `questions`：写入题干、题型、难度、年份、地区、审核状态和状态。
2. `question_options`：写入选项。
3. `question_answers`：写入答案。
4. `question_explanations`：写入解析。
5. `question_chunk_refs`：写入来源 chunk 引用。
6. 回写 `exam_question_drafts.approved_question_id`，状态改为 `approved`。

确认过程必须在数据库事务中完成。

## 抽题流程

1. 老师点击结构化任务的「开始抽题」。
2. 后端校验任务、资料、题库、班级空间权限。
3. 后端确认资料 `ingest_status=completed`，且存在可用 chunk。
4. 任务状态更新为 `extracting`。
5. 后端按 chunk 顺序构造窗口。第一版优先使用全文顺序窗口，不做复杂检索。
6. 每个窗口调用当前租户的 KnowledgeQA 模型。
7. 模型返回 JSON 后，后端执行清洗和 schema 校验。
8. 合格题目写入 `exam_question_drafts`，重复题号进行合并或跳过。
9. 有可用草稿时，任务状态更新为 `reviewing`。
10. 没有可用草稿时，任务状态更新为 `failed`，并记录错误。

## LLM 输出约束

第一版使用非流式 `Chat(ctx, messages, opts)`。模型参数建议：

- `temperature=0.1`
- `top_p=0.2`
- `max_tokens` 根据窗口大小设置上限
- 优先使用 JSON response format；若模型不支持，则使用强提示词加后端 JSON 提取

期望输出结构：

```json
{
  "questions": [
    {
      "question_no": "21",
      "question_type_code": "single_choice",
      "stem": "What does the man suggest the woman do?",
      "options": [
        { "key": "A", "content": "..." },
        { "key": "B", "content": "..." }
      ],
      "answer": { "value": "A" },
      "explanation": "...",
      "difficulty": "unknown",
      "confidence": 0.82
    }
  ]
}
```

后端校验规则：

- `question_no`、`stem` 必须非空。
- 单选题必须至少有 2 个选项。
- 客观题答案不能为空。
- `confidence` 超出范围时归一化到 0 到 1。
- 模型输出不能直接执行或拼接 SQL。
- 失败窗口只记录错误，不影响已成功写入的其他窗口。

## 后端接口

### 发起抽题

`POST /api/v1/exam/structuring-tasks/:task_id/extract`

行为：

- 校验写权限。
- 如果任务已在 `extracting`，返回当前任务状态。
- 如果已有 `pending_review` 草稿，默认不重复抽题。
- 支持后续扩展 `force=true` 重新抽题。

### 查看草稿

`GET /api/v1/exam/structuring-tasks/:task_id/drafts`

行为：

- 老师和助教可读。
- 学生不可读。
- 返回草稿列表和任务统计。

### 更新草稿

`PATCH /api/v1/exam/question-drafts/:draft_id`

可修改：

- 题号
- 题型
- 题干
- 选项
- 答案
- 解析
- 难度
- 来源 chunk 引用

### 确认草稿

`POST /api/v1/exam/question-drafts/:draft_id/approve`

行为：

- 事务写入正式题库。
- 草稿状态改为 `approved`。
- 更新结构化任务的 `structured_question_count`。
- 当任务下所有未驳回草稿都已确认时，任务状态可改为 `completed`。

### 驳回草稿

`POST /api/v1/exam/question-drafts/:draft_id/reject`

行为：

- 草稿状态改为 `rejected`。
- 不写正式题库。

## 前端设计

### 班级详情页

在「资料」Tab 的结构化任务表中增加：

- 「开始抽题」按钮。
- 「查看校对」按钮。
- 任务状态、草稿数量、已确认数量和失败原因。

### 校对工作台

新增任务校对页面或弹层。第一版推荐独立页面，便于长时间编辑：

- 左侧：题目草稿列表，显示题号、题型、状态和置信度。
- 右侧：题目编辑区，包含题干、选项、答案、解析、来源 chunk 和操作按钮。
- 顶部：任务信息、来源资料、目标题库和整体进度。
- 操作：保存草稿、确认入库、驳回草稿、返回班级。

### 题库详情页

补正式题目列表：

- 题号、题型、题干摘要、答案、状态。
- 点击后查看选项、解析和来源 chunk。

## 权限模型

- 老师、助教、班级资源管理者可以发起抽题、查看草稿、编辑草稿、确认和驳回。
- 学生不能查看未审核草稿，不能发起抽题。
- 学生后续只能使用正式题库中的题目。
- 系统管理员不绕过空间规则，除非后续新增平台审计入口。

## 错误处理

- 模型调用失败：任务进入 `failed`，记录 provider 错误摘要。
- JSON 解析失败：记录原始输出，任务进入 `failed` 或部分成功。
- 部分窗口成功：保留成功草稿，任务进入 `reviewing`，错误写入任务 `error_message`。
- 确认入库失败：草稿保留 `pending_review`，写入 `error_message`，不产生半成品正式题目。
- 重复确认：如果草稿已有 `approved_question_id`，返回已有正式题目。

## 测试策略

后端单元测试覆盖：

- 学生不能发起抽题或查看草稿。
- 老师可以对 `ready_for_review` 任务发起抽题。
- 文档未完成解析时不能抽题。
- LLM JSON 输出可以写入草稿。
- 非法 JSON 不写草稿并记录错误。
- 确认草稿时事务写入题目、选项、答案、解析和 chunk 引用。
- 已确认草稿重复确认不会重复创建正式题目。

前端验证覆盖：

- 班级结构化任务按钮显示符合角色权限。
- 校对页可以加载、保存、确认和驳回草稿。
- 题库详情页能展示正式题目。

## 验收标准

- 老师能从结构化任务发起抽题。
- 系统能基于已解析的高考英语试卷生成题目草稿。
- 老师能编辑并确认草稿。
- 确认后的题目进入正式题库，并能在题库详情页查看。
- 学生账号不能访问草稿和抽题写接口。
- 抽题失败时能看到可理解的错误信息。
- 不影响现有知识库上传、解析、向量化和基础 RAG 对话。

## 规格自检

- 没有保留待定项或占位符。
- 范围聚焦在试卷抽题、草稿校对和正式题库写入，不包含练习、批改和学情分析。
- 数据流与现有 `exam_materials`、`exam_structuring_tasks`、正式题库表保持一致。
- 权限模型延续班级空间读写隔离，不复用 WeKnora 默认租户 owner 模型来表达老师/学生业务角色。
