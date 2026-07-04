package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

type examMaterialService struct {
	materialRepo    interfaces.ExamMaterialRepository
	spaceService    interfaces.ExamSpaceService
	domainRepo      interfaces.ExamDomainRepository
	kbReader        interfaces.ExamMaterialKnowledgeBaseReader
	knowledgeReader interfaces.ExamMaterialKnowledgeReader
	chunkReader     interfaces.ExamMaterialChunkReader
	questionService interfaces.ExamQuestionService
	resourceService interfaces.ExamResourceService
}

func NewExamMaterialService(
	materialRepo interfaces.ExamMaterialRepository,
	spaceService interfaces.ExamSpaceService,
	domainRepo interfaces.ExamDomainRepository,
	kbReader interfaces.ExamMaterialKnowledgeBaseReader,
	knowledgeReader interfaces.ExamMaterialKnowledgeReader,
	chunkReader interfaces.ExamMaterialChunkReader,
	questionService interfaces.ExamQuestionService,
	resourceService interfaces.ExamResourceService,
) interfaces.ExamMaterialService {
	return &examMaterialService{
		materialRepo:    materialRepo,
		spaceService:    spaceService,
		domainRepo:      domainRepo,
		kbReader:        kbReader,
		knowledgeReader: knowledgeReader,
		chunkReader:     chunkReader,
		questionService: questionService,
		resourceService: resourceService,
	}
}

func (s *examMaterialService) RegisterMaterial(ctx context.Context, tenantID uint64, userID string, req *types.RegisterExamMaterialRequest) (*types.ExamMaterialRegistrationResult, error) {
	if req == nil || strings.TrimSpace(userID) == "" {
		return nil, ErrExamInvalidRequest
	}
	spaceID := strings.TrimSpace(req.SpaceID)
	kbID := strings.TrimSpace(req.KnowledgeBaseID)
	knowledgeID := strings.TrimSpace(req.KnowledgeID)
	domainID := strings.TrimSpace(req.DomainID)
	if spaceID == "" || kbID == "" || knowledgeID == "" || domainID == "" {
		return nil, ErrExamInvalidRequest
	}

	space, err := s.spaceService.GetSpace(ctx, tenantID, userID, spaceID)
	if err != nil {
		return nil, err
	}
	canWrite, err := s.spaceService.CanWriteSpace(ctx, tenantID, userID, spaceID)
	if err != nil {
		return nil, err
	}
	if !canWrite {
		return nil, ErrExamPermissionDenied
	}

	kb, knowledge, err := s.validateKnowledge(ctx, tenantID, kbID, knowledgeID)
	if err != nil {
		return nil, err
	}
	if err := s.validateExamDomain(ctx, domainID, req.SubjectID); err != nil {
		return nil, err
	}

	materialType := normalizeMaterialType(req.MaterialType)
	if req.CreateTask && materialType != types.ExamMaterialTypeExamPaper {
		return nil, ErrExamInvalidRequest
	}
	if err := s.ensureKnowledgeBaseBound(ctx, tenantID, userID, kb.ID, spaceID, domainID, req.SubjectID, materialType); err != nil {
		return nil, err
	}
	now := time.Now()
	materialID := uuid.New().String()
	createdAt := now
	createdBy := userID
	if existing, err := s.materialRepo.GetMaterialByKnowledge(ctx, tenantID, knowledgeID); err == nil && existing != nil {
		materialID = existing.ID
		createdAt = existing.CreatedAt
		createdBy = existing.CreatedByUserID
	} else if err != nil && !errors.Is(err, repository.ErrExamMaterialNotFound) {
		return nil, err
	}

	reviewStatus := types.ExamReviewStatusPrivate
	if space.SpaceType == types.ExamSpaceTypePublic {
		reviewStatus = types.ExamReviewStatusPending
	}
	material := &types.ExamMaterial{
		ID:              materialID,
		TenantID:        tenantID,
		SpaceID:         spaceID,
		KnowledgeBaseID: kb.ID,
		KnowledgeID:     knowledge.ID,
		DomainID:        domainID,
		SubjectID:       normalizeOptionalID(req.SubjectID),
		MaterialType:    materialType,
		Title:           resolveExamMaterialTitle(req.Title, knowledge),
		Description:     strings.TrimSpace(req.Description),
		SourceYear:      req.SourceYear,
		SourceRegion:    strings.TrimSpace(req.SourceRegion),
		PaperType:       strings.TrimSpace(req.PaperType),
		IngestStatus:    mapExamMaterialIngestStatus(knowledge.ParseStatus),
		ReviewStatus:    reviewStatus,
		Status:          "active",
		CreatedByUserID: createdBy,
		CreatedAt:       createdAt,
		UpdatedAt:       now,
	}
	if err := s.materialRepo.UpsertMaterial(ctx, material); err != nil {
		return nil, err
	}
	saved, err := s.materialRepo.GetMaterialByKnowledge(ctx, tenantID, knowledgeID)
	if err != nil {
		return nil, err
	}

	result := &types.ExamMaterialRegistrationResult{Material: saved}
	if req.CreateTask {
		task, err := s.CreateStructuringTask(ctx, tenantID, userID, saved.ID, &types.CreateExamStructuringTaskRequest{
			QuestionBankID: strings.TrimSpace(req.QuestionBankID),
			Strategy:       types.ExamStructuringStrategyManualReview,
		})
		if err != nil {
			return nil, err
		}
		result.StructuringTask = task
	}
	return result, nil
}

func (s *examMaterialService) ListMaterials(ctx context.Context, tenantID uint64, userID string, filter types.ListExamMaterialsFilter) ([]*types.ExamMaterial, error) {
	filter.SpaceID = strings.TrimSpace(filter.SpaceID)
	filter.DomainID = strings.TrimSpace(filter.DomainID)
	filter.SubjectID = strings.TrimSpace(filter.SubjectID)
	if filter.MaterialType != "" {
		filter.MaterialType = normalizeMaterialType(filter.MaterialType)
	}
	spaceIDs, err := s.resolveReadableSpaceIDs(ctx, tenantID, userID, filter.SpaceID)
	if err != nil {
		return nil, err
	}
	return s.materialRepo.ListMaterials(ctx, tenantID, filter, spaceIDs)
}

func (s *examMaterialService) CreateStructuringTask(ctx context.Context, tenantID uint64, userID string, materialID string, req *types.CreateExamStructuringTaskRequest) (*types.ExamStructuringTask, error) {
	materialID = strings.TrimSpace(materialID)
	if materialID == "" || strings.TrimSpace(userID) == "" {
		return nil, ErrExamInvalidRequest
	}
	if req == nil {
		req = &types.CreateExamStructuringTaskRequest{}
	}
	material, err := s.materialRepo.GetMaterialByIDAndTenant(ctx, materialID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamMaterialNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	if material.MaterialType != types.ExamMaterialTypeExamPaper {
		return nil, ErrExamInvalidRequest
	}
	canWrite, err := s.spaceService.CanWriteSpace(ctx, tenantID, userID, material.SpaceID)
	if err != nil {
		return nil, err
	}
	if !canWrite {
		return nil, ErrExamPermissionDenied
	}

	bank, err := s.resolveQuestionBank(ctx, tenantID, userID, material, strings.TrimSpace(req.QuestionBankID))
	if err != nil {
		return nil, err
	}
	chunkCount, err := s.countKnowledgeChunks(ctx, material.KnowledgeID)
	if err != nil {
		return nil, err
	}
	status, message := resolveStructuringTaskStatus(material.IngestStatus, chunkCount)
	strategy := req.Strategy
	if strategy == "" {
		strategy = types.ExamStructuringStrategyManualReview
	}
	if strategy != types.ExamStructuringStrategyManualReview {
		return nil, ErrExamInvalidRequest
	}

	now := time.Now()
	task := &types.ExamStructuringTask{
		ID:                      uuid.New().String(),
		TenantID:                tenantID,
		MaterialID:              material.ID,
		SpaceID:                 material.SpaceID,
		QuestionBankID:          bank.ID,
		Status:                  status,
		Strategy:                strategy,
		SourceChunkCount:        chunkCount,
		StructuredQuestionCount: 0,
		ReviewRequired:          true,
		ErrorMessage:            message,
		CreatedByUserID:         userID,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := s.materialRepo.CreateStructuringTask(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *examMaterialService) ListStructuringTasks(ctx context.Context, tenantID uint64, userID string, filter types.ListExamStructuringTasksFilter) ([]*types.ExamStructuringTask, error) {
	filter.SpaceID = strings.TrimSpace(filter.SpaceID)
	filter.MaterialID = strings.TrimSpace(filter.MaterialID)
	spaceIDs, err := s.resolveReadableSpaceIDs(ctx, tenantID, userID, filter.SpaceID)
	if err != nil {
		return nil, err
	}
	return s.materialRepo.ListStructuringTasks(ctx, tenantID, filter, spaceIDs)
}

func (s *examMaterialService) validateKnowledge(ctx context.Context, tenantID uint64, kbID string, knowledgeID string) (*types.KnowledgeBase, *types.Knowledge, error) {
	kb, err := s.kbReader.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		if errors.Is(err, repository.ErrKnowledgeBaseNotFound) {
			return nil, nil, ErrExamNotFound
		}
		return nil, nil, err
	}
	if kb == nil || kb.TenantID != tenantID {
		return nil, nil, ErrExamPermissionDenied
	}
	knowledge, err := s.knowledgeReader.GetKnowledgeByID(ctx, knowledgeID)
	if err != nil {
		if errors.Is(err, repository.ErrKnowledgeNotFound) {
			return nil, nil, ErrExamNotFound
		}
		return nil, nil, err
	}
	if knowledge == nil || knowledge.TenantID != tenantID {
		return nil, nil, ErrExamPermissionDenied
	}
	if knowledge.KnowledgeBaseID != kb.ID {
		return nil, nil, ErrExamInvalidRequest
	}
	return kb, knowledge, nil
}

func (s *examMaterialService) ensureKnowledgeBaseBound(
	ctx context.Context,
	tenantID uint64,
	userID string,
	kbID string,
	spaceID string,
	domainID string,
	subjectID *string,
	materialType types.ExamMaterialType,
) error {
	if s.resourceService == nil {
		return nil
	}
	_, err := s.resourceService.BindKnowledgeBase(ctx, tenantID, userID, kbID, &types.BindKnowledgeBaseResourceRequest{
		SpaceID:      spaceID,
		DomainID:     domainID,
		SubjectID:    normalizeOptionalID(subjectID),
		MaterialType: materialType,
	})
	return err
}

func (s *examMaterialService) validateExamDomain(ctx context.Context, domainID string, subjectID *string) error {
	if _, err := s.domainRepo.GetDomainByID(ctx, domainID); err != nil {
		if errors.Is(err, repository.ErrExamDomainNotFound) {
			return ErrExamNotFound
		}
		return err
	}
	if subjectID == nil || strings.TrimSpace(*subjectID) == "" {
		return nil
	}
	subject, err := s.domainRepo.GetSubjectByID(ctx, strings.TrimSpace(*subjectID))
	if err != nil {
		if errors.Is(err, repository.ErrExamSubjectNotFound) {
			return ErrExamNotFound
		}
		return err
	}
	if subject.DomainID != domainID {
		return ErrExamInvalidRequest
	}
	return nil
}

func (s *examMaterialService) resolveQuestionBank(ctx context.Context, tenantID uint64, userID string, material *types.ExamMaterial, bankID string) (*types.QuestionBank, error) {
	if bankID != "" {
		bank, err := s.questionService.GetQuestionBank(ctx, tenantID, userID, bankID)
		if err != nil {
			return nil, err
		}
		if bank.SpaceID != material.SpaceID || bank.DomainID != material.DomainID {
			return nil, ErrExamInvalidRequest
		}
		if material.SubjectID != nil && bank.SubjectID != nil && *bank.SubjectID != *material.SubjectID {
			return nil, ErrExamInvalidRequest
		}
		return bank, nil
	}

	return s.questionService.CreateQuestionBank(ctx, tenantID, userID, &types.CreateQuestionBankRequest{
		SpaceID:     material.SpaceID,
		DomainID:    material.DomainID,
		SubjectID:   material.SubjectID,
		Name:        fmt.Sprintf("%s 题库", material.Title),
		Description: fmt.Sprintf("由考试资料 %s 创建", material.Title),
	})
}

func (s *examMaterialService) countKnowledgeChunks(ctx context.Context, knowledgeID string) (int, error) {
	chunks, err := s.chunkReader.ListChunksByKnowledgeID(ctx, knowledgeID)
	if err != nil {
		return 0, err
	}
	return len(chunks), nil
}

func (s *examMaterialService) resolveReadableSpaceIDs(ctx context.Context, tenantID uint64, userID string, spaceID string) ([]string, error) {
	if spaceID != "" {
		ok, err := s.spaceService.CanReadSpace(ctx, tenantID, userID, spaceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrExamPermissionDenied
		}
		return []string{spaceID}, nil
	}
	spaces, err := s.spaceService.ListSpaces(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	spaceIDs := make([]string, 0, len(spaces))
	for _, space := range spaces {
		if space != nil {
			spaceIDs = append(spaceIDs, space.ID)
		}
	}
	return spaceIDs, nil
}

func normalizeOptionalID(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	out := strings.TrimSpace(*value)
	return &out
}

func resolveExamMaterialTitle(input string, knowledge *types.Knowledge) string {
	if title := strings.TrimSpace(input); title != "" {
		return title
	}
	if knowledge != nil {
		if title := strings.TrimSpace(knowledge.Title); title != "" {
			return title
		}
		if fileName := strings.TrimSpace(knowledge.FileName); fileName != "" {
			return fileName
		}
		if knowledge.ID != "" {
			return knowledge.ID
		}
	}
	return "未命名资料"
}

func mapExamMaterialIngestStatus(parseStatus string) types.ExamMaterialIngestStatus {
	switch parseStatus {
	case types.ParseStatusPending:
		return types.ExamMaterialIngestStatusPending
	case types.ParseStatusProcessing, types.ParseStatusFinalizing:
		return types.ExamMaterialIngestStatusProcessing
	case types.ParseStatusCompleted:
		return types.ExamMaterialIngestStatusCompleted
	case types.ParseStatusFailed:
		return types.ExamMaterialIngestStatusFailed
	case types.ParseStatusCancelled:
		return types.ExamMaterialIngestStatusCancelled
	default:
		return types.ExamMaterialIngestStatusUnknown
	}
}

func resolveStructuringTaskStatus(ingestStatus types.ExamMaterialIngestStatus, chunkCount int) (types.ExamStructuringTaskStatus, string) {
	if ingestStatus != types.ExamMaterialIngestStatusCompleted {
		return types.ExamStructuringTaskStatusBlocked, "文档解析尚未完成，暂不能结构化"
	}
	if chunkCount <= 0 {
		return types.ExamStructuringTaskStatusBlocked, "文档没有可用 chunk，暂不能结构化"
	}
	return types.ExamStructuringTaskStatusReadyForReview, ""
}
