import assert from 'node:assert/strict'
import test from 'node:test'
import { memberDisplayId, memberDisplayName } from './memberDisplay.ts'

test('uses generated member display fields when available', () => {
  const member = {
    display_name: '测试学生',
    display_id: '202607100001',
    user_id: 'legacy-user-id',
  }

  assert.equal(memberDisplayName(member), '测试学生')
  assert.equal(memberDisplayId(member), '编号：202607100001')
})

test('keeps a unique legacy identifier when display fields are missing', () => {
  const member = { user_id: 'legacy-user-id' }

  assert.equal(memberDisplayName(member), '未命名学生')
  assert.equal(memberDisplayId(member), '编号：legacy-user-id')
})
