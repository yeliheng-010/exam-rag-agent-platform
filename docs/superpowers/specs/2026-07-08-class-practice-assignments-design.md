# 班级练习任务发布设计

## 背景

平台已经支持班主任创建班级、审核学生加入、把资料结构化为正式题组，并支持学生按题组练习和错题复盘。下一步需要把这些能力串成教学闭环：老师在班级里发布练习任务，学生在学习中心看到任务并进入练习。

## 目标

- 老师或助教可以在班级详情里选择正式题组发布练习任务。
- 学生只能看到自己已加入班级的已发布任务。
- 学生从任务进入练习时，练习 attempt 记录关联 `assignment_id`。
- 任务列表展示题组、题数、截止时间和当前学生的完成状态。
- 权限继续以班级 active 成员关系和班级写权限为准。

## 非目标

- 本阶段不做作业编辑、撤回、重新发布。
- 本阶段不做按学生维度的班级完成率统计。
- 本阶段不做主观题批改流和教师评分。
- 本阶段不做通知、催交和支付权益限制。

## 方案选择

推荐方案是新增 `exam_class_assignments` 表，并在 `exam_practice_attempts` 增加可选 `assignment_id`。相比只把题组链接展示给学生，这个方案可以区分“同一题组被多次布置”的不同任务，也能为后续班级统计提供稳定主键。

## 数据设计

新增 `exam_class_assignments`：

- `id`: 任务 ID
- `tenant_id`: 租户
- `class_id`: 班级
- `space_id`: 班级空间，冗余用于快速过滤和隔离
- `question_bank_id`: 题库
- `group_id`: 题组
- `title`: 任务标题
- `instructions`: 老师说明
- `status`: `published` 或 `archived`
- `due_at`: 可选截止时间
- `created_by_user_id`: 发布人
- `created_at` / `updated_at`

扩展 `exam_practice_attempts.assignment_id`，学生从任务入口创建 attempt 时写入该字段。普通题组练习继续允许 `assignment_id` 为空。

## 后端接口

- `POST /api/v1/exam/classes/:class_id/assignments`
  - 班级老师或助教创建任务。
  - 题组必须属于该班级空间，避免跨班级发布。
- `GET /api/v1/exam/classes/:class_id/assignments`
  - 班级 active 成员可查看该班任务。
- `GET /api/v1/exam/assignments`
  - 当前用户查看自己所有 active 班级的任务。
- `POST /api/v1/exam/assignments/:assignment_id/attempts`
  - 当前 active 学生或老师从任务创建练习 attempt。

接口返回 `ExamAssignmentSummary`，包含 assignment、题组、题库名、题数和当前用户最近一次关联 attempt。

## 前端设计

班级详情页将原来的“作业”占位页签升级为“练习任务”：

- 老师或助教看到发布按钮。
- 发布弹窗选择题组，填写标题、说明和截止时间。
- 列表展示任务信息、题组信息、完成状态和进入练习按钮。

学习中心新增“班级任务”区域：

- 优先展示老师布置的任务。
- 每张任务卡展示班级、题组、题数、截止时间和完成状态。
- 点击后通过 assignment attempt 接口创建练习并进入现有练习页。

## 验收标准

- 学生无法给班级发布任务。
- 老师不能把非本班 space 的题组发布到班级。
- active 班级成员能看到任务，pending 或非成员不能看到。
- 从任务创建的 attempt 会写入 `assignment_id`。
- 学习中心能展示班级任务并进入练习。
