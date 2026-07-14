package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/examrag"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type examRAGDiagnosticService struct {
	questionService interfaces.ExamQuestionService
	resourceService interfaces.ExamResourceService
	questionRepo    interfaces.ExamQuestionRepository
	kbService       interfaces.KnowledgeBaseService
}

func NewExamRAGDiagnosticService(
	questionService interfaces.ExamQuestionService,
	resourceService interfaces.ExamResourceService,
	questionRepo interfaces.ExamQuestionRepository,
	kbService interfaces.KnowledgeBaseService,
) interfaces.ExamRAGDiagnosticService {
	return &examRAGDiagnosticService{
		questionService: questionService,
		resourceService: resourceService,
		questionRepo:    questionRepo,
		kbService:       kbService,
	}
}

func (s *examRAGDiagnosticService) EvaluateQuestionBank(
	ctx context.Context,
	tenantID uint64,
	userID string,
	bankID string,
	req *types.RunExamRAGDiagnosticRequest,
) (*types.ExamRAGDiagnosticResult, error) {
	preparation, err := s.PrepareQuestionBank(ctx, tenantID, userID, bankID, req)
	if err != nil {
		return nil, err
	}
	targets, err := s.buildSearchTargets(ctx, tenantID, preparation.Request.KnowledgeBaseIDs)
	if err != nil {
		return nil, err
	}

	resolver := examrag.NewExamQuestionContextResolver(examrag.ExamQuestionContextResolverConfig{
		QuestionRepo:         s.questionRepo,
		KnowledgeBaseService: s.kbService,
		SearchTargets:        targets,
		MatchCount:           preparation.Request.MatchCount,
		VectorThreshold:      preparation.Request.VectorThreshold,
		KeywordThreshold:     preparation.Request.KeywordThreshold,
	})
	summary := resolver.EvaluateRetrieval(ctx, examrag.ExamQuestionContextEvalRequest{
		TenantID:         tenantID,
		KnowledgeBaseIDs: preparation.Request.KnowledgeBaseIDs,
		Cases:            toExamRAGDiagnosticEvalCases(preparation.Request.Cases),
	})
	return &types.ExamRAGDiagnosticResult{
		QuestionBank:     preparation.QuestionBank,
		KnowledgeBaseIDs: preparation.Request.KnowledgeBaseIDs,
		UsedDefaultCases: preparation.UsedDefaultCases,
		Summary:          toExamRAGDiagnosticSummary(summary),
		Cases:            preparation.Request.Cases,
	}, nil
}

func (s *examRAGDiagnosticService) PrepareQuestionBank(
	ctx context.Context,
	tenantID uint64,
	userID string,
	bankID string,
	req *types.RunExamRAGDiagnosticRequest,
) (*types.ExamRAGDiagnosticPreparation, error) {
	if req == nil {
		req = &types.RunExamRAGDiagnosticRequest{}
	}
	bank, err := s.questionService.GetQuestionBank(ctx, tenantID, userID, strings.TrimSpace(bankID))
	if err != nil {
		return nil, err
	}
	kbIDs, err := s.resolveKnowledgeBaseIDs(ctx, tenantID, userID, bank, req.KnowledgeBaseIDs)
	if err != nil {
		return nil, err
	}
	cases, usedDefaultCases, err := buildExamRAGDiagnosticCases(req.Cases, bank)
	if err != nil {
		return nil, err
	}
	if _, err := s.buildSearchTargets(ctx, tenantID, kbIDs); err != nil {
		return nil, err
	}
	normalized, err := normalizeExamRAGDiagnosticRequest(req, kbIDs, cases)
	if err != nil {
		return nil, err
	}
	return &types.ExamRAGDiagnosticPreparation{
		QuestionBank: bank, Request: normalized, UsedDefaultCases: usedDefaultCases,
	}, nil
}

func normalizeExamRAGDiagnosticRequest(
	req *types.RunExamRAGDiagnosticRequest,
	kbIDs []string,
	cases []types.ExamRAGDiagnosticCase,
) (types.RunExamRAGDiagnosticRequest, error) {
	matchCount := req.MatchCount
	if matchCount == 0 {
		matchCount = examrag.DefaultQuestionContextMatchCount
	}
	if matchCount < 1 || matchCount > 50 {
		return types.RunExamRAGDiagnosticRequest{}, ErrExamInvalidRequest
	}
	vector, err := normalizeExamRAGThreshold(req.VectorThreshold, examrag.DefaultQuestionContextVectorThreshold)
	if err != nil {
		return types.RunExamRAGDiagnosticRequest{}, err
	}
	keyword, err := normalizeExamRAGThreshold(req.KeywordThreshold, examrag.DefaultQuestionContextKeywordThreshold)
	if err != nil {
		return types.RunExamRAGDiagnosticRequest{}, err
	}
	return types.RunExamRAGDiagnosticRequest{
		KnowledgeBaseIDs: kbIDs, Cases: cases, MatchCount: matchCount,
		VectorThreshold: &vector, KeywordThreshold: &keyword,
	}, nil
}

func normalizeExamRAGThreshold(value *float64, fallback float64) (float64, error) {
	if value == nil {
		return fallback, nil
	}
	if *value < 0 || *value > 1 {
		return 0, ErrExamInvalidRequest
	}
	return *value, nil
}

func (s *examRAGDiagnosticService) resolveKnowledgeBaseIDs(
	ctx context.Context,
	tenantID uint64,
	userID string,
	bank *types.QuestionBank,
	requested []string,
) ([]string, error) {
	kbIDs := examrag.CleanIDs(requested)
	if len(kbIDs) == 0 {
		resolved, err := s.listQuestionBankSpaceKnowledgeBaseIDs(ctx, tenantID, userID, bank)
		if err != nil {
			return nil, err
		}
		kbIDs = resolved
	}
	if len(kbIDs) == 0 {
		return nil, ErrExamInvalidRequest
	}
	for _, kbID := range kbIDs {
		ok, err := s.resourceService.CanReadKnowledgeBase(ctx, tenantID, userID, kbID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrExamPermissionDenied
		}
	}
	return kbIDs, nil
}

func (s *examRAGDiagnosticService) listQuestionBankSpaceKnowledgeBaseIDs(
	ctx context.Context,
	tenantID uint64,
	userID string,
	bank *types.QuestionBank,
) ([]string, error) {
	filter := types.ListExamResourcesFilter{
		ResourceType: types.ExamResourceTypeKnowledgeBase,
		SpaceID:      bank.SpaceID,
		DomainID:     bank.DomainID,
	}
	if bank.SubjectID != nil {
		filter.SubjectID = strings.TrimSpace(*bank.SubjectID)
	}
	resources, err := s.resourceService.ListResources(ctx, tenantID, userID, filter)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(resources))
	for _, resource := range resources {
		if resource != nil {
			ids = append(ids, resource.ResourceID)
		}
	}
	return examrag.CleanIDs(ids), nil
}

func (s *examRAGDiagnosticService) buildSearchTargets(
	ctx context.Context,
	tenantID uint64,
	kbIDs []string,
) (types.SearchTargets, error) {
	targets := make(types.SearchTargets, 0, len(kbIDs))
	for _, kbID := range kbIDs {
		kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
		if err != nil {
			if errors.Is(err, repository.ErrKnowledgeBaseNotFound) {
				return nil, ErrExamNotFound
			}
			return nil, err
		}
		if kb == nil || kb.TenantID != tenantID {
			return nil, ErrExamPermissionDenied
		}
		targets = append(targets, &types.SearchTarget{
			Type:            types.SearchTargetTypeKnowledgeBase,
			KnowledgeBaseID: kb.ID,
			TenantID:        kb.TenantID,
		})
	}
	return targets, nil
}

func buildExamRAGDiagnosticCases(raw []types.ExamRAGDiagnosticCase, bank *types.QuestionBank) ([]types.ExamRAGDiagnosticCase, bool, error) {
	usedDefault := false
	if len(raw) == 0 {
		raw = defaultExamRAGDiagnosticCasesForBank(bank)
		usedDefault = true
	}
	cases := make([]types.ExamRAGDiagnosticCase, 0, len(raw))
	for index, item := range raw {
		query := strings.TrimSpace(item.Query)
		if query == "" {
			return nil, false, ErrExamInvalidRequest
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = fmt.Sprintf("case_%d", index+1)
		}
		cases = append(cases, types.ExamRAGDiagnosticCase{
			Name:             name,
			Query:            query,
			RequiredPhrases:  examrag.CleanIDs(item.RequiredPhrases),
			ExpectedChunkIDs: examrag.CleanIDs(item.ExpectedChunkIDs),
		})
	}
	return cases, usedDefault, nil
}

func defaultExamRAGDiagnosticCases() []types.ExamRAGDiagnosticCase {
	return []types.ExamRAGDiagnosticCase{
		{
			Name:            "gaokao_english_reading_a_full_context",
			Query:           "第一篇阅读的原文、题目选项、答案和解析是什么？",
			RequiredPhrases: []string{"SoFi Stadium Events This Month", "Upcoming Football Events", "Nearby Hotels", "Parking"},
		},
		{
			Name:            "gaokao_english_reading_a_q21",
			Query:           "第一篇阅读第21题的选项、答案和解析是什么？",
			RequiredPhrases: []string{"21.", "Los Angeles Rams", "Answer: B", "Explanation:"},
		},
		{
			Name:            "gaokao_english_reading_a_q22",
			Query:           "第一篇阅读第22题的选项、答案和解析是什么？",
			RequiredPhrases: []string{"22.", "Sonder", "Answer: A", "Explanation:"},
		},
		{
			Name:            "gaokao_english_reading_a_q23",
			Query:           "第一篇阅读第23题的选项、答案和解析是什么？",
			RequiredPhrases: []string{"23.", "parking pass", "Answer: C", "Explanation:"},
		},
	}
}

func toExamRAGDiagnosticEvalCases(cases []types.ExamRAGDiagnosticCase) []examrag.ExamContextRetrievalEvalCase {
	out := make([]examrag.ExamContextRetrievalEvalCase, 0, len(cases))
	for _, item := range cases {
		out = append(out, examrag.ExamContextRetrievalEvalCase{
			Name:             item.Name,
			Query:            item.Query,
			RequiredPhrases:  item.RequiredPhrases,
			ExpectedChunkIDs: item.ExpectedChunkIDs,
		})
	}
	return out
}

func toExamRAGDiagnosticSummary(summary examrag.ExamContextRetrievalEvalSummary) types.ExamRAGDiagnosticSummary {
	results := make([]types.ExamRAGDiagnosticResultItem, 0, len(summary.Results))
	for _, item := range summary.Results {
		results = append(results, types.ExamRAGDiagnosticResultItem{
			Name:              item.Name,
			Query:             item.Query,
			Passed:            item.Passed,
			RetrievalPassed:   item.RetrievalPassed,
			AnswerPassed:      item.AnswerPassed,
			RetrievalScore:    item.RetrievalScore,
			AnswerScore:       item.AnswerScore,
			MatchedChunkIDs:   copyDiagnosticStrings(item.MatchedChunkIDs),
			MissingChunkIDs:   copyDiagnosticStrings(item.MissingChunkIDs),
			RetrievedChunkIDs: copyDiagnosticStrings(item.RetrievedChunkIDs),
			MatchedPhrases:    copyDiagnosticStrings(item.MatchedPhrases),
			MissingPhrases:    copyDiagnosticStrings(item.MissingPhrases),
			ContextLabel:      item.ContextLabel,
			SourceChunkIDs:    copyDiagnosticStrings(item.SourceChunkIDs),
			CandidateChunkIDs: copyDiagnosticStrings(item.CandidateChunkIDs),
			ContextSource:     item.ContextSource,
			GroupID:           item.GroupID,
			SearchTraces:      append([]*types.SearchTrace{}, item.SearchTraces...),
			DurationMS:        item.DurationMS,
			ReciprocalRank:    item.ReciprocalRank,
			Error:             item.Error,
		})
	}
	return types.ExamRAGDiagnosticSummary{
		Total:                    summary.Total,
		Passed:                   summary.Passed,
		RetrievalPassed:          summary.RetrievalPassed,
		AnswerPassed:             summary.AnswerPassed,
		HitRate:                  summary.HitRate,
		RetrievalHitRate:         summary.RetrievalHitRate,
		AnswerHitRate:            summary.AnswerHitRate,
		RecallAtK:                summary.RecallAtK,
		MeanReciprocalRank:       summary.MeanReciprocalRank,
		RankedCaseCount:          summary.RankedCaseCount,
		StructuredResolutionRate: summary.StructuredResolutionRate,
		StructuredResolved:       summary.StructuredResolved,
		AverageDurationMS:        summary.AverageDurationMS,
		FailedCaseCount:          summary.FailedCaseCount,
		Results:                  results,
	}
}

func copyDiagnosticStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string{}, values...)
}
