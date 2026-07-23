# 支付与权益基础接口实现计划

> **审计状态（2026-07-23）：** 权益基础层实现与验证已完成；下方未勾选测试项是历史环境阻塞记录，后续收尾回归已覆盖。第三方支付网关明确延期，不属于 Phase 1。完成证据见 [Phase 1 收尾审计报告](../reports/2026-07-23-phase1-closeout-audit.md)。

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 添加平台级支付/订阅基础接口，让系统管理员可维护套餐、订阅和订单，普通用户可读取当前租户权益状态。

**架构：** 新增 `BillingService`、`BillingRepository`、`BillingHandler`，路由分为 Viewer+ 的 `/billing/me` 和 SystemAdmin 的 `/system/admin/billing/*`。真实支付渠道后续接入，本阶段只做 provider-neutral 的订单与订阅地基。

**技术栈：** Go、Gin、GORM、PostgreSQL migration、Vue 3、TDesign。

---

### 任务 1：后端类型、接口与测试

**文件：**
- 创建：`internal/types/billing.go`
- 创建：`internal/types/interfaces/billing.go`
- 创建：`internal/application/service/billing_test.go`

- [x] **步骤 1：编写失败的 service 测试**

覆盖 `GetTenantBilling` 默认免费权益、`CreatePlan` 套餐归一化、`CreateOrder` 订单金额快照。

- [x] **步骤 2：运行测试验证失败**

运行：`go test ./internal/application/service -run Billing -count=1`
预期：编译失败或测试失败，因为实现尚未存在。

### 任务 2：Repository、Service、Handler 与路由

**文件：**
- 创建：`internal/application/repository/billing.go`
- 创建：`internal/application/service/billing.go`
- 创建：`internal/handler/billing.go`
- 创建：`internal/router/billing.go`
- 修改：`internal/container/container.go`
- 修改：`internal/router/router.go`

- [x] **步骤 1：实现最小后端代码**

实现套餐、订阅、订单的 CRUD 最小路径和 `/billing/me` 查询。

- [ ] **步骤 2：运行测试验证通过**

运行：`go test ./internal/application/service -run Billing -count=1`
预期：PASS。

> 当前本机验证阻塞：`pg_query_go/v6` 的 `Parse/Deparse` 依赖 cgo，便携 Go 默认 `CGO_ENABLED=0` 会编译失败；开启 `CGO_ENABLED=1` 后本机缺少 `gcc`，报 `runtime/cgo: C compiler "gcc" not found`。需要安装或配置 Windows C 工具链后重跑。

### 任务 3：数据库迁移

**文件：**
- 创建：`migrations/versioned/000071_billing_foundation.up.sql`
- 创建：`migrations/versioned/000071_billing_foundation.down.sql`

- [x] **步骤 1：创建表和默认免费套餐种子**

建表 `billing_plans`、`tenant_subscriptions`、`billing_orders`，插入 `free` 套餐。

- [x] **步骤 2：静态检查迁移编号**

运行：`Get-ChildItem migrations/versioned | Sort-Object Name | Select-Object -Last 4`
预期：`000071` 位于最新。

### 任务 4：路由权限测试

**文件：**
- 修改：`internal/router/exam_rbac_routes_test.go`

- [x] **步骤 1：补路由权限测试**

覆盖 `/billing/me` 使用 Viewer，`/system/admin/billing` 使用 SystemAdmin。

- [ ] **步骤 2：运行 router 测试**

运行：`go test ./internal/router -run Billing -count=1`
预期：PASS。

> 当前本机验证阻塞同上：目标包编译会先经过依赖 cgo 的 `internal/utils/inject.go`。

### 任务 5：前端页面接入

**文件：**
- 创建：`frontend/src/api/billing.ts`
- 修改：`frontend/src/views/billing/BillingHome.vue`
- 创建：`frontend/src/views/admin/BillingAdmin.vue`
- 修改：`frontend/src/views/admin/AdminHome.vue`

- [x] **步骤 1：接入普通权益页**

让 `/platform/billing` 从 `/billing/me` 读取真实套餐和权益状态。

- [x] **步骤 2：接入管理端页**

在管理端新增“支付与权益”tab，仅 SystemAdmin 显示，支持查看套餐、创建套餐、查看租户订阅和订单。

- [x] **步骤 3：运行前端构建**

运行：`npm --prefix frontend run build`
预期：构建成功。

### 任务 6：最终验证与提交

- [ ] **步骤 1：运行后端目标测试**

运行：
- `go test ./internal/application/service -run Billing -count=1`
- `go test ./internal/router -run Billing -count=1`
- `go test ./internal/handler -run Billing -count=1`

> 当前本机验证阻塞：使用便携 Go 1.26.0 时，`CGO_ENABLED=0` 下 `pg_query_go/v6` 的 `Parse/Deparse` 不参与编译；临时加入 w64devkit GCC 16.1.0 并开启 `CGO_ENABLED=1` 后，cgo 报 `cannot parse gcc output ... as ELF, Mach-O, PE, XCOFF object`，涉及 `go-sqlite3`、`gojieba`、`pg_query_go/v6/parser` 等既有 cgo 依赖。需要在机器上配置兼容的 Windows C 工具链后重跑。

- [x] **步骤 2：运行格式和 diff 检查**

运行：`gofmt`、`git diff --check`

- [ ] **步骤 3：提交并推送**

只 stage 本次相关文件，排除本地 `docker-compose.yml`。
