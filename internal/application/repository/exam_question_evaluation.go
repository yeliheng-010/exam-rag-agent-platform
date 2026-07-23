package repository

import (
	"context"
	"strings"
	"unicode"

	"github.com/Tencent/WeKnora/internal/types"
)

// ListEvaluationQuestionGroupCandidates narrows re-chunked evaluation copies
// back to question banks built from the same source files.
func (r *examQuestionRepository) ListEvaluationQuestionGroupCandidates(
	ctx context.Context,
	tenantID uint64,
	knowledgeBaseIDs []string,
) ([]*types.QuestionGroupDetail, error) {
	kbIDs := uniqueNonEmptyStrings(knowledgeBaseIDs)
	if tenantID == 0 || len(kbIDs) == 0 {
		return []*types.QuestionGroupDetail{}, nil
	}

	var fileHashes []string
	if err := r.db.WithContext(ctx).Table("knowledges").
		Distinct("file_hash").
		Where("tenant_id = ? AND knowledge_base_id IN ? AND deleted_at IS NULL", tenantID, kbIDs).
		Where("file_hash IS NOT NULL AND file_hash <> ''").
		Pluck("file_hash", &fileHashes).Error; err != nil {
		return nil, err
	}
	fileHashes = uniqueNonEmptyStrings(fileHashes)
	if len(fileHashes) == 0 {
		return []*types.QuestionGroupDetail{}, nil
	}

	bankIDs, err := r.findEvaluationSourceQuestionBankIDs(ctx, tenantID, fileHashes)
	if err != nil {
		return nil, err
	}
	if len(bankIDs) == 0 {
		return []*types.QuestionGroupDetail{}, nil
	}
	return r.listEvaluationQuestionGroups(ctx, tenantID, bankIDs)
}

// ListEvaluationQuestionGroupCandidatesForSourcePhrases locates the source
// file containing retrieval gold, then returns the groups imported from it.
// File-level fallback is required because legacy groups can lack chunk refs.
func (r *examQuestionRepository) ListEvaluationQuestionGroupCandidatesForSourcePhrases(
	ctx context.Context,
	tenantID uint64,
	knowledgeBaseIDs []string,
	sourcePhrases []string,
) ([]*types.QuestionGroupDetail, error) {
	fileHashes, err := r.findEvaluationFileHashes(ctx, tenantID, knowledgeBaseIDs)
	if err != nil || len(fileHashes) == 0 {
		return []*types.QuestionGroupDetail{}, err
	}
	phrases := normalizeEvaluationSourcePhrases(sourcePhrases)
	if len(phrases) == 0 {
		return r.ListEvaluationQuestionGroupCandidates(ctx, tenantID, knowledgeBaseIDs)
	}
	sourceFileHashes, err := r.findEvaluationSourceFileHashes(ctx, tenantID, fileHashes, phrases)
	if err != nil || len(sourceFileHashes) == 0 {
		return []*types.QuestionGroupDetail{}, err
	}
	bankIDs, err := r.findEvaluationSourceQuestionBankIDs(ctx, tenantID, sourceFileHashes)
	if err != nil || len(bankIDs) == 0 {
		return []*types.QuestionGroupDetail{}, err
	}
	return r.listEvaluationQuestionGroups(ctx, tenantID, bankIDs)
}

func (r *examQuestionRepository) findEvaluationFileHashes(
	ctx context.Context,
	tenantID uint64,
	knowledgeBaseIDs []string,
) ([]string, error) {
	kbIDs := uniqueNonEmptyStrings(knowledgeBaseIDs)
	if tenantID == 0 || len(kbIDs) == 0 {
		return []string{}, nil
	}
	var fileHashes []string
	err := r.db.WithContext(ctx).Table("knowledges").Distinct("file_hash").
		Where("tenant_id = ? AND knowledge_base_id IN ? AND deleted_at IS NULL", tenantID, kbIDs).
		Where("file_hash IS NOT NULL AND file_hash <> ''").Pluck("file_hash", &fileHashes).Error
	return uniqueNonEmptyStrings(fileHashes), err
}

type evaluationSourceChunkRow struct {
	FileHash string
	Content  string
}

func (r *examQuestionRepository) findEvaluationSourceFileHashes(
	ctx context.Context,
	tenantID uint64,
	fileHashes []string,
	phrases []string,
) ([]string, error) {
	var rows []evaluationSourceChunkRow
	err := r.db.WithContext(ctx).Table("chunks AS source_chunks").
		Select("source_knowledge.file_hash AS file_hash, source_chunks.content AS content").
		Joins("JOIN knowledges AS source_knowledge ON source_knowledge.id = source_chunks.knowledge_id").
		Where("source_chunks.tenant_id = ? AND source_knowledge.tenant_id = ?", tenantID, tenantID).
		Where("source_knowledge.file_hash IN ?", fileHashes).
		Where("source_chunks.deleted_at IS NULL AND source_knowledge.deleted_at IS NULL").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	matchedHashes := make([]string, 0, len(rows))
	for _, row := range rows {
		if evaluationSourceContainsAnyPhrase(row.Content, phrases) {
			matchedHashes = append(matchedHashes, row.FileHash)
		}
	}
	return uniqueNonEmptyStrings(matchedHashes), nil
}

func normalizeEvaluationSourcePhrases(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if value = normalizeEvaluationSourceText(value); value != "" {
			normalized = append(normalized, value)
		}
	}
	return uniqueNonEmptyStrings(normalized)
}

func evaluationSourceContainsAnyPhrase(content string, phrases []string) bool {
	content = normalizeEvaluationSourceText(content)
	for _, phrase := range phrases {
		if strings.Contains(content, phrase) {
			return true
		}
	}
	return false
}

func normalizeEvaluationSourceText(value string) string {
	var builder strings.Builder
	for _, char := range strings.ToLower(value) {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func (r *examQuestionRepository) findEvaluationSourceQuestionBankIDs(
	ctx context.Context,
	tenantID uint64,
	fileHashes []string,
) ([]string, error) {
	var bankIDs []string
	err := r.db.WithContext(ctx).Table("question_groups AS qg").
		Distinct("qg.question_bank_id").
		Joins("JOIN questions AS q ON q.group_id = qg.id").
		Joins("JOIN question_chunk_refs AS refs ON refs.question_id = q.id").
		Joins("JOIN chunks AS source_chunks ON source_chunks.id = refs.chunk_id").
		Joins("JOIN knowledges AS source_knowledge ON source_knowledge.id = source_chunks.knowledge_id").
		Where("qg.tenant_id = ? AND q.tenant_id = ?", tenantID, tenantID).
		Where("source_chunks.tenant_id = ? AND source_knowledge.tenant_id = ?", tenantID, tenantID).
		Where("source_knowledge.file_hash IN ?", fileHashes).
		Where("qg.status <> ? AND q.status <> ?", "deleted", "deleted").
		Order("qg.question_bank_id ASC").
		Pluck("qg.question_bank_id", &bankIDs).Error
	return uniqueNonEmptyStrings(bankIDs), err
}

func (r *examQuestionRepository) listEvaluationQuestionGroups(
	ctx context.Context,
	tenantID uint64,
	bankIDs []string,
) ([]*types.QuestionGroupDetail, error) {
	var groups []*types.QuestionGroup
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND question_bank_id IN ? AND status <> ?", tenantID, bankIDs, "deleted").
		Order("question_bank_id ASC, sort_order ASC, created_at ASC").
		Find(&groups).Error
	if err != nil || len(groups) == 0 {
		if err != nil {
			return nil, err
		}
		return []*types.QuestionGroupDetail{}, nil
	}
	details := make([]*types.QuestionGroupDetail, 0, len(groups))
	for _, group := range groups {
		if group != nil && strings.TrimSpace(group.ID) != "" {
			details = append(details, &types.QuestionGroupDetail{Group: group})
		}
	}
	if err := r.loadQuestionGroupsChildren(ctx, details); err != nil {
		return nil, err
	}
	return details, nil
}
