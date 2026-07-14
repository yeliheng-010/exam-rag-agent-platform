# 数学试卷结构化质量闭环实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法跟踪进度。

**目标：** 将数学题组抽取改为可观测后台任务，并用数学质量报告阻止错误草稿进入正式题库。

**架构：** HTTP 层只校验并入队，Asynq/SyncTaskExecutor 共用服务处理器；任务进度存储在 `exam_structuring_tasks.progress`。数学质量报告按草稿实时计算，聚合结果写入任务进度，前端通过现有草稿列表 API 轮询。

**技术栈：** Go、Gin、GORM、PostgreSQL JSONB、Asynq、Vue 3、TypeScript、TDesign、Vitest。

---

## 文件结构

- `internal/types/exam_structuring.go`: 进度、预检和质量报告类型。
- `internal/types/task.go`: Asynq 任务类型和 payload。
- `migrations/versioned/000079_exam_structuring_progress.*.sql`: 进度字段迁移。
- `internal/application/service/exam_question_group_quality.go`: 数学质量规则和预检。
- `internal/application/service/exam_question_group_task.go`: 入队与 worker 执行。
- `internal/application/service/exam_question_group_batch.go`: 批次回调。
- `internal/router/task.go`、`internal/router/sync_task.go`: 真实队列与 Lite 模式处理器注册。
- `frontend/src/api/exam/question-group-draft.ts`: 移除长超时并返回启动结果。
- `frontend/src/views/question-draft/*`: 轮询、进度和质量展示。
- `frontend/src/views/classes/ClassDetail.vue`: 入队后跳转与提示。
- `internal/searchutil/exam_context_eval.go`: 数学 RAG 固定用例。

### 任务 1：定义持久化进度和后台任务协议

- [x] 编写任务 payload、进度归一化和迁移相关失败测试。
- [x] 运行 `go test ./internal/application/service -run 'StructuringProgress|ExtractionTask' -count=1`，确认因类型/行为缺失失败。
- [x] 增加 `ExamStructuringProgress`、`ExamStructuringQualitySummary`、payload 和 `progress` 字段。
- [x] 增加 `000079` PostgreSQL 迁移及回滚。
- [x] 运行定向测试确认通过。

### 任务 2：实现 Asynq 入队和 worker 执行

- [x] 编写“HTTP 服务只入队”“入队失败落为 failed”“worker 解析 payload 并执行”的失败测试。
- [x] 运行定向测试并确认失败原因是缺少后台协议。
- [x] 将现有同步 `ExtractDrafts` 拆为启动方法和内部执行方法。
- [x] 注册 `exam:question_group_extract` 到 Asynq 与 `SyncTaskExecutor` 的 `question` 队列。
- [x] Handler 返回 HTTP 202。
- [x] 运行服务和路由定向测试。

### 任务 3：实现批次进度与文档预检

- [x] 编写数学分批进度递增、失败批次和 WMF 密集文档警告测试。
- [x] 运行测试确认失败。
- [x] 增加可选 `ExtractWithProgress` 接口和批次回调。
- [x] 在 worker 中持久化 `preflight|extracting|failed` 状态。
- [x] 运行定向测试确认通过。

### 任务 4：实现数学质量报告和批准门禁

- [x] 编写空选项、空答案、缺小问、缺图、短题干、空解析测试。
- [x] 运行测试确认失败。
- [x] 实现质量报告和聚合统计。
- [x] 在列表、保存响应和批准路径实时附加质量报告；批准时阻断 `error`。
- [x] 运行草稿服务完整测试。

### 任务 5：实现前端轮询与质量反馈

- [x] 编写进度计算、终止状态和质量阻断辅助函数测试。
- [x] 运行 `npm run test -- questionGroupExtractionProgress.test.ts` 确认失败。
- [x] 移除 10 分钟同步超时，启动后每 3 秒轮询。
- [x] 在审核页展示进度、预检和草稿质量问题，并在错误存在时禁用批准。
- [x] 修改班级页为“任务已启动”后跳转。
- [x] 运行前端定向测试和构建。

### 任务 6：补充数学 RAG 评测并整体验证

- [x] 编写数学题干、答案、解析和图形引用的固定 eval 用例测试。
- [x] 运行测试确认失败。
- [x] 增加数学 eval cases 并接入现有结构化上下文评估器。
- [x] 运行 `go test ./internal/application/service ./internal/router ./internal/searchutil -count=1`。
- [x] 运行 `go vet ./internal/application/service ./internal/router ./internal/searchutil`。
- [x] 运行前端测试与 `npm run build-only`。
- [x] 重建 Docker 服务并应用迁移。
- [x] 使用正式数学材料启动真实后台任务，验证 202、进度递增、最终质量汇总和批准门禁。

## 2026-07-13 验收记录

- 本地模型链路：`bge-m3`（1024 维向量）与 `gemma3:12b`（结构化抽取/摘要），模型文件位于 `D:\ollama\models`。
- 正式数学试卷任务 `0c26753c-cc13-49b8-bffc-aaf47c9d6f53` 按题号边界动态拆为 12 批，12/12 批完成。
- 19 个大题完整覆盖题号 1-19，共沉淀 28 个正式题目/小问、151 个题组资源；无重复题号、无答案页伪题。
- 质量结果为 0 个阻断错误、10 个非阻断警告；19 个草稿全部审核入库，任务状态为 `completed`。
- 题库正式 API 返回 19 个题组和 28 个题目；浏览器题库页可见 `Question 1` 与 `Question 19`，审核面板显示待审核 0、已入库 19、质量错误 0。
- 最终验证：相关 Go 包测试通过，相关范围 `go vet` 通过；前端 179 项测试通过，`npm run build-only` 成功。
