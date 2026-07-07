# 学生练习记录与错题本设计

## 背景

学生端已经可以围绕正式题组进行一次练习，并把每道题的作答写入
`exam_practice_attempts` 和 `exam_practice_answers`。下一步需要让这些
沉淀数据真正可复习：学生能回看自己的练习记录，也能快速进入错题复盘。

本阶段不引入作业发布、教师批改、班级成绩统计和错题掌握状态。目标是先
把学生个人学习闭环补齐。

## 目标

- 学生能在学习中心看到最近练习记录与错题复盘入口。
- 学生能进入复盘页，查看自己的练习记录列表和错题列表。
- 学生只能看到本人 attempt 和本人 answer。
- 错题本从 `exam_practice_answers.is_correct=false` 派生，不新建表。
- 错题项展示题干快照、我的答案、正确答案、解析快照和所属题组。

## 非目标

- 不做错题掌握状态、收藏、删除和人工标记。
- 不做教师班级统计、排行榜或作业成绩。
- 不做跨学生、跨班级的管理视图。
- 不新增数据库迁移。

## 后端设计

新增三个 viewer 接口：

- `GET /api/v1/exam/practice/attempts`
- `GET /api/v1/exam/practice/attempts/:attempt_id`
- `GET /api/v1/exam/practice/wrong-questions`

服务层继续使用 `ExamPracticeService`。列表接口先解析当前用户可读空间：

1. 如果指定 `space_id`，校验 `CanReadSpace`。
2. 未指定时，使用 `ListSpaces` 获取可读空间。
3. repository 只查询当前 `tenant_id`、`user_id`、可读 `space_id` 范围内
   的 attempt。

错题本先从最近 attempt 派生：读取当前用户最近 attempt，再读取每次
attempt 的 answer，过滤 `is_correct=false`，按 `answered_at` 倒序返回。
这避免新增表，也保留后续升级为物化错题表的空间。

## 前端设计

新增 `PracticeReviewHome.vue`，路由为：

`/platform/practice/review`

页面使用 TDesign 的工作台风格，包含两个 tab：

- 练习记录：展示题组、状态、作答数、正确数、开始时间。
- 错题本：展示题号、题干快照、我的答案、正确答案和解析摘要。

学习中心增加两个入口卡片：最近练习和错题复盘。学生仍不需要进入题库管理
中心。

## 验收标准

- 学生可以从学习中心进入复盘页。
- 复盘页能加载当前学生的练习记录和错题。
- 错题项来自当前学生自己的错误 answer。
- practice 记录与错题接口都使用 viewer guard。
- 后端 service 测试覆盖 attempt 归属与错题过滤。
- 前端 source 测试覆盖 API、路由和页面 wiring。
