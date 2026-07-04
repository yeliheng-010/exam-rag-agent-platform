package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
)

func (s *examQuestionDraftService) buildDrafts(task *types.ExamStructuringTask, material *types.ExamMaterial, candidates []*types.ExamQuestionDraftCandidate, rawOutput string) ([]*types.ExamQuestionDraft, error) {
	if len(candidates) == 0 {
		return nil, errors.New("model returned no question drafts")
	}
	drafts := make([]*types.ExamQuestionDraft, 0, len(candidates))
	now := time.Now()
	for _, candidate := range candidates {
		draft, err := buildDraftFromCandidate(task, material, candidate, rawOutput, now)
		if err != nil {
			return nil, err
		}
		drafts = append(drafts, draft)
	}
	return drafts, nil
}

func buildDraftFromCandidate(task *types.ExamStructuringTask, material *types.ExamMaterial, candidate *types.ExamQuestionDraftCandidate, rawOutput string, now time.Time) (*types.ExamQuestionDraft, error) {
	if candidate == nil || strings.TrimSpace(candidate.Stem) == "" {
		return nil, errors.New("question draft stem is required")
	}
	optionsJSON, err := marshalJSON(candidate.Options)
	if err != nil {
		return nil, err
	}
	chunkJSON, err := marshalJSON(candidate.SourceChunkIDs)
	if err != nil {
		return nil, err
	}
	difficulty := strings.TrimSpace(candidate.Difficulty)
	if difficulty == "" {
		difficulty = "unknown"
	}
	if candidate.RawModelOutput != "" {
		rawOutput = candidate.RawModelOutput
	}
	return &types.ExamQuestionDraft{
		ID:               uuid.New().String(),
		TenantID:         task.TenantID,
		SpaceID:          task.SpaceID,
		TaskID:           task.ID,
		MaterialID:       material.ID,
		QuestionBankID:   task.QuestionBankID,
		DomainID:         material.DomainID,
		SubjectID:        material.SubjectID,
		SourceChunkIDs:   chunkJSON,
		QuestionNo:       strings.TrimSpace(candidate.QuestionNo),
		QuestionTypeCode: strings.TrimSpace(candidate.QuestionTypeCode),
		Stem:             strings.TrimSpace(candidate.Stem),
		OptionsJSON:      optionsJSON,
		AnswerJSON:       normalizeAnswer(candidate.Answer),
		Explanation:      strings.TrimSpace(candidate.Explanation),
		Difficulty:       difficulty,
		Confidence:       normalizeConfidence(candidate.Confidence),
		Status:           types.ExamQuestionDraftStatusPendingReview,
		RawModelOutput:   rawOutput,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func applyDraftUpdate(draft *types.ExamQuestionDraft, req *types.UpdateExamQuestionDraftRequest) error {
	optionsJSON, err := marshalJSON(req.Options)
	if err != nil {
		return err
	}
	chunkJSON, err := marshalJSON(req.SourceChunkIDs)
	if err != nil {
		return err
	}
	draft.QuestionNo = strings.TrimSpace(req.QuestionNo)
	draft.QuestionTypeCode = strings.TrimSpace(req.QuestionTypeCode)
	draft.Stem = strings.TrimSpace(req.Stem)
	draft.OptionsJSON = optionsJSON
	draft.AnswerJSON = normalizeAnswer(req.Answer)
	draft.Explanation = strings.TrimSpace(req.Explanation)
	draft.Difficulty = strings.TrimSpace(req.Difficulty)
	if draft.Difficulty == "" {
		draft.Difficulty = "unknown"
	}
	draft.SourceChunkIDs = chunkJSON
	draft.UpdatedAt = time.Now()
	return nil
}

func buildQuestionDetailFromDraft(draft *types.ExamQuestionDraft, userID string) (*types.QuestionDetail, error) {
	options, err := draftOptions(draft)
	if err != nil {
		return nil, err
	}
	chunkIDs, err := draftSourceChunkIDs(draft)
	if err != nil {
		return nil, err
	}
	answer := draftAnswerText(draft.AnswerJSON)
	if answer == "" {
		return nil, errors.New("draft answer is required")
	}
	now := time.Now()
	questionID := uuid.New().String()
	return &types.QuestionDetail{
		Question:     buildQuestionFromDraft(draft, questionID, userID, now),
		Options:      buildQuestionOptions(questionID, options),
		Answers:      buildQuestionAnswers(questionID, answer, now),
		Explanations: buildQuestionExplanations(questionID, draft.Explanation, now),
		ChunkRefs:    buildQuestionChunkRefs(questionID, chunkIDs, draft.Confidence, now),
	}, nil
}

func buildQuestionFromDraft(draft *types.ExamQuestionDraft, questionID string, userID string, now time.Time) *types.Question {
	return &types.Question{
		ID:              questionID,
		TenantID:        draft.TenantID,
		QuestionBankID:  draft.QuestionBankID,
		DomainID:        draft.DomainID,
		SubjectID:       draft.SubjectID,
		Stem:            draft.Stem,
		Difficulty:      draft.Difficulty,
		ReviewStatus:    types.ExamReviewStatusPrivate,
		Status:          "active",
		CreatedByUserID: userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func buildQuestionOptions(questionID string, options []types.ExamQuestionDraftOption) []*types.QuestionOption {
	out := make([]*types.QuestionOption, 0, len(options))
	for i, option := range options {
		out = append(out, &types.QuestionOption{
			ID:         uuid.New().String(),
			QuestionID: questionID,
			OptionKey:  strings.TrimSpace(option.Key),
			Content:    strings.TrimSpace(option.Content),
			SortOrder:  i + 1,
		})
	}
	return out
}

func buildQuestionAnswers(questionID string, answer string, now time.Time) []*types.QuestionAnswer {
	return []*types.QuestionAnswer{{
		ID:         uuid.New().String(),
		QuestionID: questionID,
		AnswerText: answer,
		IsCorrect:  true,
		CreatedAt:  now,
	}}
}

func buildQuestionExplanations(questionID string, explanation string, now time.Time) []*types.QuestionExplanation {
	if strings.TrimSpace(explanation) == "" {
		return nil
	}
	return []*types.QuestionExplanation{{
		ID:              uuid.New().String(),
		QuestionID:      questionID,
		ExplanationText: strings.TrimSpace(explanation),
		SourceType:      "llm_review",
		CreatedAt:       now,
	}}
}

func buildQuestionChunkRefs(questionID string, chunkIDs []string, confidence float64, now time.Time) []*types.QuestionChunkRef {
	out := make([]*types.QuestionChunkRef, 0, len(chunkIDs))
	for _, chunkID := range chunkIDs {
		chunkID = strings.TrimSpace(chunkID)
		if chunkID == "" {
			continue
		}
		out = append(out, &types.QuestionChunkRef{
			QuestionID: questionID,
			ChunkID:    chunkID,
			RefType:    "evidence",
			Confidence: confidence,
			CreatedAt:  now,
		})
	}
	return out
}

func draftOptions(draft *types.ExamQuestionDraft) ([]types.ExamQuestionDraftOption, error) {
	var options []types.ExamQuestionDraftOption
	if len(draft.OptionsJSON) == 0 {
		return options, nil
	}
	if err := json.Unmarshal(draft.OptionsJSON, &options); err != nil {
		return nil, err
	}
	return options, nil
}

func draftSourceChunkIDs(draft *types.ExamQuestionDraft) ([]string, error) {
	var chunkIDs []string
	if len(draft.SourceChunkIDs) == 0 {
		return chunkIDs, nil
	}
	if err := json.Unmarshal(draft.SourceChunkIDs, &chunkIDs); err != nil {
		return nil, err
	}
	return chunkIDs, nil
}

func draftAnswerText(answer types.JSONMap) string {
	if answer == nil {
		return ""
	}
	if value, ok := answer["value"].(string); ok {
		return strings.TrimSpace(value)
	}
	if values, ok := answer["values"].([]any); ok {
		parts := make([]string, 0, len(values))
		for _, value := range values {
			parts = append(parts, strings.TrimSpace(fmt.Sprint(value)))
		}
		return strings.Join(parts, ",")
	}
	if values, ok := answer["values"].([]string); ok {
		parts := make([]string, 0, len(values))
		for _, value := range values {
			parts = append(parts, strings.TrimSpace(value))
		}
		return strings.Join(parts, ",")
	}
	raw, _ := json.Marshal(answer)
	return string(raw)
}

func normalizeAnswer(answer types.JSONMap) types.JSONMap {
	if answer == nil {
		return types.JSONMap{}
	}
	return answer
}

func marshalJSON(value any) (types.JSON, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return types.JSON(data), nil
}

func normalizeConfidence(confidence float64) float64 {
	if confidence < 0 {
		return 0
	}
	if confidence > 1 {
		return 1
	}
	return confidence
}
