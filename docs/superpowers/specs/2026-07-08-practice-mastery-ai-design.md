# 错题掌握状态与 AI 讲解设计

## 背景

学生端已经支持正式题组练习、练习记录和错题本。下一步需要让错题本不只是列表，而是能沉淀学生自己的复盘状态，并能快速带着题目上下文进入 AI 对话讲解。

## 目标

- 学生可以给自己的错题标记 `未复盘`、`复习中`、`已掌握`。
- 学生可以为每道错题记录简短复盘备注。
- 错题卡可以一键把题干、选项、我的答案、正确答案和已有解析带入 AI 对话。
- 权限继续收敛在当前租户、当前学生本人 attempt、可读 space 范围内。

## 非目标

- 本阶段不做教师端班级错题统计。
- 本阶段不做掌握度算法、复习间隔算法和错题收藏夹。
- 本阶段不新增独立错题表。
- 本阶段不新建专用 LLM 接口，复用现有聊天入口。

## 数据设计

在 `exam_practice_answers` 上新增复盘字段：

- `review_status`: `unreviewed`、`reviewing`、`mastered`
- `review_note`: 学生复盘备注
- `reviewed_at`: 最近一次复盘更新时间

这样每次作答快照、正确性和复盘状态保持在同一条 answer 上。后续如果需要“跨多次 attempt 合并同一道错题”，可以基于 `question_id + user_id` 再物化统计表。

## 后端接口

新增 viewer 接口：

`PATCH /api/v1/exam/practice/answers/:answer_id/review`

请求体：

```json
{
  "review_status": "reviewing",
  "review_note": "错因：没有定位到第二段转折句"
}
```

服务层先读取 answer，再通过 answer 的 `attempt_id` 校验当前用户是否拥有该 attempt，并再次校验 space 可读权限。非法状态、空 answer id、超长备注返回现有考试模块错误。

## 前端设计

复盘页错题卡新增：

- 掌握状态按钮组
- 复盘备注输入框，失焦保存
- `问 AI 讲解` 按钮，复用 `useStartChat()` 进入新对话

AI prompt 包含题组、题号、题干、选项、我的答案、正确答案和已有解析，并要求按错因分析、解题思路、关键知识点、下次如何避免四部分回答。

## 验收标准

- 学生只能更新自己的 answer 复盘状态。
- 非法复盘状态会被拒绝。
- 错题列表返回 `review_status`、`review_note`、`reviewed_at`。
- 复盘页可以更新状态和备注。
- `问 AI 讲解` 能把错题上下文预填到新对话。
