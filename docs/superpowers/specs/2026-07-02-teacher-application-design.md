# 班主任申请审核设计

## 目标

考试平台采用“学校”模型：平台/租户管理员负责校级管理，班主任负责班级内教学运营，学生只访问自己加入且已审核通过的班级。

本阶段实现一条完整身份链：普通用户提交班主任申请，管理员审核，通过后用户获得班主任业务身份，并可以创建班级。

## 角色边界

- 管理员：租户 `admin/owner`，负责审核班主任申请、管理成员与校级配置。支付、全局系统配置和平台安全仍由 `is_system_admin` 保护。
- 班主任：考试业务角色，不等同于租户管理员。获批后可创建班级，并在自己创建或被任命的班级里审核学生、管理班级资源。
- 学生：默认注册用户，租户角色为 `viewer`。可以申请加入班级，通过审核后访问班级学习内容。

## 数据模型

新增 `exam_teacher_applications` 表：

- `tenant_id`：学校/租户隔离边界。
- `user_id`：申请人。
- `status`：`pending`、`approved`、`rejected`。
- `reason`：申请理由。
- `reviewer_id`、`review_note`、`reviewed_at`：审核记录。

同一租户同一用户只保留一条申请记录。被拒绝后允许再次提交，重新进入 `pending`；已经通过的申请再次提交时保持 `approved`，避免班主任资格被用户自己的重复申请回退。

## 接口

- `GET /api/v1/exam/teacher-applications/me`：查看我的班主任申请/身份状态。
- `POST /api/v1/exam/teacher-applications`：提交或重新提交申请。
- `GET /api/v1/exam/admin/teacher-applications`：管理员查看申请队列。
- `POST /api/v1/exam/admin/teacher-applications/:application_id/approve`：管理员通过申请。
- `POST /api/v1/exam/admin/teacher-applications/:application_id/reject`：管理员拒绝申请。

## 权限规则

- 创建班级必须满足：当前租户内已经有 `approved` 班主任申请。
- 审核班主任申请必须满足：当前租户角色为 `admin` 或 `owner`。
- 通过申请时，将申请人在当前租户的基础角色至少提升到 `contributor`，用于复用 WeKnora 已有知识库、智能体、题库创建权限。
- 不提升到 `admin/owner`，避免班主任获得支付、租户安全配置、成员管理等校级权限。

## 前端体验

- 班级中心对普通用户展示“申请成为班主任”入口和当前审核状态。
- 只有获批班主任展示“创建班级”按钮。
- 管理端新增“班主任申请”页签，展示申请人、理由、状态和审核操作。

## 验证

- 后端服务测试覆盖：未获批不能创建班级、提交申请、管理员通过后可以创建班级、非管理员不能审核。
- 前端构建通过。
- 本地服务重启后用 API 做一次申请、审核、创建班级的 smoke test。
