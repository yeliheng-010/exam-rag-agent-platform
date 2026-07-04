# 考试资料入库与试卷结构化设计

## 背景

考试平台的下一阶段目标是让老师把高考、雅思资料沉淀到班级空间中，并为后续试卷结构化、题库生成、练习和智能体讲解建立稳定数据入口。底层文档解析、OCR、分块、embedding、检索仍复用 WeKnora 现有知识库链路；考试模块只增加考试领域元数据、班级空间隔离和结构化任务编排。

## 设计目标

- 老师可以把某个知识库中的具体文档登记为考试资料。
- 资料带有考试域、科目、年份、地区、资料类型等元数据。
- 试卷类资料可以创建结构化任务，并关联到题库。
- 学生只能读取自己可访问班级空间下的资料和任务状态，不能创建或修改。
- 第一阶段只生成“待人工校对 / 可结构化”的任务骨架，不伪装成完整自动抽题。

## 非目标

- 不在本阶段实现完整 LLM 自动抽题、答案匹配和批改。
- 不重写文档解析、分块或向量检索框架。
- 不改变 WeKnora 知识库上传、解析、chunk 和检索主流程。

## 核心实体

### `exam_materials`

表示一份考试资料，可以关联到 WeKnora 知识库中的单个 `knowledge` 文档。

关键字段：

- `tenant_id`：租户隔离。
- `space_id`：班级、个人或公共考试空间隔离。
- `knowledge_base_id`：资料所在知识库。
- `knowledge_id`：具体文档，可为空以支持后续外部来源。
- `domain_id` / `subject_id`：高考、雅思及其科目 / 模块。
- `material_type`：学习资料、试卷、答案、解析。
- `source_year` / `source_region` / `paper_type`：考试资料检索与筛选元数据。
- `ingest_status`：登记时同步知识文档解析状态。

### `exam_structuring_tasks`

表示对一份试卷资料执行结构化的任务。

关键字段：

- `material_id`：来源考试资料。
- `question_bank_id`：结构化结果的题库承接点。
- `status`：`pending`、`ready_for_review`、`blocked`、`completed`、`failed`。
- `source_chunk_count`：登记时可见的文本 chunk 数量。
- `structured_question_count`：已沉淀到题库的题目数量，第一阶段默认为 0。
- `strategy`：第一阶段为 `manual_review`，后续可扩展为 LLM 抽题策略。

## 数据流

1. 老师在 WeKnora 知识库中上传试卷或资料。
2. WeKnora 执行既有解析流程：文件解析、chunk、embedding、可选问题生成。
3. 老师在班级详情页登记考试资料，选择知识库、文档、考试域、科目和资料类型。
4. 后端校验班级空间写权限、知识库归属和文档归属，写入 `exam_materials`。
5. 如果资料类型是试卷，老师可同时创建结构化任务并选择或创建题库。
6. 后端根据知识文档解析状态和 chunk 数量设置结构化任务状态。

## 权限模型

- `GET /exam/materials`、`GET /exam/structuring-tasks`：Viewer 可读，受考试空间可读权限限制。
- `POST /exam/materials`、`POST /exam/materials/:id/structuring-tasks`：Contributor 可写，且必须对目标空间可写。
- 班级空间写权限来自班级角色：老师和助教可写，学生只读。
- 系统管理员仍通过平台权限管理角色，不绕过考试空间规则。

## 与 RAGFlow / WeKnora 的关系

- 参考 RAGFlow 的是“文档解析、OCR、chunk、检索、rerank 要形成可观测流水线”的思路。
- 复用 WeKnora 的是“知识库入库、chunk 存储、检索权限、前后端工程化落地”的实现。
- 本阶段新增考试领域编排层，不替代底层 RAG 管线。

## 验收标准

- 老师可以在班级详情页登记资料。
- 试卷资料可以创建结构化任务并关联题库。
- 学生可以看到可访问空间内的资料，但不能创建资料或任务。
- 后端服务有单元测试覆盖权限校验、资料登记和结构化任务状态。
- 前端构建通过；若 Go 测试受 Windows cgo 工具链限制，需要明确报告。
