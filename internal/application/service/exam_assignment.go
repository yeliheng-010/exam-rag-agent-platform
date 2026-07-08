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

type examAssignmentService struct {
	assignmentRepo interfaces.ExamAssignmentRepository
	classRepo      interfaces.ExamClassRepository
	questionRepo   interfaces.ExamQuestionRepository
	practiceRepo   interfaces.ExamPracticeRepository
}

func NewExamAssignmentService(
	assignmentRepo interfaces.ExamAssignmentRepository,
	classRepo interfaces.ExamClassRepository,
	questionRepo interfaces.ExamQuestionRepository,
	practiceRepo interfaces.ExamPracticeRepository,
) interfaces.ExamAssignmentService {
	return &examAssignmentService{
		assignmentRepo: assignmentRepo,
		classRepo:      classRepo,
		questionRepo:   questionRepo,
		practiceRepo:   practiceRepo,
	}
}

func (s *examAssignmentService) CreateAssignment(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	req *types.CreateExamAssignmentRequest,
) (*types.ExamAssignmentSummary, error) {
	if req == nil || strings.TrimSpace(classID) == "" || strings.TrimSpace(req.GroupID) == "" {
		return nil, ErrExamInvalidRequest
	}
	class, err := s.ensureCanWriteAssignmentClass(ctx, tenantID, userID, strings.TrimSpace(classID))
	if err != nil {
		return nil, err
	}
	detail, err := s.assignmentQuestionGroup(ctx, tenantID, strings.TrimSpace(req.GroupID))
	if err != nil {
		return nil, err
	}
	if detail.Group.SpaceID != class.SpaceID {
		return nil, ErrExamPermissionDenied
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSpace(detail.Group.Title)
	}
	if title == "" {
		title = "练习任务"
	}
	instructions := strings.TrimSpace(req.Instructions)
	if len([]rune(title)) > 255 || len([]rune(instructions)) > 2000 {
		return nil, ErrExamInvalidRequest
	}
	now := time.Now()
	assignment := &types.ExamClassAssignment{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		ClassID:         class.ID,
		SpaceID:         class.SpaceID,
		QuestionBankID:  detail.Group.QuestionBankID,
		GroupID:         detail.Group.ID,
		Title:           title,
		Instructions:    instructions,
		Status:          types.ExamAssignmentStatusPublished,
		DueAt:           req.DueAt,
		CreatedByUserID: userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.assignmentRepo.CreateAssignment(ctx, assignment); err != nil {
		return nil, err
	}
	return s.assignmentSummary(ctx, tenantID, userID, assignment)
}

func (s *examAssignmentService) ListClassAssignments(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	filter types.ListExamAssignmentsFilter,
) ([]*types.ExamAssignmentSummary, error) {
	if strings.TrimSpace(classID) == "" {
		return nil, ErrExamInvalidRequest
	}
	if _, err := s.ensureActiveClassMember(ctx, tenantID, userID, strings.TrimSpace(classID)); err != nil {
		return nil, err
	}
	assignments, err := s.assignmentRepo.ListAssignmentsByClass(ctx, tenantID, strings.TrimSpace(classID), filter.Limit)
	if err != nil {
		return nil, err
	}
	return s.assignmentSummaries(ctx, tenantID, userID, assignments)
}

func (s *examAssignmentService) ListMyAssignments(
	ctx context.Context,
	tenantID uint64,
	userID string,
	filter types.ListExamAssignmentsFilter,
) ([]*types.ExamAssignmentSummary, error) {
	assignments, err := s.assignmentRepo.ListAssignmentsByUserClasses(ctx, tenantID, userID, filter.Limit)
	if err != nil {
		return nil, err
	}
	return s.assignmentSummaries(ctx, tenantID, userID, assignments)
}

func (s *examAssignmentService) CreateAssignmentAttempt(
	ctx context.Context,
	tenantID uint64,
	userID string,
	assignmentID string,
) (*types.CreatePracticeAttemptResult, error) {
	assignment, err := s.assignmentByID(ctx, tenantID, strings.TrimSpace(assignmentID))
	if err != nil {
		return nil, err
	}
	if assignment.Status != types.ExamAssignmentStatusPublished {
		return nil, ErrExamInvalidRequest
	}
	if _, err := s.ensureActiveClassMember(ctx, tenantID, userID, assignment.ClassID); err != nil {
		return nil, err
	}
	detail, err := s.assignmentQuestionGroup(ctx, tenantID, assignment.GroupID)
	if err != nil {
		return nil, err
	}
	assignmentIDCopy := assignment.ID
	now := time.Now()
	attempt := &types.ExamPracticeAttempt{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		UserID:         userID,
		SpaceID:        assignment.SpaceID,
		QuestionBankID: assignment.QuestionBankID,
		GroupID:        assignment.GroupID,
		AssignmentID:   &assignmentIDCopy,
		Status:         types.ExamPracticeAttemptStatusInProgress,
		QuestionCount:  len(detail.Questions),
		StartedAt:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.practiceRepo.CreateAttempt(ctx, attempt); err != nil {
		return nil, err
	}
	return &types.CreatePracticeAttemptResult{
		Attempt: attempt,
		Group:   practiceQuestionGroupView(detail),
	}, nil
}

func (s *examAssignmentService) ensureCanWriteAssignmentClass(ctx context.Context, tenantID uint64, userID string, classID string) (*types.ExamClass, error) {
	class, err := s.classRepo.GetByIDAndTenant(ctx, classID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	member, err := s.classRepo.GetMember(ctx, class.ID, tenantID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassMemberNotFound) {
			return nil, ErrExamPermissionDenied
		}
		return nil, err
	}
	if !member.Role.CanWrite() {
		return nil, ErrExamPermissionDenied
	}
	return class, nil
}

func (s *examAssignmentService) ensureActiveClassMember(ctx context.Context, tenantID uint64, userID string, classID string) (*types.ExamClassMember, error) {
	if strings.TrimSpace(classID) == "" {
		return nil, ErrExamInvalidRequest
	}
	member, err := s.classRepo.GetMember(ctx, strings.TrimSpace(classID), tenantID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassMemberNotFound) {
			return nil, ErrExamPermissionDenied
		}
		return nil, err
	}
	return member, nil
}

func (s *examAssignmentService) assignmentByID(ctx context.Context, tenantID uint64, assignmentID string) (*types.ExamClassAssignment, error) {
	if assignmentID == "" {
		return nil, ErrExamInvalidRequest
	}
	assignment, err := s.assignmentRepo.GetAssignmentByIDAndTenant(ctx, tenantID, assignmentID)
	if err != nil {
		if errors.Is(err, repository.ErrExamClassAssignmentNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	return assignment, nil
}

func (s *examAssignmentService) assignmentQuestionGroup(ctx context.Context, tenantID uint64, groupID string) (*types.QuestionGroupDetail, error) {
	if strings.TrimSpace(groupID) == "" {
		return nil, ErrExamInvalidRequest
	}
	detail, err := s.questionRepo.GetQuestionGroupDetailByIDAndTenant(ctx, tenantID, strings.TrimSpace(groupID))
	if err != nil {
		if errors.Is(err, repository.ErrQuestionNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	if detail == nil || detail.Group == nil || detail.Group.Status == "deleted" {
		return nil, ErrExamNotFound
	}
	return detail, nil
}

func (s *examAssignmentService) assignmentSummaries(
	ctx context.Context,
	tenantID uint64,
	userID string,
	assignments []*types.ExamClassAssignment,
) ([]*types.ExamAssignmentSummary, error) {
	assignmentIDs := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment != nil {
			assignmentIDs = append(assignmentIDs, assignment.ID)
		}
	}
	attempts, err := s.practiceRepo.ListLatestAttemptsByAssignments(ctx, tenantID, userID, assignmentIDs)
	if err != nil {
		return nil, err
	}
	out := make([]*types.ExamAssignmentSummary, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment == nil {
			continue
		}
		summary, err := s.assignmentSummaryWithAttempts(ctx, tenantID, assignment, attempts)
		if err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
	return out, nil
}

func (s *examAssignmentService) assignmentSummary(ctx context.Context, tenantID uint64, userID string, assignment *types.ExamClassAssignment) (*types.ExamAssignmentSummary, error) {
	attempts, err := s.practiceRepo.ListLatestAttemptsByAssignments(ctx, tenantID, userID, []string{assignment.ID})
	if err != nil {
		return nil, err
	}
	return s.assignmentSummaryWithAttempts(ctx, tenantID, assignment, attempts)
}

func (s *examAssignmentService) assignmentSummaryWithAttempts(
	ctx context.Context,
	tenantID uint64,
	assignment *types.ExamClassAssignment,
	attempts map[string]*types.ExamPracticeAttempt,
) (*types.ExamAssignmentSummary, error) {
	detail, err := s.assignmentQuestionGroup(ctx, tenantID, assignment.GroupID)
	if err != nil {
		return nil, err
	}
	bankName := ""
	if bank, err := s.questionRepo.GetQuestionBankByIDAndTenant(ctx, assignment.QuestionBankID, tenantID); err == nil && bank != nil {
		bankName = bank.Name
	} else if err != nil && !errors.Is(err, repository.ErrQuestionBankNotFound) {
		return nil, err
	}
	return &types.ExamAssignmentSummary{
		Assignment:    assignment,
		Group:         detail.Group,
		BankName:      bankName,
		QuestionCount: len(detail.Questions),
		LastAttempt:   attempts[assignment.ID],
	}, nil
}
