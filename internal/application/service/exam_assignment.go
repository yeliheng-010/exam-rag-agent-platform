package service

import (
	"context"
	"errors"
	"sort"
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
	spaceService   interfaces.ExamSpaceService
	now            func() time.Time
}

func NewExamAssignmentService(
	assignmentRepo interfaces.ExamAssignmentRepository,
	classRepo interfaces.ExamClassRepository,
	questionRepo interfaces.ExamQuestionRepository,
	practiceRepo interfaces.ExamPracticeRepository,
	spaceService interfaces.ExamSpaceService,
) interfaces.ExamAssignmentService {
	return &examAssignmentService{
		assignmentRepo: assignmentRepo,
		classRepo:      classRepo,
		questionRepo:   questionRepo,
		practiceRepo:   practiceRepo,
		spaceService:   spaceService,
		now:            time.Now,
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
		detail, err = s.importAssignmentGroupToClassSpace(ctx, tenantID, userID, class, detail)
		if err != nil {
			return nil, err
		}
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
	now := s.now()
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
	notifications, err := s.assignmentNotificationsForStudents(
		ctx, tenantID, userID, assignment, types.ExamAssignmentNotificationKindPublished, now,
	)
	if err != nil {
		return nil, err
	}
	if err := s.assignmentRepo.CreateAssignmentWithNotifications(ctx, assignment, notifications); err != nil {
		return nil, err
	}
	return s.assignmentSummary(ctx, tenantID, userID, assignment)
}

func (s *examAssignmentService) UpdateAssignment(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	assignmentID string,
	req *types.UpdateExamAssignmentRequest,
) (*types.ExamAssignmentSummary, error) {
	title, instructions, err := validateAssignmentUpdate(req)
	if err != nil {
		return nil, err
	}
	assignment, err := s.writableAssignment(ctx, tenantID, userID, classID, assignmentID)
	if err != nil {
		return nil, err
	}
	if assignment.Status == types.ExamAssignmentStatusPublished && assignmentExpired(assignment, s.now()) {
		return nil, ErrExamStateConflict
	}
	if !assignmentStatusAllowed(assignment.Status, types.ExamAssignmentStatusPublished, types.ExamAssignmentStatusWithdrawn) {
		return nil, ErrExamStateConflict
	}
	err = s.assignmentRepo.UpdateAssignmentMetadata(
		ctx, tenantID, assignment.ClassID, assignment.ID, []types.ExamAssignmentStatus{assignment.Status},
		title, instructions, req.DueAt, s.now(),
	)
	if err != nil {
		return nil, mapAssignmentStateError(err)
	}
	return s.refreshedAssignmentSummary(ctx, tenantID, userID, assignment.ID)
}

func (s *examAssignmentService) WithdrawAssignment(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	assignmentID string,
) (*types.ExamAssignmentSummary, error) {
	assignment, err := s.writableAssignment(ctx, tenantID, userID, classID, assignmentID)
	if err != nil {
		return nil, err
	}
	return s.transitionAssignment(ctx, tenantID, userID, assignment, types.ExamAssignmentStatusPublished, types.ExamAssignmentStatusWithdrawn)
}

func (s *examAssignmentService) RepublishAssignment(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	assignmentID string,
) (*types.ExamAssignmentSummary, error) {
	assignment, err := s.writableAssignment(ctx, tenantID, userID, classID, assignmentID)
	if err != nil {
		return nil, err
	}
	if assignment.Status != types.ExamAssignmentStatusWithdrawn || assignmentExpired(assignment, s.now()) {
		return nil, ErrExamStateConflict
	}
	return s.transitionAssignment(ctx, tenantID, userID, assignment, types.ExamAssignmentStatusWithdrawn, types.ExamAssignmentStatusPublished)
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
	member, err := s.ensureActiveClassMember(ctx, tenantID, userID, strings.TrimSpace(classID))
	if err != nil {
		return nil, err
	}
	statuses := []types.ExamAssignmentStatus{types.ExamAssignmentStatusPublished}
	if member.Role.CanWrite() {
		statuses = append(statuses, types.ExamAssignmentStatusWithdrawn)
	}
	assignments, err := s.assignmentRepo.ListAssignmentsByClass(ctx, tenantID, strings.TrimSpace(classID), statuses, filter.Limit)
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
	if _, err := s.ensureActiveClassMember(ctx, tenantID, userID, assignment.ClassID); err != nil {
		return nil, err
	}
	if assignment.Status != types.ExamAssignmentStatusPublished || assignmentExpired(assignment, s.now()) {
		return nil, ErrExamStateConflict
	}
	detail, err := s.assignmentQuestionGroup(ctx, tenantID, assignment.GroupID)
	if err != nil {
		return nil, err
	}
	assignmentIDCopy := assignment.ID
	now := s.now()
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
	if err := s.practiceRepo.CreateAssignmentAttemptIfOpen(ctx, attempt, now); err != nil {
		if errors.Is(err, repository.ErrExamAssignmentAttemptClosed) {
			return nil, ErrExamStateConflict
		}
		return nil, err
	}
	return &types.CreatePracticeAttemptResult{
		Attempt: attempt,
		Group:   practiceQuestionGroupView(detail),
	}, nil
}

func (s *examAssignmentService) GetAssignmentProgress(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	assignmentID string,
) (*types.ExamAssignmentProgressSummary, error) {
	class, err := s.ensureCanWriteAssignmentClass(ctx, tenantID, userID, strings.TrimSpace(classID))
	if err != nil {
		return nil, err
	}
	assignment, err := s.assignmentByID(ctx, tenantID, strings.TrimSpace(assignmentID))
	if err != nil {
		return nil, err
	}
	if assignment.ClassID != class.ID || !assignmentStatusAllowed(
		assignment.Status,
		types.ExamAssignmentStatusPublished,
		types.ExamAssignmentStatusWithdrawn,
	) {
		return nil, ErrExamNotFound
	}
	members, err := s.classRepo.ListMembers(ctx, class.ID, tenantID, []types.ExamClassMemberStatus{
		types.ExamClassMemberStatusActive,
	})
	if err != nil {
		return nil, err
	}
	students := make([]*types.ExamClassMember, 0, len(members))
	userIDs := make([]string, 0, len(members))
	for _, member := range members {
		if member == nil || member.Role != types.ExamClassRoleStudent {
			continue
		}
		students = append(students, member)
		userIDs = append(userIDs, member.UserID)
	}
	sort.Slice(students, func(i, j int) bool {
		if students[i].JoinedAt.Equal(students[j].JoinedAt) {
			return students[i].UserID < students[j].UserID
		}
		return students[i].JoinedAt.Before(students[j].JoinedAt)
	})
	attempts, err := s.practiceRepo.ListLatestAttemptsByAssignmentUsers(ctx, tenantID, assignment.ID, userIDs)
	if err != nil {
		return nil, err
	}
	summary := &types.ExamAssignmentProgressSummary{
		Assignment:    assignment,
		TotalStudents: len(students),
		Members:       make([]*types.ExamAssignmentMemberProgress, 0, len(students)),
	}
	var correctRateTotal float64
	for _, student := range students {
		attempt := attempts[student.UserID]
		status := types.ExamAssignmentProgressStatusNotStarted
		correctRate := 0.0
		if attempt != nil {
			summary.StartedCount++
			status = types.ExamAssignmentProgressStatusInProgress
			if attempt.QuestionCount > 0 {
				correctRate = float64(attempt.CorrectCount) / float64(attempt.QuestionCount)
			}
			correctRateTotal += correctRate
			if attempt.Status == types.ExamPracticeAttemptStatusCompleted {
				status = types.ExamAssignmentProgressStatusCompleted
				summary.CompletedCount++
			}
		}
		summary.Members = append(summary.Members, &types.ExamAssignmentMemberProgress{
			Member:      student,
			Attempt:     attempt,
			Status:      status,
			CorrectRate: correctRate,
		})
	}
	if summary.StartedCount > 0 {
		summary.AverageCorrectRate = correctRateTotal / float64(summary.StartedCount)
	}
	return summary, nil
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
	if member.Status != types.ExamClassMemberStatusActive {
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
	if member.Status != types.ExamClassMemberStatusActive {
		return nil, ErrExamPermissionDenied
	}
	return member, nil
}

func validateAssignmentUpdate(req *types.UpdateExamAssignmentRequest) (string, string, error) {
	if req == nil {
		return "", "", ErrExamInvalidRequest
	}
	title := strings.TrimSpace(req.Title)
	instructions := strings.TrimSpace(req.Instructions)
	if title == "" || len([]rune(title)) > 255 || len([]rune(instructions)) > 2000 {
		return "", "", ErrExamInvalidRequest
	}
	return title, instructions, nil
}

func (s *examAssignmentService) writableAssignment(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	assignmentID string,
) (*types.ExamClassAssignment, error) {
	class, err := s.ensureCanWriteAssignmentClass(ctx, tenantID, userID, strings.TrimSpace(classID))
	if err != nil {
		return nil, err
	}
	assignment, err := s.assignmentByID(ctx, tenantID, strings.TrimSpace(assignmentID))
	if err != nil {
		return nil, err
	}
	if assignment.ClassID != class.ID {
		return nil, ErrExamNotFound
	}
	return assignment, nil
}

func (s *examAssignmentService) transitionAssignment(
	ctx context.Context,
	tenantID uint64,
	userID string,
	assignment *types.ExamClassAssignment,
	expected types.ExamAssignmentStatus,
	next types.ExamAssignmentStatus,
) (*types.ExamAssignmentSummary, error) {
	if assignment.Status != expected {
		return nil, ErrExamStateConflict
	}
	now := s.now()
	notificationKind := types.ExamAssignmentNotificationKindWithdrawn
	if next == types.ExamAssignmentStatusPublished {
		notificationKind = types.ExamAssignmentNotificationKindRepublished
	}
	notifications, err := s.assignmentNotificationsForStudents(
		ctx, tenantID, userID, assignment, notificationKind, now,
	)
	if err != nil {
		return nil, err
	}
	err = s.assignmentRepo.TransitionAssignmentStatusWithNotifications(
		ctx, tenantID, assignment.ClassID, assignment.ID, expected, next, now, notifications,
	)
	if err != nil {
		return nil, mapAssignmentStateError(err)
	}
	return s.refreshedAssignmentSummary(ctx, tenantID, userID, assignment.ID)
}

func (s *examAssignmentService) refreshedAssignmentSummary(
	ctx context.Context,
	tenantID uint64,
	userID string,
	assignmentID string,
) (*types.ExamAssignmentSummary, error) {
	assignment, err := s.assignmentByID(ctx, tenantID, assignmentID)
	if err != nil {
		return nil, err
	}
	return s.assignmentSummary(ctx, tenantID, userID, assignment)
}

func mapAssignmentStateError(err error) error {
	if errors.Is(err, repository.ErrExamClassAssignmentStateConflict) {
		return ErrExamStateConflict
	}
	return err
}

func assignmentExpired(assignment *types.ExamClassAssignment, now time.Time) bool {
	return assignment != nil && assignment.DueAt != nil && !assignment.DueAt.After(now)
}

func assignmentStatusAllowed(status types.ExamAssignmentStatus, allowed ...types.ExamAssignmentStatus) bool {
	for _, candidate := range allowed {
		if status == candidate {
			return true
		}
	}
	return false
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
