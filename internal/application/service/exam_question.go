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

type examQuestionService struct {
	questionRepo interfaces.ExamQuestionRepository
	spaceService interfaces.ExamSpaceService
	domainRepo   interfaces.ExamDomainRepository
}

func NewExamQuestionService(
	questionRepo interfaces.ExamQuestionRepository,
	spaceService interfaces.ExamSpaceService,
	domainRepo interfaces.ExamDomainRepository,
) interfaces.ExamQuestionService {
	return &examQuestionService{
		questionRepo: questionRepo,
		spaceService: spaceService,
		domainRepo:   domainRepo,
	}
}

func (s *examQuestionService) CreateQuestionBank(ctx context.Context, tenantID uint64, userID string, req *types.CreateQuestionBankRequest) (*types.QuestionBank, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrExamInvalidRequest
	}
	canWrite, err := s.spaceService.CanWriteSpace(ctx, tenantID, userID, req.SpaceID)
	if err != nil {
		return nil, err
	}
	if !canWrite {
		return nil, ErrExamPermissionDenied
	}
	if _, err := s.domainRepo.GetDomainByID(ctx, req.DomainID); err != nil {
		if errors.Is(err, repository.ErrExamDomainNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
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
		subjectID := subject.ID
		req.SubjectID = &subjectID
	} else {
		req.SubjectID = nil
	}

	now := time.Now()
	bank := &types.QuestionBank{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		SpaceID:         req.SpaceID,
		DomainID:        req.DomainID,
		SubjectID:       req.SubjectID,
		Name:            name,
		Description:     strings.TrimSpace(req.Description),
		SourceType:      "manual",
		ReviewStatus:    types.ExamReviewStatusPrivate,
		Status:          "active",
		CreatedByUserID: userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.questionRepo.CreateQuestionBank(ctx, bank); err != nil {
		return nil, err
	}
	return bank, nil
}

func (s *examQuestionService) ListQuestionBanks(ctx context.Context, tenantID uint64, userID string, spaceID string) ([]*types.QuestionBank, error) {
	var spaceIDs []string
	if strings.TrimSpace(spaceID) != "" {
		ok, err := s.spaceService.CanReadSpace(ctx, tenantID, userID, spaceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrExamPermissionDenied
		}
		spaceIDs = []string{spaceID}
	} else {
		spaces, err := s.spaceService.ListSpaces(ctx, tenantID, userID)
		if err != nil {
			return nil, err
		}
		spaceIDs = make([]string, 0, len(spaces))
		for _, space := range spaces {
			if space != nil {
				spaceIDs = append(spaceIDs, space.ID)
			}
		}
	}
	return s.questionRepo.ListQuestionBanks(ctx, tenantID, spaceIDs)
}

func (s *examQuestionService) GetQuestionBank(ctx context.Context, tenantID uint64, userID string, bankID string) (*types.QuestionBank, error) {
	bank, err := s.questionRepo.GetQuestionBankByIDAndTenant(ctx, bankID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrQuestionBankNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	ok, err := s.spaceService.CanReadSpace(ctx, tenantID, userID, bank.SpaceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrExamPermissionDenied
	}
	return bank, nil
}
