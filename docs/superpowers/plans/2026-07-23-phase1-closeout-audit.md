# 二次开发 Phase 1 收尾审计与三角色验收实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 `superpowers:executing-plans` 串行执行。步骤使用复选框（`- [ ]`）跟踪进度。审计阶段不创建 worktree、不 commit、不 push；用户后续明确授权完成收尾后，按任务 6 形成职责清晰的本地提交，仍不 push。

**目标：** 对 27 份既有计划建立统一的真实完成度矩阵，并以管理员、教师、学生三种角色验证非评测业务主链路，给出 Phase 1 是否可以收尾的证据和剩余阻塞项。

**架构：** 原计划复选框仅作为输入，不直接代表实现状态。审计以四类证据交叉判定：生产代码/路由存在、自动化测试通过、当前容器 API 正常、浏览器角色页面可见；无法取得足够证据的项目标记为“待验收”，不推断完成。

**技术栈：** Go test、Node test、Vue/TypeScript/Vite、PostgreSQL 只读查询、Docker Compose、Playwright WebKit。

---

## 文件结构

- 创建：`docs/superpowers/reports/2026-07-23-phase1-closeout-audit.md`，统一计划矩阵、验证结果、剩余事项和收尾结论。
- 创建：`.artifacts/phase1-closeout-browser.py`，三角色浏览器验收脚本；只保存到忽略目录，不进入产品源码。
- 创建：`.artifacts/phase1-closeout-browser.json`，浏览器/API 结构化证据。
- 创建：`.artifacts/phase1-closeout-*.png`，管理员、教师、学生关键页面截图。
- 修改：`docs/superpowers/plans/2026-07-23-phase1-closeout-audit.md`，随执行勾选任务。

### 任务 1：建立 27 份计划的证据矩阵

- [x] 枚举 `docs/superpowers/plans/*.md`，按真实任务复选框统计完成度，排除说明文字中的示例 `- [ ]`。
- [x] 对未勾选的旧计划，核对对应生产文件、路由、迁移、测试文件和后续替代计划，不用“文件存在”单独判定完成。
- [x] 使用 `git log --oneline --all` 核对已提交阶段；使用 `git status --short` 单列尚未交付的工作区改动。
- [x] 在报告中把每份计划分类为：`已闭环`、`实现完成但文档滞后`、`基础完成/范围受限`、`待真实验收`。

验证：报告应包含 27 行计划记录，且每行至少有一项代码、测试、提交或运行时证据。

### 任务 2：选择只读验收身份和现有业务样本

- [x] 通过 PostgreSQL 只读查询核对管理员、Contributor/教师、Viewer/学生的当前身份，不在报告中记录密码、token 或 password hash。
- [x] 选择已有班级、题库、题组、作业和学生 attempt，优先使用名称或账号带有测试标识的数据。
- [x] 若缺少某角色或样本，将该链路标记为阻塞；不直接插入、更新或删除数据库记录。

验证：只读查询记录每个角色的用户 ID、租户角色和可用样本数量；公开报告只记录“身份可用/不可用”和样本类型，不记录凭据。

### 任务 3：运行当前工作树的静态与自动化回归

- [x] 运行后端业务核心包：

```powershell
go test ./internal/application/repository ./internal/application/service ./internal/handler ./internal/router ./internal/types -count=1
```

- [x] 在 `frontend` 运行：

```powershell
node --test
npm run type-check
npm run build-only
```

- [x] 在仓库根目录运行：

```powershell
git diff --check
docker compose ps
```

验证：所有命令退出码为 0；Node 测试失败数为 0；核心容器运行，要求声明了 healthcheck 的服务为 healthy。

### 任务 4：完成管理员、教师、学生浏览器验收

- [x] 创建 Playwright WebKit 脚本，通过真实登录表单分别登录三个角色；凭据只从运行时输入读取，不写入脚本、报告或截图文件名。
- [x] 管理员验证 `/platform/admin`：页面非空，教师申请与平台管理相关 API 无 4xx/5xx，控制台和页面错误为 0。
- [x] 教师验证 `/platform/classes/:classId`：班级详情、作业列表、分析/建议和设置入口可见；Contributor 题库/结构化入口可访问。
- [x] 学生验证 `/platform/learning`、已有作业入口和 `/platform/practice/review`：作业可定位、已有 attempt 可继续、错题复盘和 AI 讲解入口可见。
- [x] 每个角色至少保存一张截图并记录最终 URL、页面文本长度、捕获 API 状态、console/page error 和横向溢出。

验证：三个角色均完成登录；目标页面导航状态为 200；API failure、console error、page error 均为 0；桌面视口无横向溢出。任何权限跳转或缺少真实样本必须记录为缺口，不通过伪造数据绕过。

### 任务 5：形成收尾结论与下一步清单

- [x] 将自动化和浏览器证据写入 `docs/superpowers/reports/2026-07-23-phase1-closeout-audit.md`。
- [x] 区分“功能缺失”“仅文档滞后”“尚未浏览器验收”“明确延期”，给出是否进入 Phase 1 收尾的结论。
- [x] 为真实失败建立最小修复项；若没有生产缺陷，不新增功能或无关重构。
- [x] 重跑受影响验证并执行 `git diff --check`，回读报告和计划，确认没有泄露凭据。

验证：报告包含结论、27 份计划矩阵、三角色结果、自动化结果、工作区交付风险和按优先级排序的剩余清单；本计划无未勾选的真实任务。

### 任务 6：发布前工作区整理

- [x] 按 RAG 质量、前端 TypeScript 债务、知识库复制进度、Phase 1 审计与 i18n 修复拆分本地提交。
- [x] 每次提交前检查 staged diff、职责边界、空白错误和敏感信息，不混入无关文件。
- [x] 保留 `docker-compose.yml` 本机 DNS 与端口映射改动，不暂存、不提交。
- [x] 提交后重跑 Go 核心包、前端 Node 测试、类型检查、生产构建和 `git diff --check`。
- [x] 保留当前开发分支，不 push、不合并、不删除工作树。

验证：功能改动均进入可回溯本地提交；工作区只保留既有 `docker-compose.yml` 本机配置；最终验证退出码为 0。
