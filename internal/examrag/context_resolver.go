package examrag

import (
	"context"
	"fmt"
	"strings"

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
	VectorThreshold      float64
	KeywordThreshold     float64
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
	Bundle            *searchutil.ExamQuestionContextBundle
	Detail            *types.QuestionGroupDetail
	TenantID          uint64
	GroupID           string
	RetrievedChunkIDs []string
	CandidateChunkIDs []string
	SourceChunkIDs    []string
}

func NewExamQuestionContextResolver(cfg ExamQuestionContextResolverConfig) *ExamQuestionContextResolver {
	matchCount := cfg.MatchCount
	if matchCount <= 0 {
		matchCount = DefaultQuestionContextMatchCount
	}
	vectorThreshold := cfg.VectorThreshold
	if vectorThreshold <= 0 {
		vectorThreshold = DefaultQuestionContextVectorThreshold
	}
	keywordThreshold := cfg.KeywordThreshold
	if keywordThreshold <= 0 {
		keywordThreshold = DefaultQuestionContextKeywordThreshold
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
	if r == nil {
		return nil, fmt.Errorf("exam question context resolver is not configured")
	}
	if r.questionRepo == nil {
		return nil, fmt.Errorf("exam question repository is not configured")
	}

	query := strings.TrimSpace(req.Query)
	groupID := strings.TrimSpace(req.GroupID)
	chunkIDs := CleanIDs(req.ChunkIDs)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	tenantID, err := r.resolveTenantID(ctx, req.TenantID, req.KnowledgeBaseIDs)
	if err != nil {
		return nil, err
	}

	result := &ExamQuestionContextResolveResult{
		TenantID: tenantID,
		GroupID:  groupID,
	}
	if groupID == "" && len(chunkIDs) == 0 {
		chunkIDs, err = r.searchChunkIDsForQuery(ctx, query, req.KnowledgeBaseIDs, tenantID)
		if err != nil {
			return nil, err
		}
		result.RetrievedChunkIDs = append([]string{}, chunkIDs...)
	}
	result.CandidateChunkIDs = append([]string{}, chunkIDs...)
	if len(result.RetrievedChunkIDs) == 0 {
		result.RetrievedChunkIDs = append([]string{}, chunkIDs...)
	}

	detail, err := r.loadQuestionGroupDetail(ctx, tenantID, groupID, chunkIDs)
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.Group == nil {
		return result, nil
	}

	bundle := searchutil.BuildStructuredExamQuestionContextBundle(query, detail)
	if bundle == nil {
		return result, nil
	}
	result.Detail = detail
	result.Bundle = bundle
	result.GroupID = detail.Group.ID
	result.SourceChunkIDs = append([]string{}, bundle.SourceChunkIDs...)
	return result, nil
}

func (r *ExamQuestionContextResolver) EvalResolver(
	tenantID uint64,
	knowledgeBaseIDs []string,
) searchutil.ExamContextBundleResolver {
	return func(ctx context.Context, query string) (*searchutil.ExamQuestionContextBundle, []string, error) {
		result, err := r.Resolve(ctx, ExamQuestionContextResolveRequest{
			Query:            query,
			TenantID:         tenantID,
			KnowledgeBaseIDs: knowledgeBaseIDs,
		})
		if result == nil {
			return nil, nil, err
		}
		return result.Bundle, result.RetrievedChunkIDs, err
	}
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

func (r *ExamQuestionContextResolver) searchChunkIDsForQuery(
	ctx context.Context,
	query string,
	kbIDs []string,
	tenantID uint64,
) ([]string, error) {
	if r.knowledgeBaseService == nil {
		return nil, fmt.Errorf("knowledge base service is not configured")
	}

	targets, err := r.selectSearchTargets(kbIDs, tenantID)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no accessible knowledge bases available for exam question context search")
	}

	seen := make(map[string]bool)
	chunkIDs := make([]string, 0, r.matchCount)
	for _, target := range targets {
		if target == nil || strings.TrimSpace(target.KnowledgeBaseID) == "" {
			continue
		}
		params := types.SearchParams{
			QueryText:        query,
			MatchCount:       r.matchCount,
			VectorThreshold:  r.vectorThreshold,
			KeywordThreshold: r.keywordThreshold,
			KnowledgeIDs:     CleanIDs(target.KnowledgeIDs),
			TagIDs:           CleanIDs(target.TagIDs),
		}
		results, err := r.knowledgeBaseService.HybridSearch(ctx, target.KnowledgeBaseID, params)
		if err != nil {
			return nil, fmt.Errorf("failed to search knowledge base %s: %w", target.KnowledgeBaseID, err)
		}
		for _, result := range results {
			if result == nil {
				continue
			}
			chunkIDs = appendChunkID(chunkIDs, seen, result.ID)
			for _, subID := range result.SubChunkID {
				chunkIDs = appendChunkID(chunkIDs, seen, subID)
			}
		}
	}
	return chunkIDs, nil
}

func (r *ExamQuestionContextResolver) selectSearchTargets(
	kbIDs []string,
	tenantID uint64,
) (types.SearchTargets, error) {
	requested := CleanIDs(kbIDs)
	requestedSet := make(map[string]bool, len(requested))
	for _, kbID := range requested {
		requestedSet[kbID] = true
	}

	targets := make(types.SearchTargets, 0, len(r.searchTargets))
	for _, target := range r.searchTargets {
		if target == nil || strings.TrimSpace(target.KnowledgeBaseID) == "" {
			continue
		}
		if len(requestedSet) > 0 {
			if requestedSet[target.KnowledgeBaseID] {
				targets = append(targets, target)
				delete(requestedSet, target.KnowledgeBaseID)
			}
			continue
		}
		if tenantID != 0 && target.TenantID != 0 && target.TenantID != tenantID {
			continue
		}
		targets = append(targets, target)
	}
	if len(requestedSet) > 0 {
		for kbID := range requestedSet {
			return nil, fmt.Errorf("knowledge base %s is not accessible", kbID)
		}
	}
	return targets, nil
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

func appendChunkID(ids []string, seen map[string]bool, id string) []string {
	id = strings.TrimSpace(id)
	if id == "" || seen[id] {
		return ids
	}
	seen[id] = true
	return append(ids, id)
}
