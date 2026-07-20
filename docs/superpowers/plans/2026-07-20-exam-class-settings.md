# 班级设置与可逆归档实现计划

> **执行方式：** 使用 `superpowers:executing-plans` 串行执行；所有生产代码遵循 red -> green -> refactor。完成前使用 `requesting-code-review` 和 `verification-before-completion`。

**目标：** 完成班主任专属的班级元数据编辑、可逆归档和真实成员上限，并保证归档班级不能产生新的作业行为。

**设计：** `docs/superpowers/specs/2026-07-20-exam-class-settings-design.md`

**技术栈：** Go、Gin、GORM、PostgreSQL、SQLite repository tests、Vue 3、TypeScript、TDesign、Node test runner、Docker Compose。

---

## 任务 1：类型与接口契约

**文件：**

- 修改：`internal/types/exam_class.go`
- 修改：`internal/types/interfaces/exam_class.go`
- 新建：`internal/types/exam_class_settings_test.go`

- [ ] 编写红灯测试，固定 `UpdateExamClassRequest`、含归档读取、元数据更新、状态转换和受限审批接口。
- [ ] 运行 `go test ./internal/types -run TestExamClassSettingsContract -count=1`，确认因契约缺失失败。
- [ ] 实现最小类型和接口；`member_limit` 约定 0 表示不限。
- [ ] 运行类型测试，预期 PASS。
- [ ] 提交：`feat(exam): 增加班级设置与归档契约`

## 任务 2：Repository 原子更新与成员上限

**文件：**

- 修改：`internal/application/repository/exam_class.go`
- 修改：`internal/application/repository/exam_class_test.go`

- [ ] 编写红灯测试，覆盖默认列表只返回 active、管理列表包含 archived、含归档详情 tenant 隔离、owner 元数据更新、降低上限回滚、expected-status 转换和成员审批限额。
- [ ] 运行 `go test ./internal/application/repository -run TestExamClassRepository -count=1`，确认新增方法缺失。
- [ ] 实现状态过滤读取、事务内 class 行锁、active 成员计数、元数据更新、状态转换和 `ApproveMemberWithinLimit`。
- [ ] Repository 错误固定为 not found、state conflict、member limit conflict，禁止用错误字符串分支。
- [ ] 运行 repository 测试，预期 PASS。
- [ ] 提交：`feat(exam): 实现班级设置持久化与成员限额`

## 任务 3：Service 权限、校验与归档读取

**文件：**

- 修改：`internal/application/service/exam_class.go`
- 修改：`internal/application/service/exam_class_test.go`

- [ ] 编写红灯测试，覆盖 owner 成功、assistant/student 拒绝、字段边界、active-only 默认列表、include archived、归档详情可读、归档/恢复并发冲突和审批超限。
- [ ] 运行 `go test ./internal/application/service -run TestExamClass -count=1`，确认新行为失败。
- [ ] 实现 `UpdateClass`、`ArchiveClass`、`RestoreClass`；owner 必须是 active member。
- [ ] `ListClasses` 接收 `IncludeArchived` 过滤；`GetClass` 使用含归档读取但仍检查 active membership。
- [ ] 将 repository 冲突统一映射为 `ErrExamStateConflict`。
- [ ] 将成员审批切换为事务内限额方法。
- [ ] 运行 service 测试，预期 PASS。
- [ ] 提交：`feat(exam): 实现班级设置权限与可逆归档`

## 任务 4：归档班级的作业安全边界

**文件：**

- 修改：`internal/application/service/exam_assignment.go`
- 修改：`internal/application/service/exam_assignment_notification.go`
- 修改：`internal/application/service/exam_assignment_test.go`
- 按需修改：`internal/types/interfaces/exam_class.go`

- [ ] 编写红灯测试：archived class 不能创建新 assignment attempt；通知无 attempt 时 `can_start=false`；已有 attempt 仍优先返回。
- [ ] 运行 `go test ./internal/application/service -run "TestExamAssignment.*Archived|TestExamAssignmentNotification.*Archived" -count=1`，确认旧行为失败。
- [ ] 在新 attempt 路径显式校验 active class。
- [ ] 通知列表批量读取相关 class 状态，避免 N+1；`assignmentNotificationItem` 将 class active 纳入 `can_start`。
- [ ] 运行 assignment service 测试，预期 PASS。
- [ ] 提交：`fix(exam): 阻止归档班级产生新作业行为`

## 任务 5：HTTP 路由与错误语义

**文件：**

- 修改：`internal/handler/exam_class.go`
- 新建：`internal/handler/exam_class_settings_test.go`
- 修改：`internal/router/exam.go`
- 修改：`internal/router/exam_rbac_routes_test.go`

- [ ] 编写红灯测试，固定 `include_archived`、PUT、archive、restore 路由和 400/403/404/409 映射。
- [ ] 运行 `go test ./internal/handler ./internal/router -run "TestExamClassSettings|TestExamClassRouteGuardSourceMatrix" -count=1`，确认路由和 handler 缺失。
- [ ] 实现请求绑定、service 调用和统一 JSON 响应；三个写接口继续使用 viewer guard，由 service 执行业务权限。
- [ ] 运行 handler/router 测试，预期 PASS。
- [ ] 提交：`feat(exam): 开放班级设置与归档接口`

## 任务 6：前端类型、API 与权限纯函数

**文件：**

- 修改：`frontend/src/types/exam.ts`
- 修改：`frontend/src/api/exam/class.ts`
- 新建：`frontend/src/views/classes/classSettings.ts`
- 新建：`frontend/src/views/classes/classSettings.test.ts`
- 新建：`frontend/src/views/classes/classSettingsSource.test.ts`

- [ ] 编写红灯测试，覆盖 owner-only 判断、active/archived 命令状态、设置 payload 和四个 API 契约。
- [ ] 运行 `node --test src/views/classes/classSettings.test.ts src/views/classes/classSettingsSource.test.ts`，确认文件或导出缺失。
- [ ] 实现类型、API 和纯函数，不在组件里复制权限判断。
- [ ] 运行定向前端测试，预期 PASS。
- [ ] 提交：`feat(exam): 增加班级设置前端契约`

## 任务 7：班级列表与设置页

**文件：**

- 修改：`frontend/src/views/classes/ClassList.vue`
- 修改：`frontend/src/views/classes/ClassDetail.vue`
- 修改：`frontend/src/views/classes/classSettingsSource.test.ts`

- [ ] 先扩展 source 红灯测试，覆盖管理列表包含 archived、真实设置 tab、owner 表单、只读角色、保存、确认归档、恢复、归档业务 tab 禁用和移动端单列布局。
- [ ] 运行定向测试，确认页面尚未接入。
- [ ] ClassList 请求 `include_archived=true` 并保留状态标签。
- [ ] ClassDetail 用真实设置 tab 替换占位；实现表单快照、重复提交保护、成功刷新和错误提示。
- [ ] archived 时禁用成员、资料、练习任务和分析 tab；归档后保持设置 tab。
- [ ] 设置页使用现有 TDesign 控件和响应式布局，不新增依赖。
- [ ] 运行定向与 `npm test`，预期 fail 0。
- [ ] 运行 `npm run build-only`，预期 exit 0。
- [ ] 提交：`feat(exam): 完成班级设置与可逆归档页面`

## 任务 8：审查、全量验证与真实运行

**文件：** 检查本计划全部改动；不提交 `docker-compose.yml`。

- [ ] 使用 `requesting-code-review` 检查 tenant/owner 隔离、active-only 边界、状态转换、成员上限事务、归档通知跳转和移动端溢出。
- [ ] 所有 Critical/Important 问题先补红灯测试再修复。
- [ ] 运行：

```powershell
go test ./internal/types ./internal/application/repository ./internal/application/service ./internal/handler ./internal/router -count=1
go vet ./internal/types ./internal/application/repository ./internal/application/service ./internal/handler ./internal/router
Set-Location frontend
npm test
npm run build-only
Set-Location ..
```

- [ ] 重建并启动：

```powershell
docker compose build app frontend
docker compose up -d --no-deps app frontend
docker compose ps app frontend
docker compose logs app --tail 160
```

- [ ] 验证 migration 仍为 `83 / dirty: false`、app healthy、frontend 200、未认证新路由返回 401 而非 404。
- [ ] 在 1440px 与 355px 浏览器视口检查登录页运行和横向溢出；只有有效登录会话可用时才执行 owner/assistant/student 写操作验收。
- [ ] 最终运行 `git diff --check`、`git status --short`、`git log -12 --oneline`；工作区只保留用户的 `docker-compose.yml`，不 push。
