import assert from 'node:assert/strict'
import { existsSync } from 'node:fs'
import test from 'node:test'
import type { ExamAssignmentSummary, ExamClassAssignment, ExamPracticeAttempt } from '../../types/exam'

const lifecycleUrl = new URL('./assignmentLifecycle.ts', import.meta.url)

const loadLifecycle = async () => {
  assert.equal(existsSync(lifecycleUrl), true, 'assignmentLifecycle.ts must exist')
  return import(lifecycleUrl.href)
}

const makeAssignment = (overrides: Partial<ExamClassAssignment> = {}): ExamClassAssignment => ({
  id: 'assignment-1',
  tenant_id: 10000,
  class_id: 'class-1',
  space_id: 'space-1',
  question_bank_id: 'bank-1',
  group_id: 'group-1',
  title: '练习',
  instructions: '',
  status: 'published',
  created_by_user_id: 'teacher-1',
  created_at: '2026-07-19T08:00:00Z',
  updated_at: '2026-07-19T08:00:00Z',
  ...overrides,
})

const makeSummary = (overrides: Partial<ExamAssignmentSummary> = {}): ExamAssignmentSummary => ({
  assignment: makeAssignment(),
  bank_name: '题库',
  question_count: 1,
  ...overrides,
})

test('derives lifecycle state and label without persisting expired status', async () => {
  const { assignmentLifecycleState, assignmentStatusLabel } = await loadLifecycle()
  const now = Date.parse('2026-07-20T08:00:00Z')

  assert.equal(assignmentLifecycleState(makeAssignment({ due_at: '2026-07-20T09:00:00Z' }), now), 'active')
  assert.equal(assignmentLifecycleState(makeAssignment({ due_at: '2026-07-20T08:00:00Z' }), now), 'expired')
  assert.equal(assignmentLifecycleState(makeAssignment({ status: 'withdrawn' }), now), 'withdrawn')
  assert.equal(assignmentLifecycleState(makeAssignment({ status: 'archived' }), now), 'archived')
  assert.equal(assignmentStatusLabel(makeAssignment({ due_at: '2026-07-20T08:00:00Z' }), now), '已截止')
})

test('exposes only lifecycle actions allowed by the current state', async () => {
  const { canEditAssignment, canWithdrawAssignment, canRepublishAssignment } = await loadLifecycle()
  const now = Date.parse('2026-07-20T08:00:00Z')
  const active = makeAssignment({ due_at: '2026-07-20T09:00:00Z' })
  const expired = makeAssignment({ due_at: '2026-07-20T08:00:00Z' })
  const withdrawn = makeAssignment({ status: 'withdrawn', due_at: '2026-07-20T09:00:00Z' })

  assert.equal(canEditAssignment(active, now), true)
  assert.equal(canWithdrawAssignment(active), true)
  assert.equal(canEditAssignment(expired, now), false)
  assert.equal(canWithdrawAssignment(expired), true)
  assert.equal(canEditAssignment(withdrawn, now), true)
  assert.equal(canRepublishAssignment(withdrawn, now), true)
  assert.equal(canRepublishAssignment({ ...withdrawn, due_at: '2026-07-20T08:00:00Z' }, now), false)
  assert.equal(canEditAssignment(makeAssignment({ status: 'archived' }), now), false)
})

test('continues the latest attempt instead of creating another one', async () => {
  const { canCreateAssignmentAttempt, existingAssignmentAttemptID } = await loadLifecycle()
  const attempt = { id: 'attempt-1', status: 'in_progress' } as ExamPracticeAttempt
  const item = makeSummary({ last_attempt: attempt })

  assert.equal(existingAssignmentAttemptID(item), 'attempt-1')
  assert.equal(canCreateAssignmentAttempt(item, Date.now()), false)
  assert.equal(canCreateAssignmentAttempt(makeSummary(), Date.now()), true)
  assert.equal(canCreateAssignmentAttempt(makeSummary({ assignment: makeAssignment({ status: 'withdrawn' }) }), Date.now()), false)
})
