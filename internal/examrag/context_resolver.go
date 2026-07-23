package examrag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/searchutil"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const (
	DefaultQuestionContextMatchCount       = 8
	DefaultQuestionContextVectorThreshold  = 0.5
	DefaultQuestionContextKeywordThreshold = 0.3
)

type ExamQuestionContextResolverConfig struct {
	QuestionRepo         interfaces.ExamQuestionRepository
	KnowledgeBaseService interfaces.KnowledgeBaseService
	SearchTargets        types.SearchTargets
	MatchCount           int
	VectorThreshold      *float64
	KeywordThreshold     *float64
}

type ExamQuestionContextResolver struct {
	questionRepo         interfaces.ExamQuestionRepository
	knowledgeBaseService interfaces.KnowledgeBaseService
	searchTargets        types.SearchTargets
	matchCount           int
	vectorThreshold      float64
	keywordThreshold     float64
}

type ExamQuestionContextResolveRequest struct {
	Query            string
	TenantID         uint64
	KnowledgeBaseIDs []string
	ChunkIDs         []string
	GroupID          string
}

type ExamQuestionContextResolveResult struct {
	Bundle                *searchutil.ExamQuestionContextBundle
	Detail                *types.QuestionGroupDetail
	TenantID              uint64
	GroupID               string
	RetrievedChunkIDs     []string
	RetrievedContents     []string
	RetrievedItems        []ExamContextRankedItem
	CandidateChunkIDs     []string
	CandidateItems        []ExamContextRankedItem
	SourceChunkIDs        []string
	ContextSource         types.ExamRAGContextSource
	AssociationConfidence float64
	SearchTraces          []*types.SearchTrace
	DurationMS            int64
}

func NewExamQuestionContextResolver(cfg ExamQuestionContextResolverConfig) *ExamQuestionContextResolver {
	matchCount := cfg.MatchCount
	if matchCount <= 0 {
		matchCount = DefaultQuestionContextMatchCount
	}
	vectorThreshold := DefaultQuestionContextVectorThreshold
	if cfg.VectorThreshold != nil {
		vectorThreshold = *cfg.VectorThreshold
	}
	keywordThreshold := DefaultQuestionContextKeywordThreshold
	if cfg.KeywordThreshold != nil {
		keywordThreshold = *cfg.KeywordThreshold
	}

	return &ExamQuestionContextResolver{
		questionRepo:         cfg.QuestionRepo,
		knowledgeBaseService: cfg.KnowledgeBaseService,
		searchTargets:        cfg.SearchTargets,
		matchCount:           matchCount,
		vectorThreshold:      vectorThreshold,
		keywordThreshold:     keywordThreshold,
	}
}

func (r *ExamQuestionContextResolver) Resolve(
	ctx context.Context,
	req ExamQuestionContextResolveRequest,
) (*ExamQuestionContextResolveResult, error) {
	startedAt := time.Now()
	query, groupID, chunkIDs, tenantID, err := r.resolveRequestScope(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &ExamQuestionContextResolveResult{
		TenantID:      tenantID,
		ContextSource: types.ExamRAGContextSourceNone,
	}
	if groupID == "" && len(chunkIDs) == 0 {
		result.RetrievedItems, result.CandidateItems, result.SearchTraces, err = r.searchRankedItemsForKBs(
			ctx, query, req.KnowledgeBaseIDs, tenantID,
		)
		if err != nil {
			return nil, err
		}
		result.RetrievedChunkIDs = rankedItemChunkIDs(result.RetrievedItems)
		result.RetrievedContents = rankedItemContents(result.RetrievedItems)
		result.CandidateChunkIDs = rankedItemChunkIDs(result.CandidateItems)
		chunkIDs = append([]string{}, result.RetrievedChunkIDs...)
	}
	if len(result.CandidateChunkIDs) == 0 {
		result.CandidateChunkIDs = append([]string{}, chunkIDs...)
	}
	if len(result.RetrievedChunkIDs) == 0 {
		result.RetrievedChunkIDs = append([]string{}, chunkIDs...)
	}

	detail, err := r.loadQuestionGroupDetail(ctx, tenantID, groupID, chunkIDs)
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.Group == nil {
		result.DurationMS = time.Since(startedAt).Milliseconds()
		return result, nil
	}

	if !setResolvedQuestionGroup(
		result, query, detail, types.ExamRAGContextSourceStructuredQuestionGroup, 0,
	) {
		result.DurationMS = time.Since(startedAt).Milliseconds()
		return result, nil
	}
	result.DurationMS = time.Since(startedAt).Milliseconds()
	return result, nil
}

func (r *ExamQuestionContextResolver) resolveRequestScope(
	ctx context.Context,
	req ExamQuestionContextResolveRequest,
) (string, string, []string, uint64, error) {
	if r == nil {
		return "", "", nil, 0, fmt.Errorf("exam question context resolver is not configured")
	}
	if r.questionRepo == nil {
		return "", "", nil, 0, fmt.Errorf("exam question repository is not configured")
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return "", "", nil, 0, fmt.Errorf("query is required")
	}
	tenantID, err := r.resolveTenantID(ctx, req.TenantID, req.KnowledgeBaseIDs)
	if err != nil {
		return "", "", nil, 0, err
	}
	return query, strings.TrimSpace(req.GroupID), CleanIDs(req.ChunkIDs), tenantID, nil
}

func setResolvedQuestionGroup(
	result *ExamQuestionContextResolveResult,
	query string,
	detail *types.QuestionGroupDetail,
	source types.ExamRAGContextSource,
	confidence float64,
) bool {
	if result == nil || detail == nil || detail.Group == nil {
		return false
	}
	bundle := searchutil.BuildStructuredExamQuestionContextBundle(query, detail)
	if bundle == nil {
		return false
	}
	result.Detail = detail
	result.Bundle = bundle
	result.GroupID = detail.Group.ID
	result.SourceChunkIDs = append([]string{}, bundle.SourceChunkIDs...)
	result.ContextSource = source
	result.AssociationConfidence = confidence
	return true
}

func (r *ExamQuestionContextResolver) loadQuestionGroupDetail(
	ctx context.Context,
	tenantID uint64,
	groupID string,
	chunkIDs []string,
) (*types.QuestionGroupDetail, error) {
	if groupID != "" {
		return r.questionRepo.GetQuestionGroupDetailByIDAndTenant(ctx, tenantID, groupID)
	}
	if len(chunkIDs) == 0 {
		return nil, nil
	}
	return r.questionRepo.FindQuestionGroupDetailByChunkIDs(ctx, tenantID, chunkIDs)
}

func (r *ExamQuestionContextResolver) resolveTenantID(
	ctx context.Context,
	tenantID uint64,
	kbIDs []string,
) (uint64, error) {
	if tenantID != 0 {
		if err := r.ensureRequestedKBsInTenant(kbIDs, tenantID); err != nil {
			return 0, err
		}
		return tenantID, nil
	}
	requestedKBs := CleanIDs(kbIDs)
	if len(requestedKBs) > 0 {
		return r.tenantIDFromRequestedKBs(requestedKBs)
	}
	if ctxTenantID, ok := types.TenantIDFromContext(ctx); ok && ctxTenantID != 0 {
		return ctxTenantID, nil
	}
	return r.singleSearchTargetTenantID()
}

func (r *ExamQuestionContextResolver) ensureRequestedKBsInTenant(kbIDs []string, tenantID uint64) error {
	for _, kbID := range CleanIDs(kbIDs) {
		candidate := r.searchTargets.GetTenantIDForKB(kbID)
		if candidate == 0 {
			return fmt.Errorf("knowledge base %s is not accessible", kbID)
		}
		if candidate != tenantID {
			return fmt.Errorf("knowledge_base_ids must belong to the same tenant")
		}
	}
	return nil
}

func (r *ExamQuestionContextResolver) tenantIDFromRequestedKBs(kbIDs []string) (uint64, error) {
	var tenantID uint64
	for _, kbID := range kbIDs {
		candidate := r.searchTargets.GetTenantIDForKB(kbID)
		if candidate == 0 {
			return 0, fmt.Errorf("knowledge base %s is not accessible", kbID)
		}
		if tenantID != 0 && tenantID != candidate {
			return 0, fmt.Errorf("knowledge_base_ids must belong to the same tenant")
		}
		tenantID = candidate
	}
	return tenantID, nil
}

func (r *ExamQuestionContextResolver) singleSearchTargetTenantID() (uint64, error) {
	var tenantID uint64
	for _, target := range r.searchTargets {
		if target == nil || target.TenantID == 0 {
			continue
		}
		if tenantID != 0 && tenantID != target.TenantID {
			return 0, fmt.Errorf("knowledge_base_ids is required when search targets span multiple tenants")
		}
		tenantID = target.TenantID
	}
	if tenantID == 0 {
		return 0, fmt.Errorf("tenant scope is required")
	}
	return tenantID, nil
}

func CleanIDs(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
