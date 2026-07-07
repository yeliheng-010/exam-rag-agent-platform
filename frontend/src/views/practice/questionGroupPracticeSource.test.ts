import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const router = readFileSync(new URL('../../router/index.ts', import.meta.url), 'utf8')
const learningHome = readFileSync(new URL('../learning/LearningHome.vue', import.meta.url), 'utf8')
const practiceApi = readFileSync(new URL('../../api/exam/practice.ts', import.meta.url), 'utf8')
const practiceViewUrl = new URL('./QuestionGroupPractice.vue', import.meta.url)
const practiceView = readFileSync(practiceViewUrl, 'utf8')

test('wires student question group practice route and learning entry', () => {
  assert.equal(existsSync(practiceViewUrl), true)
  assert.match(router, /name:\s*["']questionGroupPractice["']/)
  assert.match(router, /QuestionGroupPractice\.vue/)
  assert.doesNotMatch(router, /questionGroupPractice[\s\S]{0,180}minRole/)
  assert.match(learningHome, /listPracticeQuestionGroups/)
  assert.match(learningHome, /题组练习/)
  assert.doesNotMatch(learningHome, /canUseQuestionBanks[\s\S]{0,120}listPracticeQuestionGroups/)
})

test('declares practice API endpoints', () => {
  assert.match(practiceApi, /practice\/question-groups/)
  assert.match(practiceApi, /practice\/question-groups\/\$\{groupId\}\/attempts/)
  assert.match(practiceApi, /practice\/attempts\/\$\{attemptId\}\/answers/)
  assert.match(practiceApi, /practice\/attempts\/\$\{attemptId\}\/complete/)
})

test('practice page supports answer submission and AI explanation prefill', () => {
  assert.match(practiceView, /submitPracticeAnswer/)
  assert.match(practiceView, /completePracticeAttempt/)
  assert.match(practiceView, /setPrefillQuery/)
  assert.match(practiceView, /\/platform\/creatChat/)
  assert.match(practiceView, /listExamResources/)
})
