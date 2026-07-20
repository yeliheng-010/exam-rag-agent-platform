import type { ExamAssignmentNotificationItem, ExamAssignmentSummary } from '@/types/exam'

export type AssignmentNotificationTarget =
  | { kind: 'attempt'; groupId: string; attemptId: string }
  | { kind: 'assignment'; assignmentId: string }
  | { kind: 'blocked' }

export function notificationBadgeCount(unread: number, invitations: number) {
  return Math.min(99, Math.max(0, unread) + Math.max(0, invitations))
}

export function assignmentNotificationTarget(item: ExamAssignmentNotificationItem): AssignmentNotificationTarget {
  if (item.last_attempt_id) {
    return {
      kind: 'attempt',
      groupId: item.notification.group_id,
      attemptId: item.last_attempt_id,
    }
  }
  if (item.can_start) {
    return { kind: 'assignment', assignmentId: item.notification.assignment_id }
  }
  return { kind: 'blocked' }
}

export function assignmentWindowForNotification(
  assignments: ExamAssignmentSummary[],
  assignmentId: string,
  limit: number,
) {
  const visible = assignments.slice(0, limit)
  if (!assignmentId || visible.some(item => item.assignment.id === assignmentId)) return visible
  const target = assignments.find(item => item.assignment.id === assignmentId)
  if (!target || limit <= 0) return visible
  return [...visible.slice(0, limit - 1), target]
}
