package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
)

func buildQuestionGroupDrafts(task *types.ExamStructuringTask, material *types.ExamMaterial, candidates []*types.ExamQuestionGroupDraftCandidate, rawOutput string) ([]*types.ExamQuestionGroupDraft, error) {
	if len(candidates) == 0 {
		return nil, errors.New("model returned no question group drafts")
	}
	now := time.Now()
	out := make([]*types.ExamQuestionGroupDraft, 0, len(candidates))
	for _, candidate := range candidates {
		draft, err := buildQuestionGroupDraftFromCandidate(task, material, candidate, rawOutput, now)
		if err != nil {
			return nil, err
		}
		out = append(out, draft)
	}
	return out, nil
}

func buildQuestionGroupDraftFromCandidate(task *types.ExamStructuringTask, material *types.ExamMaterial, candidate *types.ExamQuestionGroupDraftCandidate, rawOutput string, now time.Time) (*types.ExamQuestionGroupDraft, error) {
	if task == nil || material == nil || candidate == nil || len(candidate.Questions) == 0 {
		return nil, errors.New("question group draft questions are required")
	}
	questionsJSON, err := marshalJSON(candidate.Questions)
	if err != nil {
		return nil, err
	}
	assetsJSON, err := marshalJSON(candidate.Assets)
	if err != nil {
		return nil, err
	}
	chunkJSON, err := marshalJSON(candidate.SourceChunkIDs)
	if err != nil {
		return nil, err
	}
	if candidate.RawModelOutput != "" {
		rawOutput = candidate.RawModelOutput
	}
	groupType := defaultString(candidate.GroupType, "single_question")
	return &types.ExamQuestionGroupDraft{
		ID:             uuid.New().String(),
		TenantID:       task.TenantID,
		SpaceID:        task.SpaceID,
		TaskID:         task.ID,
		MaterialID:     material.ID,
		QuestionBankID: task.QuestionBankID,
		DomainID:       material.DomainID,
		SubjectID:      material.SubjectID,
		GroupType:      groupType,
		Title:          strings.TrimSpace(candidate.Title),
		MaterialText:   strings.TrimSpace(candidate.MaterialText),
		MaterialFormat: defaultString(candidate.MaterialFormat, "plain_text"),
		QuestionsJSON:  questionsJSON,
		AssetsJSON:     assetsJSON,
		SourceChunkIDs: chunkJSON,
		StrategyCode:   strings.TrimSpace(candidate.StrategyCode),
		Confidence:     normalizeConfidence(candidate.Confidence),
		Status:         types.ExamQuestionGroupDraftStatusPendingReview,
		RawModelOutput: rawOutput,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func applyQuestionGroupDraftUpdate(draft *types.ExamQuestionGroupDraft, req *types.UpdateExamQuestionGroupDraftRequest) error {
	questionsJSON, err := marshalJSON(req.Questions)
	if err != nil {
		return err
	}
	assetsJSON, err := marshalJSON(req.Assets)
	if err != nil {
		return err
	}
	chunkJSON, err := marshalJSON(req.SourceChunkIDs)
	if err != nil {
		return err
	}
	draft.GroupType = defaultString(req.GroupType, "single_question")
	draft.Title = strings.TrimSpace(req.Title)
	draft.MaterialText = strings.TrimSpace(req.MaterialText)
	draft.MaterialFormat = defaultString(req.MaterialFormat, "plain_text")
	draft.QuestionsJSON = questionsJSON
	draft.AssetsJSON = assetsJSON
	draft.SourceChunkIDs = chunkJSON
	draft.UpdatedAt = time.Now()
	return nil
}

func buildQuestionGroupDetailFromDraft(draft *types.ExamQuestionGroupDraft, userID string) (*types.QuestionGroupDetail, error) {
	questions, err := draftGroupQuestions(draft)
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, errors.New("question group draft questions are required")
	}
	assets, err := draftGroupAssets(draft)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	groupID := uuid.New().String()
	detail := &types.QuestionGroupDetail{
		Group:  buildQuestionGroupFromDraft(draft, groupID, userID, now),
		Assets: buildQuestionGroupAssets(groupID, draft.TenantID, assets, now),
	}
	for _, candidate := range questions {
		qd, err := buildQuestionDetailFromGroupCandidate(draft, groupID, candidate, userID, now)
		if err != nil {
			return nil, err
		}
		detail.Questions = append(detail.Questions, qd)
	}
	return detail, nil
}

func buildQuestionGroupFromDraft(draft *types.ExamQuestionGroupDraft, groupID string, userID string, now time.Time) *types.QuestionGroup {
	return &types.QuestionGroup{
		ID:              groupID,
		TenantID:        draft.TenantID,
		SpaceID:         draft.SpaceID,
		QuestionBankID:  draft.QuestionBankID,
		DomainID:        draft.DomainID,
		SubjectID:       draft.SubjectID,
		GroupType:       defaultString(draft.GroupType, "single_question"),
		Title:           strings.TrimSpace(draft.Title),
		MaterialText:    strings.TrimSpace(draft.MaterialText),
		MaterialFormat:  defaultString(draft.MaterialFormat, "plain_text"),
		AssetRefs:       draft.AssetsJSON,
		SourceChunkIDs:  draft.SourceChunkIDs,
		ReviewStatus:    types.ExamReviewStatusPrivate,
		Status:          "active",
		CreatedByUserID: userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func buildQuestionGroupAssets(groupID string, tenantID uint64, assets []types.ExamQuestionGroupDraftAssetCandidate, now time.Time) []*types.QuestionGroupAsset {
	out := make([]*types.QuestionGroupAsset, 0, len(assets))
	for i, asset := range assets {
		bbox := asset.BBox
		if bbox == nil {
			bbox = types.JSONMap{}
		}
		metadata := asset.Metadata
		if metadata == nil {
			metadata = types.JSONMap{}
		}
		sortOrder := asset.SortOrder
		if sortOrder <= 0 {
			sortOrder = i + 1
		}
		out = append(out, &types.QuestionGroupAsset{
			ID:            uuid.New().String(),
			TenantID:      tenantID,
			GroupID:       groupID,
			AssetType:     defaultString(asset.AssetType, "image"),
			StorageURI:    strings.TrimSpace(asset.StorageURI),
			AltText:       strings.TrimSpace(asset.AltText),
			SourceChunkID: strings.TrimSpace(asset.SourceChunkID),
			BBox:          bbox,
			Metadata:      metadata,
			SortOrder:     sortOrder,
			CreatedAt:     now,
		})
	}
	return out
}

func buildQuestionDetailFromGroupCandidate(draft *types.ExamQuestionGroupDraft, groupID string, candidate types.ExamQuestionGroupDraftQuestionCandidate, userID string, now time.Time) (*types.QuestionDetail, error) {
	if strings.TrimSpace(candidate.Stem) == "" {
		return nil, errors.New("question stem is required")
	}
	answer := draftAnswerText(candidate.Answer)
	if answer == "" {
		return nil, errors.New("question answer is required")
	}
	questionID := uuid.New().String()
	metadata := candidate.Metadata
	if metadata == nil {
		metadata = types.JSONMap{}
	}
	order := candidate.OrderInGroup
	if order <= 0 {
		order = 1
	}
	return &types.QuestionDetail{
		Question: &types.Question{
			ID:               questionID,
			TenantID:         draft.TenantID,
			QuestionBankID:   draft.QuestionBankID,
			DomainID:         draft.DomainID,
			SubjectID:        draft.SubjectID,
			GroupID:          &groupID,
			QuestionNo:       strings.TrimSpace(candidate.QuestionNo),
			OrderInGroup:     order,
			Stem:             strings.TrimSpace(candidate.Stem),
			QuestionMetadata: metadata,
			Difficulty:       defaultString(candidate.Difficulty, "unknown"),
			ReviewStatus:     types.ExamReviewStatusPrivate,
			Status:           "active",
			CreatedByUserID:  userID,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		Options:      buildQuestionOptions(questionID, candidate.Options),
		Answers:      buildQuestionAnswers(questionID, answer, now),
		Explanations: buildQuestionExplanations(questionID, candidate.Explanation, now),
		ChunkRefs:    buildQuestionChunkRefs(questionID, candidate.SourceChunkIDs, candidate.Confidence, now),
	}, nil
}

func draftGroupQuestions(draft *types.ExamQuestionGroupDraft) ([]types.ExamQuestionGroupDraftQuestionCandidate, error) {
	var questions []types.ExamQuestionGroupDraftQuestionCandidate
	if draft == nil || len(draft.QuestionsJSON) == 0 {
		return questions, nil
	}
	if err := json.Unmarshal(draft.QuestionsJSON, &questions); err != nil {
		return nil, err
	}
	return questions, nil
}

func draftGroupAssets(draft *types.ExamQuestionGroupDraft) ([]types.ExamQuestionGroupDraftAssetCandidate, error) {
	var assets []types.ExamQuestionGroupDraftAssetCandidate
	if draft == nil || len(draft.AssetsJSON) == 0 {
		return assets, nil
	}
	if err := json.Unmarshal(draft.AssetsJSON, &assets); err != nil {
		return nil, err
	}
	return assets, nil
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
