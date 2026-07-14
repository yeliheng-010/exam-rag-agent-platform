# 智能教学干预与自适应练习 Agent 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）跟踪。

**目标：** 用共享推荐服务把班级/学生诊断转化为正式题组推荐，并让教师人工确认后复用现有班级任务发布流程。

**架构：** `ExamInterventionService` 聚合诊断证据和可读题组，进行确定性排序；HTTP 和 `exam_practice_recommendation` Agent 工具共用该服务。推荐过程只读，不新增数据库表或自动发布能力。

**技术栈：** Go、Gin、现有 Exam Analytics/Practice/Assignment 模块、Vue 3、TypeScript、TDesign、Go testing、Node test。

---

## 文件结构

- 创建：`internal/types/exam_intervention.go`，推荐请求、证据、理由和结果类型。
- 创建：`internal/types/interfaces/exam_intervention.go`，领域服务接口。
- 创建：`internal/application/service/exam_intervention.go`，权限和候选编排。
- 创建：`internal/application/service/exam_intervention_score.go`，确定性评分与排序。
- 创建：`internal/application/service/exam_intervention_test.go`，班级/学生推荐测试。
- 修改：`internal/types/exam_analytics.go` 与 analytics 聚合，保留错题来源题组。
- 创建：`internal/handler/exam_intervention.go`，班级和学生推荐 HTTP handler。
- 修改：`internal/router/exam.go`、`internal/container/container.go`，注册服务与路由。
- 创建：`internal/agent/tools/exam_practice_recommendation.go`、`exam_practice_recommendation_format.go` 及测试。
- 修改：Agent 定义、capability、service、preset 和 builtin 配置。
- 创建：`frontend/src/api/exam/intervention.ts`。
- 修改：`frontend/src/types/exam.ts`、`ClassDetail.vue` 和多语言工具文案。
- 创建：`frontend/src/views/classes/classInterventionSource.test.ts`。

### 任务 1：保留班级错题来源并定义推荐契约

- [x] 编写失败测试，验证高频错题包含来源 `group_id`。
- [x] 运行 analytics 定向测试确认失败。
- [x] 将 latest attempt 上下文从 user ID 扩展为 user/group/assignment，并写入错题聚合。
- [x] 定义推荐请求、证据、推荐项和结果类型。
- [x] 运行 analytics 测试确认通过。

### 任务 2：实现班级与学生推荐服务

- [x] 编写失败测试：班级模式排除已发布题组并优先同学科同题型候选。
- [x] 编写失败测试：学生模式忽略 mastered 错题并优先原题复习。
- [x] 编写失败测试：无诊断信号时返回 supplemental 候选和 warning。
- [x] 运行 `go test ./internal/application/service -run 'TestExamIntervention' -count=1` 确认失败。
- [x] 实现候选加载、诊断信号归一化、评分、稳定排序和 limit。
- [x] 修复班级 scope 误用教师个人最近作答的评分污染。
- [x] 使用候选 group IDs 精确反查全部历史发布记录，不受最近 100 条限制。
- [x] 运行定向测试确认通过。

### 任务 3：接入 HTTP 和 Agent 工具

- [x] 编写 Agent 工具失败测试，覆盖 class/student scope、结果格式、默认注册和错误路径。
- [x] 实现 `exam_practice_recommendation`，只从 session context 读取 tenant/user。
- [x] 增加班级 `GET /exam/classes/:class_id/practice-recommendations` 和学生 `GET /exam/practice/recommendations`。
- [x] 注册 container、Agent service、tool definitions、capability 和 presets。
- [x] 运行 Agent、router、container 定向测试。

### 任务 4：实现教师人工发布交互

- [x] 编写前端 source test，要求推荐 API、推荐按钮、推荐对话框和发布预填函数存在。
- [x] 运行前端定向测试确认失败。
- [x] 增加 intervention API 和 TypeScript 类型。
- [x] 在班级分析页增加推荐列表、诊断提示和“用于发布”。
- [x] 增加“包含已发布题组”开关，显式支持原题复习。
- [x] 复用现有发布表单，不直接从推荐对话框写入 assignment。
- [x] 运行前端定向测试与全量测试。

### 任务 5：审查与真实验证

- [x] 运行 gofmt、Go 相关包测试和 go vet。
- [x] 运行前端 `npm test`、`npm run build`；记录既有 type-check 阻塞。
- [x] 独立审查权限、数据范围、评分稳定性和前端人工确认边界。
- [x] 重建 app/frontend Docker 镜像并确认健康；五个核心容器运行正常，app 为 healthy。
- [x] 使用真实教师/学生账号验证：教师班级推荐 200、非班级学生访问班级推荐 403、学生个人推荐 200。
- [x] 从推荐项预填并发布一项练习，确认跨空间题组复制和现有 assignment 流程可用。
- [x] 验证跨空间已发布识别：默认排除已发布题组，显式包含时优先班级副本并按结构化内容身份去重。
- [x] 在 390x844 与 1440x900 下完成浏览器验收：移动端使用平铺列表、桌面端使用表格、无页面横向滚动，发布表单预填与窄屏布局正常。
- [x] 处理独立审查问题：推荐加载只接受最后一次请求并在重载时清空旧结果；旧成员缺少展示编号时保留唯一用户标识作为兼容回退。
- [x] 复查包含五个题组的 390x844 弹窗，外框完整位于视口内，题组列表使用内部滚动。
