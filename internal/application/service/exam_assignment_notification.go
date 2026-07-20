package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"

	"github.com/Tencent/WeKnora/internal/application/repository"
)

func (s *examAssignmentService) SendAssignmentReminders(
	ctx context.Context,
	tenantID uint64,
	userID string,
	classID string,
	assignmentID string,
	req *types.SendExamAssignmentReminderRequest,
) (*types.SendExamAssignmentReminderResult, error) {
	if req == nil {
		return nil, ErrExamInvalidRequest
	}
	assignment, err := s.writableAssignment(ctx, tenantID, userID, classID, assignmentID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if assignment.Status != types.ExamAssignmentStatusPublished || assignmentExpired(assignment, now) {
		return nil, ErrExamStateConflict
	}
	requested, err := reminderRecipientSet(req.RecipientUserIDs)
	if err != nil {
		return nil, err
	}
	candidates, err := s.assignmentNotificationsForStudents(
		ctx, tenantID, userID, assignment, types.ExamAssignmentNotificationKindReminder, now,
	)
	if err != nil {
		return nil, err
	}
	candidates, err = selectReminderCandidates(candidates, requested)
	if err != nil {
		return nil, err
	}
	result, err := s.notificationRepo.CreateRemindersIfEligible(ctx, tenantID, assignment.ID, candidates, now)
	if err != nil {
		return nil, mapAssignmentStateError(err)
	}
	return result, nil
}

func (s *examAssignmentService) ListAssignmentNotifications(
	ctx context.Context,
	tenantID uint64,
	userID string,
	limit int,
) (*types.ExamAssignmentNotificationList, error) {
	notifications, unreadCount, err := s.notificationRepo.ListForRecipient(ctx, tenantID, userID, limit)
	if err != nil {
		return nil, err
	}
	assignmentIDs := notificationAssignmentIDs(notifications)
	assignments, err := s.assignmentRepo.ListAssignmentsByIDsAndTenant(ctx, tenantID, assignmentIDs)
	if err != nil {
		return nil, err
	}
	attempts, err := s.practiceRepo.ListLatestAttemptsByAssignments(ctx, tenantID, userID, assignmentIDs)
	if err != nil {
		return nil, err
	}
	now := s.now()
	items := make([]*types.ExamAssignmentNotificationItem, 0, len(notifications))
	for _, notification := range notifications {
		if notification == nil {
			continue
		}
		item := assignmentNotificationItem(notification, assignments[notification.AssignmentID], attempts[notification.AssignmentID], now)
		items = append(items, item)
	}
	return &types.ExamAssignmentNotificationList{Items: items, UnreadCount: unreadCount}, nil
}

func (s *examAssignmentService) MarkAssignmentNotificationRead(
	ctx context.Context,
	tenantID uint64,
	userID string,
	notificationID string,
) error {
	if strings.TrimSpace(notificationID) == "" {
		return ErrExamInvalidRequest
	}
	err := s.notificationRepo.MarkRead(ctx, tenantID, userID, strings.TrimSpace(notificationID), s.now())
	if errors.Is(err, repository.ErrExamAssignmentNotificationNotFound) {
		return ErrExamNotFound
	}
	return err
}

func (s *examAssignmentService) MarkAllAssignmentNotificationsRead(
	ctx context.Context,
	tenantID uint64,
	userID string,
) error {
	return s.notificationRepo.MarkAllRead(ctx, tenantID, userID, s.now())
}

func (s *examAssignmentService) assignmentNotificationsForStudents(
	ctx context.Context,
	tenantID uint64,
	actorUserID string,
	assignment *types.ExamClassAssignment,
	kind types.ExamAssignmentNotificationKind,
	createdAt time.Time,
) ([]*types.ExamAssignmentNotification, error) {
	class, err := s.classRepo.GetByIDAndTenant(ctx, assignment.ClassID, tenantID)
	if err != nil {
		return nil, err
	}
	members, err := s.classRepo.ListMembers(ctx, assignment.ClassID, tenantID, []types.ExamClassMemberStatus{
		types.ExamClassMemberStatusActive,
	})
	if err != nil {
		return nil, err
	}
	title, content := assignmentNotificationText(kind, class.Name, assignment)
	notifications := make([]*types.ExamAssignmentNotification, 0, len(members))
	for _, member := range members {
		if member == nil || member.Role != types.ExamClassRoleStudent || member.Status != types.ExamClassMemberStatusActive {
			continue
		}
		notifications = append(notifications, &types.ExamAssignmentNotification{
			ID: uuid.New().String(), TenantID: tenantID, ClassID: assignment.ClassID,
			AssignmentID: assignment.ID, GroupID: assignment.GroupID,
			RecipientUserID: member.UserID, ActorUserID: actorUserID,
			Kind: kind, Title: title, Content: content, CreatedAt: createdAt,
		})
	}
	return notifications, nil
}

func assignmentNotificationText(
	kind types.ExamAssignmentNotificationKind,
	className string,
	assignment *types.ExamClassAssignment,
) (string, string) {
	dueAt := assignmentNotificationDueAt(assignment.DueAt)
	switch kind {
	case types.ExamAssignmentNotificationKindReminder:
		return "作业待完成：" + assignment.Title, fmt.Sprintf("%s · 请尽快完成 · 截止时间：%s", className, dueAt)
	case types.ExamAssignmentNotificationKindRepublished:
		return "作业重新发布：" + assignment.Title, fmt.Sprintf("%s · 截止时间：%s", className, dueAt)
	case types.ExamAssignmentNotificationKindWithdrawn:
		return "作业已撤回：" + assignment.Title, className + " · 已有答题记录仍然保留"
	default:
		return "新作业：" + assignment.Title, fmt.Sprintf("%s · 截止时间：%s", className, dueAt)
	}
}

func reminderRecipientSet(userIDs []string) (map[string]bool, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	out := make(map[string]bool, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			return nil, ErrExamInvalidRequest
		}
		out[userID] = true
	}
	return out, nil
}

func selectReminderCandidates(
	candidates []*types.ExamAssignmentNotification,
	requested map[string]bool,
) ([]*types.ExamAssignmentNotification, error) {
	if len(requested) == 0 {
		return candidates, nil
	}
	selected := make([]*types.ExamAssignmentNotification, 0, len(requested))
	found := make(map[string]bool, len(requested))
	for _, candidate := range candidates {
		if candidate != nil && requested[candidate.RecipientUserID] {
			selected = append(selected, candidate)
			found[candidate.RecipientUserID] = true
		}
	}
	if len(found) != len(requested) {
		return nil, ErrExamInvalidRequest
	}
	return selected, nil
}

func assignmentNotificationDueAt(dueAt *time.Time) string {
	if dueAt == nil {
		return "无固定截止时间"
	}
	return dueAt.In(time.Local).Format("2006-01-02 15:04")
}

func notificationAssignmentIDs(notifications []*types.ExamAssignmentNotification) []string {
	seen := make(map[string]bool, len(notifications))
	assignmentIDs := make([]string, 0, len(notifications))
	for _, notification := range notifications {
		if notification == nil || notification.AssignmentID == "" || seen[notification.AssignmentID] {
			continue
		}
		seen[notification.AssignmentID] = true
		assignmentIDs = append(assignmentIDs, notification.AssignmentID)
	}
	return assignmentIDs
}

func assignmentNotificationItem(
	notification *types.ExamAssignmentNotification,
	assignment *types.ExamClassAssignment,
	attempt *types.ExamPracticeAttempt,
	now time.Time,
) *types.ExamAssignmentNotificationItem {
	item := &types.ExamAssignmentNotificationItem{Notification: notification}
	if assignment != nil {
		item.AssignmentStatus = assignment.Status
		item.AssignmentDueAt = assignment.DueAt
		item.CanStart = attempt == nil && assignment.Status == types.ExamAssignmentStatusPublished && !assignmentExpired(assignment, now)
	}
	if attempt != nil {
		item.LastAttemptID = attempt.ID
	}
	return item
}
