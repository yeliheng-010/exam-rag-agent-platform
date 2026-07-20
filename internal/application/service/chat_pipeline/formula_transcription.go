package chatpipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

const (
	maxFormulaTranscriptionImages = 12
	maxFormulaTranscriptionRunes  = 2000
	noFormulaTranscription        = "NO_FORMULA"
	formulaAnswerTemperature      = 0.1
)

const formulaTranscriptionPrompt = `You are a verbatim mathematical formula transcriber. Inspect only this image. Output only LaTeX wrapped in $...$. Verify every numerator and denominator; never omit a fraction bar.`

const formulaAnswerSystemInstruction = `数学公式讲解必须遵守以下输出契约：
1. 回答开头按图号顺序逐字复制用户消息中“公式图片逐张转写”的每个相关 LaTeX 公式，并保留 $ 分隔符。
2. 全文数学表达式只使用 LaTeX，不得改写成 Unicode 或纯文本。
3. 只按已引用公式的顺序说明等价操作，不得新增上下文中没有的中间等式，不得展示矛盾的试算、自我纠正或失败过程。
4. 解释含分式等式的化简时，必须一步同乘所有分母的最小公倍式并直接得到下一条已引用公式；禁止先乘部分因子，禁止插入额外中间式。
5. 若证明某数列为等差数列，必须先用与题目和参考答案一致的等式定义目标数列 $b_n$；不得把不同数列写成相等，不得把目标数列偷换为原数列。推导后必须另起一行写出化简后的 $b_{n+1}-b_n=d$，并说明公差 $d$。
6. 输出只按“原始条件、等价变形、定义目标数列、相邻差与公差、复盘建议”生成一次；完成目标数列结论和一条复盘建议后立即停止，不得重新证明，不得继续讨论其他数列是否为等差数列。`

func transcribeInlineFormulaImages(
	ctx context.Context,
	model chat.Chat,
	query string,
	images []string,
) []string {
	imageURLs := selectInlineFormulaImages(query, images)
	results := ParallelMap(imageURLs, 2, func(_ int, imageURL string) string {
		return transcribeFormulaImage(ctx, model, imageURL)
	})

	transcriptions := make([]string, 0, len(results))
	for _, result := range results {
		if result != "" {
			transcriptions = append(transcriptions, result)
		}
	}
	return transcriptions
}

func transcribeFormulaImage(ctx context.Context, model chat.Chat, imageURL string) string {
	var preferred string
	for attempt := 1; attempt <= 2; attempt++ {
		thinking := false
		response, err := model.Chat(ctx, []chat.Message{{
			Role:    "user",
			Content: formulaTranscriptionPrompt,
			Images:  []string{imageURL},
		}}, &chat.ChatOptions{
			Temperature:         0.1,
			MaxCompletionTokens: 300,
			Thinking:            &thinking,
		})
		if err != nil {
			pipelineWarn(ctx, "FormulaTranscription", "model_call", map[string]interface{}{
				"image_url": imageURL,
				"attempt":   attempt,
				"error":     err.Error(),
			})
			continue
		}
		candidate := sanitizeFormulaTranscription(response.Content)
		preferred = preferStructurallyCompleteTranscription(preferred, candidate)
	}
	return preferred
}

func preferStructurallyCompleteTranscription(current, candidate string) string {
	if formulaStructureScore(candidate) > formulaStructureScore(current) {
		return candidate
	}
	return current
}

func formulaStructureScore(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, `\frac`)*100 +
		strings.Count(content, "_")*10 +
		strings.Count(content, "^")*10 +
		strings.Count(content, "{") +
		len([]rune(content))
}

func selectInlineFormulaImages(query string, images []string) []string {
	selected := make([]string, 0, min(len(images), maxFormulaTranscriptionImages))
	seen := make(map[string]struct{}, len(images))
	for _, imageURL := range images {
		if len(selected) >= maxFormulaTranscriptionImages {
			break
		}
		if !strings.HasPrefix(imageURL, "local://") || !strings.Contains(query, imageURL) {
			continue
		}
		if _, exists := seen[imageURL]; exists {
			continue
		}
		seen[imageURL] = struct{}{}
		selected = append(selected, imageURL)
	}
	return selected
}

func sanitizeFormulaTranscription(raw string) string {
	content := strings.TrimSpace(raw)
	for _, fence := range []string{"```latex", "```tex", "```"} {
		content = strings.TrimSpace(strings.TrimPrefix(content, fence))
	}
	content = strings.TrimSpace(strings.TrimSuffix(content, "```"))
	if content == "" || strings.EqualFold(content, noFormulaTranscription) {
		return ""
	}
	runes := []rune(content)
	if len(runes) > maxFormulaTranscriptionRunes {
		content = string(runes[:maxFormulaTranscriptionRunes])
	}
	return content
}

func buildFormulaTranscriptionContext(transcriptions []string) string {
	if len(transcriptions) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("【公式图片逐张转写】\n")
	builder.WriteString("以下是由单图逐字识别得到的候选公式。请结合当前小题文字筛选，只采用与当前小题直接相关的条件；回答开头必须按图号顺序原样引用下方 LaTeX 中的每个相关公式，不得转写成 Unicode 或纯文本，也不得自行补充与这些公式矛盾的变形：\n")
	for i, transcription := range transcriptions {
		builder.WriteString(fmt.Sprintf("图 %d：%s\n", i+1, transcription))
	}
	return strings.TrimSpace(builder.String())
}

func applyFormulaTranscriptionContext(chatManage *types.ChatManage, formulaContext string) string {
	formulaContext = strings.TrimSpace(formulaContext)
	if formulaContext == "" {
		return chatManage.Query
	}

	queryForRewrite := appendContext(chatManage.Query, formulaContext)
	chatManage.QuotedContext = appendContext(chatManage.QuotedContext, formulaContext)
	baseSystemPrompt := chatManage.SystemPromptOverride
	if strings.TrimSpace(baseSystemPrompt) == "" {
		baseSystemPrompt = chatManage.SummaryConfig.Prompt
	}
	chatManage.SystemPromptOverride = appendContext(baseSystemPrompt, formulaAnswerSystemInstruction)
	chatManage.SummaryConfig.Temperature = formulaAnswerTemperature
	return queryForRewrite
}

func appendContext(existing, supplemental string) string {
	if existing == "" {
		return supplemental
	}
	if strings.Contains(existing, supplemental) {
		return existing
	}
	return existing + "\n\n" + supplemental
}
