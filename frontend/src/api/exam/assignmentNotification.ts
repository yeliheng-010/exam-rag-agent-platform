import { get, post } from '@/utils/request'
import type {
  ApiResponse,
  ExamAssignmentNotificationList,
  SendExamAssignmentReminderRequest,
  SendExamAssignmentReminderResult,
} from '@/types/exam'

export function listAssignmentNotifications(limit = 50) {
  return get(`/api/v1/exam/notifications?limit=${limit}`) as unknown as Promise<ApiResponse<ExamAssignmentNotificationList>>
}

export function sendAssignmentReminders(
  classId: string,
  assignmentId: string,
  data: SendExamAssignmentReminderRequest,
) {
  return post(`/api/v1/exam/classes/${classId}/assignments/${assignmentId}/reminders`, data) as unknown as Promise<ApiResponse<SendExamAssignmentReminderResult>>
}

export function markAssignmentNotificationRead(notificationId: string) {
  return post(`/api/v1/exam/notifications/${notificationId}/read`, {}) as unknown as Promise<void>
}

export function markAllAssignmentNotificationsRead() {
  return post('/api/v1/exam/notifications/read-all', {}) as unknown as Promise<void>
}
