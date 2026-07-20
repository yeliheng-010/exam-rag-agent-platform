import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const classApi = readFileSync(new URL('../../api/exam/class.ts', import.meta.url), 'utf8')
const examTypes = readFileSync(new URL('../../types/exam.ts', import.meta.url), 'utf8')
const settingsUrl = new URL('./classSettings.ts', import.meta.url)
const settingsPanelUrl = new URL('./ClassSettingsPanel.vue', import.meta.url)
const settingsPanelStyleUrl = new URL('./classSettingsPanel.less', import.meta.url)
const classList = readFileSync(new URL('./ClassList.vue', import.meta.url), 'utf8')
const classDetail = readFileSync(new URL('./ClassDetail.vue', import.meta.url), 'utf8')

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

test('class management list requests archived classes', () => {
  assert.match(classList, /listExamClasses\(\{\s*includeArchived:\s*true\s*\}\)/)
  assert.match(classList, /item\.status\s*===\s*['"]active['"]/)
})

test('class detail renders settings and disables archived business tabs', () => {
  assert.match(classDetail, /value="settings"/)
  assert.match(classDetail, /ClassSettingsPanel/)
  assert.match(classDetail, /handleClassSettingsUpdated/)
  assert.match(classDetail, /:disabled="isArchivedClass"/)
  assert.doesNotMatch(classDetail, /\{\s*value:\s*['"]settings['"]/)
})

test('settings panel supports owner commands and read-only roles', () => {
  assert.equal(existsSync(settingsPanelUrl), true)
  const panel = existsSync(settingsPanelUrl) ? readFileSync(settingsPanelUrl, 'utf8') : ''
  assert.match(panel, /canManageClassSettings/)
  assert.match(panel, /updateExamClass/)
  assert.match(panel, /archiveExamClass/)
  assert.match(panel, /restoreExamClass/)
  assert.match(panel, /DialogPlugin\.confirm/)
  assert.match(panel, /v-if="canManage"/)
  assert.match(panel, /v-else/)
  assert.match(panel, /submitting/)
  assert.match(panel, /formSnapshot/)
  assert.match(panel, /archiveDialogOpen/)
})

test('settings panel uses a single-column mobile layout', () => {
  const panel = [settingsPanelUrl, settingsPanelStyleUrl]
    .filter(url => existsSync(url))
    .map(url => readFileSync(url, 'utf8'))
    .join('\n')
  assert.match(panel, /@media\s*\(max-width:\s*600px\)/)
  assert.match(panel, /grid-template-columns:\s*1fr/)
  assert.match(panel, /flex-wrap:\s*wrap/)
})
