# 作业通知与催交设计

## 背景

班级作业已经具备发布、编辑、软撤回、重新发布、截止控制、学生作答和教师进度查看，但学生只能通过主动打开学习中心发现任务，教师也无法针对未完成学生发起可追踪的催交。项目现有右上角铃铛仅处理租户邀请，IM 通知链路仅服务文件解析，不存在可直接复用的通用站内通知模型。

本阶段在作业域内补齐持久化站内通知和手动催交，不扩展为全站消息平台，也不接入外部渠道。

## 目标

- 作业发布、重新发布和撤回时，自动通知事件发生时的 active 学生。
- 教师或助教可以一键催交所有未完成学生，也可以单独催交某位未完成学生。
- 同一作业、同一学生在滚动 24 小时内最多收到一次手动催交。
- 学生可以通过统一铃铛查看未读数、最近通知、已读状态，并跳转或继续对应任务。
- 自动通知与作业生命周期写入同一事务，避免任务状态已改变但通知遗漏。
- 通知始终受 tenant、班级成员状态和作业权限约束。

## 非目标

- 不接入短信、邮件、企业微信、飞书、Slack 或其他 IM 渠道。
- 不实现 WebSocket、SSE 或浏览器系统通知，首期使用轻量轮询。
- 不实现定时催交、截止前自动提醒、通知模板管理或用户通知偏好。
- 不建设跨业务域的通用通知平台，不迁移租户邀请的数据模型。
- 不允许自定义催交文案，首期使用确定性系统文案。
- 不为通知提供删除、撤回或管理员审计页面。

## 方案选择

采用“作业域专用持久化通知”方案。新增 `exam_assignment_notifications`，只承载作业发布、重发、撤回和催交四类事件。相比全站通用通知平台，该方案不会提前引入模板、渠道和偏好系统；相比前端 toast 或即时推送，持久化记录能够支持未读状态、历史查看、限频和可靠测试。

现有租户邀请继续保留原模型和接口。前端将邀请入口与作业通知聚合到同一个全局铃铛，铃铛角标是“作业未读数 + 待处理邀请数”，但两类数据仍由各自 API 管理。

## 通知类型与接收人

`ExamAssignmentNotificationKind` 包含：

- `published`：首次发布作业。
- `republished`：撤回后重新发布。
- `withdrawn`：已发布作业被撤回。
- `reminder`：教师或助教手动催交。

自动通知以生命周期操作开始时查询到的 active student 成员为接收人。pending、rejected、removed 成员不接收。成员在事件发生后才加入班级时，不补发历史自动通知；其任务发现入口仍是学习中心。

手动催交只面向 active student 且当前进度不是 `completed` 的成员，包括 `not_started` 和 `in_progress`。服务端重新计算接收人，不能信任前端传入的完成状态。

## 数据模型

新增迁移版本 `000083_exam_assignment_notifications` 和类型 `ExamAssignmentNotification`：

```text
id                  varchar(36) primary key
tenant_id           bigint not null
class_id            varchar(36) not null
assignment_id       varchar(36) not null
group_id            varchar(36) not null
recipient_user_id   varchar(36) not null
actor_user_id       varchar(36) not null
kind                varchar(32) not null
title               varchar(255) not null
content             text not null
read_at             timestamp null
created_at           timestamp not null
```

索引：

- `(tenant_id, recipient_user_id, read_at, created_at DESC)`：未读数与收件箱。
- `(tenant_id, assignment_id, recipient_user_id, kind, created_at DESC)`：催交限频与作业追踪。
- `(tenant_id, class_id, created_at DESC)`：班级范围排查。

`title`、`content` 和 `group_id` 保存事件发生时的快照。后续编辑作业标题不会改写旧通知；`group_id` 在作业生命周期内不可变，可用于直接继续已有 attempt。通知不依赖作业当前是否对学生列表可见。

迁移 down 只删除新表和索引，不触碰作业、attempt 或租户邀请数据。

## 文案规则

服务端生成确定性文案，前端只负责展示：

- published：标题“新作业：{assignment_title}”，内容包含班级名和截止时间。
- republished：标题“作业重新发布：{assignment_title}”，内容包含新的截止时间。
- withdrawn：标题“作业已撤回：{assignment_title}”，内容说明已有答题记录仍然保留。
- reminder：标题“作业待完成：{assignment_title}”，内容包含发送人、当前进度和截止时间。

没有截止时间时统一显示“无固定截止时间”。文案不包含富文本或用户输入的 HTML，前端按纯文本渲染。

## 生命周期事务边界

首次发布：

1. service 校验班级写权限、题组和 active student 列表。
2. repository 在一个数据库事务中插入 assignment 和每位学生的 published 通知。
3. 任一写入失败则整体回滚，接口返回错误，不留下半完成发布。

撤回与重新发布：

1. service 校验写权限并准备接收人和通知快照。
2. repository 在一个事务中锁定 assignment 行，重新检查期望状态和截止条件。
3. 更新状态并批量插入 withdrawn 或 republished 通知。
4. 状态冲突或通知写入失败时整体回滚。

生命周期 repository 的事务方法接收已构造的通知列表，不在仓储层查询班级成员。成员选择和文案属于 service 职责，状态原子性和持久化属于 repository 职责。

自动通知生成失败不是 best-effort 警告，而是生命周期操作失败；调用方可以安全重试，因为失败事务不会留下状态变更。

## 催交规则与并发

新增：

```http
POST /api/v1/exam/classes/:class_id/assignments/:assignment_id/reminders
```

请求体：

```json
{
  "recipient_user_ids": ["optional-student-id"]
}
```

- `recipient_user_ids` 为空或缺省：尝试催交全部未完成学生。
- 非空：只尝试指定学生，但服务端仍要求其是该班 active student 且未完成。
- 只有 active teacher/assistant 可以调用。
- assignment 必须是未截止的 `published`；withdrawn、archived 或已截止返回 `409`。

repository 在事务中锁定 assignment 行，以便与撤回、重发和并发催交串行化。随后读取接收人的最新 attempt 和最近 reminder：

- 已完成：跳过，记入 `completed_skipped_count`。
- 最近 reminder 距当前不足 24 小时：跳过，记入 `cooldown_skipped_count`。
- 其他未完成成员：批量插入 reminder 通知。

完成状态和限频按事务查询时的数据库快照判断。学生恰好在催交事务同时完成作业时，允许收到一次边界提醒，不回滚学生完成结果。

响应：

```json
{
  "sent_count": 3,
  "completed_skipped_count": 1,
  "cooldown_skipped_count": 2,
  "sent_user_ids": ["student-1", "student-2", "student-3"]
}
```

## 进度接口扩展

`ExamAssignmentMemberProgress` 增加：

```text
last_reminded_at *time.Time
can_remind       bool
```

`can_remind` 由服务端根据作业状态、截止时间、学生完成状态和 24 小时限频计算。前端不自行推断权限。作业进度顶部显示“一键催交”按钮；表格增加“催交”列：

- 可催交：显示发送图标按钮，tooltip 为“催交该学生”。
- 24 小时内已催：禁用并显示“24 小时内已催”。
- 已完成：不显示催交按钮。

批量操作完成后重新加载进度，确保按钮状态、计数和最近催交时间来自服务端。

## 学生通知接口

新增 tenant 和当前用户范围内的接口：

```http
GET  /api/v1/exam/notifications?limit=50
POST /api/v1/exam/notifications/:notification_id/read
POST /api/v1/exam/notifications/read-all
```

列表按 `created_at DESC, id DESC` 返回最多 50 条最近通知，并同时返回 `unread_count`。首期不做翻页；达到 50 条时仍保留数据库历史，只是 UI 展示最近 50 条。

列表项包含通知字段和可选的 `last_attempt_id`。service 批量查询当前用户在相关 assignment 下的最近 attempt：

- 存在 attempt：前端直接进入原题组并携带 `attempt_id`，无论作业当前是否撤回或截止。
- 不存在 attempt 且任务仍可开始：跳转学习中心并定位对应任务。
- 不存在 attempt 且任务已撤回或截止：标记已读并提示当前不可开始，不创建 attempt。

单条已读必须同时匹配 tenant、recipient 和 notification ID。`read-all` 只更新当前 tenant 下当前用户的未读作业通知。

错误语义沿用 exam 模块：非法请求 `400`、无权限 `403`、通知或任务不可见 `404`、状态或截止冲突 `409`。

## 统一铃铛交互

平台当前只挂载 `GlobalInvitationBell`。本阶段将其替换为 `GlobalNotificationBell`，保留原邀请对话框作为通知中心中的“待处理邀请”入口。

铃铛行为：

- 登录后的 platform 页面始终显示 32px 图标按钮，不因未读数为 0 而消失。
- badge 总数为 assignment unread count 与 pending invitation count 之和，最大显示 99。
- 首次挂载立即加载，之后每 30 秒轮询；页面不可见时暂停，恢复可见后立即刷新。
- 点击打开单层通知对话框，不嵌套卡片。
- 顶部显示“通知”和“全部已读”；存在邀请时显示独立的“待处理邀请”行。
- 作业通知按时间倒序展示未读点、类型图标、标题、纯文本摘要和时间。
- 点击通知先标记已读，再按 `last_attempt_id` 和作业当前状态决定跳转。
- 空状态显示简洁的“暂无通知”。

桌面对话框最大宽度 560px；移动端使用 `calc(100vw - 24px)`，通知标题和长内容允许换行，操作区不产生横向滚动。图标按钮使用 TDesign 已有 notification、send、check 等图标并提供 tooltip。

## 前端数据流

新增 assignment notification API 与纯函数 helper，分别负责：

- 规范化未读总数。
- 计算铃铛总 badge。
- 根据 notification、last attempt 和 lifecycle 状态决定点击目标。
- 格式化通知类型标签和相对时间所需的稳定输入。

`ClassDetail.vue` 只负责调用催交 API 和刷新进度；通知中心独立组件负责轮询、已读和跳转；`LearningHome.vue` 读取 `assignment_id` 查询参数，在列表加载后滚动并短暂突出对应任务。三者不共享可变 store，避免把作业通知扩散到全局认证状态。

## 代码边界

后端新增或修改：

- `migrations/versioned/000083_exam_assignment_notifications.{up,down}.sql`
- `internal/types/exam_assignment_notification.go`
- `internal/types/interfaces/exam_assignment_notification.go`
- `internal/application/repository/exam_assignment_notification.go`
- `internal/application/service/exam_assignment_notification.go`
- `internal/handler/exam_assignment_notification.go`
- 现有 assignment repository/service/handler/router 的事务接入点
- container provider 注册

前端新增或修改：

- `frontend/src/types/exam.ts`
- `frontend/src/api/exam/assignmentNotification.ts`
- `frontend/src/views/classes/ClassDetail.vue`
- `frontend/src/views/learning/LearningHome.vue`
- `frontend/src/components/GlobalNotificationBell.vue`
- `frontend/src/components/ExamAssignmentNotificationDialog.vue`
- `frontend/src/views/platform/index.vue`
- 配套纯函数与 source contract 测试

删除或改名 `GlobalInvitationBell.vue` 仅在统一铃铛接入完成后进行；原 `MyInvitationsDialog.vue` 和邀请 API 保持不变。

## 测试策略

Repository：

- migration up/down 结构正确。
- assignment 创建与通知批量插入同事务成功或整体回滚。
- 状态转换与通知同事务，旧状态冲突不产生通知。
- notification 列表、未读数、单条已读和全部已读严格按 tenant/user 隔离。
- 并发催交在 assignment 行锁下遵守 24 小时限频。

Service：

- student、pending 成员和跨班用户不能催交。
- published/republished/withdrawn 自动通知只发送给 active student。
- 已完成学生不被催交，not_started/in_progress 可以被催交。
- 指定用户催交不能绕过成员、完成状态和限频校验。
- withdrawn/expired 作业不能催交。
- 进度返回 `last_reminded_at` 和可信的 `can_remind`。
- 通知列表只包含当前 tenant/user，并正确附带最近 attempt。

HTTP：

- 新路由和 Viewer guard 存在。
- 请求绑定、权限错误和 `409` 状态映射正确。
- 已读接口不能修改其他用户通知。

Frontend：

- API 路径、方法和 payload 正确。
- badge 合并作业未读和邀请待处理数量。
- 轮询在隐藏页面暂停并正确清理 timer。
- 通知点击优先继续已有 attempt，否则定位学习中心或显示不可开始提示。
- 批量和单人催交操作后刷新进度。
- 24 小时内已催和已完成行不可重复操作。
- 355px 移动视口无横向溢出。

最终验证包括 Go 定向与全量测试、`go vet`、前端全量测试、Vite 构建、Docker 重建、迁移版本 `83 / dirty: false`、真实 API 状态以及桌面和移动端浏览器检查。登录会话不可用时必须明确报告浏览器验收缺口，不得用未认证健康检查替代业务验收。

## 验收标准

- 生命周期自动通知与 assignment 写入具备事务一致性。
- active student 能看到发布、重发和撤回通知，其他成员状态不能收到。
- 教师/助教能批量或单独催交，学生不能调用催交接口。
- completed 学生不被催交，同一未完成学生 24 小时内不能重复收到催交。
- 学生能查看最近 50 条通知、未读数并执行单条或全部已读。
- 已有 attempt 的通知可直接继续；撤回或截止不会创建新 attempt。
- 统一铃铛同时承载作业未读和待处理邀请，邀请原行为不回归。
- 桌面和移动端通知中心、进度催交操作均可用且无横向溢出。
- 所有测试、构建、迁移和运行健康检查通过；真实登录业务验收有可追溯证据或明确阻塞说明。
