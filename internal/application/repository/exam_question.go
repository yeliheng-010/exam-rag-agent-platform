package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrQuestionBankNotFound = errors.New("question bank not found")
var ErrQuestionNotFound = errors.New("question not found")

type examQuestionRepository struct {
	db *gorm.DB
}

func NewExamQuestionRepository(db *gorm.DB) interfaces.ExamQuestionRepository {
	return &examQuestionRepository{db: db}
}

func (r *examQuestionRepository) CreateQuestionBank(ctx context.Context, bank *types.QuestionBank) error {
	return r.db.WithContext(ctx).Create(bank).Error
}

func (r *examQuestionRepository) ListQuestionBanks(ctx context.Context, tenantID uint64, spaceIDs []string) ([]*types.QuestionBank, error) {
	if len(spaceIDs) == 0 {
		return []*types.QuestionBank{}, nil
	}
	var banks []*types.QuestionBank
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND space_id IN ? AND status = ?", tenantID, spaceIDs, "active").
		Order("created_at DESC").
		Find(&banks).Error
	return banks, err
}

func (r *examQuestionRepository) GetQuestionBankByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.QuestionBank, error) {
	var bank types.QuestionBank
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status = ?", id, tenantID, "active").
		First(&bank).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionBankNotFound
		}
		return nil, err
	}
	return &bank, nil
}

func (r *examQuestionRepository) CreateQuestionDetail(ctx context.Context, detail *types.QuestionDetail) error {
	if detail == nil || detail.Question == nil {
		return errors.New("question detail is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(detail.Question).Error; err != nil {
			return err
		}
		if len(detail.Options) > 0 {
			if err := tx.Create(&detail.Options).Error; err != nil {
				return err
			}
		}
		if len(detail.Answers) > 0 {
			if err := tx.Create(&detail.Answers).Error; err != nil {
				return err
			}
		}
		if len(detail.Explanations) > 0 {
			if err := tx.Create(&detail.Explanations).Error; err != nil {
				return err
			}
		}
		if len(detail.ChunkRefs) > 0 {
			if err := tx.Create(&detail.ChunkRefs).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *examQuestionRepository) CreateQuestionGroupDetail(ctx context.Context, detail *types.QuestionGroupDetail) error {
	if detail == nil || detail.Group == nil {
		return errors.New("question group detail is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(detail.Group).Error; err != nil {
			return err
		}
		if len(detail.Assets) > 0 {
			if err := tx.Create(&detail.Assets).Error; err != nil {
				return err
			}
		}
		for _, question := range detail.Questions {
			if question == nil || question.Question == nil {
				return errors.New("question detail is required")
			}
			if err := tx.Create(question.Question).Error; err != nil {
				return err
			}
			if len(question.Options) > 0 {
				if err := tx.Create(&question.Options).Error; err != nil {
					return err
				}
			}
			if len(question.Answers) > 0 {
				if err := tx.Create(&question.Answers).Error; err != nil {
					return err
				}
			}
			if len(question.Explanations) > 0 {
				if err := tx.Create(&question.Explanations).Error; err != nil {
					return err
				}
			}
			if len(question.ChunkRefs) > 0 {
				if err := tx.Create(&question.ChunkRefs).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *examQuestionRepository) ListQuestionDetailsByBank(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionDetail, error) {
	var questions []*types.Question
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND question_bank_id = ? AND status <> ?", tenantID, bankID, "deleted").
		Order("created_at DESC").
		Find(&questions).Error
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return []*types.QuestionDetail{}, nil
	}
	ids := make([]string, 0, len(questions))
	details := make([]*types.QuestionDetail, 0, len(questions))
	byID := make(map[string]*types.QuestionDetail, len(questions))
	for _, question := range questions {
		ids = append(ids, question.ID)
		detail := &types.QuestionDetail{Question: question}
		details = append(details, detail)
		byID[question.ID] = detail
	}
	if err := r.loadQuestionChildren(ctx, ids, byID); err != nil {
		return nil, err
	}
	return details, nil
}

func (r *examQuestionRepository) ListQuestionGroupDetailsByBank(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionGroupDetail, error) {
	groups, err := r.listStoredQuestionGroups(ctx, tenantID, bankID)
	if err != nil {
		return nil, err
	}
	details := make([]*types.QuestionGroupDetail, 0, len(groups))
	for _, group := range groups {
		details = append(details, &types.QuestionGroupDetail{Group: group})
	}
	if err := r.loadQuestionGroupsChildren(ctx, details); err != nil {
		return nil, err
	}
	legacy, err := r.listLegacyQuestionGroups(ctx, tenantID, bankID)
	if err != nil {
		return nil, err
	}
	return append(details, legacy...), nil
}

func (r *examQuestionRepository) GetQuestionDetailByIDAndTenant(ctx context.Context, tenantID uint64, questionID string) (*types.QuestionDetail, error) {
	var question types.Question
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status <> ?", questionID, tenantID, "deleted").
		First(&question).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	detail := &types.QuestionDetail{Question: &question}
	if err := r.loadQuestionChildren(ctx, []string{question.ID}, map[string]*types.QuestionDetail{question.ID: detail}); err != nil {
		return nil, err
	}
	return detail, nil
}

func (r *examQuestionRepository) GetQuestionGroupDetailByIDAndTenant(ctx context.Context, tenantID uint64, groupID string) (*types.QuestionGroupDetail, error) {
	var group types.QuestionGroup
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status <> ?", groupID, tenantID, "deleted").
		First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	detail := &types.QuestionGroupDetail{Group: &group}
	if err := r.loadQuestionGroupsChildren(ctx, []*types.QuestionGroupDetail{detail}); err != nil {
		return nil, err
	}
	return detail, nil
}

func (r *examQuestionRepository) FindQuestionGroupDetailByChunkIDs(ctx context.Context, tenantID uint64, chunkIDs []string) (*types.QuestionGroupDetail, error) {
	chunkIDs = uniqueNonEmptyStrings(chunkIDs)
	if len(chunkIDs) == 0 {
		return nil, nil
	}

	var row struct {
		GroupID string
	}
	err := r.db.WithContext(ctx).
		Table("question_chunk_refs AS refs").
		Select("questions.group_id").
		Joins("JOIN questions ON questions.id = refs.question_id").
		Joins("JOIN question_groups ON question_groups.id = questions.group_id").
		Where("questions.tenant_id = ?", tenantID).
		Where("refs.chunk_id IN ?", chunkIDs).
		Where("questions.group_id IS NOT NULL AND questions.group_id <> ''").
		Where("questions.status <> ? AND question_groups.status <> ?", "deleted", "deleted").
		Order("refs.confidence DESC, questions.order_in_group ASC, refs.created_at ASC").
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.GroupID == "" {
		groupID, err := r.findQuestionGroupIDBySourceChunkIDs(ctx, tenantID, chunkIDs)
		if err != nil || groupID == "" {
			return nil, err
		}
		row.GroupID = groupID
	}
	return r.GetQuestionGroupDetailByIDAndTenant(ctx, tenantID, row.GroupID)
}

func (r *examQuestionRepository) findQuestionGroupIDBySourceChunkIDs(
	ctx context.Context,
	tenantID uint64,
	chunkIDs []string,
) (string, error) {
	whereSQL, args := sourceChunkIDsContainsAnySQL(r.db.Dialector.Name(), chunkIDs)
	if whereSQL == "" {
		return "", nil
	}

	var row struct {
		GroupID string
	}
	err := r.db.WithContext(ctx).
		Table("question_groups").
		Select("id AS group_id").
		Where("tenant_id = ?", tenantID).
		Where("status <> ?", "deleted").
		Where(whereSQL, args...).
		Order("sort_order ASC, created_at DESC").
		Limit(1).
		Scan(&row).Error
	return row.GroupID, err
}

func sourceChunkIDsContainsAnySQL(dialect string, chunkIDs []string) (string, []interface{}) {
	chunkIDs = uniqueNonEmptyStrings(chunkIDs)
	if len(chunkIDs) == 0 {
		return "", nil
	}

	clauses := make([]string, 0, len(chunkIDs))
	args := make([]interface{}, 0, len(chunkIDs))
	for _, chunkID := range chunkIDs {
		if dialect == "postgres" {
			payload, err := json.Marshal([]string{chunkID})
			if err != nil {
				continue
			}
			clauses = append(clauses, "source_chunk_ids @> ?::jsonb")
			args = append(args, string(payload))
			continue
		}
		clauses = append(clauses, "source_chunk_ids LIKE ?")
		args = append(args, `%"`+chunkID+`"%`)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func (r *examQuestionRepository) ListQuestionGroupPracticeSummaries(
	ctx context.Context,
	tenantID uint64,
	spaceIDs []string,
	filter types.ListPracticeQuestionGroupsFilter,
) ([]*types.QuestionGroupPracticeSummary, error) {
	if len(spaceIDs) == 0 {
		return []*types.QuestionGroupPracticeSummary{}, nil
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 24
	}

	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND space_id IN ? AND status = ?", tenantID, spaceIDs, "active")
	if filter.DomainID != "" {
		query = query.Where("domain_id = ?", filter.DomainID)
	}
	if filter.SubjectID != "" {
		query = query.Where("subject_id = ?", filter.SubjectID)
	}
	if filter.SpaceID != "" {
		query = query.Where("space_id = ?", filter.SpaceID)
	}

	var groups []*types.QuestionGroup
	if err := query.Order("created_at DESC").Limit(limit).Find(&groups).Error; err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []*types.QuestionGroupPracticeSummary{}, nil
	}

	bankIDs := make([]string, 0, len(groups))
	groupIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		if group == nil {
			continue
		}
		bankIDs = append(bankIDs, group.QuestionBankID)
		groupIDs = append(groupIDs, group.ID)
	}

	bankNames, err := r.questionBankNames(ctx, tenantID, bankIDs)
	if err != nil {
		return nil, err
	}
	counts, err := r.questionCountsByGroup(ctx, tenantID, groupIDs)
	if err != nil {
		return nil, err
	}
	assetsByGroup, err := r.questionGroupAssetsByGroup(ctx, tenantID, groupIDs)
	if err != nil {
		return nil, err
	}

	out := make([]*types.QuestionGroupPracticeSummary, 0, len(groups))
	for _, group := range groups {
		if group == nil {
			continue
		}
		out = append(out, &types.QuestionGroupPracticeSummary{
			Group:         group,
			BankName:      bankNames[group.QuestionBankID],
			QuestionCount: counts[group.ID],
			Assets:        assetsByGroup[group.ID],
		})
	}
	return out, nil
}

func (r *examQuestionRepository) listStoredQuestionGroups(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionGroup, error) {
	var groups []*types.QuestionGroup
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND question_bank_id = ? AND status <> ?", tenantID, bankID, "deleted").
		Order("sort_order ASC, created_at DESC").
		Find(&groups).Error
	return groups, err
}

func (r *examQuestionRepository) listLegacyQuestionGroups(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionGroupDetail, error) {
	questions, err := r.listQuestionsByGroupState(ctx, tenantID, bankID, false)
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return []*types.QuestionGroupDetail{}, nil
	}
	ids := make([]string, 0, len(questions))
	byID := make(map[string]*types.QuestionDetail, len(questions))
	out := make([]*types.QuestionGroupDetail, 0, len(questions))
	for _, question := range questions {
		ids = append(ids, question.ID)
		detail := &types.QuestionDetail{Question: question}
		byID[question.ID] = detail
		out = append(out, &types.QuestionGroupDetail{
			Group: &types.QuestionGroup{
				ID:             "legacy-" + question.ID,
				TenantID:       question.TenantID,
				QuestionBankID: question.QuestionBankID,
				DomainID:       question.DomainID,
				SubjectID:      question.SubjectID,
				GroupType:      "single_question",
				Title:          question.QuestionNo,
				MaterialText:   question.Stem,
				MaterialFormat: "plain_text",
				ReviewStatus:   question.ReviewStatus,
				Status:         question.Status,
				CreatedAt:      question.CreatedAt,
				UpdatedAt:      question.UpdatedAt,
			},
			Questions: []*types.QuestionDetail{detail},
		})
	}
	if err := r.loadQuestionChildren(ctx, ids, byID); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *examQuestionRepository) listQuestionsByGroupState(ctx context.Context, tenantID uint64, bankID string, grouped bool) ([]*types.Question, error) {
	var questions []*types.Question
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND question_bank_id = ? AND status <> ?", tenantID, bankID, "deleted")
	if grouped {
		query = query.Where("group_id IS NOT NULL AND group_id <> ''")
	} else {
		query = query.Where("group_id IS NULL OR group_id = ''")
	}
	err := query.Order("created_at DESC").Find(&questions).Error
	return questions, err
}

func (r *examQuestionRepository) questionBankNames(ctx context.Context, tenantID uint64, bankIDs []string) (map[string]string, error) {
	names := make(map[string]string, len(bankIDs))
	if len(bankIDs) == 0 {
		return names, nil
	}
	var banks []*types.QuestionBank
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ?", tenantID, bankIDs).
		Find(&banks).Error; err != nil {
		return nil, err
	}
	for _, bank := range banks {
		if bank != nil {
			names[bank.ID] = bank.Name
		}
	}
	return names, nil
}

func (r *examQuestionRepository) questionCountsByGroup(ctx context.Context, tenantID uint64, groupIDs []string) (map[string]int, error) {
	counts := make(map[string]int, len(groupIDs))
	if len(groupIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		GroupID string
		Count   int
	}
	if err := r.db.WithContext(ctx).
		Model(&types.Question{}).
		Select("group_id, COUNT(*) AS count").
		Where("tenant_id = ? AND group_id IN ? AND status <> ?", tenantID, groupIDs, "deleted").
		Group("group_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.GroupID] = row.Count
	}
	return counts, nil
}

func (r *examQuestionRepository) questionGroupAssetsByGroup(ctx context.Context, tenantID uint64, groupIDs []string) (map[string][]*types.QuestionGroupAsset, error) {
	assetsByGroup := make(map[string][]*types.QuestionGroupAsset, len(groupIDs))
	if len(groupIDs) == 0 {
		return assetsByGroup, nil
	}
	var assets []*types.QuestionGroupAsset
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND group_id IN ?", tenantID, groupIDs).
		Order("sort_order ASC").
		Find(&assets).Error; err != nil {
		return nil, err
	}
	for _, asset := range assets {
		if asset != nil {
			assetsByGroup[asset.GroupID] = append(assetsByGroup[asset.GroupID], asset)
		}
	}
	return assetsByGroup, nil
}

func (r *examQuestionRepository) loadQuestionGroupsChildren(ctx context.Context, details []*types.QuestionGroupDetail) error {
	if len(details) == 0 {
		return nil
	}
	groupIDs := make([]string, 0, len(details))
	byGroupID := make(map[string]*types.QuestionGroupDetail, len(details))
	for _, detail := range details {
		if detail == nil || detail.Group == nil {
			continue
		}
		groupIDs = append(groupIDs, detail.Group.ID)
		byGroupID[detail.Group.ID] = detail
	}
	if len(groupIDs) == 0 {
		return nil
	}
	var assets []*types.QuestionGroupAsset
	if err := r.db.WithContext(ctx).
		Where("group_id IN ?", groupIDs).
		Order("sort_order ASC").
		Find(&assets).Error; err != nil {
		return err
	}
	for _, asset := range assets {
		if detail := byGroupID[asset.GroupID]; detail != nil {
			detail.Assets = append(detail.Assets, asset)
		}
	}
	var questions []*types.Question
	if err := r.db.WithContext(ctx).
		Where("group_id IN ?", groupIDs).
		Order("order_in_group ASC, created_at ASC").
		Find(&questions).Error; err != nil {
		return err
	}
	questionIDs := make([]string, 0, len(questions))
	byQuestionID := make(map[string]*types.QuestionDetail, len(questions))
	for _, question := range questions {
		questionIDs = append(questionIDs, question.ID)
		qd := &types.QuestionDetail{Question: question}
		byQuestionID[question.ID] = qd
		if question.GroupID != nil {
			if detail := byGroupID[*question.GroupID]; detail != nil {
				detail.Questions = append(detail.Questions, qd)
			}
		}
	}
	if len(questionIDs) > 0 {
		return r.loadQuestionChildren(ctx, questionIDs, byQuestionID)
	}
	return nil
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func (r *examQuestionRepository) loadQuestionChildren(ctx context.Context, questionIDs []string, byID map[string]*types.QuestionDetail) error {
	var options []*types.QuestionOption
	if err := r.db.WithContext(ctx).
		Where("question_id IN ?", questionIDs).
		Order("sort_order ASC, option_key ASC").
		Find(&options).Error; err != nil {
		return err
	}
	for _, item := range options {
		if detail := byID[item.QuestionID]; detail != nil {
			detail.Options = append(detail.Options, item)
		}
	}

	var answers []*types.QuestionAnswer
	if err := r.db.WithContext(ctx).
		Where("question_id IN ?", questionIDs).
		Order("created_at ASC").
		Find(&answers).Error; err != nil {
		return err
	}
	for _, item := range answers {
		if detail := byID[item.QuestionID]; detail != nil {
			detail.Answers = append(detail.Answers, item)
		}
	}

	var explanations []*types.QuestionExplanation
	if err := r.db.WithContext(ctx).
		Where("question_id IN ?", questionIDs).
		Order("created_at ASC").
		Find(&explanations).Error; err != nil {
		return err
	}
	for _, item := range explanations {
		if detail := byID[item.QuestionID]; detail != nil {
			detail.Explanations = append(detail.Explanations, item)
		}
	}

	var refs []*types.QuestionChunkRef
	if err := r.db.WithContext(ctx).
		Where("question_id IN ?", questionIDs).
		Order("created_at ASC").
		Find(&refs).Error; err != nil {
		return err
	}
	for _, item := range refs {
		if detail := byID[item.QuestionID]; detail != nil {
			detail.ChunkRefs = append(detail.ChunkRefs, item)
		}
	}
	return nil
}
