package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
)

func (s *examAssignmentService) importAssignmentGroupToClassSpace(
	ctx context.Context,
	tenantID uint64,
	userID string,
	class *types.ExamClass,
	source *types.QuestionGroupDetail,
) (*types.QuestionGroupDetail, error) {
	if class == nil || source == nil || source.Group == nil || s.spaceService == nil {
		return nil, ErrExamPermissionDenied
	}
	canRead, err := s.spaceService.CanReadSpace(ctx, tenantID, userID, source.Group.SpaceID)
	if err != nil {
		if errors.Is(err, ErrExamNotFound) || errors.Is(err, ErrExamPermissionDenied) {
			return nil, ErrExamPermissionDenied
		}
		return nil, err
	}
	if !canRead {
		return nil, ErrExamPermissionDenied
	}

	sourceBank, err := s.questionRepo.GetQuestionBankByIDAndTenant(ctx, source.Group.QuestionBankID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrQuestionBankNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	now := time.Now()
	targetBank := cloneQuestionBankForClassImport(sourceBank, class.SpaceID, userID, now)
	if err := s.questionRepo.CreateQuestionBank(ctx, targetBank); err != nil {
		return nil, err
	}
	targetDetail := cloneQuestionGroupDetailForClassImport(source, targetBank.ID, class.SpaceID, userID, now)
	if err := s.questionRepo.CreateQuestionGroupDetail(ctx, targetDetail); err != nil {
		return nil, err
	}
	return targetDetail, nil
}

func cloneQuestionBankForClassImport(source *types.QuestionBank, spaceID string, userID string, now time.Time) *types.QuestionBank {
	bank := &types.QuestionBank{}
	if source != nil {
		cp := *source
		bank = &cp
	}
	bank.ID = uuid.New().String()
	bank.SpaceID = spaceID
	if strings.TrimSpace(bank.Name) == "" {
		bank.Name = "Imported question bank"
	}
	bank.SourceType = "assignment_import"
	bank.ReviewStatus = types.ExamReviewStatusPrivate
	bank.Status = "active"
	bank.CreatedByUserID = userID
	bank.CreatedAt = now
	bank.UpdatedAt = now
	return bank
}

func cloneQuestionGroupDetailForClassImport(source *types.QuestionGroupDetail, bankID string, spaceID string, userID string, now time.Time) *types.QuestionGroupDetail {
	groupID := uuid.New().String()
	group := cloneQuestionGroupForClassImport(source.Group, groupID, bankID, spaceID, userID, now)
	out := &types.QuestionGroupDetail{
		Group:     group,
		Assets:    make([]*types.QuestionGroupAsset, 0, len(source.Assets)),
		Questions: make([]*types.QuestionDetail, 0, len(source.Questions)),
	}
	for _, asset := range source.Assets {
		out.Assets = append(out.Assets, cloneQuestionGroupAssetForClassImport(asset, groupID, now))
	}
	for _, question := range source.Questions {
		out.Questions = append(out.Questions, cloneQuestionDetailForClassImport(question, bankID, groupID, userID, now))
	}
	return out
}

func cloneQuestionGroupForClassImport(source *types.QuestionGroup, groupID string, bankID string, spaceID string, userID string, now time.Time) *types.QuestionGroup {
	group := &types.QuestionGroup{}
	if source != nil {
		cp := *source
		group = &cp
	}
	group.ID = groupID
	group.SpaceID = spaceID
	group.QuestionBankID = bankID
	group.SubjectID = cloneStringPtr(group.SubjectID)
	group.AssetRefs = cloneJSON(group.AssetRefs)
	group.SourceChunkIDs = cloneJSON(group.SourceChunkIDs)
	group.SourceYear = cloneIntPtr(group.SourceYear)
	group.ReviewStatus = types.ExamReviewStatusPrivate
	group.Status = "active"
	group.CreatedByUserID = userID
	group.CreatedAt = now
	group.UpdatedAt = now
	return group
}

func cloneQuestionGroupAssetForClassImport(source *types.QuestionGroupAsset, groupID string, now time.Time) *types.QuestionGroupAsset {
	asset := &types.QuestionGroupAsset{}
	if source != nil {
		cp := *source
		asset = &cp
	}
	asset.ID = uuid.New().String()
	asset.GroupID = groupID
	asset.BBox = cloneJSONMap(asset.BBox)
	asset.Metadata = cloneJSONMap(asset.Metadata)
	asset.CreatedAt = now
	return asset
}

func cloneQuestionDetailForClassImport(source *types.QuestionDetail, bankID string, groupID string, userID string, now time.Time) *types.QuestionDetail {
	questionID := uuid.New().String()
	out := &types.QuestionDetail{
		Question:     cloneQuestionForClassImport(source.Question, questionID, bankID, groupID, userID, now),
		Options:      make([]*types.QuestionOption, 0, len(source.Options)),
		Answers:      make([]*types.QuestionAnswer, 0, len(source.Answers)),
		Explanations: make([]*types.QuestionExplanation, 0, len(source.Explanations)),
		ChunkRefs:    make([]*types.QuestionChunkRef, 0, len(source.ChunkRefs)),
	}
	for _, option := range source.Options {
		out.Options = append(out.Options, cloneQuestionOptionForClassImport(option, questionID))
	}
	for _, answer := range source.Answers {
		out.Answers = append(out.Answers, cloneQuestionAnswerForClassImport(answer, questionID, now))
	}
	for _, explanation := range source.Explanations {
		out.Explanations = append(out.Explanations, cloneQuestionExplanationForClassImport(explanation, questionID, now))
	}
	for _, ref := range source.ChunkRefs {
		out.ChunkRefs = append(out.ChunkRefs, cloneQuestionChunkRefForClassImport(ref, questionID, now))
	}
	return out
}

func cloneQuestionForClassImport(source *types.Question, questionID string, bankID string, groupID string, userID string, now time.Time) *types.Question {
	question := &types.Question{}
	if source != nil {
		cp := *source
		question = &cp
	}
	question.ID = questionID
	question.QuestionBankID = bankID
	question.GroupID = &groupID
	question.SubjectID = cloneStringPtr(question.SubjectID)
	question.QuestionTypeID = cloneStringPtr(question.QuestionTypeID)
	question.QuestionMetadata = cloneJSONMap(question.QuestionMetadata)
	question.SourceYear = cloneIntPtr(question.SourceYear)
	question.ReviewStatus = types.ExamReviewStatusPrivate
	question.Status = "active"
	question.CreatedByUserID = userID
	question.CreatedAt = now
	question.UpdatedAt = now
	return question
}

func cloneQuestionOptionForClassImport(source *types.QuestionOption, questionID string) *types.QuestionOption {
	option := &types.QuestionOption{}
	if source != nil {
		cp := *source
		option = &cp
	}
	option.ID = uuid.New().String()
	option.QuestionID = questionID
	return option
}

func cloneQuestionAnswerForClassImport(source *types.QuestionAnswer, questionID string, now time.Time) *types.QuestionAnswer {
	answer := &types.QuestionAnswer{}
	if source != nil {
		cp := *source
		answer = &cp
	}
	answer.ID = uuid.New().String()
	answer.QuestionID = questionID
	answer.CreatedAt = now
	return answer
}

func cloneQuestionExplanationForClassImport(source *types.QuestionExplanation, questionID string, now time.Time) *types.QuestionExplanation {
	explanation := &types.QuestionExplanation{}
	if source != nil {
		cp := *source
		explanation = &cp
	}
	explanation.ID = uuid.New().String()
	explanation.QuestionID = questionID
	explanation.CreatedAt = now
	return explanation
}

func cloneQuestionChunkRefForClassImport(source *types.QuestionChunkRef, questionID string, now time.Time) *types.QuestionChunkRef {
	ref := &types.QuestionChunkRef{}
	if source != nil {
		cp := *source
		ref = &cp
	}
	ref.QuestionID = questionID
	ref.CreatedAt = now
	return ref
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cp := *value
	return &cp
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	cp := *value
	return &cp
}

func cloneJSON(value types.JSON) types.JSON {
	if len(value) == 0 {
		return nil
	}
	cp := make([]byte, len(value))
	copy(cp, value)
	return types.JSON(cp)
}

func cloneJSONMap(value types.JSONMap) types.JSONMap {
	if value == nil {
		return nil
	}
	cp := make(types.JSONMap, len(value))
	for key, item := range value {
		cp[key] = item
	}
	return cp
}
