import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import test from 'node:test'

const router = readFileSync(new URL('../../router/index.ts', import.meta.url), 'utf8')
const learningHome = readFileSync(new URL('../learning/LearningHome.vue', import.meta.url), 'utf8')
const practiceApi = readFileSync(new URL('../../api/exam/practice.ts', import.meta.url), 'utf8')
const reviewViewUrl = new URL('./PracticeReviewHome.vue', import.meta.url)
const reviewView = readFileSync(reviewViewUrl, 'utf8')

test('wires practice review route and learning center entry', () => {
  assert.equal(existsSync(reviewViewUrl), true)
  assert.match(router, /name:\s*["']practiceReview["']/)
  assert.match(router, /PracticeReviewHome\.vue/)
  assert.doesNotMatch(router, /practiceReview[\s\S]{0,180}minRole/)
  assert.match(learningHome, /\/platform\/practice\/review/)
  assert.match(learningHome, /错题复盘/)
})

test('declares practice review APIs', () => {
  assert.match(practiceApi, /listPracticeAttempts/)
  assert.match(practiceApi, /getPracticeAttempt/)
  assert.match(practiceApi, /listWrongQuestions/)
  assert.match(practiceApi, /updatePracticeAnswerReview/)
  assert.match(practiceApi, /practice\/attempts/)
  assert.match(practiceApi, /practice\/wrong-questions/)
  assert.match(practiceApi, /practice\/answers\/\$\{answerId\}\/review/)
})

test('practice review page loads attempts and wrong questions', () => {
  assert.match(reviewView, /listPracticeAttempts/)
  assert.match(reviewView, /listWrongQuestions/)
  assert.match(reviewView, /wrongQuestions/)
  assert.match(reviewView, /练习记录/)
  assert.match(reviewView, /错题本/)
})

test('practice review page supports mastery state and AI explanation prefill', () => {
  assert.match(reviewView, /updatePracticeAnswerReview/)
  assert.match(reviewView, /review_status/)
  assert.match(reviewView, /review_note/)
  assert.match(reviewView, /mastered/)
  assert.match(reviewView, /askAiExplanation/)
  assert.match(reviewView, /useStartChat/)
})
