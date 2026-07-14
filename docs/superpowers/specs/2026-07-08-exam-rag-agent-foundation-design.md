# 考试 RAG 与 Agent 底层增强设计

## 目标

把考试平台的核心学习能力从“页面功能可用”推进到“底层 RAG/Agent 可解释、可复用、可评测”。第一阶段先建立结构化题组到 RAG 上下文的标准表达，让题组、原文、题目、选项、答案、解析和 chunk 引用可以被普通对话、Agent 工具、错题复盘和教师分析共同消费。

## 当前基础

- WeKnora 已有知识库解析、chunk、混合检索、rerank、Chat pipeline 和 Agent `knowledge_search` 工具。
- 考试模块已沉淀 `QuestionGroupDetail`，包含 `MaterialText`、`Questions`、`Options`、`Answers`、`Explanations`、`ChunkRefs`。
- 当前 `searchutil.BuildExamQuestionContextBundle` 主要基于原始 chunk 启发式补全阅读原文和答案，适合作为兜底，但不应成为长期主路径。

## 方案选择

推荐采用“结构化题组优先，chunk 启发式兜底”的两层 RAG 设计：

1. 结构化题组上下文包
   - 输入：`QuestionGroupDetail`
   - 输出：统一文本上下文，包含题组元信息、原文、题目、选项、答案、解析、证据 chunk。
   - 作用：作为 Chat/Agent/错题/讲评共用的上下文表达。

2. 检索接入层
   - 当检索结果命中考试 chunk 或题目 chunk ref 时，优先找到对应题组并生成结构化上下文包。
   - 找不到结构化题组时，继续使用现有 chunk 启发式阅读补全。

3. Agent 工具层
   - 让 Agent 在需要解释、诊断、生成练习建议时可以直接拿到结构化题组上下文。
   - 短期复用 `knowledge_search` 的结果增强；中期新增考试专属工具，如 `exam_question_context`、`exam_learning_diagnosis`。

## 第一阶段边界

本阶段只做底层可测地基：

- 新增结构化题组上下文包构建器。
- 保留现有 chunk 启发式逻辑不变。
- 先不改数据库 schema。
- 先不新增页面。
- 后续再把上下文包接入 Chat pipeline 和 Agent tool。

## 验收标准

- 给定一个阅读题组，能生成包含原文、题号、选项、答案、解析、chunk 引用的上下文文本。
- 上下文长度可控，避免一次塞爆模型上下文。
- 代码不依赖具体页面，能被 RAG 和 Agent 层复用。
- 单元测试覆盖阅读题组、答案解析、chunk 引用和长度裁剪。
