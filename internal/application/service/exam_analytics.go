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
)

type examAnalyticsService struct {
	classRepo      interfaces.ExamClassRepository
	assignmentRepo interfaces.ExamAssignmentRepository
	practiceRepo   interfaces.ExamPracticeRepository
}

func NewExamAnalyticsService(
	classRepo interfaces.ExamClassRepository,
	assignmentRepo interfaces.ExamAssignmentRepository,
	practiceRepo interfaces.ExamPracticeRepository,
) interfaces.ExamAnalyticsService {
	return &examAnalyticsService{
		classRepo:      classRepo,
		assignmentRepo: assignmentRepo,
		practiceRepo:   practiceRepo,
	}
}

func (s *examAnalyticsService) GetClassAnalytics(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
) (*types.ExamClassAnalyticsSummary, error) {
	class, err := s.ensureCanAnalyzeClass(ctx, tenantID, userID, classID)
	if err != nil {
		return nil, err
	}
	students, err := s.activeStudents(ctx, tenantID, class.ID)
	if err != nil {
		return nil, err
	}
	assignments, err := s.assignmentRepo.ListAssignmentsByClass(
		ctx,
		tenantID,
		class.ID,
		[]types.ExamAssignmentStatus{types.ExamAssignmentStatusPublished},
		100,
	)
	if err != nil {
		return nil, err
	}
	return s.summarizeClassAnalytics(ctx, tenantID, class, students, assignments)
}

func (s *examAnalyticsService) ensureCanAnalyzeClass(ctx context.Context, tenantID uint64, userID string, classID string) (*types.ExamClass, error) {
	class, err := s.classRepo.GetByIDAndTenant(ctx, strings.TrimSpace(classID), tenantID)
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

func (s *examAnalyticsService) activeStudents(ctx context.Context, tenantID uint64, classID string) ([]*types.ExamClassMember, error) {
	members, err := s.classRepo.ListMembers(ctx, classID, tenantID, []types.ExamClassMemberStatus{types.ExamClassMemberStatusActive})
	if err != nil {
		return nil, err
	}
	students := make([]*types.ExamClassMember, 0, len(members))
	for _, member := range members {
		if member != nil && member.Role == types.ExamClassRoleStudent {
			students = append(students, member)
		}
	}
	sort.Slice(students, func(i, j int) bool {
		if students[i].JoinedAt.Equal(students[j].JoinedAt) {
			return students[i].UserID < students[j].UserID
		}
		return students[i].JoinedAt.Before(students[j].JoinedAt)
	})
	return students, nil
}

func (s *examAnalyticsService) summarizeClassAnalytics(
	ctx context.Context,
	tenantID uint64,
	class *types.ExamClass,
	students []*types.ExamClassMember,
	assignments []*types.ExamClassAssignment,
) (*types.ExamClassAnalyticsSummary, error) {
	summary := newClassAnalyticsSummary(class, students, assignments)
	memberByUser := analyticsMembersByUser(summary.Members)
	studentIDs := analyticsStudentIDs(students)
	latestAttempts := map[string]analyticsAttemptContext{}
	for _, assignment := range assignments {
		if assignment == nil {
			continue
		}
		assignmentSummary, attempts, err := s.summarizeAssignmentAnalytics(ctx, tenantID, assignment, studentIDs, memberByUser)
		if err != nil {
			return nil, err
		}
		for _, attempt := range attempts {
			if attempt != nil {
				latestAttempts[attempt.ID] = newAnalyticsAttemptContext(attempt)
			}
		}
		mergeAssignmentAnalytics(summary, assignmentSummary)
	}
	wrongQuestions, err := s.frequentWrongQuestions(ctx, tenantID, latestAttempts)
	if err != nil {
		return nil, err
	}
	summary.FrequentWrongQuestions = wrongQuestions
	finalizeClassAnalytics(summary)
	return summary, nil
}

func (s *examAnalyticsService) summarizeAssignmentAnalytics(
	ctx context.Context,
	tenantID uint64,
	assignment *types.ExamClassAssignment,
	studentIDs []string,
	memberByUser map[string]*types.ExamClassAnalyticsMember,
) (*types.ExamClassAnalyticsAssignment, map[string]*types.ExamPracticeAttempt, error) {
	attempts, err := s.practiceRepo.ListLatestAttemptsByAssignmentUsers(ctx, tenantID, assignment.ID, studentIDs)
	if err != nil {
		return nil, nil, err
	}
	item := &types.ExamClassAnalyticsAssignment{Assignment: assignment}
	for _, userID := range studentIDs {
		member := memberByUser[userID]
		if member != nil {
			member.AssignmentCount++
		}
		applyAnalyticsAttempt(item, member, attempts[userID])
	}
	finalizeAssignmentAnalytics(item, len(studentIDs))
	return item, attempts, nil
}

func newClassAnalyticsSummary(
	class *types.ExamClass,
	students []*types.ExamClassMember,
	assignments []*types.ExamClassAssignment,
) *types.ExamClassAnalyticsSummary {
	members := make([]*types.ExamClassAnalyticsMember, 0, len(students))
	for _, student := range students {
		members = append(members, &types.ExamClassAnalyticsMember{Member: student})
	}
	return &types.ExamClassAnalyticsSummary{
		Class:                  class,
		TotalStudents:          len(students),
		AssignmentCount:        len(assignments),
		TotalAssignmentSlots:   len(students) * len(assignments),
		Members:                members,
		Assignments:            make([]*types.ExamClassAnalyticsAssignment, 0, len(assignments)),
		FrequentWrongQuestions: make([]*types.ExamClassFrequentWrongQuestion, 0),
	}
}

func analyticsMembersByUser(items []*types.ExamClassAnalyticsMember) map[string]*types.ExamClassAnalyticsMember {
	out := make(map[string]*types.ExamClassAnalyticsMember, len(items))
	for _, item := range items {
		if item != nil && item.Member != nil {
			out[item.Member.UserID] = item
		}
	}
	return out
}

func analyticsStudentIDs(students []*types.ExamClassMember) []string {
	ids := make([]string, 0, len(students))
	for _, student := range students {
		if student != nil {
			ids = append(ids, student.UserID)
		}
	}
	return ids
}

func applyAnalyticsAttempt(
	assignment *types.ExamClassAnalyticsAssignment,
	member *types.ExamClassAnalyticsMember,
	attempt *types.ExamPracticeAttempt,
) {
	if attempt == nil {
		return
	}
	rate := analyticsAttemptCorrectRate(attempt)
	assignment.StartedCount++
	assignment.AverageCorrectRate += rate
	if member != nil {
		member.StartedCount++
		member.AverageCorrectRate += rate
		member.LastActivityAt = laterAnalyticsTime(member.LastActivityAt, analyticsAttemptActivityAt(attempt))
	}
	if attempt.Status != types.ExamPracticeAttemptStatusCompleted {
		return
	}
	assignment.CompletedCount++
	if member != nil {
		member.CompletedCount++
	}
}

func mergeAssignmentAnalytics(summary *types.ExamClassAnalyticsSummary, assignment *types.ExamClassAnalyticsAssignment) {
	summary.Assignments = append(summary.Assignments, assignment)
	summary.StartedCount += assignment.StartedCount
	summary.CompletedCount += assignment.CompletedCount
	summary.AverageCorrectRate += assignment.AverageCorrectRate * float64(assignment.StartedCount)
}

func finalizeClassAnalytics(summary *types.ExamClassAnalyticsSummary) {
	if summary.TotalAssignmentSlots > 0 {
		summary.CompletionRate = float64(summary.CompletedCount) / float64(summary.TotalAssignmentSlots)
	}
	if summary.StartedCount > 0 {
		summary.AverageCorrectRate = summary.AverageCorrectRate / float64(summary.StartedCount)
	}
	for _, member := range summary.Members {
		finalizeMemberAnalytics(member)
	}
	sortAnalyticsMembers(summary.Members)
	sortAnalyticsAssignments(summary.Assignments)
}

func finalizeAssignmentAnalytics(item *types.ExamClassAnalyticsAssignment, studentCount int) {
	if studentCount > 0 {
		item.CompletionRate = float64(item.CompletedCount) / float64(studentCount)
	}
	if item.StartedCount > 0 {
		item.AverageCorrectRate = item.AverageCorrectRate / float64(item.StartedCount)
	}
}

func finalizeMemberAnalytics(item *types.ExamClassAnalyticsMember) {
	if item == nil {
		return
	}
	if item.AssignmentCount > 0 {
		item.CompletionRate = float64(item.CompletedCount) / float64(item.AssignmentCount)
	}
	if item.StartedCount > 0 {
		item.AverageCorrectRate = item.AverageCorrectRate / float64(item.StartedCount)
	}
}

func analyticsAttemptCorrectRate(attempt *types.ExamPracticeAttempt) float64 {
	if attempt == nil || attempt.QuestionCount <= 0 {
		return 0
	}
	return float64(attempt.CorrectCount) / float64(attempt.QuestionCount)
}

func analyticsAttemptActivityAt(attempt *types.ExamPracticeAttempt) *time.Time {
	if attempt == nil {
		return nil
	}
	if attempt.CompletedAt != nil {
		return attempt.CompletedAt
	}
	return &attempt.UpdatedAt
}

func laterAnalyticsTime(current *time.Time, candidate *time.Time) *time.Time {
	if candidate == nil || (current != nil && !candidate.After(*current)) {
		return current
	}
	cp := *candidate
	return &cp
}
