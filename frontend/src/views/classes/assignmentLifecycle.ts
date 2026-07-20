import type { ExamAssignmentSummary, ExamClassAssignment } from '@/types/exam'

export type AssignmentLifecycleState = 'active' | 'expired' | 'withdrawn' | 'archived'

export function assignmentLifecycleState(
  assignment: ExamClassAssignment,
  nowMs = Date.now(),
): AssignmentLifecycleState {
  if (assignment.status === 'withdrawn') return 'withdrawn'
  if (assignment.status === 'archived') return 'archived'
  if (assignment.due_at && Date.parse(assignment.due_at) <= nowMs) return 'expired'
  return 'active'
}

export function assignmentStatusLabel(assignment: ExamClassAssignment, nowMs = Date.now()) {
  const state = assignmentLifecycleState(assignment, nowMs)
  return ({
    active: '进行中',
    expired: '已截止',
    withdrawn: '已撤回',
    archived: '已归档',
  } as const)[state]
}

export function canEditAssignment(assignment: ExamClassAssignment, nowMs = Date.now()) {
  const state = assignmentLifecycleState(assignment, nowMs)
  return state === 'active' || state === 'withdrawn'
}

export function canWithdrawAssignment(assignment: ExamClassAssignment) {
  return assignment.status === 'published'
}

export function canRepublishAssignment(assignment: ExamClassAssignment, nowMs = Date.now()) {
  if (assignment.status !== 'withdrawn') return false
  return !assignment.due_at || Date.parse(assignment.due_at) > nowMs
}

export function existingAssignmentAttemptID(item: ExamAssignmentSummary) {
  return item.last_attempt?.id || ''
}

export function canCreateAssignmentAttempt(item: ExamAssignmentSummary, nowMs = Date.now()) {
  return !existingAssignmentAttemptID(item)
    && assignmentLifecycleState(item.assignment, nowMs) === 'active'
}
