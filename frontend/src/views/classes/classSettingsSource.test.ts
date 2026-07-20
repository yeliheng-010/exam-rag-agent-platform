import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const classApi = readFileSync(new URL('../../api/exam/class.ts', import.meta.url), 'utf8')
const examTypes = readFileSync(new URL('../../types/exam.ts', import.meta.url), 'utf8')
const settingsUrl = new URL('./classSettings.ts', import.meta.url)

test('declares class settings types and pure helpers', () => {
  assert.equal(existsSync(settingsUrl), true)
  const settings = existsSync(settingsUrl) ? readFileSync(settingsUrl, 'utf8') : ''
  assert.match(examTypes, /ExamClassStatus\s*=\s*['"]active['"]\s*\|\s*['"]archived['"]/)
  assert.match(examTypes, /UpdateExamClassPayload/)
  assert.match(examTypes, /status:\s*ExamClassStatus/)
  assert.match(settings, /canManageClassSettings/)
  assert.match(settings, /classSettingsCommand/)
  assert.match(settings, /buildClassSettingsPayload/)
})

test('declares list update archive and restore API contracts', () => {
  assert.match(classApi, /import \{ get, post, put \}/)
  assert.match(classApi, /listExamClasses/)
  assert.match(classApi, /include_archived/)
  assert.match(classApi, /updateExamClass/)
  assert.match(classApi, /archiveExamClass/)
  assert.match(classApi, /restoreExamClass/)
  assert.match(classApi, /put\(`\/api\/v1\/exam\/classes\/\$\{classId\}`/)
  assert.match(classApi, /post\(`\/api\/v1\/exam\/classes\/\$\{classId\}\/archive`/)
  assert.match(classApi, /post\(`\/api\/v1\/exam\/classes\/\$\{classId\}\/restore`/)
})
