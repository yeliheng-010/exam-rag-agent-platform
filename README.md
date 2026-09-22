# Exam RAG Agent Platform（教学考试 RAG 平台）

> 基于腾讯开源 [WeKnora](https://github.com/Tencent/WeKnora) 的教学考试场景深度二次开发：
> 面向高考试卷在 **DOCX / 文本层 PDF / 扫描件** 三种形态下的解析差异，以及跨切块召回、AI 讲解上下文不完整等问题，
> 构建 **试卷入库 → 检索 → 讲解 → 练习 → 离线评测** 的完整闭环。

个人项目（2026.06 - 2026.08）｜ 作者：[@yeliheng-010](https://github.com/yeliheng-010) ｜ 博客：[blog.miansu.eu.cc](https://blog.miansu.eu.cc)

---

## ✨ 相对上游 WeKnora 的新增能力

| 模块 | 能力 | 代码位置 |
|------|------|----------|
| 📄 文档解析 | DocReader 质量感知路由：文本层优先、真扫描件降级 PaddleOCR-VL（本地/云双实现），统一处理三类试卷并保留解析质量告警 | `internal/infrastructure/docparser/paddleocr_vl_*.go` |
| 🧩 切块与检索 | 围绕试卷结构的 `text` / `parent_text` / `summary` 三类切块；Dense/BM25 双路召回 + RRF 融合排序；引用溯源与缺失片段诊断 | `internal/types/chunk.go`、`internal/application/service/` |
| 🤖 Agent 编排 | 扩展 Deep Agent 四阶段 ReAct 链路（Think-Analyze-Act-Finalize）：AgentState 记录轮次/检索证据/工具结果并流式输出；上下文达窗口 50% 触发摘要压缩，失败重试 3 次后降级归档 | `internal/agent/` |
| 📝 考试工具 | 新增 4 个 Agent 工具：题目上下文 `exam_question_context`、学习诊断 `exam_learning_diagnosis`、班级诊断 `exam_class_diagnosis`、练习推荐 `exam_practice_recommendation` | `internal/agent/tools/exam_*.go`（含单测） |
| 📊 评测中心 | RAG 离线评测：答案覆盖率、Recall@K、MRR、结构化解析率；支持基线对比与回归原因归类；前端可视化 | `internal/application/service/exam_evaluation_metrics.go`、`frontend/src/views/evaluation/` |
| 🏫 班级视图 | 班级作答记录、诊断报告与个性化练习推荐页面 | `frontend/src/views/classes/` |

## 📈 关键评测结果

基于多套语数外真题构建的离线评测集（覆盖 DOCX、文本层 PDF、扫描件三种形态）：

| 指标 | 上游基线 | 本项目 | 说明 |
|------|---------|--------|------|
| 语文 auto 切块 Recall@5 | 76.7% | **90.0%** | 三类切块 + 混合召回 |
| 语文 auto 切块 Recall@20 | — | **100%** | |
| 文本层 PDF 金标覆盖率 | — | **96%** | |
| 扫描数学卷规范化 Recall@20 | — | **80%** | 经 PaddleOCR-VL 解析 |
| DocReader 定向回归 | — | **46 passed / 3 skipped** | |

## 🏗️ 架构概览

```
试卷 (DOCX / PDF / 扫描件)
        │
        ▼
┌─────────────────────┐
│  DocReader 质量感知路由 │  文本层优先，真扫描件 → PaddleOCR-VL
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│  三类结构化切块        │  text / parent_text / summary
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│  Dense/BM25 + RRF    │  混合召回、融合排序、引用溯源
└─────────┬───────────┘
          ▼
┌─────────────────────┐      ┌──────────────────┐
│  Deep Agent 四阶段    │ ───▶ │ 4 个考试工具       │
│  Think-Analyze-       │      │ 题目上下文/学习诊断/  │
│  Act-Finalize        │      │ 班级诊断/练习推荐    │
└─────────┬───────────┘      └──────────────────┘
          ▼
┌─────────────────────┐
│  离线评测中心          │  Recall@K / MRR / 答案覆盖率 / 回归对比
└─────────────────────┘
```

## 📚 设计文档与过程记录

本项目的完整设计、实验与复盘记录（中文）：

- 设计稿：[`docs/superpowers/specs/`](docs/superpowers/specs/)（评测体系设计、切块质量方案等）
- 实验报告：[`docs/superpowers/reports/`](docs/superpowers/reports/)（切块 A/B 实验、全局 topk、质量差距分析）
- 实施计划：[`docs/superpowers/plans/`](docs/superpowers/plans/)

## 🚀 快速开始

与上游 WeKnora 一致（Docker Compose）：

```bash
cp .env.example .env   # 按需修改模型配置
docker compose up -d
```

详见 [`docs/QA.md`](docs/QA.md) 与上游文档。

## 🛠️ 技术栈

**后端**：Go（Gin / GORM）、PostgreSQL（pgvector）、Redis、PaddleOCR-VL
**前端**：Vue 3、TDesign
**Agent**：Deep Agent（langchain-ai/deepagents）、ReAct、Function Calling、MCP
**检索**：Dense Embedding + BM25 + RRF 融合
**评测**：离线评测集 + 定向回归测试

## 🙏 致谢与许可

本项目基于 [Tencent/WeKnora](https://github.com/Tencent/WeKnora) 二次开发，遵循上游许可证（见 [LICENSE](LICENSE)）。
感谢 WeKnora 团队的开源工作。本人亦向上游提交过修复：[PR #3548](https://github.com/Tencent/WeKnora/pull/3548)（分块列表多模态切片可见性）。
