# 考试 RAG 与 Agent 底层增强实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框语法来跟踪进度。

**目标：** 建立结构化题组到 RAG/Agent 上下文的第一层底座。

**架构：** 在 `internal/searchutil` 新增结构化题组上下文包构建器，输入 `QuestionGroupDetail`，输出可被检索结果和 Agent 工具复用的紧凑文本上下文。后续 Chat pipeline 和 Agent `knowledge_search` 都优先消费该上下文包，原有 chunk 启发式作为兜底。

**技术栈：** Go、WeKnora `types.QuestionGroupDetail`、`searchutil`、单元测试。

---

## 文件结构

- 创建：`internal/searchutil/exam_structured_context.go`
  - 负责把 `QuestionGroupDetail` 序列化为紧凑、可引用的考试 RAG 上下文包。
- 创建：`internal/searchutil/exam_structured_context_test.go`
  - 覆盖阅读原文、题目、选项、答案、解析、chunk 引用和长度限制。
- 修改：`internal/searchutil/exam_context.go`
  - 只复用既有 `ExamQuestionContextBundle` 和裁剪工具，不改原有 chunk 启发式行为。

## 任务 1：结构化上下文构建器

- [x] **步骤 1：编写失败测试**

创建 `internal/searchutil/exam_structured_context_test.go`，测试输入一个 `QuestionGroupDetail`，期望输出包含：

```go
func TestBuildStructuredExamQuestionContextBundleReadingGroup(t *testing.T) {
	t.Parallel()

	bundle := BuildStructuredExamQuestionContextBundle("第一篇阅读第21题答案和解析", newStructuredReadingGroup())
	if bundle == nil {
		t.Fatal("expected structured context bundle")
	}
	if !strings.Contains(bundle.Content, "[Exam Structured Context]") {
		t.Fatalf("missing structured context header:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "SoFi Stadium is the go-to destination.") {
		t.Fatalf("missing material text:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "21. Which team will play the most games?") {
		t.Fatalf("missing question stem:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "B. Los Angeles Rams") {
		t.Fatalf("missing option:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "答案: B") {
		t.Fatalf("missing answer:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "证据 chunk: chunk-21") {
		t.Fatalf("missing chunk ref:\n%s", bundle.Content)
	}
}
```

- [x] **步骤 2：运行测试确认失败**

运行：

```powershell
go test ./internal/searchutil -run TestBuildStructuredExamQuestionContextBundleReadingGroup -count=1
```

结果：首次红灯被本机 Go 环境拦截，`go` 不在 PATH。改用 `D:\rag\.cache\go-sdk-1.26.0\go\bin\go.exe` 后，编译进入既有 cgo 依赖问题：`internal/utils/inject.go` 中 `pg_query.Parse` / `pg_query.Deparse` 未定义。

- [x] **步骤 3：实现最小构建器**

创建 `internal/searchutil/exam_structured_context.go`：

```go
package searchutil

import (
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const defaultStructuredExamContextRuneLimit = 9000

func BuildStructuredExamQuestionContextBundle(query string, detail *types.QuestionGroupDetail) *ExamQuestionContextBundle {
	if detail == nil || detail.Group == nil {
		return nil
	}
	content := BuildStructuredExamQuestionContextText(detail, defaultStructuredExamContextRuneLimit)
	if strings.TrimSpace(content) == "" {
		return nil
	}
	return &ExamQuestionContextBundle{
		Label:   structuredExamContextLabel(detail),
		Content: content,
	}
}
```

- [x] **步骤 4：运行测试确认通过**

运行：

```powershell
go test ./internal/searchutil -run TestBuildStructuredExamQuestionContextBundleReadingGroup -count=1
```

结果：PASS。

本机 Go/cgo 阻塞已修复：Go 使用 `D:\rag\.cache\go-sdk-1.26.0\go`，GCC 使用 WinLibs GCC 14.2，`GOPATH/GOMODCACHE/GOCACHE/GOTMPDIR/TEMP/TMP` 迁到 `D:\rag\.cache`，`GOPROXY=https://goproxy.cn,direct`，并补充 SQLite amalgamation 头文件供 `sqlite-vec` CGO 编译。

## 任务 2：长度限制与空值稳定性

- [x] **步骤 1：补充长度限制测试**

在同一测试文件中添加超长原文场景：

```go
func TestBuildStructuredExamQuestionContextBundleTrimsLongMaterial(t *testing.T) {
	t.Parallel()

	detail := newStructuredReadingGroup()
	detail.Group.MaterialText = strings.Repeat("long material ", 1000)
	text := BuildStructuredExamQuestionContextText(detail, 240)
	if !strings.Contains(text, "[truncated]") {
		t.Fatalf("expected truncated marker:\n%s", text)
	}
}
```

- [x] **步骤 2：实现可配置文本构建函数**

实现 `BuildStructuredExamQuestionContextText(detail, runeLimit)`，用 `trimRunes` 控制输出长度。

- [x] **步骤 3：运行 searchutil 定向测试**

运行：

```powershell
go test ./internal/searchutil -count=1
```

结果：PASS。

## 任务 3：结构化上下文接入 Chat 与 Agent

- [x] **步骤 1：Repository 回链**

新增 `FindQuestionGroupDetailByChunkIDs`，从检索命中的 chunk refs 回到 `QuestionGroupDetail`。

- [x] **步骤 2：Chat pipeline 接入**

`PluginSearch.enrichExamQuestionContext` 先查结构化题组上下文，找不到时继续使用原 chunk 启发式兜底。

- [x] **步骤 3：Agent tool 接入**

`knowledge_search` 同样优先输出结构化题组上下文，确保 Agent 和普通对话共享同一套考试 RAG 表达。

- [x] **步骤 4：最终验证**

运行：

```powershell
go test ./internal/searchutil -count=1
go test ./internal/application/repository -run 'TestExamQuestionRepository_(ListQuestionGroupDetailsByBankWrapsLegacyQuestions|FindQuestionGroupDetailByChunkIDs)' -count=1
go test ./internal/application/service/chat_pipeline -count=1
go test ./internal/agent/tools -run TestKnowledgeSearchToolAppendExamContextBundlesPrefersStructuredGroup -count=1
go test ./internal/application/service -run '^$' -count=1
go test ./internal/container -run '^$' -count=1
```

结果：PASS。

## 下一步

- 建立高考英语阅读样例 query 集，验证“第几篇阅读原文/答案/解析”的命中率。
- 在 Agent 层设计考试专属工具，如 `exam_question_context`、`exam_learning_diagnosis`，减少通用 `knowledge_search` 的提示压力。
- 将结构化上下文输出的 metadata 暴露给前端 citation / 调试面板，便于观察模型到底引用了题组还是原始 chunk。
