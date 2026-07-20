# 班级设置与可逆归档设计

## 背景

考试平台已经支持创建班级、审核成员、绑定资料、发布练习任务、查看学情以及作业通知与催交，但班级详情中的“设置”仍是占位页。班级创建后无法修改名称、描述或成员上限，也无法结束一个班级的运营周期。

现有 `exam_classes.status` 已定义 `active` 和 `archived`，因此本阶段复用现有字段完成软归档，不新增迁移，不删除班级、成员、作业、练习记录或通知。

## 目标

- 班主任可以修改班级名称、描述和成员上限。
- 班主任可以归档 active 班级，也可以恢复 archived 班级。
- 助教和学生可以查看设置摘要，但不能修改、归档或恢复。
- 归档班级保留历史数据，同时禁止加入、发布与维护作业、催交以及创建新的作业 attempt。
- 班级列表可以显示当前用户所属的归档班级，学习中心默认仍只展示 active 班级。
- 成员上限成为真实约束，而不是只展示的字段。

## 非目标

- 不删除班级或级联删除任何业务数据。
- 不允许修改班级考试域、班级空间、班主任或邀请码。
- 不建设题集、主观题批改、教师评语或支付权益。
- 不自动撤回已发布作业，也不在恢复班级时自动重新发布作业。
- 不修改已存在 attempt 的查看、继续作答和复盘能力。

## 方案选择

### 方案 A：独立的含归档读取与原子状态转换，推荐

保留现有 active-only repository 方法作为业务写保护，新增含归档的读取方法供班级列表、详情和设置使用；归档与恢复使用 expected-status 条件更新。优点是不会因为放宽一个共享查询而意外开放归档班级的作业、资源或分析操作。

### 方案 B：取消 repository 的 active 过滤并在每个 service 手动判断

改动表面较少，但所有现有调用者都需要重新审计，遗漏一个状态判断就会让归档班级继续写入。拒绝采用。

### 方案 C：删除班级

会破坏成员、作业、attempt、通知和审计记录，不满足可恢复要求。拒绝采用。

## 权限模型

- 更新、归档、恢复：必须满足当前 tenant、`class.owner_user_id == current_user_id`，且班主任仍是 active class member。
- 查看归档班级详情：当前用户必须仍是 active class member。
- 助教继续拥有 active 班级的成员审核、资料、作业和分析权限，但不能修改班级设置或归档状态。
- 学生只能查看设置摘要。
- 对跨 tenant、非成员和错误班级 ID 继续使用现有 404/403 语义，不暴露额外信息。

## 数据与 Repository 边界

不新增数据库字段。扩展 `ExamClassRepository`：

```go
ListByUserWithStatuses(ctx, tenantID, userID, statuses)
GetByIDAndTenantIncludingArchived(ctx, classID, tenantID)
ListByIDsAndTenantIncludingArchived(ctx, tenantID, classIDs)
UpdateClassMetadata(ctx, tenantID, classID, ownerUserID, name, description, memberLimit, updatedAt)
TransitionClassStatus(ctx, tenantID, classID, ownerUserID, expected, next, updatedAt)
ApproveMemberWithinLimit(ctx, tenantID, classID, userID, joinedAt)
```

`UpdateClassMetadata` 在事务中锁定 class 行，确认 active 与 owner，再统计 active 成员。非零 `member_limit` 小于当前 active 成员数时返回状态冲突。

`ApproveMemberWithinLimit` 在同一事务中锁定 active class 行，读取当前 active 成员数并更新指定 pending student。`member_limit == 0` 表示不限；达到上限时返回状态冲突。这样并发审批不能越过上限。

`TransitionClassStatus` 使用 `tenant_id + id + owner_user_id + expected status` 条件更新；状态已经变化时返回状态冲突。

## Service 与接口

新增请求：

```go
type UpdateExamClassRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    MemberLimit int    `json:"member_limit"`
}
```

校验规则：

- name 去除首尾空白后长度为 1 到 255 个字符。
- description 去除首尾空白后最多 2000 个字符。
- member_limit 为 0 到 1000；0 表示不限。
- 更新只允许 active 班级。

HTTP 路由：

```http
GET  /api/v1/exam/classes?include_archived=true
PUT  /api/v1/exam/classes/:class_id
POST /api/v1/exam/classes/:class_id/archive
POST /api/v1/exam/classes/:class_id/restore
```

成功响应继续使用 `{"success":true,"data":class}`。非法字段返回 400，无权限返回 403，不可见返回 404，成员上限或并发状态冲突返回 409。

## 归档后的业务约束

现有 `GetByIDAndTenant` 继续只返回 active 班级，因此依赖它的资料、作业、分析和催交写路径保持关闭。另补两处显式约束：

- 创建 assignment attempt 前必须确认 class 仍为 active；历史 attempt 的读取和继续作答不受影响。
- 通知列表计算 `can_start` 时批量读取班级状态；归档班级无 attempt 的通知不能创建新 attempt，已有 attempt 仍优先跳转。

加入班级继续使用 active-only invite code 查询。学习中心的班级列表不请求归档数据，因此归档班级不会出现在学习入口和可开始任务中。

## 前端设计

### 班级列表

班级管理页请求 `include_archived=true`，卡片继续展示状态标签。归档班级可进入详情，以便班主任恢复；学习中心保持默认 active-only 请求。

### 班级详情设置页

将占位“设置”升级为真实 tab：

- 名称输入框、描述文本域、成员上限数字输入。
- 班主任看到保存按钮与归档/恢复命令。
- 助教和学生看到只读摘要及权限提示。
- 归档使用确认对话框；恢复是明确的命令按钮。
- 请求期间禁用重复提交，成功后刷新 `classInfo` 和表单快照。

班级为 archived 时，成员、资料、练习任务和分析 tab 禁用，概览和设置可用。归档成功后保持在设置 tab，不触发已禁用业务请求。

### 移动端

设置表单使用单列布局，命令区允许换行；归档确认框使用现有 TDesign 响应式宽度，不产生横向滚动。

## 错误与并发

- 两个并发更新或状态转换只有满足 expected status/owner 的事务成功。
- 并发成员审批通过 class 行锁串行化，不能超过 member limit。
- 更新时成员数已经超过新上限则整体回滚。
- archive 和作业写入都依赖 class 行状态；后到达的写请求看到 archived 后失败。
- 前端失败时保留原表单与当前状态，显示服务端错误，不做乐观状态修改。

## 测试与验收

- Repository 覆盖 tenant/owner/status 条件、元数据更新回滚、状态转换和并发安全的成员上限检查。
- Service 覆盖 owner 成功、assistant/student 拒绝、字段校验、归档详情可读、默认列表 active-only、include archived 列表。
- Assignment 覆盖 archived class 不能创建新 attempt；通知列表对 archived class 返回 `can_start=false`，已有 attempt 仍可继续。
- Handler 与 router 覆盖三个新写接口和 `include_archived` 查询。
- 前端测试覆盖 API、设置页权限、保存、归档、恢复、禁用业务 tab 和移动端布局。
- 最终运行 Go 测试、`go vet`、前端全量测试和生产构建。
- Docker 重建后验证迁移仍为 `83 / dirty: false`、app healthy、frontend 200。
- 有效 owner/assistant/student 会话可用时完成真实权限与归档恢复验收；会话不可用时明确保留该缺口，不猜测凭据。
