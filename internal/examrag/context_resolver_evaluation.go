package examrag

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/searchutil"
	"github.com/Tencent/WeKnora/internal/types"
)

type evaluationQuestionGroupCandidateRepository interface {
	ListEvaluationQuestionGroupCandidates(
		ctx context.Context,
		tenantID uint64,
		knowledgeBaseIDs []string,
	) ([]*types.QuestionGroupDetail, error)
}

type evaluationQuestionGroupSourceCandidateRepository interface {
	ListEvaluationQuestionGroupCandidatesForSourcePhrases(
		ctx context.Context,
		tenantID uint64,
		knowledgeBaseIDs []string,
		sourcePhrases []string,
	) ([]*types.QuestionGroupDetail, error)
}

func (r *ExamQuestionContextResolver) EvalResolver(
	tenantID uint64,
	knowledgeBaseIDs []string,
) searchutil.ExamContextBundleResolver {
	return func(ctx context.Context, query string, evaluationAnchors []string) (*searchutil.ExamContextResolution, error) {
		result, err := r.Resolve(ctx, ExamQuestionContextResolveRequest{
			Query: query, TenantID: tenantID, KnowledgeBaseIDs: knowledgeBaseIDs,
		})
		if err == nil && result != nil && result.Bundle == nil {
			fallbackStarted := time.Now()
			err = r.resolveEvaluationAnchorMatch(ctx, query, evaluationAnchors, knowledgeBaseIDs, result)
			result.DurationMS += time.Since(fallbackStarted).Milliseconds()
		}
		if result == nil {
			return nil, err
		}
		return evaluationResolution(result), err
	}
}

func evaluationResolution(result *ExamQuestionContextResolveResult) *searchutil.ExamContextResolution {
	return &searchutil.ExamContextResolution{
		Bundle:                result.Bundle,
		RetrievedChunkIDs:     append([]string{}, result.RetrievedChunkIDs...),
		RetrievedContents:     append([]string{}, result.RetrievedContents...),
		RetrievedItems:        cloneRankedItems(result.RetrievedItems),
		CandidateChunkIDs:     append([]string{}, result.CandidateChunkIDs...),
		CandidateItems:        cloneRankedItems(result.CandidateItems),
		SourceChunkIDs:        append([]string{}, result.SourceChunkIDs...),
		ContextSource:         result.ContextSource,
		GroupID:               result.GroupID,
		AssociationConfidence: result.AssociationConfidence,
		SearchTraces:          append([]*types.SearchTrace{}, result.SearchTraces...),
		DurationMS:            result.DurationMS,
	}
}

func (r *ExamQuestionContextResolver) resolveEvaluationAnchorMatch(
	ctx context.Context,
	query string,
	evaluationAnchors []string,
	knowledgeBaseIDs []string,
	result *ExamQuestionContextResolveResult,
) error {
	repo, ok := r.questionRepo.(evaluationQuestionGroupCandidateRepository)
	if !ok || result == nil || result.Bundle != nil {
		return nil
	}
	candidates, err := r.listEvaluationQuestionGroupCandidates(
		ctx, repo, result.TenantID, knowledgeBaseIDs, evaluationAnchors,
	)
	if err != nil {
		return err
	}
	detail, confidence := matchEvaluationQuestionGroupWithAnchors(
		query, evaluationAnchors, result.RetrievedContents, candidates,
	)
	if detail != nil {
		setResolvedQuestionGroup(
			result, query, detail, types.ExamRAGContextSourceEvaluationAnchorMatch, confidence,
		)
	}
	return nil
}

func (r *ExamQuestionContextResolver) listEvaluationQuestionGroupCandidates(
	ctx context.Context,
	repo evaluationQuestionGroupCandidateRepository,
	tenantID uint64,
	knowledgeBaseIDs []string,
	evaluationAnchors []string,
) ([]*types.QuestionGroupDetail, error) {
	if sourceRepo, ok := r.questionRepo.(evaluationQuestionGroupSourceCandidateRepository); ok &&
		len(evaluationAnchors) > 0 {
		return sourceRepo.ListEvaluationQuestionGroupCandidatesForSourcePhrases(
			ctx, tenantID, knowledgeBaseIDs, evaluationAnchors,
		)
	}
	return repo.ListEvaluationQuestionGroupCandidates(ctx, tenantID, knowledgeBaseIDs)
}
