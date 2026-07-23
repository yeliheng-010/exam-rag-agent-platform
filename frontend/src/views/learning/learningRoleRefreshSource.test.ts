import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const learningHome = readFileSync(new URL('./LearningHome.vue', import.meta.url), 'utf8')
const loadData = learningHome.match(/const loadData = async \(\) => \{[\s\S]*?\n\}/)?.[0] || ''

test('learning center refreshes the tenant role before gated requests', () => {
  const refreshIndex = loadData.indexOf('await authStore.refreshFromAuthMe()')
  const roleCheckIndex = loadData.indexOf("authStore.hasRole('contributor')")

  assert.notEqual(refreshIndex, -1)
  assert.notEqual(roleCheckIndex, -1)
  assert.ok(refreshIndex < roleCheckIndex)
})

test('question bank failure cannot discard core learning data', () => {
  const coreRequests = loadData.match(/Promise\.all\(\[([\s\S]*?)\]\)/)?.[1] || ''

  assert.match(coreRequests, /listExamDomains\(\)/)
  assert.match(coreRequests, /ensurePersonalExamSpace\(\)/)
  assert.match(coreRequests, /listExamClasses\(\)/)
  assert.doesNotMatch(coreRequests, /listQuestionBanks\(\)/)
  assert.match(loadData, /classes\.value = classRes\.data \|\| \[\][\s\S]*listQuestionBanks\(\)/)
})
