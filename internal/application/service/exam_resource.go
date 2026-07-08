package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

type examResourceService struct {
	resourceRepo interfaces.ExamResourceRepository
	spaceService interfaces.ExamSpaceService
	domainRepo   interfaces.ExamDomainRepository
	kbService    interfaces.KnowledgeBaseService
}

func NewExamResourceService(
	resourceRepo interfaces.ExamResourceRepository,
	spaceService interfaces.ExamSpaceService,
	domainRepo interfaces.ExamDomainRepository,
	kbService interfaces.KnowledgeBaseService,
) interfaces.ExamResourceService {
	return &examResourceService{
		resourceRepo: resourceRepo,
		spaceService: spaceService,
		domainRepo:   domainRepo,
		kbService:    kbService,
	}
}

func (s *examResourceService) BindKnowledgeBase(ctx context.Context, tenantID uint64, userID string, knowledgeBaseID string, req *types.BindKnowledgeBaseResourceRequest) (*types.ExamSpaceResource, error) {
	if strings.TrimSpace(knowledgeBaseID) == "" || req == nil {
		return nil, ErrExamInvalidRequest
	}
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, knowledgeBaseID)
	if err != nil {
		if errors.Is(err, repository.ErrKnowledgeBaseNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	if kb.TenantID != tenantID {
		return nil, ErrExamPermissionDenied
	}

	space, err := s.spaceService.GetSpace(ctx, tenantID, userID, strings.TrimSpace(req.SpaceID))
	if err != nil {
		return nil, err
	}
	if space.SpaceType == types.ExamSpaceTypePublic {
		if !canBindKnowledgeBase(ctx, kb, userID) {
			return nil, ErrExamPermissionDenied
		}
	} else {
		canWrite, err := s.spaceService.CanWriteSpace(ctx, tenantID, userID, space.ID)
		if err != nil {
			return nil, err
		}
		if !canWrite {
			return nil, ErrExamPermissionDenied
		}
	}
	if _, err := s.domainRepo.GetDomainByID(ctx, strings.TrimSpace(req.DomainID)); err != nil {
		if errors.Is(err, repository.ErrExamDomainNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}

	var subjectID *string
	if req.SubjectID != nil && strings.TrimSpace(*req.SubjectID) != "" {
		subject, err := s.domainRepo.GetSubjectByID(ctx, strings.TrimSpace(*req.SubjectID))
		if err != nil {
			if errors.Is(err, repository.ErrExamSubjectNotFound) {
				return nil, ErrExamNotFound
			}
			return nil, err
		}
		if subject.DomainID != req.DomainID {
			return nil, ErrExamInvalidRequest
		}
		id := subject.ID
		subjectID = &id
	}

	materialType := normalizeMaterialType(req.MaterialType)
	reviewStatus := types.ExamReviewStatusPrivate
	if space.SpaceType == types.ExamSpaceTypePublic {
		reviewStatus = types.ExamReviewStatusPending
	}

	now := time.Now()
	resource := &types.ExamSpaceResource{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		SpaceID:         space.ID,
		ResourceType:    types.ExamResourceTypeKnowledgeBase,
		ResourceID:      knowledgeBaseID,
		DomainID:        strings.TrimSpace(req.DomainID),
		SubjectID:       subjectID,
		MaterialType:    materialType,
		ReviewStatus:    reviewStatus,
		Status:          types.ExamSpaceResourceStatusActive,
		CreatedByUserID: userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.resourceRepo.Upsert(ctx, resource); err != nil {
		return nil, err
	}
	return s.resourceRepo.GetByResource(ctx, tenantID, types.ExamResourceTypeKnowledgeBase, knowledgeBaseID)
}

func (s *examResourceService) GetKnowledgeBaseBinding(ctx context.Context, tenantID uint64, userID string, knowledgeBaseID string) (*types.ExamSpaceResource, error) {
	resource, err := s.resourceRepo.GetByResource(ctx, tenantID, types.ExamResourceTypeKnowledgeBase, knowledgeBaseID)
	if err != nil {
		if errors.Is(err, repository.ErrExamResourceNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	ok, err := s.spaceService.CanReadSpace(ctx, tenantID, userID, resource.SpaceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrExamPermissionDenied
	}
	return resource, nil
}

func (s *examResourceService) CanReadKnowledgeBase(ctx context.Context, tenantID uint64, userID string, knowledgeBaseID string) (bool, error) {
	resource, err := s.resourceRepo.GetByResource(ctx, tenantID, types.ExamResourceTypeKnowledgeBase, knowledgeBaseID)
	if err != nil {
		if errors.Is(err, repository.ErrExamResourceNotFound) {
			return false, nil
		}
		return false, err
	}
	ok, err := s.spaceService.CanReadSpace(ctx, tenantID, userID, resource.SpaceID)
	if err != nil {
		if errors.Is(err, ErrExamNotFound) || errors.Is(err, ErrExamPermissionDenied) {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

func (s *examResourceService) ListResources(ctx context.Context, tenantID uint64, userID string, filter types.ListExamResourcesFilter) ([]*types.ExamSpaceResource, error) {
	if filter.ResourceType == "" {
		filter.ResourceType = types.ExamResourceTypeKnowledgeBase
	}
	if filter.ResourceType != types.ExamResourceTypeKnowledgeBase {
		return nil, ErrExamInvalidRequest
	}
	if filter.MaterialType != "" {
		filter.MaterialType = normalizeMaterialType(filter.MaterialType)
	}
	spaceIDs, err := s.resolveReadableSpaceIDs(ctx, tenantID, userID, strings.TrimSpace(filter.SpaceID))
	if err != nil {
		return nil, err
	}
	return s.resourceRepo.List(ctx, tenantID, filter, spaceIDs)
}

func (s *examResourceService) resolveReadableSpaceIDs(ctx context.Context, tenantID uint64, userID string, spaceID string) ([]string, error) {
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

func canBindKnowledgeBase(ctx context.Context, kb *types.KnowledgeBase, userID string) bool {
	if kb == nil || strings.TrimSpace(userID) == "" {
		return false
	}
	if kb.CreatorID != "" && kb.CreatorID == userID {
		return true
	}
	return types.TenantRoleFromContext(ctx).HasPermission(types.TenantRoleAdmin)
}

func normalizeMaterialType(materialType types.ExamMaterialType) types.ExamMaterialType {
	switch materialType {
	case types.ExamMaterialTypeExamPaper,
		types.ExamMaterialTypeAnswerKey,
		types.ExamMaterialTypeExplanation:
		return materialType
	default:
		return types.ExamMaterialTypeLearningMaterial
	}
}
