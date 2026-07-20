import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const apiUrl = new URL('../../api/exam/assignmentNotification.ts', import.meta.url)
const apiSource = existsSync(apiUrl) ? readFileSync(apiUrl, 'utf8') : ''
const typesSource = readFileSync(new URL('../../types/exam.ts', import.meta.url), 'utf8')

test('declares assignment notification frontend APIs', () => {
  assert.equal(existsSync(apiUrl), true)
  assert.match(apiSource, /listAssignmentNotifications/)
  assert.match(apiSource, /sendAssignmentReminders/)
  assert.match(apiSource, /markAssignmentNotificationRead/)
  assert.match(apiSource, /markAllAssignmentNotificationsRead/)
  assert.match(apiSource, /exam\/notifications\?limit=/)
  assert.match(apiSource, /exam\/notifications\/read-all/)
  assert.match(apiSource, /exam\/notifications\/\$\{notificationId\}\/read/)
  assert.match(apiSource, /assignments\/\$\{assignmentId\}\/reminders/)
})

test('declares notification, reminder and progress fields', () => {
  assert.match(typesSource, /ExamAssignmentNotificationKind/)
  assert.match(typesSource, /ExamAssignmentNotificationItem/)
  assert.match(typesSource, /ExamAssignmentNotificationList/)
  assert.match(typesSource, /SendExamAssignmentReminderResult/)
  assert.match(typesSource, /last_reminded_at\?:\s*string/)
  assert.match(typesSource, /can_remind:\s*boolean/)
})
