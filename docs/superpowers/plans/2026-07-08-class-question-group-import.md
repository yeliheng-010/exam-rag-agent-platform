# 班级题组导入发布实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 老师在班级发布练习时，可以选择自己可访问的题库中心题组；如果题组不属于当前班级空间，后端自动克隆到班级空间后再发布任务。

**架构：** 修正空间资源绑定的唯一键，让同一知识库可绑定多个班级；扩展班级作业服务，在创建作业时对跨空间题组做可读校验、题库准备和深度克隆；前端发布弹窗加载所有可访问题组而不是只加载当前班级空间题组。

**技术栈：** Go、Gin、GORM、PostgreSQL migration、Vue 3、TDesign、Docker Compose。

---

### 任务 1：修复资源绑定唯一键

**文件：**
- 修改：`internal/application/repository/exam_resource.go`
- 修改：`internal/types/interfaces/exam_resource.go`
- 修改：`internal/application/service/exam_resource.go`
- 测试：`internal/application/service/exam_resource_test.go`
- 创建：`migrations/versioned/000078_exam_space_resource_scope.down.sql`
- 创建：`migrations/versioned/000078_exam_space_resource_scope.up.sql`

- [x] **步骤 1：编写失败测试**

新增测试覆盖同一个 `knowledge_base` 绑定到两个不同 `space_id` 时应保留两条绑定，`ListResources(space_id=...)` 能分别查到对应空间。

- [x] **步骤 2：运行测试确认失败**

运行：

```powershell
docker run --rm -v "D:\rag\WeKnora:/app" -w /app golang:1.26 go test ./internal/application/service -run "ExamResource.*Multi|ExamResourceBind" -count=1
```

预期：新增测试失败，表现为第二次绑定覆盖第一次绑定。

- [x] **步骤 3：实现绑定按空间维度 upsert**

将 repository `Upsert` 的冲突列改为 `tenant_id, space_id, resource_type, resource_id`，并新增按空间读取绑定的方法，服务层绑定后按空间返回对应记录。

- [x] **步骤 4：添加数据库迁移**

迁移删除旧唯一约束 `exam_space_resources_tenant_id_resource_type_resource_id_key`，新增唯一索引 `idx_exam_space_resources_unique_space_resource`。

### 任务 2：实现发布练习自动导入题组

**文件：**
- 修改：`internal/application/service/exam_assignment.go`
- 修改：`internal/container/container.go`
- 修改：`internal/application/service/exam_assignment_test.go`

- [x] **步骤 1：编写失败测试**

新增测试：老师发布一个来自个人空间的题组时，服务会克隆到班级空间并创建作业；学生仍不能发布；不可读空间的题组仍被拒绝。

- [x] **步骤 2：运行测试确认失败**

运行：

```powershell
docker run --rm -v "D:\rag\WeKnora:/app" -w /app golang:1.26 go test ./internal/application/service -run "ExamAssignmentCreate.*Import|ExamAssignmentCreate" -count=1
```

预期：旧逻辑返回 `ErrExamPermissionDenied`。

- [x] **步骤 3：实现深度克隆**

给 `examAssignmentService` 注入 `ExamSpaceService`；当源题组不在班级空间时，校验用户可读源空间，创建班级空间题库，克隆题组、素材、题目、选项、答案、解析和 chunk 引用。

### 任务 3：前端发布弹窗加载可导入题组

**文件：**
- 修改：`frontend/src/views/classes/ClassDetail.vue`

- [x] **步骤 1：调整题组加载**

把 `loadAssignmentGroups` 从只传 `space_id` 改为加载所有可访问题组，并在下拉 label 中区分“班级题组”和“可导入题组”。

- [x] **步骤 2：补充空状态提示**

当没有任何题组时，提示需要先在题库中心完成结构化题组，而不是只显示空下拉。

### 任务 4：验证和数据补偿

**文件：**
- 不新增源码文件，使用 SQL/API 验证。

- [x] **步骤 1：运行后端定向测试**

运行：

```powershell
docker run --rm -v "D:\rag\WeKnora:/app" -w /app golang:1.26 go test ./internal/application/service ./internal/application/repository ./internal/router ./internal/handler -run "Exam|Assignment|Practice|Question|Resource" -count=1
```

- [x] **步骤 2：重建后端和前端**

运行：

```powershell
docker compose up -d --build app frontend
```

- [x] **步骤 3：验证现有账号**

使用 `yeliheng3@gmail.com` 登录，确认班级 `123` 的发布任务弹窗能看到个人空间的 5 篇阅读；发布后班级空间有克隆题组，任务可创建。
