import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildClassSettingsPayload,
  canManageClassSettings,
  classSettingsCommand,
  isArchivedExamClass,
} from './classSettings.ts'

const activeClass = {
  owner_user_id: 'owner-1',
  status: 'active' as const,
}

const archivedClass = {
  owner_user_id: 'owner-1',
  status: 'archived' as const,
}

test('only the class owner can manage settings', () => {
  assert.equal(canManageClassSettings(activeClass, 'owner-1'), true)
  assert.equal(canManageClassSettings(activeClass, 'assistant-1'), false)
  assert.equal(canManageClassSettings(activeClass, 'student-1'), false)
  assert.equal(canManageClassSettings(undefined, 'owner-1'), false)
})

test('selects archive or restore command from class status', () => {
  assert.equal(classSettingsCommand(activeClass, 'owner-1'), 'archive')
  assert.equal(classSettingsCommand(archivedClass, 'owner-1'), 'restore')
  assert.equal(classSettingsCommand(archivedClass, 'assistant-1'), null)
  assert.equal(isArchivedExamClass(activeClass), false)
  assert.equal(isArchivedExamClass(archivedClass), true)
})

test('builds a normalized settings payload', () => {
  assert.deepEqual(
    buildClassSettingsPayload({
      name: '  高三一班  ',
      description: '  冲刺班  ',
      member_limit: 0,
    }),
    {
      name: '高三一班',
      description: '冲刺班',
      member_limit: 0,
    },
  )
})
