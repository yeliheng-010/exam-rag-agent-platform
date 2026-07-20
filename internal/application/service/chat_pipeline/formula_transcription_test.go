package chatpipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

type formulaTranscriptionChat struct {
	mu        sync.Mutex
	responses map[string]string
	errors    map[string]error
	calls     []chat.Message
}

func (m *formulaTranscriptionChat) Chat(
	_ context.Context,
	messages []chat.Message,
	_ *chat.ChatOptions,
) (*types.ChatResponse, error) {
	message := messages[len(messages)-1]
	m.mu.Lock()
	m.calls = append(m.calls, message)
	m.mu.Unlock()

	imageURL := message.Images[0]
	if err := m.errors[imageURL]; err != nil {
		return nil, err
	}
	return &types.ChatResponse{Content: m.responses[imageURL]}, nil
}

func (m *formulaTranscriptionChat) ChatStream(
	context.Context,
	[]chat.Message,
	*chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *formulaTranscriptionChat) GetModelName() string { return "formula-test" }
func (m *formulaTranscriptionChat) GetModelID() string   { return "formula-test" }

func TestTranscribeInlineFormulaImagesOneAtATime(t *testing.T) {
	first := "local://42/exports/first.png"
	second := "local://42/exports/second.png"
	model := &formulaTranscriptionChat{
		responses: map[string]string{
			first:  `$a_1=3$`,
			second: "```latex\n$\\frac{a_{n+1}}{n}=\\frac{a_n}{n+1}+\\frac{1}{n(n+1)}$\n```",
		},
		errors: map[string]error{},
	}
	query := "题面：![条件一](" + first + ")\n![条件二](" + second + ")"

	got := transcribeInlineFormulaImages(context.Background(), model, query, []string{
		first,
		second,
		"data:image/png;base64,ignored",
		"local://42/exports/not-in-query.png",
	})

	want := []string{
		`$a_1=3$`,
		`$\frac{a_{n+1}}{n}=\frac{a_n}{n+1}+\frac{1}{n(n+1)}$`,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d transcriptions, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("transcription %d = %q, want %q", i, got[i], want[i])
		}
	}

	model.mu.Lock()
	defer model.mu.Unlock()
	if len(model.calls) != 4 {
		t.Fatalf("model calls = %d, want 4", len(model.calls))
	}
	for i, call := range model.calls {
		if len(call.Images) != 1 {
			t.Fatalf("call %d has %d images, want exactly 1", i, len(call.Images))
		}
		if !strings.Contains(call.Content, "numerator") || !strings.Contains(call.Content, "denominator") {
			t.Fatalf("call %d prompt does not require numerator/denominator verification: %q", i, call.Content)
		}
	}
}

func TestPreferStructurallyCompleteFormulaTranscription(t *testing.T) {
	omittedDenominator := `$a_{n+1}=\frac{a_n}{n+1}+\frac{1}{n(n+1)}$`
	complete := `$\frac{a_{n+1}}{n}=\frac{a_n}{n+1}+\frac{1}{n(n+1)}$`

	if got := preferStructurallyCompleteTranscription(omittedDenominator, complete); got != complete {
		t.Fatalf("preferred transcription = %q, want %q", got, complete)
	}
	if got := preferStructurallyCompleteTranscription("", complete); got != complete {
		t.Fatalf("preferred non-empty transcription = %q, want %q", got, complete)
	}
}

func TestTranscribeFormulaImageRetriesIndependently(t *testing.T) {
	imageURL := "local://42/exports/recurrence.png"
	firstCandidate := `$a_{n+1}=\frac{a_n}{n+1}+\frac{1}{n(n+1)}$`
	model := &formulaTranscriptionChat{
		responses: map[string]string{imageURL: firstCandidate},
		errors:    map[string]error{},
	}

	transcribeFormulaImage(context.Background(), model, imageURL)

	model.mu.Lock()
	defer model.mu.Unlock()
	if len(model.calls) != 2 {
		t.Fatalf("model calls = %d, want 2", len(model.calls))
	}
	if model.calls[1].Content != model.calls[0].Content {
		t.Fatalf("second pass prompt = %q, want independent retry with %q", model.calls[1].Content, model.calls[0].Content)
	}
	if strings.Contains(model.calls[1].Content, firstCandidate) {
		t.Fatalf("second pass prompt %q is anchored to first candidate %q", model.calls[1].Content, firstCandidate)
	}
}

func TestFormulaTranscriptionPromptStaysSinglePurpose(t *testing.T) {
	if strings.Contains(formulaTranscriptionPrompt, noFormulaTranscription) {
		t.Fatalf("formula transcription prompt should not add a no-formula classification branch: %q", formulaTranscriptionPrompt)
	}
	if len([]rune(formulaTranscriptionPrompt)) > 250 {
		t.Fatalf("formula transcription prompt is too verbose for reliable small-formula OCR: %d runes", len([]rune(formulaTranscriptionPrompt)))
	}
}

func TestSelectInlineFormulaImagesKeepsEightCurrentAnswerImages(t *testing.T) {
	images := make([]string, 8)
	var query strings.Builder
	for i := range images {
		images[i] = fmt.Sprintf("local://42/exports/formula-%d.png", i+1)
		query.WriteString(fmt.Sprintf("![公式 %d](%s)\n", i+1, images[i]))
	}

	got := selectInlineFormulaImages(query.String(), images)

	if len(got) != len(images) {
		t.Fatalf("selected %d images, want %d: %#v", len(got), len(images), got)
	}
}

func TestTranscribeInlineFormulaImagesSkipsFailuresAndNonFormula(t *testing.T) {
	failed := "local://42/exports/failed.png"
	plain := "local://42/exports/plain.png"
	model := &formulaTranscriptionChat{
		responses: map[string]string{plain: "NO_FORMULA"},
		errors:    map[string]error{failed: errors.New("vision unavailable")},
	}
	query := "![失败](" + failed + ")\n![非公式](" + plain + ")"

	got := transcribeInlineFormulaImages(context.Background(), model, query, []string{failed, plain})

	if len(got) != 0 {
		t.Fatalf("got %#v, want no transcriptions", got)
	}
}

func TestBuildFormulaTranscriptionContext(t *testing.T) {
	got := buildFormulaTranscriptionContext([]string{`$a_1=3$`, `$\frac{a_{n+1}}{n}=1$`})
	for _, required := range []string{
		"【公式图片逐张转写】",
		"候选公式",
		"原样引用下方 LaTeX",
		"图 1：$a_1=3$",
		`图 2：$\frac{a_{n+1}}{n}=1$`,
	} {
		if !strings.Contains(got, required) {
			t.Fatalf("context %q does not contain %q", got, required)
		}
	}
}

func TestApplyFormulaTranscriptionContextFeedsRewriteAndFinalAnswer(t *testing.T) {
	chatManage := &types.ChatManage{
		PipelineRequest: types.PipelineRequest{
			Query:         "原始题目",
			SummaryConfig: types.SummaryConfig{Prompt: "基础系统提示"},
		},
		PipelineState: types.PipelineState{QuotedContext: "已有引用"},
	}
	formulaContext := "【公式图片逐张转写】\n图 1：$a_1=3$"

	queryForRewrite := applyFormulaTranscriptionContext(chatManage, formulaContext)

	if queryForRewrite != "原始题目\n\n"+formulaContext {
		t.Fatalf("rewrite query = %q", queryForRewrite)
	}
	if chatManage.QuotedContext != "已有引用\n\n"+formulaContext {
		t.Fatalf("final-answer context = %q", chatManage.QuotedContext)
	}
	for _, required := range []string{
		"基础系统提示",
		"逐字",
		"不得新增上下文中没有的中间等式",
		"所有分母的最小公倍式",
		"必须一步同乘",
		"禁止先乘部分因子",
		`$b_{n+1}-b_n=d$`,
		"不得把不同数列写成相等",
		"不得把目标数列偷换为原数列",
		"完成目标数列结论和一条复盘建议后立即停止",
		"不得继续讨论其他数列是否为等差数列",
		"另起一行",
	} {
		if !strings.Contains(chatManage.SystemPromptOverride, required) {
			t.Fatalf("formula system prompt %q does not contain %q", chatManage.SystemPromptOverride, required)
		}
	}
	if chatManage.SummaryConfig.Temperature != 0.1 {
		t.Fatalf("formula answer temperature = %v, want 0.1", chatManage.SummaryConfig.Temperature)
	}
}
