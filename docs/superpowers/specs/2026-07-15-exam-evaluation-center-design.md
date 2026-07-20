# 平台级 RAG/Agent 评测与回归中心设计

## 目标

在现有题库级 RAG 可观测评测和 Agent 行为评测之上，建设面向平台管理员与班主任的统一评测中心。该中心必须支持跨题库查看运行、保存可复用评测集、追加不可变版本、设置基线、确定性识别回归，并能下钻到原有题库级检索 trace 或 Agent 工具轨迹。

本阶段继续使用确定性断言，不引入 LLM-as-judge。学生角色不开放该模块。

## 采用方案

采用“运行快照生成评测集”的方案：

- 题库级页面继续负责编辑和验证具体 RAG 案例或 Agent 场景。
- 任一已完成运行都可以在平台中心保存为命名评测集。
- 同一评测集可以从兼容的已完成运行追加新版本，历史版本不可覆盖。
- 平台中心可以按指定版本重新运行，运行记录保存评测集 ID 和版本号。
- 基线是运行记录的元数据，不修改原始请求和结果快照。

该方案复用已经完成的题库级案例编辑器和真实执行链路，避免在平台页复制两套复杂编辑器，同时仍形成可复现的版本化评测资产。

## 权限与数据隔离

- HTTP 路由统一要求 Contributor 及以上角色，Viewer 返回 403。
- Admin/Owner 可以查看当前 tenant 的全部题库、运行和评测集。
- Contributor 只能查看其现有题库权限允许访问的题库、运行和评测集。
- Service 层根据调用上下文中的 tenant role 计算可见题库范围；Repository 不接受未经校验的用户输入作为授权依据。
- 所有读取和写入都必须包含 tenant_id；非管理员查询还必须包含可见 question_bank_id 集合。
- 设置基线、创建评测集、追加版本和按版本运行前，必须再次校验目标运行或题库可见性。

## 持久化模型

迁移 `000082_exam_evaluation_center`：

1. 扩展 `exam_rag_evaluation_runs`
   - `is_baseline BOOLEAN NOT NULL DEFAULT FALSE`
   - `evaluation_set_id VARCHAR(36)`
   - `evaluation_set_version INTEGER`
   - 同一 `(tenant_id, question_bank_id, evaluation_kind, agent_id)` 最多一个基线。

2. 新增 `exam_evaluation_sets`
   - ID、tenant、题库、类型、Agent、名称、描述、当前版本、创建者、状态和时间戳。
   - RAG 的 agent_id 为空；Agent 类型必须保存 agent_id。

3. 新增 `exam_evaluation_set_versions`
   - ID、set_id、tenant、version、source_run_id、definition_snapshot、created_by 和 created_at。
   - `(set_id, version)` 唯一；版本只追加，不更新和删除。
   - definition_snapshot 直接复制来源运行的 request_snapshot，保证重跑契约与原运行一致。

## API

### 聚合与基线

- `GET /api/v1/exam/evaluation-center`
  - 参数：`kind`、`bank_id`、`agent_id`、`status`、`regression`、`limit`。
  - 返回可见题库选项、汇总指标以及带基线比较的运行列表。
- `PUT /api/v1/exam/evaluation-center/baseline`
  - body：`{"run_id":"..."}`。
  - 仅允许把已完成运行设为基线；事务内清除同作用域旧基线。

### 评测集与版本

- `GET /api/v1/exam/evaluation-sets`
- `POST /api/v1/exam/evaluation-sets`
  - body：`run_id`、`name`、`description`。
  - 从已完成运行创建评测集及版本 1。
- `POST /api/v1/exam/evaluation-sets/:set_id/versions`
  - body：`run_id`。
  - 来源运行必须与评测集的题库、类型和 Agent 一致。
- `POST /api/v1/exam/evaluation-sets/:set_id/runs`
  - body：可选 `version`，缺省使用当前版本。
  - 复用现有 RAG/Agent 运行服务创建真实异步运行，并记录评测集来源。

## 回归计算

每条已完成运行与同作用域基线比较：

- RAG：总体通过率、召回通过率、答案覆盖率、Recall@K、MRR、结构化解析率和平均耗时。
- Agent：总体通过率、工具顺序、工具参数、证据、引用、答案、groundedness 和平均耗时。
- 质量指标下降超过 1 个百分点记为回归。
- 平均耗时同时增加超过 10% 且超过 100ms 记为性能回归。
- 无基线、非完成状态和基线自身标记为 `uncompared`，不误报回归。
- 返回具体回归原因和每个指标的 delta，前端不自行推导业务规则。

## 前端信息架构

新增一级路由 `/platform/evaluations` 和菜单“评测中心”，最低角色 Contributor。

页面分为两个视图：

1. 回归运行
   - RAG/Agent 分段切换。
   - 题库、Agent、状态和回归状态筛选。
   - 汇总带显示运行数、完成数、基线数、回归数和平均通过率。
   - 稠密运行列表显示题库、评测集版本、核心指标、基线 delta 和状态。
   - 操作包括设置基线、保存为评测集、打开题库级 RAG/Agent 详情。

2. 评测集
   - 显示名称、题库、类型、Agent、当前版本和更新时间。
   - 支持从兼容运行追加版本、运行当前/指定版本。
   - 版本历史在行内展开，显示来源运行和创建时间。

桌面端使用全宽工作台布局；390px 下筛选单列、表格转堆叠条目、长 ID 和错误信息可换行，不产生页面横向滚动。

## 错误处理

- 运行未完成：不能设为基线、创建评测集或作为新版本来源。
- 版本来源不兼容：返回 validation error，不创建半成品版本。
- 评测集重跑时题库、Agent 或权限已失效：返回明确业务错误，不入队。
- 设置基线和版本递增使用事务，避免并发产生多个基线或重复版本。
- 聚合列表中的损坏快照不阻塞整页；该运行标记为不可比较并保留原始状态。

## 验收标准

- Admin/Owner 能查看 tenant 内跨题库运行；Contributor 只能看到可访问题库；Viewer API 返回 403。
- 可以从 RAG 和 Agent 已完成运行分别创建评测集、追加版本并重跑。
- 设置新基线后旧基线被自动替换，列表返回确定性 delta 和回归原因。
- 平台运行可下钻到现有题库级 RAG trace 或 Agent 工具轨迹页面。
- Docker 迁移、真实 API、PostgreSQL 状态、1440px 桌面页面和 390px 移动页面均完成验证。

## 非目标

- 不使用 LLM-as-judge。
- 不自动生成评测案例。
- 不在平台页复制题库级案例编辑器。
- 不允许跨 tenant 共享评测集。
- 本阶段不做评测集删除、协作编辑或外部导入导出。
