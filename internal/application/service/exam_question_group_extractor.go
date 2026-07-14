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

const examQuestionGroupExtractionSystemPrompt = `You are an exam paper structuring assistant. Extract real question groups, not isolated options. Return JSON only.`

var errNoValidQuestionGroupDrafts = errors.New("model returned no valid question group drafts")

type examQuestionGroupExtractor struct {
	modelService interfaces.ModelService
	registry     *QuestionGroupStrategyRegistry
}

func NewExamQuestionGroupExtractor(modelService interfaces.ModelService) interfaces.ExamQuestionGroupExtractor {
	return &examQuestionGroupExtractor{
		modelService: modelService,
		registry:     NewQuestionGroupStrategyRegistry(),
	}
}

func (e *examQuestionGroupExtractor) Extract(ctx context.Context, material *types.ExamMaterial, task *types.ExamStructuringTask, chunks []*types.Chunk) ([]*types.ExamQuestionGroupDraftCandidate, string, error) {
	return e.ExtractWithProgress(ctx, material, task, chunks, nil)
}

func (e *examQuestionGroupExtractor) ExtractWithProgress(
	ctx context.Context,
	material *types.ExamMaterial,
	task *types.ExamStructuringTask,
	chunks []*types.Chunk,
	observer interfaces.ExamQuestionGroupBatchObserver,
) ([]*types.ExamQuestionGroupDraftCandidate, string, error) {
	if e == nil || e.modelService == nil {
		return nil, "", errors.New("exam question group extractor model service is not configured")
	}
	strategy := e.strategy(material, task)
	modelID, err := e.resolveChatModelID(ctx)
	if err != nil {
		return nil, "", err
	}
	chatModel, err := e.modelService.GetChatModel(ctx, modelID)
	if err != nil {
		return nil, "", err
	}
	return e.extractBatches(ctx, chatModel, strategy, material, task, chunks, observer)
}

func (e *examQuestionGroupExtractor) strategy(material *types.ExamMaterial, task *types.ExamStructuringTask) ExamQuestionGroupExtractionStrategy {
	if e != nil && e.registry != nil {
		return e.registry.Match(material, task)
	}
	return NewQuestionGroupStrategyRegistry().Match(material, task)
}

func (e *examQuestionGroupExtractor) resolveChatModelID(ctx context.Context) (string, error) {
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
		return "", errors.New("no available KnowledgeQA model for exam question group extraction")
	}
	return fallback, nil
}

func callQuestionGroupExtractionModel(ctx context.Context, chatModel chat.Chat, prompt string, maxTokens int) (string, error) {
	thinking := false
	response, err := chatModel.Chat(ctx, []chat.Message{
		{Role: "system", Content: examQuestionGroupExtractionSystemPrompt},
		{Role: "user", Content: prompt},
	}, &chat.ChatOptions{
		Temperature: 0.1,
		MaxTokens:   maxTokens,
		Thinking:    &thinking,
		Format:      json.RawMessage(`"json"`),
	})
	if err != nil {
		return "", fmt.Errorf("extract exam question groups: %w", err)
	}
	if response == nil || strings.TrimSpace(response.Content) == "" {
		return "", errors.New("empty model response")
	}
	return response.Content, nil
}

func parseQuestionGroupDraftCandidates(raw string) ([]*types.ExamQuestionGroupDraftCandidate, error) {
	payload := extractJSONPayload(raw)
	var envelope struct {
		QuestionGroups []*types.ExamQuestionGroupDraftCandidate `json:"question_groups"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err == nil && len(envelope.QuestionGroups) > 0 {
		return validateQuestionGroupDraftCandidates(envelope.QuestionGroups)
	}
	var candidates []*types.ExamQuestionGroupDraftCandidate
	if err := json.Unmarshal([]byte(payload), &candidates); err != nil {
		return nil, fmt.Errorf("invalid model json: %w", err)
	}
	return validateQuestionGroupDraftCandidates(candidates)
}

func validateQuestionGroupDraftCandidates(candidates []*types.ExamQuestionGroupDraftCandidate) ([]*types.ExamQuestionGroupDraftCandidate, error) {
	valid := make([]*types.ExamQuestionGroupDraftCandidate, 0, len(candidates))
	rejected := make([]string, 0)
	for index, candidate := range candidates {
		normalizeQuestionGroupCandidate(candidate)
		if err := validateQuestionGroupCandidate(candidate, false); err != nil {
			rejected = append(rejected, fmt.Sprintf("candidate %d: %v", index+1, err))
			continue
		}
		valid = append(valid, candidate)
	}
	if len(valid) == 0 {
		if len(rejected) == 0 {
			return nil, errNoValidQuestionGroupDrafts
		}
		return nil, fmt.Errorf("%w: %s", errNoValidQuestionGroupDrafts, strings.Join(rejected, "; "))
	}
	return valid, nil
}

func normalizeQuestionGroupCandidate(candidate *types.ExamQuestionGroupDraftCandidate) {
	if candidate == nil {
		return
	}
	candidate.GroupNo = strings.TrimSpace(candidate.GroupNo)
	candidate.GroupType = strings.TrimSpace(candidate.GroupType)
	if candidate.GroupType == "" {
		candidate.GroupType = "single_question"
	}
	candidate.Title = strings.TrimSpace(candidate.Title)
	candidate.MaterialText = strings.TrimSpace(candidate.MaterialText)
	candidate.MaterialFormat = strings.TrimSpace(candidate.MaterialFormat)
	if candidate.MaterialFormat == "" {
		candidate.MaterialFormat = "plain_text"
	}
	candidate.Confidence = normalizeConfidence(candidate.Confidence)
	if candidate.SourceChunkIDs == nil {
		candidate.SourceChunkIDs = []string{}
	}
	if candidate.Assets == nil {
		candidate.Assets = []types.ExamQuestionGroupDraftAssetCandidate{}
	}
	for i := range candidate.Assets {
		normalizeQuestionGroupAssetCandidate(&candidate.Assets[i], i+1)
	}
	if candidate.Questions == nil {
		candidate.Questions = []types.ExamQuestionGroupDraftQuestionCandidate{}
	}
	for i := range candidate.Questions {
		normalizeQuestionGroupQuestionCandidate(&candidate.Questions[i], i+1)
	}
}

func normalizeQuestionGroupAssetCandidate(candidate *types.ExamQuestionGroupDraftAssetCandidate, order int) {
	if candidate == nil {
		return
	}
	candidate.AssetType = strings.TrimSpace(candidate.AssetType)
	if candidate.AssetType == "" {
		candidate.AssetType = "image"
	}
	candidate.StorageURI = strings.TrimSpace(candidate.StorageURI)
	candidate.AltText = strings.TrimSpace(candidate.AltText)
	candidate.SourceChunkID = strings.TrimSpace(candidate.SourceChunkID)
	if candidate.BBox == nil {
		candidate.BBox = types.JSONMap{}
	}
	if candidate.Metadata == nil {
		candidate.Metadata = types.JSONMap{}
	}
	if candidate.SortOrder <= 0 {
		candidate.SortOrder = order
	}
}

func normalizeQuestionGroupQuestionCandidate(candidate *types.ExamQuestionGroupDraftQuestionCandidate, order int) {
	if candidate == nil {
		return
	}
	candidate.QuestionNo = strings.TrimSpace(candidate.QuestionNo)
	candidate.QuestionTypeCode = strings.TrimSpace(candidate.QuestionTypeCode)
	if candidate.QuestionTypeCode == "" {
		candidate.QuestionTypeCode = "unknown"
	}
	candidate.Stem = strings.TrimSpace(candidate.Stem)
	if candidate.Options == nil {
		candidate.Options = []types.ExamQuestionDraftOption{}
	}
	for i := range candidate.Options {
		candidate.Options[i].Key = strings.TrimSpace(candidate.Options[i].Key)
		candidate.Options[i].Content = strings.TrimSpace(candidate.Options[i].Content)
	}
	if candidate.Answer == nil {
		candidate.Answer = types.JSONMap{}
	}
	candidate.Explanation = strings.TrimSpace(candidate.Explanation)
	if candidate.Evidence == nil {
		candidate.Evidence = []types.JSONMap{}
	}
	if candidate.Metadata == nil {
		candidate.Metadata = types.JSONMap{}
	}
	candidate.Difficulty = strings.TrimSpace(candidate.Difficulty)
	if candidate.Difficulty == "" {
		candidate.Difficulty = "unknown"
	}
	candidate.Confidence = normalizeConfidence(candidate.Confidence)
	if candidate.OrderInGroup <= 0 {
		candidate.OrderInGroup = order
	}
	if candidate.SourceChunkIDs == nil {
		candidate.SourceChunkIDs = []string{}
	}
}

func validateQuestionGroupCandidate(candidate *types.ExamQuestionGroupDraftCandidate, requireMaterial bool) error {
	if candidate == nil {
		return errors.New("question group candidate is nil")
	}
	if strings.TrimSpace(candidate.GroupType) == "" {
		return errors.New("question group type is required")
	}
	if requireMaterial && strings.TrimSpace(candidate.MaterialText) == "" {
		return errors.New("question group material text is required")
	}
	if len(candidate.Questions) == 0 {
		return errors.New("question group must contain questions")
	}
	for _, question := range candidate.Questions {
		if strings.TrimSpace(question.Stem) == "" {
			return errors.New("question stem is required")
		}
	}
	return nil
}

func isChoiceQuestion(questionTypeCode string) bool {
	switch strings.TrimSpace(questionTypeCode) {
	case "single_choice", "multiple_choice":
		return true
	default:
		return false
	}
}
