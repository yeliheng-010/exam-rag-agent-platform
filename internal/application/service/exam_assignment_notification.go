package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
)

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
	case types.ExamAssignmentNotificationKindRepublished:
		return "作业重新发布：" + assignment.Title, fmt.Sprintf("%s · 截止时间：%s", className, dueAt)
	case types.ExamAssignmentNotificationKindWithdrawn:
		return "作业已撤回：" + assignment.Title, className + " · 已有答题记录仍然保留"
	default:
		return "新作业：" + assignment.Title, fmt.Sprintf("%s · 截止时间：%s", className, dueAt)
	}
}

func assignmentNotificationDueAt(dueAt *time.Time) string {
	if dueAt == nil {
		return "无固定截止时间"
	}
	return dueAt.In(time.Local).Format("2006-01-02 15:04")
}
