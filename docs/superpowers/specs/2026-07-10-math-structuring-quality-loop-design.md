# 数学试卷结构化质量闭环设计

## 目标

把高考数学试卷从“HTTP 请求中同步调用模型”改造成可恢复、可观测、可审核的后台结构化流程，并在错误草稿进入正式题库前阻断。

本阶段不引入尚未配置的 VLM，也不承诺从 WMF 公式图片中恢复不可见内容。系统必须明确提示输入文档的视觉信息风险，并建议使用 PDF/VLM 重新解析。

## 范围

- 高考数学题组抽取的 Asynq 后台执行。
- 批次级进度、失败批次、质量汇总和文档预检结果持久化。
- 数学草稿质量检测与批准门禁。
- 前端启动任务、轮询状态、展示进度和质量问题。
- 数学结构化上下文的固定 RAG 评测用例。

暂不包含：

- 教师/班级诊断 Agent。
- OCR/VLM 模型接入和 WMF 转公式引擎。
- 自动修复所有低质量题目；本阶段先检测、阻断并允许教师编辑后重新检查。

## 架构

### 请求与任务

`POST /exam/structuring-tasks/:task_id/group-extract` 只负责：

1. 校验租户、用户、空间写权限和任务状态。
2. 将任务状态更新为 `extracting`，写入 `queued` 进度。
3. 向 `question` 队列提交 `exam:question_group_extract`。
4. 返回 HTTP 202 和当前任务快照。

Asynq 处理器使用 payload 中的 `tenant_id`、`user_id`、`task_id`、`force` 执行原有抽取流程。Lite 模式注册同一个处理器，由 `SyncTaskExecutor` 后台执行。

为避免重复收费和重复草稿，该任务使用 `MaxRetry(0)`。任务内部捕获错误并把状态持久化为 `failed`；用户可以在前端明确发起重试。

### 状态与进度

在 `exam_structuring_tasks` 增加 `progress JSONB NOT NULL DEFAULT '{}'`。Go 类型为 `ExamStructuringProgress`，包含：

- `phase`: `queued|preflight|extracting|quality_check|completed|failed`
- `total_batches`、`completed_batches`、`current_batch`、`failed_batch`
- `percent`、`message`
- `warnings`: 文档级预检问题
- `quality_summary`: 草稿质量统计
- `started_at`、`finished_at`

状态写入使用 `context.WithoutCancel`，即使 HTTP 已结束或 worker 上下文取消，也尽量保留最终进度。

### 批次进度

生产抽取器实现可选的 `ExamQuestionGroupProgressExtractor`：

- 开始批次前回调 `(completed=index, current=index+1, total)`。
- 批次成功后回调 `(completed=index+1, current=index+1, total)`。
- 失败时记录 `failed_batch=current`。

不支持批次回调的测试替身和兼容实现仍可通过原 `Extract` 接口运行，并按单批任务记录。

## 数学质量规则

质量问题分为 `error` 和 `warning`。只有 `error` 阻止批准。

### 阻断错误

- 选择题存在空选项内容。
- 答案为空或答案 JSON 不含有效值。
- 题号包含子问序号但中间子问缺失，例如存在 `(1)`、`(3)` 而没有 `(2)`。
- 题干引用“如图/下图/图中/图示”，但没有资源；或资源只有空 `storage_uri`。

### 审核警告

- 数学题干过短，可能被模型概括或截断。
- 解析为空。
- 资源缺少替代文本。

质量报告按草稿实时计算，不额外持久化，避免教师编辑后报告过期。任务完成时仅把聚合统计写入 `progress.quality_summary`。列表和更新 API 返回每个草稿最新的 `quality_report`。

## 文档预检

在抽取前扫描 chunk 内容中的 `x-wmf`、`.wmf`、`image/x-wmf` 等引用。达到阈值时产生：

- code: `formula_heavy_docx`
- level: `warning`
- 消息：DOCX 中包含较多 WMF/公式图片，纯文本解析可能丢失公式或图形。
- 建议：优先上传带可选文本层的 PDF；配置 VLM 后重新解析图像内容。

该警告不阻塞文本题目抽取，因为当前文档仍可能包含可用题干和答案。

## 前端交互

- 启动抽取后立即进入草稿审核页。
- 审核页每 3 秒请求 `group-drafts`，直到任务进入 `reviewing|completed|failed`。
- 抽取中显示稳定的百分比、批次文案和预检警告。
- 每个草稿显示错误/警告数量及明细。
- 有阻断错误时禁用“确认入库”；教师保存修正后质量报告立即刷新。

## 验证标准

1. 抽取 API 在模型调用开始前返回 202。
2. 数学多批任务能持久化递增批次进度。
3. worker 失败后任务状态和失败批次可查询。
4. 空答案、空选项、缺图等错误草稿无法批准。
5. WMF 密集文档返回明确预检警告。
6. 前端不会等待 10 分钟请求，刷新页面后仍能恢复轮询。
7. 数学 RAG 固定用例能检查题干、答案、解析和图形引用的结构化可回答性。
