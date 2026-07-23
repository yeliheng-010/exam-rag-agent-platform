# 前端 TypeScript 债务清理实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 `superpowers:executing-plans` 串行执行。当前工作区已有大量未提交改动，不创建 worktree，不 commit、不 push。

**目标：** 将 `frontend` 的 `npm run type-check` 从稳定复现的 101 条错误清理到 0，并保持现有运行时行为不变。

**架构：** 先修复两个公共类型根因：Axios 响应拦截器与请求包装器签名不一致、Vue 编译宏被显式导入。随后按类型边界、计时器/可空值、组件模型和第三方库配置四组处理剩余局部错误；每批后完整重跑 `type-check`，只依据新的错误集继续。

**技术栈：** Vue 3.5、TypeScript 6、vue-tsc 3、Axios 1.16、Node test、Vite。

---

## 基线

```text
npm run type-check
exit code: 2
errors: 101
files: 42
```

错误码分布：`TS2558=33`、`TS2322=19`、`TS7006=17`、`TS2440=12`、`TS2345=10`，其余 10 条。

### 任务 1：校正请求包装器的响应类型契约

**文件：**
- 修改：`frontend/src/utils/request.ts`
- 编译契约：`frontend/src/api/agent/index.ts`
- 编译契约：`frontend/src/api/embed/index.ts`
- 编译契约：`frontend/src/api/skill/index.ts`
- 编译契约：`frontend/src/api/user-favorites.ts`
- 编译契约：`frontend/src/api/chunker/index.ts`
- 编译契约：`frontend/src/api/vector-store.ts`

- [x] 保留当前 101 条红灯输出，确认 `TS2558` 的 33 条均来自无泛型的 `get/post/put/del` 包装器。
- [x] 将 `get/post/put/patch/del` 改为泛型响应包装器，并使用 Axios 的第二泛型参数表达响应拦截器返回的业务数据：

```ts
export function get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> {
  return instance.get<unknown, T>(url, config)
}
```

- [x] 将 `getDown` 的返回值声明为 `Promise<Blob>`，与 `responseType: 'blob'` 和响应拦截器行为一致。
- [x] 运行 `npm run type-check`，记录剩余错误数；33 条 `TS2558` 全部消失，API 返回值不再被误判为 `AxiosResponse`。

### 任务 2：移除 Vue 编译宏的显式导入

**文件：**
- 修改：`frontend/src/views/chat/components/ToolResultRenderer.vue`
- 修改：`frontend/src/views/chat/components/tool-results/ChunkDetail.vue`
- 修改：`frontend/src/views/chat/components/tool-results/DocumentInfo.vue`
- 修改：`frontend/src/views/chat/components/tool-results/GraphQueryResults.vue`
- 修改：`frontend/src/views/chat/components/tool-results/KnowledgeBaseList.vue`
- 修改：`frontend/src/views/chat/components/tool-results/RelatedChunks.vue`
- 修改：`frontend/src/views/chat/components/tool-results/ThinkingDisplay.vue`
- 修改：`frontend/src/views/knowledge/components/KbUploadSourceDropdown.vue`
- 修改：`frontend/src/views/knowledge/components/UploadConfirmDialog.vue`
- 修改：`frontend/src/views/knowledge/settings/GraphSettings.vue`
- 修改：`frontend/src/views/knowledge/settings/KBAdvancedSettings.vue`
- 修改：`frontend/src/views/knowledge/settings/KBChunkingSettings.vue`

- [x] 对照 Vue `<script setup>` 的正常文件，确认 `defineProps`、`withDefaults` 由编译器提供且无需从 `vue` 导入。
- [x] 只从现有 import 中删除发生冲突的宏名称，不改 props 声明和运行逻辑。
- [x] 运行 `npm run type-check`，确认 12 条 `TS2440` 全部消失。

### 任务 3：修复回调、计时器与可空值边界

**文件：**
- 修改：`frontend/src/components/AgentEmbedChannelPanel.vue`
- 修改：`frontend/src/components/IMChannelPanel.vue`
- 修改：`frontend/src/composables/useEmbedChatSession.ts`
- 修改：`frontend/src/views/knowledge/components/DocumentListView.vue`
- 修改：`frontend/src/views/knowledge/components/FAQEntryManager.vue`
- 修改：`frontend/src/views/knowledge/components/KbTagManageDrawer.vue`
- 修改：`frontend/src/views/knowledge/KnowledgeBase.vue`
- 修改：`frontend/src/views/chat/components/McpOAuthCard.vue`

- [x] 从组件 props、TDesign 回调签名和集合元素类型反向追踪 17 个隐式 `any`，使用已有领域类型或 `unknown` 收窄，禁止新增裸 `any`。
- [x] 将浏览器计时器变量统一为 `ReturnType<typeof window.setTimeout>` 或直接使用 `number`，与实际调用来源保持一致。
- [x] 对可选标签数组使用局部空数组回退，避免模板读取 `undefined`，不改变空标签展示。
- [x] 运行 `npm run type-check`，确认本组错误清零并记录新的剩余集合。

### 任务 4：修复组件与 API 数据模型边界

**文件：**
- 修改：`frontend/src/components/AgentSelector.vue`
- 修改：`frontend/src/components/menu.vue`
- 修改：`frontend/src/components/ModelEditorDialog.vue`
- 修改：`frontend/src/components/document-preview.vue`
- 修改：`frontend/src/components/manual-knowledge-editor.vue`
- 修改：`frontend/src/composables/useChatStreamHandler.ts`
- 修改：`frontend/src/views/agent/AgentEditorModal.vue`
- 修改：`frontend/src/views/chat/components/AgentStreamDisplay.vue`
- 修改：`frontend/src/views/embed/EmbedChatCore.vue`
- 修改：`frontend/src/views/knowledge/settings/DataSourceEditorDialog.vue`
- 修改：`frontend/src/views/settings/ApiInfo.vue`
- 修改：`frontend/src/stores/editorResources.ts`
- 修改：`frontend/src/composables/useEmbedBridge.ts`

- [x] 对每条错误读取生产者类型和消费者类型，优先修正源类型或建立显式映射，不使用双重断言绕过检查。
- [x] 为 `CustomAgent`、会话消息、模型 ID、提示词占位符、Embed 附件和租户 ID 建立最小且真实的边界收窄。
- [x] 运行 `npm run type-check`，确认模型边界错误清零。

### 任务 5：处理第三方库配置与最后残余

**文件：**
- 修改：`frontend/src/utils/security.ts`
- 修改：`frontend/src/stores/settings.ts`
- 修改：`frontend/src/views/settings/StorageEngineSettings.vue`

- [x] 依据 DOMPurify 当前 `Config` 定义修正只读数组/Hook 参数，不扩大允许标签或 URI 范围。
- [x] 修正 `selectedTools` 的实际设置模型来源，以及存储检查响应字段的真实层级。
- [x] 若前四批仍有残余，先把精确文件和根因补入对应任务，再实施修复。
- [x] 运行 `npm run type-check`，退出码 0、错误 0。

### 任务 6：完成门禁与运行时回归

- [x] 运行 `node --test`，记录全部测试数量与结果。
- [x] 运行 `npm run build-only`，确认生产构建退出码 0。
- [x] 运行 `git diff --check` 并审查所有本轮前端改动，不接受 `@ts-ignore`、`@ts-expect-error` 或新增裸 `any`。
- [x] 重建 `frontend` 容器，在桌面和移动端打开现有考试 RAG 页面，确认 API 200、控制台 error 0、页面无横向溢出。
- [x] 将最终错误基线、修复批次、测试和浏览器证据写入 `docs/superpowers/reports/2026-07-23-frontend-type-debt.md`。
