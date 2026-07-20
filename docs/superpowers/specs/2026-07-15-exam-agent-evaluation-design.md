# 题库级 Agent 行为评测设计

## 目标

在题库级 RAG 评测中心之后增加 Agent 行为评测，验证一个真实 Agent 是否按预期调用工具、传递参数、取得证据并生成有依据的最终回答。评测必须使用生产 Agent 引擎和真实工具，不使用模拟回答，也不额外调用裁判模型。

## 方案选择

采用“复用运行生命周期，区分评测类型”的方案：

1. 扩展 `exam_rag_evaluation_runs`，增加 `evaluation_kind` 和 `agent_id`。
2. `rag` 和 `agent` 运行共用 queued/running/completed/failed、进度、请求快照和结果快照。
3. RAG 与 Agent 使用独立 service、worker 和 API，repository 按 `evaluation_kind` 隔离列表和详情。
4. 保留题库作用域，Agent 评测仍从题库详情进入，跨题库平台看板留到后续阶段。

不创建第二套运行表，避免重复实现历史、轮询、权限和失败恢复；不把 Agent 结果塞进现有 RAG JSON 结构，避免两种指标相互污染。

## 场景契约

每个场景包含：

- `name`：场景名称。
- `input`：发送给 Agent 的用户输入。
- `expected_tool_calls`：按顺序排列的工具名和参数子集。
- `expected_evidence_phrases`：必须出现在成功工具结果中的证据短语。
- `expected_citations`：必须同时出现在工具证据和最终回答中的引用标识。
- `expected_answer_phrases`：必须出现在最终回答中的关键结论。
- `grounded_phrases`：必须同时出现在工具证据和最终回答中的事实短语。

工具参数采用递归子集匹配，允许模型携带额外可选参数。工具序列默认精确匹配，意外调用和漏调用都会失败。

## 执行隔离

每个场景使用独立的临时 session/message ID，不写入聊天会话和消息表，不加载多轮历史。评测副本强制：

- 关闭 MCP、Skills、Web Search 和多轮记忆。
- 只保留内置只读工具白名单。
- 拒绝期望序列中出现写工具或未知工具。
- 使用运行创建者的 tenant/user 上下文执行现有权限校验。
- 场景串行执行，单个失败不阻塞后续场景，worker 持续写入进度。

首批只读白名单覆盖考试诊断、题目上下文、练习推荐和知识检索工具，不开放 Wiki 修改、数据库写入、MCP 或外部副作用工具。

## 评分口径

每个场景输出五类断言：

1. `tool_sequence_score`：实际工具序列与期望序列是否一致。
2. `tool_arguments_score`：期望参数子集的命中比例。
3. `evidence_score`：期望证据短语在工具结果中的覆盖率。
4. `citation_score`：期望引用同时在工具结果和最终回答中的覆盖率。
5. `groundedness_score`：grounded 短语同时被证据和回答支持的比例；未配置时回退到答案短语覆盖率。

场景通过要求：执行无错误，工具序列通过，参数断言全部通过，且所有已配置的证据、引用、答案和 groundedness 断言均为 1。汇总保存通过率、各断言平均值、平均耗时和失败场景数。

## 数据与 API

迁移 `000081`：

- `evaluation_kind VARCHAR(16) NOT NULL DEFAULT 'rag'`
- `agent_id VARCHAR(36)`
- 新增 `(tenant_id, question_bank_id, evaluation_kind, created_at DESC)` 索引。

Agent API：

- `POST /api/v1/exam/question-banks/:bank_id/agent-evaluation-runs`
- `GET /api/v1/exam/question-banks/:bank_id/agent-evaluation-runs`
- `GET /api/v1/exam/question-banks/:bank_id/agent-evaluation-runs/:run_id`

三条路由均要求 Contributor；service 再校验题库可读和 Agent 属于当前租户。列表只返回配置和汇总，详情返回完整场景结果。

## 前端

新增题库级 Agent 评测页面，与 RAG 评测中心互相跳转。页面包括：

- Agent 选择器和结构化场景编辑器。
- 场景新增、删除和参数断言编辑。
- 异步运行状态、历史列表和轮询。
- 汇总指标、实际工具时间线、参数差异、证据/引用/回答断言详情。

桌面端使用左侧历史和右侧详情；窄屏改为单列，长输入、英文场景名、JSON 参数和工具输出允许换行，不产生页面横向滚动。

## 验收标准

1. Contributor 可为当前题库选择智能推理 Agent 并创建 1-20 个场景的运行。
2. Viewer/学生创建、列表和详情均返回 403。
3. 写工具和未知工具在创建阶段被拒绝。
4. worker 使用真实 Agent 引擎，保存实际工具顺序、参数、成功状态、耗时、证据摘要和最终回答。
5. 工具、参数、证据、引用和 groundedness 分数由确定性测试覆盖。
6. RAG 历史不出现 Agent 运行，Agent 历史不出现 RAG 运行。
7. Docker 部署后，使用现有考试 Agent 完成至少一次真实运行，并在浏览器中看到结果。
