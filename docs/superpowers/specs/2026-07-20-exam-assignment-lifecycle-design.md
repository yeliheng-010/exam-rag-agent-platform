# 班级作业运营闭环设计

## 背景

班级练习任务已经支持发布、学生进入练习、attempt 关联和教师查看完成进度，但任务发布后不能编辑、撤回或重新发布。当前 `due_at` 只用于展示，截止后仍可创建新 attempt；前端“继续练习”也会再次创建 attempt。需要在不破坏历史答题记录的前提下补齐首期运营闭环。

## 目标

- 班级老师或助教可以编辑尚未结束的任务元数据。
- 已发布任务可以软撤回，已撤回任务可以编辑后重新发布。
- 撤回、截止和重新发布遵循明确且可并发校验的状态规则。
- 历史 attempt 始终保留原 `assignment_id`，不因撤回或重新发布而删除或改绑。
- 学生不能在任务撤回或截止后创建新 attempt，但已经开始的 attempt 可以继续完成。
- 教师仍能查看已撤回任务及其历史进度，班级分析继续只统计当前已发布任务。

## 非目标

- 本阶段不做通知、站内信、短信、邮件或催交。
- 本阶段不做主观题批改、教师评分和退回重做。
- 本阶段不做任务版本链、变更审计表或操作日志页面。
- 本阶段不允许更换已发布任务的题组。
- 本阶段不新增定时任务自动修改任务状态。
- 本阶段不物理删除任务或 attempt。

## 方案选择

采用“同一任务软撤回并原 ID 重发”的方案。新增 `withdrawn` 状态，使用 `published -> withdrawn -> published` 状态机。相比复用 `archived`，该方案能区分可恢复撤回和历史归档；相比重新发布时创建新任务版本，该方案不需要版本链，也不会拆散同一次作业的历史 attempt 与进度。

`archived` 保留为兼容既有数据的终态。本阶段不提供进入 `archived` 的新操作，也不允许从 `archived` 编辑或重新发布。

## 任务状态与截止语义

持久化状态只有 `published`、`withdrawn` 和兼容状态 `archived`。“已截止”是由 `status == published && due_at <= now` 计算出的展示状态，不写回数据库。

| 状态 | 学生列表可见 | 可创建新 attempt | 已有 attempt 可继续 | 可编辑元数据 | 可撤回 | 可重新发布 |
| --- | --- | --- | --- | --- | --- | --- |
| `published`，未截止 | 是 | 是 | 是 | 是 | 是 | 否 |
| `published`，已截止 | 是 | 否 | 是 | 否 | 是 | 否 |
| `withdrawn` | 否 | 否 | 是 | 是 | 否 | 是，前提是新截止时间为空或晚于当前时间 |
| `archived` | 否 | 否 | 是 | 否 | 否 | 否 |

已截止任务若需延期，老师必须先撤回，再修改截止时间，最后重新发布。这样可以避免学生正在进入任务时直接改变截止规则，也让运营动作具有清晰顺序。

## 可编辑字段

编辑接口只接受并完整替换以下元数据：

- `title`：去除首尾空白后必须为 1 到 255 个字符。
- `instructions`：去除首尾空白后最多 2000 个字符，可以为空。
- `due_at`：可以为空；非空时必须是有效时间。

`class_id`、`space_id`、`question_bank_id`、`group_id`、`created_by_user_id` 和 `created_at` 永久不可编辑。题组不可变可以确保所有历史 attempt 的题目语义保持一致。

采用完整元数据更新而不是稀疏 PATCH，以便用 `due_at: null` 明确清除截止时间：

```json
{
  "title": "函数综合练习",
  "instructions": "完成后复盘错题",
  "due_at": "2026-07-30T15:59:00Z"
}
```

## 数据模型

`exam_class_assignments.status` 已是无枚举约束的 `VARCHAR(32)`，因此无需数据库迁移。后端和前端类型增加 `withdrawn`：

```text
ExamAssignmentStatusPublished = "published"
ExamAssignmentStatusWithdrawn = "withdrawn"
ExamAssignmentStatusArchived  = "archived"
```

状态变化和元数据更新都刷新 `updated_at`。本阶段不新增 `withdrawn_at` 或版本字段；需要精细审计时再独立设计操作日志。

## 后端接口

新增三个班级任务接口：

- `PUT /api/v1/exam/classes/:class_id/assignments/:assignment_id`
  - 完整更新 `title`、`instructions`、`due_at`。
  - 仅允许班级老师或助教操作。
  - 仅允许未截止的 `published` 或 `withdrawn` 任务。
- `POST /api/v1/exam/classes/:class_id/assignments/:assignment_id/withdraw`
  - 原子地执行 `published -> withdrawn`。
  - 已截止的 `published` 任务也允许撤回。
- `POST /api/v1/exam/classes/:class_id/assignments/:assignment_id/republish`
  - 原子地执行 `withdrawn -> published`。
  - `due_at` 非空且不晚于当前时间时拒绝重新发布。

三个接口都返回更新后的 `ExamAssignmentSummary`。路由继续使用租户 `Viewer` guard，实际写权限由 service 校验 active 班级成员且角色为 teacher 或 assistant，保持与现有发布接口一致。

错误语义：

- 请求字段非法返回 `400`。
- 非班级成员或学生写操作返回 `403`。
- 任务不存在、租户或班级不匹配返回 `404`。
- 当前状态不允许操作、任务已截止或并发状态已经变化返回 `409`。

新增 `ErrExamStateConflict`，由 `writeExamError` 映射为 conflict error。状态转换使用带期望旧状态的条件更新；更新影响行数不是 1 时返回冲突，避免重复点击或并发请求产生非法转换。

## 列表、进度与分析

现有班级任务列表按调用者角色返回不同状态：

- teacher/assistant：返回 `published` 和 `withdrawn`，按 `created_at DESC` 排序。
- student：只返回 `published`。
- `GET /api/v1/exam/assignments`：继续只返回当前用户 active 班级的 `published` 任务。

Repository 的班级列表方法接收明确的状态集合，调用方不能依赖隐式默认值。班级分析显式传入 `[published]`，因此撤回任务不会继续占用当前作业数或完成率分母。

`GET /classes/:class_id/assignments/:assignment_id/progress` 对 teacher/assistant 允许读取 `published` 和 `withdrawn`，从而保留撤回后的历史查看入口；`archived` 不进入本阶段管理列表。

## Attempt 行为

`POST /assignments/:assignment_id/attempts` 在原有租户、班级成员和状态校验上增加截止校验：

- 任务必须为 `published`。
- `due_at` 为空或晚于当前时间。
- 不满足条件返回 `409`，不得创建数据库记录。

继续已有 attempt 不调用创建接口。前端若 `last_attempt` 存在，直接携带其 ID 进入现有练习页；只有没有 attempt 的任务才调用创建接口。这样撤回或截止不会中断已经开始的答题，也不会因点击“继续”生成重复 attempt。

提交答案和完成 attempt 的接口不新增 assignment 状态检查。其职责仍是维护一个已存在 attempt 的生命周期，避免老师撤回任务时破坏学生已产生的答题数据。

## 前端交互

### 教师班级任务页

教师和助教看到 `published` 与 `withdrawn` 任务。每张任务卡展示一个明确状态标签：

- 未截止的 `published`：`进行中`
- 已截止的 `published`：`已截止`
- `withdrawn`：`已撤回`

任务卡操作保持紧凑：

- 未截止 published：编辑、查看结果、撤回。
- 已截止 published：查看结果、撤回。
- withdrawn：编辑、查看结果、重新发布。

编辑复用现有发布表单的标题、说明和截止时间控件，但题组只读展示。撤回使用确认对话框，明确提示“学生入口将隐藏，历史答题不会删除”。重新发布前端先检查截止时间，后端仍做最终校验。

### 学生入口

学习中心和班级详情只收到 `published` 任务：

- 未截止且无 attempt：按钮为“开始任务”，创建 attempt 后进入练习。
- 已有 attempt：按钮为“继续任务”，直接进入最近 attempt。
- 已截止且无 attempt：显示“已截止”，按钮禁用。
- 已截止且有 attempt：仍可“继续任务”。

撤回任务在下一次列表刷新后消失。已打开的练习页可以继续完成。

所有写操作按钮在请求期间进入 loading 状态，成功后刷新任务列表；冲突响应显示后端消息并重新加载当前任务状态。

## 代码边界

后端改动集中在现有 assignment 模块：

- `internal/types/exam_assignment.go`：状态和更新请求。
- `internal/types/interfaces/exam_assignment.go`：service/repository 新方法。
- `internal/application/repository/exam_assignment.go`：按状态列表、条件更新和状态转换。
- `internal/application/service/exam_assignment.go`：权限、截止规则和状态机。
- `internal/handler/exam_assignment.go`、`internal/handler/exam_error.go`、`internal/router/exam.go`：接口和错误映射。
- `internal/application/service/exam_analytics.go`：显式只读取 `published`。

前端沿用现有班级任务页面，不新建独立管理页：

- `frontend/src/types/exam.ts`：状态类型。
- `frontend/src/api/exam/assignment.ts`：更新、撤回和重发 API。
- `frontend/src/views/classes/ClassDetail.vue`：教师管理操作和学生截止状态。
- `frontend/src/views/learning/LearningHome.vue`：截止状态与继续已有 attempt。

不修改 shared schema、practice answer 数据结构和现有路由路径。

## 测试策略

后端使用 service 和 repository 测试覆盖：

- 学生不能编辑、撤回或重新发布。
- 跨 tenant 或跨 class 任务不能操作。
- 未截止 published 可编辑，已截止 published 编辑返回冲突。
- withdrawn 可编辑，且不能更换 group。
- published 可撤回，重复撤回返回冲突。
- withdrawn 在有效截止时间下可重发，过期截止时间重发返回冲突。
- 并发或旧状态条件更新影响 0 行时返回冲突。
- 学生列表不含 withdrawn，教师列表包含 withdrawn。
- withdrawn 仍可读取进度，班级分析只统计 published。
- 截止或撤回任务不能创建新 attempt，且失败时没有新增记录。

前端测试覆盖：

- API 路径、方法和 payload。
- 三种任务状态对应的标签与操作。
- 编辑表单不允许修改题组。
- 撤回确认文案和操作后的刷新。
- 有 `last_attempt` 时不调用创建接口，直接继续。
- 已截止且无 attempt 时禁用开始按钮。
- 移动端任务卡操作不产生横向溢出。

最终验证包括相关 Go 包测试、`go vet`、前端全量测试、Vite 生产构建，以及使用真实 Docker 服务验证教师编辑/撤回/重发、学生列表变化和历史进度保留。

## 验收标准

- teacher/assistant 能编辑未截止 published 任务和 withdrawn 任务，student 被拒绝。
- 已截止 published 任务不能直接编辑，必须撤回后编辑再重发。
- 撤回后学生任务列表不再返回该任务，且不能创建新 attempt。
- 撤回不会删除、清空或改绑任何历史 attempt。
- 已有 attempt 在截止或撤回后仍能继续提交并完成。
- 重发沿用原 assignment ID，历史进度继续聚合在同一任务下。
- 班级分析不把 withdrawn 任务计入当前统计。
- 桌面端和移动端均能完成编辑、撤回、重发和状态识别。
