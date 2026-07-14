# 班级学习诊断 Agent 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 新增权限受控的 `exam_class_diagnosis` 工具，提供班级完成情况、正确率趋势、高频错题和需关注学生。

**架构：** 扩展现有 `ExamAnalyticsService` 生成可信聚合，repository 批量读取最新任务 attempt 的答案；Agent 工具只解析输入、复用会话身份并格式化统计结果。工具不依赖知识库能力，完整题目材料继续由 `exam_question_context` 提供。

**技术栈：** Go、GORM、WeKnora Agent Tool、现有 Exam Analytics/Practice 模块、Go testing。

---

## 文件结构

- 修改：`internal/types/exam_analytics.go`，增加高频错题统计类型。
- 修改：`internal/types/interfaces/exam_analytics.go`，增加可分析班级列表方法。
- 修改：`internal/types/interfaces/exam_practice.go`，增加批量答案读取方法。
- 修改：`internal/application/repository/exam_practice.go`，实现 tenant-scoped 批量答案查询。
- 修改：`internal/application/service/exam_analytics.go`，聚合高频错题和可分析班级。
- 修改：`internal/application/service/exam_analytics_test.go`，覆盖权限和聚合口径。
- 创建：`internal/agent/tools/exam_class_diagnosis.go`，实现工具输入、范围解析和调用。
- 创建：`internal/agent/tools/exam_class_diagnosis_format.go`，实现趋势和诊断上下文格式化。
- 创建：`internal/agent/tools/exam_class_diagnosis_test.go`，覆盖工具行为与注册。
- 修改：`internal/agent/tools/definitions.go`、`capabilities.go`，声明工具。
- 修改：`internal/application/service/agent_service.go`，注入统计服务并注册工具。
- 修改：`config/agent_type_presets.yaml`、`config/builtin_agents.yaml`，允许考试 Agent 使用工具。
- 修改：`frontend/src/utils/tool-capabilities.ts`、多语言文件，展示工具名称与能力。

### 任务 1：批量答案和高频错题聚合

- [x] 编写失败测试：两个学生在同一题答错时聚合为一条，验证 `answer_count=2`、`wrong_count=2`、`affected_student_count=2`。
- [x] 编写失败测试：同一学生同一任务的旧 attempt 不计入统计。
- [x] 运行 `go test ./internal/application/service -run 'TestExamClassAnalytics.*Wrong' -count=1`，确认因类型或聚合缺失失败。
- [x] 增加 `ListAnswersByAttempts(ctx, tenantID, attemptIDs)`，查询必须带 tenant 和 attempt ID 集合。
- [x] 收集每项任务每名学生的最新 attempt ID，并聚合 `FrequentWrongQuestions`。
- [x] 运行上述定向测试确认通过。

### 任务 2：班级选择和权限边界

- [x] 编写失败测试：教师只能列出自己 active 且可写的班级，学生班级被过滤。
- [x] 运行 `go test ./internal/application/service -run 'TestExamAnalyticsListAnalyzableClasses' -count=1` 确认失败。
- [x] 在 `ExamAnalyticsService` 增加 `ListAnalyzableClasses`，复用 `ListByUser` 和 `GetMember` 过滤 `CanWrite()`。
- [x] 运行定向测试确认通过，并回归 `go test ./internal/application/service -run 'TestExamClassAnalytics' -count=1`。

### 任务 3：Agent 工具和确定性诊断表达

- [x] 编写失败测试：唯一班级自动诊断、多个班级返回选择、学生权限错误、参数上限归一化。
- [x] 编写失败测试：输出包含 `class_summary`、`mastery_trend`、`frequent_wrong_questions`、`at_risk_students` 和结构化 Data。
- [x] 运行 `go test ./internal/agent/tools -run 'TestExamClassDiagnosis' -count=1` 确认失败。
- [x] 实现 `ExamClassDiagnosisTool`，仅从 context 读取 tenant/user。
- [x] 实现 5 个百分点阈值的趋势计算和需关注学生排序。
- [x] 运行定向测试确认通过。

### 任务 4：注册、前端能力和整体验证

- [x] 在定义、默认工具、capability、Agent service、preset 和 builtin agent 中注册 `exam_class_diagnosis`。
- [x] 在前端 Agent 工具清单和多语言资源中增加显示文案。
- [x] 运行 `gofmt` 和相关 Go 包测试。
- [x] 运行 `go vet ./internal/agent/tools ./internal/application/service ./internal/application/repository`。
- [x] 运行前端定向测试、全量测试和生产构建。
- [x] 重建 app/frontend Docker 镜像并确认容器健康。
- [x] 使用真实班级 API/数据库验证教师可见、学生被拒绝、无作答班级不产生虚假错题。
