# 支付与权益基础接口设计

## 目标

为考试平台补上支付/订阅的第一层真实工程地基：系统管理员可维护套餐、查看和设置租户订阅、创建平台订单记录；普通登录用户可读取自己当前租户的权益状态，用于前端“支付与权益”页展示。

## 边界

本阶段不接入微信、支付宝、Stripe 等真实支付渠道，不保存第三方密钥，不处理支付回调签名。真实渠道接入将在后续独立阶段实现，并复用本阶段的订单、套餐和订阅表。

## 数据模型

- `billing_plans`：平台套餐。保存名称、周期、价格、币种、权益 JSON、是否启用。
- `tenant_subscriptions`：租户当前订阅。每个租户最多一条当前订阅，指向套餐并保存周期、状态和来源订单。
- `billing_orders`：平台订单记录。创建时快照套餐价格、币种和权益，避免套餐后续变更影响历史订单。

## 接口设计

- `GET /api/v1/billing/me`：Viewer+ 可读当前租户权益。无订阅时返回内置免费权益。
- `GET /api/v1/system/admin/billing/plans`：SystemAdmin 查看套餐。
- `POST /api/v1/system/admin/billing/plans`：SystemAdmin 创建套餐。
- `GET /api/v1/system/admin/billing/subscriptions`：SystemAdmin 查看租户订阅。
- `PUT /api/v1/system/admin/billing/subscriptions/:tenant_id`：SystemAdmin 设置租户订阅。
- `POST /api/v1/system/admin/billing/orders`：SystemAdmin 创建订单记录，默认 `manual` 渠道。

## 权限

支付配置影响全平台安全运行，所有写操作只允许 `User.IsSystemAdmin=true`。租户内 Owner/Admin/班主任不能维护平台套餐，也不能直接修改租户订阅。

## 验证

- Service 单元测试覆盖默认免费权益、套餐创建归一化、订单金额快照。
- Router 测试覆盖普通用户读取 `/billing/me`，非系统管理员不能访问 `/system/admin/billing/*`。
- 运行 Go 目标包测试和前端构建。
