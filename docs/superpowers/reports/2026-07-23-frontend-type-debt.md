# 前端 TypeScript 债务清理验收报告

## 结论

`frontend` 的全局 TypeScript 错误已从 42 个文件中的 101 条降为 0。请求包装器继续以 `unknown` 作为默认响应类型，没有回退到默认 `any`，现有 Node 测试、生产构建、容器运行和考试 RAG 页面均通过回归验收。

本轮未修改数据库或正式题库/RAG 数据，未 commit、未 push。

## 错误收敛

完整 `npm run type-check` 每批重新执行，错误数按以下序列收敛：

```text
101 -> 62 -> 50 -> 26 -> 3 -> 0
```

| 批次 | 主要处理内容 | 剩余错误 |
|---|---|---:|
| 基线 | 42 个文件，`TS2558=33`、`TS2322=19`、`TS7006=17`、`TS2440=12`、`TS2345=10` | 101 |
| 请求契约 | Axios 请求配置、业务响应泛型、上传进度和请求 body 类型 | 62 |
| Vue 宏 | 移除 `<script setup>` 编译宏的显式导入 | 50 |
| 边界收窄 | 回调、计时器、可空值、DOM ref、流式处理和组件 props | 26 |
| 数据模型 | API 响应、数据源联合类型、共享智能体、附件、存储检查和租户 ID | 3 |
| 最后残余 | 第三方配置和剩余组件边界 | 0 |

## 关键修复

- `get/post/put/patch/del` 以 Axios 第二泛型参数表达响应拦截器返回的业务数据，默认响应保持 `unknown`；`getDown` 返回 `Promise<Blob>`。
- 上传进度改用 `AxiosProgressEvent`，请求配置改用 `AxiosRequestConfig`，请求体边界使用 `unknown`。
- 数据源接口按真实响应建模为联合类型，并通过 `unwrapMaybeWrapped` 统一处理直接响应和包装响应。
- 修复 Vue 编译宏冲突、流式回调、计时器、可空值、DOM 引用、共享智能体和 Embed 附件等边界。
- `DataSourceEditorDialog.testConnection()` 使用 `finally` 保证 loading 在成功和失败路径均复位。
- 租户 ID 按后端 `ParseUint` 契约转换；DOMPurify 配置按当前只读类型收窄，未扩大允许范围。
- 删除 `index.html` 和 `embed.html` viewport 中已失效的 `minimal-ui`，消除浏览器控制台错误。

## 静态验证

2026-07-23 最终重新执行：

```text
npm run type-check
exit 0

node --test
tests 255, pass 255, fail 0

npm run build-only
exit 0, 6366 modules transformed, built in 43.75s

git diff --check
exit 0
```

构建仍有项目既有的动态/静态混合导入和大包体警告，不影响退出状态。

类型逃逸审计：

```text
新增 @ts-ignore / @ts-expect-error: 0
全量 tracked frontend/src @ts-ignore / @ts-expect-error: 0
新增显式 any: 0
新增默认 any 泛型: 0
```

## 容器与浏览器

最新前端源码已通过 `docker compose up -d --no-deps --build frontend` 重建。最终 `docker compose ps` 显示：

- `frontend` 运行中，端口 `80`。
- `app`、`postgres`、`docreader` 为 healthy。
- `redis` 运行中。

真实登录后打开现有考试 RAG 观测页，分别使用 `1440x1000` 和 `390x844` 验收：

```text
navigation status: 200
final route: /platform/question-banks/8e5ffbbd-f0ab-4e96-ac98-435307f06dae/rag-observability
captured API responses: 18/18 HTTP 200 (desktop), 18/18 HTTP 200 (mobile)
API failures: 0
console errors: 0
page errors: 0
horizontal overflow: false
guide visible: false
```

截图：

- `.artifacts/frontend-type-debt-desktop.png`
- `.artifacts/frontend-type-debt-mobile.png`

## 交付边界

- 本轮清理的是前端全局 TypeScript 债务，不改变 RAG 评测算法、题库数据或数据库结构。
- 工作区原有大量未提交业务/RAG 改动均保留，未做清理、回退或提交。
