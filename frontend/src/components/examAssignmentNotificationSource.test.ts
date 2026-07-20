import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const bellUrl = new URL('./GlobalNotificationBell.vue', import.meta.url)
const dialogUrl = new URL('./ExamAssignmentNotificationDialog.vue', import.meta.url)
const oldBellUrl = new URL('./GlobalInvitationBell.vue', import.meta.url)
const platform = readFileSync(new URL('../views/platform/index.vue', import.meta.url), 'utf8')
const bell = existsSync(bellUrl) ? readFileSync(bellUrl, 'utf8') : ''
const dialog = existsSync(dialogUrl) ? readFileSync(dialogUrl, 'utf8') : ''

test('global notification bell polls safely and combines unread sources', () => {
  assert.equal(existsSync(bellUrl), true)
  assert.match(bell, /notificationBadgeCount/)
  assert.match(bell, /pendingInvitationCount/)
  assert.match(bell, /setInterval\(loadNotifications, 30_000\)/)
  assert.match(bell, /visibilitychange/)
  assert.match(bell, /document\.hidden/)
  assert.match(bell, /clearInterval/)
  assert.doesNotMatch(bell, /v-if="badgeCount > 0"/)
})

test('notification dialog supports reads, invitations and all click targets', () => {
  assert.equal(existsSync(dialogUrl), true)
  assert.match(dialog, /markAssignmentNotificationRead/)
  assert.match(dialog, /markAllAssignmentNotificationsRead/)
  assert.match(dialog, /MyInvitationsDialog/)
  assert.match(dialog, /assignmentNotificationTarget/)
  assert.match(dialog, /attempt_id=\$\{target\.attemptId\}/)
  assert.match(dialog, /assignment_id:\s*target\.assignmentId/)
  assert.match(dialog, /暂无通知/)
})

test('platform uses one responsive unified notification surface', () => {
  assert.match(platform, /GlobalNotificationBell/)
  assert.doesNotMatch(platform, /GlobalInvitationBell/)
  assert.equal(existsSync(oldBellUrl), false)
  assert.match(dialog, /width="560px"/)
  assert.match(dialog, /calc\(100vw - 24px\)/)
  assert.match(dialog, /overflow-wrap:\s*anywhere/)
})
