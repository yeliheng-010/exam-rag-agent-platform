import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const apiUrl = new URL('../../api/exam/intervention.ts', import.meta.url)
const componentUrl = new URL('./ClassPracticeRecommendations.vue', import.meta.url)
const classDetail = readFileSync(new URL('./ClassDetail.vue', import.meta.url), 'utf8')
const platformLayout = readFileSync(new URL('../platform/index.vue', import.meta.url), 'utf8')
const platformMenu = readFileSync(new URL('../../components/menu.vue', import.meta.url), 'utf8')
const examTypes = readFileSync(new URL('../../types/exam.ts', import.meta.url), 'utf8')
const agentEditor = readFileSync(new URL('../agent/AgentEditorModal.vue', import.meta.url), 'utf8')
const toolCapabilities = readFileSync(new URL('../../utils/tool-capabilities.ts', import.meta.url), 'utf8')

test('declares class and student intervention APIs', () => {
  assert.equal(existsSync(apiUrl), true)
  const api = readFileSync(apiUrl, 'utf8')
  assert.match(api, /getClassPracticeRecommendations/)
  assert.match(api, /classes\/\$\{classId\}\/practice-recommendations/)
  assert.match(api, /getStudentPracticeRecommendations/)
  assert.match(api, /practice\/recommendations/)
})

test('class analytics uses a dedicated recommendation dialog and publish prefill', () => {
  assert.equal(existsSync(componentUrl), true)
  const component = readFileSync(componentUrl, 'utf8')
  assert.match(component, /智能练习推荐/)
  assert.match(component, /getClassPracticeRecommendations/)
  assert.match(component, /emit\(['"]publish['"]/)
  assert.match(classDetail, /ClassPracticeRecommendations/)
  assert.match(classDetail, /prefillRecommendedAssignment/)
  assert.match(classDetail, /assignmentForm\.value\.group_id/)
})

test('recommendation dialog can explicitly include published groups for review', () => {
  const component = readFileSync(componentUrl, 'utf8')
  assert.match(component, /包含已发布题组/)
  assert.match(component, /include_assigned:\s*includeAssigned\.value/)
  assert.match(component, /createLatestRequestRunner/)
  assert.match(component, /result\.value = null/)
  assert.match(component, /outcome\.status === 'stale'/)
})

test('recommendation dialog has a viewport-safe mobile layout', () => {
  const component = readFileSync(componentUrl, 'utf8')
  assert.match(component, /min\(920px, calc\(100vw - 24px\)\)/)
  assert.match(component, /recommendation-mobile-list/)
  assert.match(component, /@media \(max-width: 720px\)/)
  assert.match(component, /max-height:\s*calc\(100vh - 300px\)/)
})

test('platform removes horizontal overflow and collapses navigation on narrow viewports', () => {
  assert.match(platformLayout, /@media \(max-width: 720px\)[\s\S]*?\.main[\s\S]*?min-width:\s*0/)
  assert.match(platformMenu, /const isNarrowViewport = ref/)
  assert.match(platformMenu, /effectiveSidebarCollapsed = computed/)
  assert.match(platformMenu, /'aside_box--collapsed': effectiveSidebarCollapsed/)
  assert.match(platformMenu, /window\.matchMedia\(MOBILE_LAYOUT_QUERY\)/)
})

test('assignment prefill dialog remains usable on narrow viewports', () => {
  assert.match(classDetail, /v-model:visible="assignmentVisible"[\s\S]*?width="min\(720px, calc\(100vw - 16px\)\)"/)
  assert.match(classDetail, /class="assignment-form"/)
  assert.match(classDetail, /class="assignment-group-control"/)
  assert.match(classDetail, /\.assignment-form\s*\{[\s\S]*?max-height:\s*calc\(100vh - 220px\)/)
  assert.match(classDetail, /\.assignment-group-control\s*\{[\s\S]*?flex-direction:\s*column/)
})

test('frontend types and Agent editor expose practice recommendation', () => {
  assert.match(examTypes, /ExamPracticeRecommendationResult/)
  assert.match(examTypes, /ExamPracticeRecommendationReason/)
  assert.match(agentEditor, /exam_practice_recommendation/)
  assert.match(toolCapabilities, /exam_practice_recommendation:\s*\{\s*consumesFiles:\s*false/)
})
