import assert from 'node:assert/strict'
import { existsSync } from 'node:fs'
import test from 'node:test'
import type { ExamAssignmentNotificationItem } from '../../types/exam'

const notificationUrl = new URL('./assignmentNotification.ts', import.meta.url)

const loadNotification = async () => {
  assert.equal(existsSync(notificationUrl), true, 'assignmentNotification.ts must exist')
  return import(notificationUrl.href)
}

const makeItem = (overrides: Partial<ExamAssignmentNotificationItem> = {}): ExamAssignmentNotificationItem => ({
  notification: {
    id: 'notification-1',
    tenant_id: 10000,
    class_id: 'class-1',
    assignment_id: 'assignment-1',
    group_id: 'group-1',
    recipient_user_id: 'student-1',
    actor_user_id: 'teacher-1',
    kind: 'published',
    title: 'New assignment',
    content: 'Start now',
    created_at: '2026-07-20T08:00:00Z',
  },
  assignment_status: 'published',
  can_start: true,
  ...overrides,
})

test('combines assignment unread and invitations into a capped badge', async () => {
  const { notificationBadgeCount } = await loadNotification()
  assert.equal(notificationBadgeCount(5, 2), 7)
  assert.equal(notificationBadgeCount(90, 20), 99)
  assert.equal(notificationBadgeCount(-1, -2), 0)
})

test('opens the latest attempt before considering assignment state', async () => {
  const { assignmentNotificationTarget } = await loadNotification()
  assert.deepEqual(assignmentNotificationTarget(makeItem({
    last_attempt_id: 'attempt-1', assignment_status: 'withdrawn', can_start: false,
  })), {
    kind: 'attempt', groupId: 'group-1', attemptId: 'attempt-1',
  })
})

test('locates an open assignment and blocks a closed one without an attempt', async () => {
  const { assignmentNotificationTarget } = await loadNotification()
  assert.deepEqual(assignmentNotificationTarget(makeItem()), {
    kind: 'assignment', assignmentId: 'assignment-1',
  })
  assert.deepEqual(assignmentNotificationTarget(makeItem({ can_start: false })), { kind: 'blocked' })
})

test('keeps an older notification assignment inside the visible assignment window', async () => {
  const { assignmentWindowForNotification } = await loadNotification()
  const assignments = Array.from({ length: 10 }, (_, index) => ({
    assignment: { id: `assignment-${index + 1}` },
  }))

  const visible = assignmentWindowForNotification(assignments, 'assignment-10', 8)

  assert.equal(visible.length, 8)
  assert.deepEqual(
    visible.map((item: { assignment: { id: string } }) => item.assignment.id),
    ['assignment-1', 'assignment-2', 'assignment-3', 'assignment-4', 'assignment-5', 'assignment-6', 'assignment-7', 'assignment-10'],
  )
})
