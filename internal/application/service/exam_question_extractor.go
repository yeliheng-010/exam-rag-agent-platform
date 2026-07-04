package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const examQuestionExtractionSystemPrompt = `你是考试试卷结构化助手。请从用户提供的试卷 chunk 中抽取题目草稿。
只返回 JSON，不要输出 Markdown、解释或代码块。JSON 结构必须是：
{
  "questions": [
    {
      "question_no": "题号",
      "question_type_code": "single_choice|multiple_choice|fill_blank|reading|writing|unknown",
      "stem": "题干",
      "options": [{"key":"A","content":"选项内容"}],
      "answer": {"value":"A"},
      "explanation": "解析",
      "difficulty": "easy|medium|hard|unknown",
      "confidence": 0.8,
      "source_chunk_ids": ["chunk id"]
    }
  ]
}
如果无法确定题型，使用 unknown；如果没有选项，options 返回空数组。`

const (
	questionExtractionBodyPromptLimit   = 18000
	questionExtractionAnswerPromptLimit = 7000
	questionExtractionChunkRuneLimit    = 3000
)

type examQuestionExtractor struct {
	modelService interfaces.ModelService
}

func NewExamQuestionExtractor(modelService interfaces.ModelService) interfaces.ExamQuestionExtractor {
	return &examQuestionExtractor{modelService: modelService}
}

func (e *examQuestionExtractor) Extract(ctx context.Context, material *types.ExamMaterial, task *types.ExamStructuringTask, chunks []*types.Chunk) ([]*types.ExamQuestionDraftCandidate, string, error) {
	if e == nil || e.modelService == nil {
		return nil, "", errors.New("exam question extractor model service is not configured")
	}
	modelID, err := e.resolveChatModelID(ctx)
	if err != nil {
		return nil, "", err
	}
	chatModel, err := e.modelService.GetChatModel(ctx, modelID)
	if err != nil {
		return nil, "", err
	}
	raw, err := callQuestionExtractionModel(ctx, chatModel, buildQuestionExtractionPrompt(material, task, chunks))
	if err != nil {
		return nil, raw, err
	}
	candidates, err := parseQuestionDraftCandidates(raw)
	return candidates, raw, err
}

func (e *examQuestionExtractor) resolveChatModelID(ctx context.Context) (string, error) {
	models, err := e.modelService.ListModels(ctx)
	if err != nil {
		return "", fmt.Errorf("list chat models: %w", err)
	}
	var fallback string
	for _, model := range models {
		if model == nil || model.Type != types.ModelTypeKnowledgeQA {
			continue
		}
		if model.Status != "" && model.Status != types.ModelStatusActive {
			continue
		}
		if model.IsDefault {
			return model.ID, nil
		}
		if fallback == "" {
			fallback = model.ID
		}
	}
	if fallback == "" {
		return "", errors.New("no available KnowledgeQA model for exam question extraction")
	}
	return fallback, nil
}

func callQuestionExtractionModel(ctx context.Context, chatModel chat.Chat, prompt string) (string, error) {
	thinking := false
	response, err := chatModel.Chat(ctx, []chat.Message{
		{Role: "system", Content: examQuestionExtractionSystemPrompt},
		{Role: "user", Content: prompt},
	}, &chat.ChatOptions{
		Temperature: 0.1,
		MaxTokens:   4096,
		Thinking:    &thinking,
	})
	if err != nil {
		return "", fmt.Errorf("extract exam questions: %w", err)
	}
	if response == nil || strings.TrimSpace(response.Content) == "" {
		return "", errors.New("empty model response")
	}
	return response.Content, nil
}

func buildQuestionExtractionPrompt(material *types.ExamMaterial, task *types.ExamStructuringTask, chunks []*types.Chunk) string {
	var builder strings.Builder
	builder.WriteString("请根据以下试卷内容抽取题目草稿，优先抽取高考英语第一篇阅读或第一个完整客观题组，并把题干、选项、答案和来源 chunk 关联起来。本轮最多返回 8 道题，不要为了覆盖整卷返回过长 JSON。")
	if material != nil {
		builder.WriteString(fmt.Sprintf("\n资料标题：%s\n考试域：%s\n科目：%s\n", material.Title, material.DomainID, promptSubjectID(material)))
	}
	if task != nil {
		builder.WriteString(fmt.Sprintf("目标题库：%s\n", task.QuestionBankID))
	}
	builder.WriteString("\n试卷 chunks：\n")
	bodyChunks := selectQuestionExtractionBodyChunks(chunks)
	answerChunks := selectLikelyAnswerChunks(chunks)
	included := make(map[string]bool, len(chunks))
	for _, chunk := range bodyChunks {
		if chunk == nil {
			continue
		}
		writeQuestionPromptChunk(&builder, chunk)
		included[questionPromptChunkKey(chunk)] = true
		if builder.Len() > questionExtractionBodyPromptLimit {
			builder.WriteString("\n[正文 chunk 因长度限制省略；以下会保留识别到的答案区 chunk]\n")
			break
		}
	}
	appendAnswerPromptChunks(&builder, answerChunks, included)
	return builder.String()
}

func selectQuestionExtractionBodyChunks(chunks []*types.Chunk) []*types.Chunk {
	start := -1
	for i, chunk := range chunks {
		if chunk == nil {
			continue
		}
		if isLikelyFirstReadingStart(chunk.Content) {
			start = i
			break
		}
	}
	if start < 0 {
		return chunks
	}
	selected := make([]*types.Chunk, 0, 10)
	for i := start; i < len(chunks); i++ {
		chunk := chunks[i]
		if chunk == nil {
			continue
		}
		selected = append(selected, chunk)
		if len(selected) >= 2 && isLikelyNextReadingPassage(chunk.Content) {
			break
		}
		if len(selected) >= 10 {
			break
		}
	}
	if len(selected) == 0 {
		return chunks
	}
	return selected
}

func isLikelyFirstReadingStart(content string) bool {
	normalized := strings.ToLower(content)
	return strings.Contains(content, "阅读理解") ||
		strings.Contains(content, "阅读下列短文") ||
		strings.Contains(normalized, "reading comprehension") ||
		strings.Contains(normalized, "passage 1")
}

func isLikelyNextReadingPassage(content string) bool {
	normalized := "\n" + strings.TrimSpace(content) + "\n"
	return strings.Contains(normalized, "\n**B**\n") ||
		strings.Contains(normalized, "\nB\n") ||
		strings.Contains(normalized, "\nPassage 2\n")
}

func promptSubjectID(material *types.ExamMaterial) string {
	if material == nil || material.SubjectID == nil {
		return ""
	}
	return strings.TrimSpace(*material.SubjectID)
}

func writeQuestionPromptChunk(builder *strings.Builder, chunk *types.Chunk) {
	builder.WriteString(fmt.Sprintf(
		"\n--- chunk_id: %s | chunk_index: %d ---\n%s\n",
		chunk.ID,
		chunk.ChunkIndex,
		truncateRunes(chunk.Content, questionExtractionChunkRuneLimit),
	))
}

func appendAnswerPromptChunks(builder *strings.Builder, chunks []*types.Chunk, included map[string]bool) {
	if len(chunks) == 0 {
		return
	}
	builder.WriteString("\n答案区 chunks：\n")
	addedLen := 0
	for _, chunk := range chunks {
		key := questionPromptChunkKey(chunk)
		if included[key] {
			continue
		}
		before := builder.Len()
		writeQuestionPromptChunk(builder, chunk)
		included[key] = true
		addedLen += builder.Len() - before
		if addedLen > questionExtractionAnswerPromptLimit {
			builder.WriteString("\n[答案区 chunk 因长度限制省略]\n")
			break
		}
	}
}

func selectLikelyAnswerChunks(chunks []*types.Chunk) []*types.Chunk {
	out := make([]*types.Chunk, 0)
	for _, chunk := range chunks {
		if chunk == nil || !isLikelyAnswerChunk(chunk.Content) {
			continue
		}
		out = append(out, chunk)
	}
	return out
}

func isLikelyAnswerChunk(content string) bool {
	normalized := strings.ToLower(content)
	return strings.Contains(content, "答案") ||
		strings.Contains(content, "参考答案") ||
		strings.Contains(normalized, "answer key") ||
		strings.Contains(normalized, "answers")
}

func questionPromptChunkKey(chunk *types.Chunk) string {
	if chunk.ID != "" {
		return chunk.ID
	}
	return fmt.Sprintf("chunk-index-%d", chunk.ChunkIndex)
}

func parseQuestionDraftCandidates(raw string) ([]*types.ExamQuestionDraftCandidate, error) {
	payload := extractJSONPayload(raw)
	var envelope struct {
		Questions []*types.ExamQuestionDraftCandidate `json:"questions"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err == nil && len(envelope.Questions) > 0 {
		return validateQuestionDraftCandidates(envelope.Questions)
	}
	var candidates []*types.ExamQuestionDraftCandidate
	if err := json.Unmarshal([]byte(payload), &candidates); err != nil {
		return nil, fmt.Errorf("invalid model json: %w", err)
	}
	return validateQuestionDraftCandidates(candidates)
}

func validateQuestionDraftCandidates(candidates []*types.ExamQuestionDraftCandidate) ([]*types.ExamQuestionDraftCandidate, error) {
	valid := make([]*types.ExamQuestionDraftCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || strings.TrimSpace(candidate.Stem) == "" {
			continue
		}
		if candidate.QuestionTypeCode == "" {
			candidate.QuestionTypeCode = "unknown"
		}
		if candidate.Difficulty == "" {
			candidate.Difficulty = "unknown"
		}
		if candidate.Answer == nil {
			candidate.Answer = types.JSONMap{}
		}
		if candidate.Options == nil {
			candidate.Options = []types.ExamQuestionDraftOption{}
		}
		valid = append(valid, candidate)
	}
	if len(valid) == 0 {
		return nil, errors.New("model returned no valid question drafts")
	}
	return valid, nil
}

func extractJSONPayload(raw string) string {
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	startObj := strings.Index(text, "{")
	startArr := strings.Index(text, "[")
	start := startObj
	if start < 0 || (startArr >= 0 && startArr < start) {
		start = startArr
	}
	if start < 0 {
		return text
	}
	endObj := strings.LastIndex(text, "}")
	endArr := strings.LastIndex(text, "]")
	end := endObj
	if endArr > end {
		end = endArr
	}
	if end >= start {
		return text[start : end+1]
	}
	return text[start:]
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "\n[chunk 内容截断]"
}
